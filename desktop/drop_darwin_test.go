//go:build darwin

package desktop

import (
	"image"
	"slices"
	"testing"

	"gioui.org/app"

	"github.com/vibrantgio/mvu"
)

// The tests below exercise the registration bookkeeping — the per-view
// routing map and its lifecycle — without a native window, using fake view
// identities the way mvu's own view-event tests do. adopt and release are
// the state half of the view-event handling, split from the AppKit half
// (class augmentation and registerForDraggedTypes) precisely so this can be
// tested headless; the AppKit half needs a live window and is verified by
// the launch checks and the manual drag script.

// bareTarget builds a DropTarget without the constructor's view-event
// subscription, so a test controls the lifecycle by hand. The tracker
// goroutine is not started; raw is drained directly.
func bareTarget() *DropTarget {
	return &DropTarget{
		w:     mvu.NewWindow(app.Title("routing test")),
		zones: &ZoneGroup{},
		raw:   make(chan dragEvent, rawBuffer),
	}
}

// rawEvents drains and returns whatever sits in the target's raw channel.
func rawEvents(d *DropTarget) []dragEvent {
	var evs []dragEvent
	for {
		select {
		case ev := <-d.raw:
			evs = append(evs, ev)
		default:
			return evs
		}
	}
}

// hasTarget reports whether the routing map holds an entry for view.
func hasTarget(view uintptr) bool {
	viewMu.Lock()
	defer viewMu.Unlock()
	return viewTargets[view] != nil
}

// TestDispatchRoutesPerView pins the multi-window contract at the routing
// layer: two targets with distinct views each receive exactly their own
// view's events — no cross-talk — and an unknown view's events deliver
// nowhere, silently.
func TestDispatchRoutesPerView(t *testing.T) {
	a, b := bareTarget(), bareTarget()
	const viewA, viewB, viewUnknown = 0xA10, 0xB20, 0xC30
	a.adopt(viewA)
	b.adopt(viewB)
	defer a.release()
	defer b.release()

	dispatchRaw(viewA, dragEvent{kind: dragEnter, pos: image.Pt(1, 1)})
	dispatchRaw(viewB, dragEvent{kind: dragDrop, pos: image.Pt(2, 2), paths: []string{"/tmp/b"}})
	dispatchRaw(viewUnknown, dragEvent{kind: dragDrop, pos: image.Pt(3, 3), paths: []string{"/tmp/lost"}})

	evsA, evsB := rawEvents(a), rawEvents(b)
	if len(evsA) != 1 || evsA[0].kind != dragEnter {
		t.Fatalf("target A received %#v, want its one enter event", evsA)
	}
	if len(evsB) != 1 || evsB[0].kind != dragDrop || evsB[0].paths[0] != "/tmp/b" {
		t.Fatalf("target B received %#v, want its one drop event", evsB)
	}
}

// TestAdoptIsIdempotentAndSupersedes pins re-registration: the same view
// adopted again (every valid view event re-registers) leaves exactly one
// live entry, and a NEW view for the same target supersedes the old entry so
// nothing leaks.
func TestAdoptIsIdempotentAndSupersedes(t *testing.T) {
	d := bareTarget()
	const viewOld, viewNew = 0xD40, 0xD50

	d.adopt(viewOld)
	d.adopt(viewOld) // the repeated attach of the same view
	if !hasTarget(viewOld) {
		t.Fatal("repeated adopt lost the entry")
	}

	d.adopt(viewNew) // the view was replaced under the same window
	if hasTarget(viewOld) {
		t.Fatal("adopting a new view leaked the old entry")
	}
	if !hasTarget(viewNew) {
		t.Fatal("adopting a new view did not store it")
	}

	d.release()
	if hasTarget(viewNew) {
		t.Fatal("release leaked the entry")
	}
}

// TestReleaseStopsDeliveryAndLeaksNothing pins the detach contract: after
// the invalid view event's release, the view's events deliver nowhere, a
// second release is harmless, and the map holds nothing for the view.
func TestReleaseStopsDeliveryAndLeaksNothing(t *testing.T) {
	d := bareTarget()
	const view = 0xE60
	d.adopt(view)
	d.release()
	d.release() // idempotent

	dispatchRaw(view, dragEvent{kind: dragDrop, pos: image.Pt(1, 1), paths: []string{"/tmp/late"}})
	if evs := rawEvents(d); len(evs) != 0 {
		t.Fatalf("released target still received %#v", evs)
	}
	if hasTarget(view) {
		t.Fatal("released view still in the routing map")
	}
}

// TestHandleViewEventLifecycle drives the real view-event handler with the
// platform's concrete event type — valid adopts, invalid releases — pinning
// the seam between mvu's forwarded events and the registration state. The
// AppKit half is excluded by using a target whose window has no driver: the
// register call routes through Window.Run, which without a driver would run
// AppKit code on this goroutine, so the test stops at the state half via
// adopt/release directly through handleViewEvent's own switch. A valid event
// must therefore not be sent here; only the invalid one is safe headless.
func TestHandleViewEventLifecycle(t *testing.T) {
	d := bareTarget()
	const view = 0xF70
	d.adopt(view) // stands in for the valid event's state half

	// The invalid event — the detach — must run the release path.
	d.handleViewEvent(app.AppKitViewEvent{})
	if hasTarget(view) {
		t.Fatal("invalid view event did not release the view")
	}
}

// The tests below exercise the promised-file half of a drop without AppKit:
// the native side opens a drop with the plain URLs it carried and how many
// files its promises owe, then reports each file as its reader writes it, and
// the drop becomes one event when the last one is in. That counting and
// assembly is Go's, so a test drives it directly — see drop_darwin.m for why the receiving itself cannot
// be driven from a test, and drop_window_darwin_test.go for the real drops on
// a real window.

// beginPromisedDropAfterExit opens a promised drop and takes the exit the drop
// itself delivers, leaving the raw channel holding only what follows.
func beginPromisedDropAfterExit(t *testing.T, d *DropTarget, view uintptr, pos image.Point, plain []string, promised int) uint64 {
	t.Helper()
	drop := beginPromisedDrop(view, pos, plain, promised)
	evs := rawEvents(d)
	if len(evs) != 1 || evs[0].kind != dragExit {
		t.Fatalf("a drop taken for its promises delivered %v, want the hover's exit", evs)
	}
	return drop
}

// promisedDropCount reports how many drops are still being received.
func promisedDropCount() int {
	promiseMu.Lock()
	defer promiseMu.Unlock()
	return len(promiseDrops)
}

// TestPromisedDropDeliversOnceTheLastFileIsIn pins the shape of a promise
// drag: nothing crosses while files are still being written, and the drop then
// arrives as a single event carrying every received file in the order its
// promise stood on the pasteboard.
func TestPromisedDropDeliversOnceTheLastFileIsIn(t *testing.T) {
	d := bareTarget()
	const view = 0xD10
	d.adopt(view)
	defer d.release()

	drop := beginPromisedDropAfterExit(t, d, view, image.Pt(7, 9), nil, 2)
	addPromisedFile(drop, "/tmp/mvu-desktop/one/first.m4a")
	if evs := rawEvents(d); len(evs) != 0 {
		t.Fatalf("the drop crossed with one file still promised: %v", evs)
	}
	addPromisedFile(drop, "/tmp/mvu-desktop/one/second.m4a")

	evs := rawEvents(d)
	if len(evs) != 1 {
		t.Fatalf("got %d events, want the one drop: %v", len(evs), evs)
	}
	if evs[0].kind != dragDrop {
		t.Fatalf("got kind %v, want dragDrop", evs[0].kind)
	}
	if evs[0].pos != image.Pt(7, 9) {
		t.Fatalf("got pos %v, want (7,9)", evs[0].pos)
	}
	want := []string{"/tmp/mvu-desktop/one/first.m4a", "/tmp/mvu-desktop/one/second.m4a"}
	if !slices.Equal(evs[0].paths, want) {
		t.Fatalf("got paths %v, want %v", evs[0].paths, want)
	}
	if promisedDropCount() != 0 {
		t.Fatal("the delivered drop is still being received")
	}
}

// TestPromisedDropCarriesPlainURLsFirst pins the mixed drag — a source
// dragging one file that exists and one it promises — as one message, the
// plain URL ahead of the received file.
func TestPromisedDropCarriesPlainURLsFirst(t *testing.T) {
	d := bareTarget()
	const view = 0xD20
	d.adopt(view)
	defer d.release()

	drop := beginPromisedDropAfterExit(t, d, view, image.Pt(1, 2), []string{"/Users/someone/exists.txt"}, 1)
	addPromisedFile(drop, "/tmp/mvu-desktop/two/promised.m4a")

	evs := rawEvents(d)
	if len(evs) != 1 {
		t.Fatalf("got %d events, want one drop carrying both: %v", len(evs), evs)
	}
	want := []string{"/Users/someone/exists.txt", "/tmp/mvu-desktop/two/promised.m4a"}
	if !slices.Equal(evs[0].paths, want) {
		t.Fatalf("got paths %v, want %v", evs[0].paths, want)
	}
}

// TestFailedPromiseDeliversWhatArrived pins the failure contract: a promise
// the source does not fulfil reports no path, and costs its own file only.
func TestFailedPromiseDeliversWhatArrived(t *testing.T) {
	d := bareTarget()
	const view = 0xD30
	d.adopt(view)
	defer d.release()

	drop := beginPromisedDropAfterExit(t, d, view, image.Pt(0, 0), []string{"/Users/someone/exists.txt"}, 2)
	addPromisedFile(drop, "/tmp/mvu-desktop/three/arrived.m4a")
	addPromisedFile(drop, "") // the promise the source did not fulfil

	evs := rawEvents(d)
	if len(evs) != 1 {
		t.Fatalf("got %d events, want the drop with what arrived: %v", len(evs), evs)
	}
	want := []string{"/Users/someone/exists.txt", "/tmp/mvu-desktop/three/arrived.m4a"}
	if !slices.Equal(evs[0].paths, want) {
		t.Fatalf("got paths %v, want %v", evs[0].paths, want)
	}
}

// TestEveryPromiseFailedDeliversNothing pins the one silence: a drop whose
// every promise failed carries no file, and a message with no paths would say
// a drop happened and hand the application nothing.
func TestEveryPromiseFailedDeliversNothing(t *testing.T) {
	d := bareTarget()
	const view = 0xD40
	d.adopt(view)
	defer d.release()

	addPromisedFile(beginPromisedDropAfterExit(t, d, view, image.Pt(0, 0), nil, 1), "")

	if evs := rawEvents(d); len(evs) != 0 {
		t.Fatalf("a drop with no file crossed anyway: %v", evs)
	}
	if promisedDropCount() != 0 {
		t.Fatal("the empty drop is still being received")
	}
}

// TestPromisedFilesAfterTheDropAreIgnored pins that a reader arriving late
// cannot deliver a second drop or panic: the drop is gone, so what it reports
// goes nowhere.
func TestPromisedFilesAfterTheDropAreIgnored(t *testing.T) {
	d := bareTarget()
	const view = 0xD50
	d.adopt(view)
	defer d.release()

	drop := beginPromisedDropAfterExit(t, d, view, image.Pt(0, 0), nil, 1)
	addPromisedFile(drop, "/tmp/mvu-desktop/four/only.m4a")
	addPromisedFile(drop, "/tmp/mvu-desktop/four/late.m4a") // after the drop landed

	if evs := rawEvents(d); len(evs) != 1 {
		t.Fatalf("got %d events, want one: %v", len(evs), evs)
	}
}

// TestPromisedDropEndsTheHoverAtOnce pins that the drag is over when the drop
// is taken, not when its promised files arrive: the hover's exit crosses
// immediately, so a highlighted zone clears while the files are still being
// written.
func TestPromisedDropEndsTheHoverAtOnce(t *testing.T) {
	d := bareTarget()
	const view = 0xD60
	d.adopt(view)
	defer d.release()

	drop := beginPromisedDrop(view, image.Pt(3, 4), nil, 1)

	evs := rawEvents(d)
	if len(evs) != 1 {
		t.Fatalf("got %d events, want the hover's exit alone: %v", len(evs), evs)
	}
	if evs[0].kind != dragExit {
		t.Fatalf("got kind %v, want dragExit", evs[0].kind)
	}

	addPromisedFile(drop, "/tmp/mvu-desktop/five/arrived.m4a")
	evs = rawEvents(d)
	if len(evs) != 1 || evs[0].kind != dragDrop {
		t.Fatalf("got %v, want the drop once its file was in", evs)
	}
}
