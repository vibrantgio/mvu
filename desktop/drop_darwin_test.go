//go:build darwin

package desktop

import (
	"image"
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
