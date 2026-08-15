package desktop

import (
	"log"
	"os"
	"sync"

	"gioui.org/app"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/mvu"
)

// rawBuffer is the capacity of the per-target raw-event channel between the
// native drag callbacks and the tracker goroutine. The callbacks run on the
// platform's UI thread and must never block, so the send is non-blocking;
// 64 is far more than a human hover produces between pipeline reads, and
// [DropTarget.sendRaw] documents what happens when it fills anyway.
const rawBuffer = 64

// msgBuffer is the capacity of the per-target message channel between the
// tracker goroutine and the [DropTarget.Messages] subscriber. The tracker
// owns its goroutine and may block here briefly; the buffer only smooths
// bursts. An application that merges Messages into its loop drains it.
const msgBuffer = 16

// DropTarget makes one window accept file drags from the OS, delivering
// them as messages: [FilesEntered] and [FilesExited] while a drag hovers,
// [FilesDropped] when it lands, each resolved against the window's
// [ZoneGroup]. Construct it with [NewDropTarget] before the window starts
// rendering and merge [DropTarget.Messages] into the application's message
// stream; everything else — native registration on view attach, teardown on
// detach, re-registration when the view returns — is automatic.
//
// A DropTarget serves exactly one window and lives as long as that window;
// there is no detach method, and its message stream completes when the
// window is destroyed. Windows never share a DropTarget: each window gets
// its own, with its own ZoneGroup, and drops are delivered only to the
// target of the window they landed in.
type DropTarget struct {
	w     *mvu.Window
	zones *ZoneGroup
	kinds []string

	raw  chan dragEvent   // native callbacks -> tracker; send never blocks
	msgs chan mvu.Message // tracker -> Messages subscriber

	closeOnce sync.Once

	mu   sync.Mutex
	view uintptr // the native view currently registered; 0 while detached
}

// NewDropTarget registers w as a drop target for the given payload kinds and
// returns the target delivering its messages. Passing no kinds registers
// [FileURLs]; passing any kind this package does not implement panics, so an
// unsupported kind fails at construction rather than as silently refused
// drags. zones is the window's zone registry and must not be nil — an
// application that wants window-level drops without distinct targets records
// a single zone covering the window.
//
// Call NewDropTarget once per window, before the window's event loop starts
// rendering. The target subscribes the window's ViewEvents stream, which the
// mvu window documents as single-subscriber: constructing a DropTarget
// claims that stream, and the application must not subscribe it as well.
//
// On platforms without drop support the constructor succeeds and the target
// is inert: it delivers no messages and touches nothing native.
func NewDropTarget(w *mvu.Window, zones *ZoneGroup, kinds ...string) *DropTarget {
	if zones == nil {
		panic("desktop: NewDropTarget requires a ZoneGroup (record one zone covering the window for window-level drops)")
	}
	if len(kinds) == 0 {
		kinds = []string{FileURLs}
	}
	for _, k := range kinds {
		if k != FileURLs {
			panic("desktop: unsupported drop kind " + k + " (desktop.FileURLs is the one kind registered today)")
		}
	}
	d := &DropTarget{
		w:     w,
		zones: zones,
		kinds: kinds,
		raw:   make(chan dragEvent, rawBuffer),
		msgs:  make(chan mvu.Message, msgBuffer),
	}
	go d.track()
	// The view-event subscription is the registration lifecycle: a valid
	// event means the native view joined a window (register the drop target
	// on it — registration is per view instance, so every attach registers
	// again), an invalid one means it left (drop every native reference),
	// and completion means the window was destroyed (end the pipeline).
	w.ViewEvents().Subscribe(rx.GoroutineContext(), func(e app.ViewEvent, err error, done bool) {
		if done {
			d.close()
			return
		}
		d.handleViewEvent(e)
	})
	return d
}

// Messages returns the target's message stream: [FilesEntered],
// [FilesExited] and [FilesDropped] values, ready to merge into the
// application's message stream beside the window's own. Subscribe it once —
// it is backed by a per-target channel, and two subscriptions would compete
// for the same messages. The stream completes when the window is destroyed.
func (d *DropTarget) Messages() rx.Observable[mvu.Message] {
	return rx.Recv(d.msgs)
}

// track is the drop pipeline: one goroutine per target that drains the raw
// channel, runs the hover tracker against the zone registry, and forwards
// the resolved messages. It exits when the raw channel closes and completes
// the message stream on the way out.
func (d *DropTarget) track() {
	tr := newHoverTracker(d.zones.Hit)
	for ev := range d.raw {
		msgs := tr.step(ev)
		if ev.kind == dragDrop && len(msgs) == 0 {
			debugf("desktop: drop of %d path(s) at %v outside every zone — silence by design", len(ev.paths), ev.pos)
		}
		for _, m := range msgs {
			d.msgs <- m
		}
	}
	close(d.msgs)
}

// sendRaw hands a raw event from a native drag callback to the tracker
// without ever blocking the calling thread — the callbacks run on the
// platform's UI thread, where blocking freezes the compositor mid-drag.
//
// The policy on a full buffer is evict-oldest, never drop-newest: buffered
// hover events are superseded by every later one, and the newest event may
// be the drop itself — the one event that must not be lost while stale moves
// occupy the buffer. Eviction is only reachable when the tracker has stalled
// for dozens of events, which no draining application produces.
func (d *DropTarget) sendRaw(ev dragEvent) {
	for {
		select {
		case d.raw <- ev:
			return
		default:
		}
		select {
		case <-d.raw: // full: evict the oldest, then retry the send
		default:
		}
	}
}

// close ends the pipeline exactly once: the window is destroyed, so the
// native references are released and the raw channel closes, which lets the
// tracker drain and complete the message stream. The platform side
// guarantees no native callback can reach sendRaw after release returns.
func (d *DropTarget) close() {
	d.closeOnce.Do(func() {
		d.release()
		close(d.raw)
	})
}

// debugEnabled reports whether VGIO_DROP_DEBUG asks for drop-path logging.
// Read lazily so an application can set the variable early in main, before
// any window exists.
var debugEnabled = sync.OnceValue(func() bool {
	return os.Getenv("VGIO_DROP_DEBUG") != ""
})

// debugf logs the drop path's lifecycle — registration, teardown, and
// deliberate silences — when VGIO_DROP_DEBUG is set. Off by default: a
// library logs nothing unasked.
func debugf(format string, args ...any) {
	if debugEnabled() {
		log.Printf(format, args...)
	}
}
