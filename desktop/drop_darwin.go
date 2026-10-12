//go:build darwin

package desktop

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AppKit

#include "drop_darwin.h"
*/
import "C"

import (
	"image"
	"strings"
	"sync"
	"unsafe"

	"gioui.org/app"
)

// addDropMethodsOnce guards the class augmentation. Gio's windows all share
// one view class, so adding the drag selectors to it is per-class and
// therefore process-global and permanent: exactly once per process, however
// many windows and drop targets exist. Per-view registration is the separate,
// repeated half — see `DropTarget.register`.
var addDropMethodsOnce sync.Once

// viewTargets maps each registered native view pointer to the DropTarget
// serving its window. It must be package-level state: the drag callbacks
// re-enter Go from Objective-C carrying nothing but the view's `self`
// pointer, so no instance, closure or argument can carry the owning target
// across, and a pointer-keyed registry behind a mutex is the only bridge. It
// is not an event bus and nothing subscribes to it: entries are owned by the
// window lifecycle —
// stored when a valid view event attaches a view, deleted when the invalid
// event detaches it — so the map holds exactly the live views and leaks
// nothing. dispatchRaw holds viewMu across lookup AND delivery so that
// release can guarantee no callback still touches a target after the entry
// is gone.
var (
	viewMu      sync.Mutex
	viewTargets = map[uintptr]*DropTarget{}
)

// handleViewEvent is the darwin half of the registration lifecycle. A valid
// view event announces the native view joined a window: adopt it (state) and
// register it (AppKit). Registration is per view instance, so EVERY valid
// event registers again — a view that leaves and rejoins a window arrives
// here as a fresh attach and gets a fresh registration; the class
// augmentation inside register stays once-per-process. The invalid event
// announces the view left its window: it is a real event, not an error, and
// every native reference taken from the previous event is dead — release
// drops them all.
func (d *DropTarget) handleViewEvent(e app.ViewEvent) {
	ake, ok := e.(app.AppKitViewEvent)
	if !ok {
		return
	}
	if ake.Valid() {
		d.adopt(ake.View)
		d.register(ake.View)
	} else {
		d.release()
	}
}

// adopt records view as the target's current native view and routes its
// callbacks here. Pure bookkeeping — no AppKit — so it is testable without a
// window. Idempotent for a repeated attach of the same view; an attach of a
// different view supersedes the old entry so nothing leaks.
func (d *DropTarget) adopt(view uintptr) {
	d.mu.Lock()
	old := d.view
	d.view = view
	d.mu.Unlock()

	viewMu.Lock()
	if old != 0 && old != view {
		delete(viewTargets, old)
	}
	viewTargets[view] = d
	viewMu.Unlock()
}

// release drops every reference to the target's native view: the view left
// its window (or the window is being destroyed), and a retained stale view
// pointer is a use-after-free waiting for a window close. Deliberately no
// AppKit call is made — not even an unregister — because touching the view
// on its way out is exactly the hazard. Holding viewMu for the delete
// guarantees that once release returns, no drag callback can reach this
// target again.
func (d *DropTarget) release() {
	d.mu.Lock()
	view := d.view
	d.view = 0
	d.mu.Unlock()
	if view == 0 {
		return
	}
	viewMu.Lock()
	if viewTargets[view] == d {
		delete(viewTargets, view)
	}
	viewMu.Unlock()
	debugf("desktop: drop target released view %#x", view)
}

// `register` performs the AppKit half on the main thread, through the
// window's Run — the only sanctioned door to the native event-loop thread;
// calling AppKit from any other goroutine appears to work and then fails
// intermittently. The class augmentation runs once per process; the
// per-instance registration runs for every view, every time it attaches.
func (d *DropTarget) register(view uintptr) {
	d.w.Window().Run(func() {
		addDropMethodsOnce.Do(func() {
			n := C.vgio_drop_add_methods(C.uintptr_t(view))
			debugf("desktop: drop selectors added to the shared view class (%d added)", int(n))
		})
		C.vgio_drop_register(C.uintptr_t(view))
		debugf("desktop: drop target registered for view %#x", view)
	})
}

// dispatchRaw routes one raw event from a drag callback to the target owning
// the view. An unknown view — detached moments ago, or never registered —
// delivers nowhere, silently: the window it belonged to is gone or leaving.
// The lock is held across the delivery; sendRaw never blocks, so the AppKit
// main thread is never meaningfully held up.
func dispatchRaw(view uintptr, ev dragEvent) {
	viewMu.Lock()
	defer viewMu.Unlock()
	if d := viewTargets[view]; d != nil {
		d.sendRaw(ev)
	}
}

// vgioDropUpdate is called from draggingEntered:/draggingUpdated:/
// draggingExited: on the AppKit main thread. kind is the dragKind ordinal;
// x/y/viewHeight are view coordinates in points and scale the backing scale
// factor re-read for this very event — the pure half of the transform runs
// here, Go-side, where its tests live. For an exit the coordinates are
// meaningless and ignored by the tracker.
//
//export vgioDropUpdate
func vgioDropUpdate(view C.uintptr_t, kind C.int, x, y, viewHeight, scale C.double) {
	dispatchRaw(uintptr(view), dragEvent{
		kind: dragKind(kind),
		pos:  gioPoint(float64(x), float64(y), float64(viewHeight), float64(scale)),
	})
}

// vgioDropPaths is called from performDragOperation: on the AppKit main
// thread for a drop whose files all exist already. paths is a buffer of count
// NUL-terminated UTF-8 paths laid end to end ('\0' being the one byte a POSIX
// path cannot contain); the coordinate components are as in vgioDropUpdate.
//
//export vgioDropPaths
func vgioDropPaths(view C.uintptr_t, paths *C.char, length, count C.int, x, y, viewHeight, scale C.double) {
	dispatchRaw(uintptr(view), dragEvent{
		kind:  dragDrop,
		pos:   gioPoint(float64(x), float64(y), float64(viewHeight), float64(scale)),
		paths: dropPaths(paths, length, count),
	})
}

// dropPaths unpacks the native path buffer: count NUL-terminated UTF-8 paths
// laid end to end. An empty buffer is no paths at all, not one empty path,
// which is the case a drop of promises alone arrives in.
func dropPaths(paths *C.char, length, count C.int) []string {
	if count <= 0 || length <= 0 {
		return nil
	}
	raw := C.GoBytes(unsafe.Pointer(paths), length)
	split := strings.Split(strings.TrimSuffix(string(raw), "\x00"), "\x00")
	if len(split) != int(count) {
		debugf("desktop: drop path count mismatch: got %d, expected %d", len(split), int(count))
	}
	return split
}

// vgioDropPromiseBegin is called from performDragOperation: on the AppKit main
// thread for a drop that carries file promises, and returns the id the drop's
// files are reported under. promised is how many files the receivers promise
// between them; the plain paths and the coordinate components are as in
// vgioDropPaths.
//
//export vgioDropPromiseBegin
func vgioDropPromiseBegin(view C.uintptr_t, paths *C.char, length, count, promised C.int, x, y, viewHeight, scale C.double) C.uint64_t {
	return C.uint64_t(beginPromisedDrop(uintptr(view),
		gioPoint(float64(x), float64(y), float64(viewHeight), float64(scale)),
		dropPaths(paths, length, count), int(promised)))
}

// vgioDropPromiseFile is called from a promise reader, on the queue the
// package hands the receivers rather than the main thread. Exactly one of the
// two strings is set: path for a file that was written, failure for one that
// was not. Reaching Go from that queue is safe — the view registry is behind a
// mutex and the hand-off to the window's stream never blocks — so a promised
// drop joins the stream the same way an ordinary drop does.
//
//export vgioDropPromiseFile
func vgioDropPromiseFile(drop C.uint64_t, path, failure *C.char) {
	if path != nil {
		addPromisedFile(uint64(drop), C.GoString(path))
		return
	}
	reason := ""
	if failure != nil {
		reason = C.GoString(failure)
	}
	debugf("desktop: a promised file did not arrive: %s", reason)
	addPromisedFile(uint64(drop), "")
}

// A drop that carries promises completes after the drag is over: the receivers
// write their files on the package's queue, one file per reader call, and the
// drop becomes one FilesDropped only once the last reader has run. The table
// below holds the drops still being received, keyed by the id the native side
// passes back with every file. It is package-level for the same reason the view
// registry is: the reader calls re-enter Go carrying nothing but that id.
//
// Nothing here waits for a window. A window destroyed mid-receiving takes its
// entry out of the view registry, so the assembled drop is routed nowhere and
// the entry is gone either way — the table holds only drops still arriving.
var (
	promiseMu    sync.Mutex
	promiseLast  uint64
	promiseDrops = map[uint64]*promisedDrop{}
)

// promisedDrop is one drop being received: the plain URLs it also carried, the
// files received so far, and how many readers are still to run. received grows
// in the order the files were written, which is an order rather than a race
// because the package's queue runs one reader at a time.
type promisedDrop struct {
	view      uintptr
	pos       image.Point
	plain     []string
	received  []string
	remaining int
}

// beginPromisedDrop opens a drop whose promised files are being received and
// returns its id. It also ends the hover then and there: the drag is over the
// moment the drop is taken, while the drop's own event follows only once the
// files are written — so a zone left to be un-highlighted by the delivery
// would stay highlighted for as long as writing them takes, with nothing
// dragging over it.
func beginPromisedDrop(view uintptr, pos image.Point, plain []string, promised int) uint64 {
	promiseMu.Lock()
	promiseLast++
	drop := promiseLast
	promiseDrops[drop] = &promisedDrop{
		view: view, pos: pos, plain: plain, remaining: promised,
	}
	promiseMu.Unlock()

	dispatchRaw(view, dragEvent{kind: dragExit})
	return drop
}

// addPromisedFile records one promised file against its drop — an empty path
// being a promise the source did not fulfil — and delivers the drop once the
// last reader has run: the plain URLs in the source's order followed by the
// received files in the order they were written.
//
// A failed promise costs its own file and nothing else, because a source that
// promises five files and writes four has still handed the application four.
// A drop that received nothing at all is the one silence: a message with no
// paths would say a drop happened and hand the application nothing.
func addPromisedFile(drop uint64, path string) {
	promiseMu.Lock()
	d := promiseDrops[drop]
	if d == nil {
		promiseMu.Unlock()
		debugf("desktop: a promised file arrived for drop %d, which is not being received", drop)
		return
	}
	if path != "" {
		d.received = append(d.received, path)
	}
	d.remaining--
	if d.remaining > 0 {
		promiseMu.Unlock()
		return
	}
	delete(promiseDrops, drop)
	promiseMu.Unlock()

	paths := make([]string, 0, len(d.plain)+len(d.received))
	paths = append(paths, d.plain...)
	paths = append(paths, d.received...)
	if len(paths) == 0 {
		debugf("desktop: a drop of promises alone delivered no file")
		return
	}
	dispatchRaw(d.view, dragEvent{kind: dragDrop, pos: d.pos, paths: paths})
}
