package mvu

import (
	"testing"

	"gioui.org/app"
)

// The tests below exercise the Option/OnConfigure seam directly, the way
// message_test.go exercises the collector seam: Window.Render's FrameEvent arm
// calls notifyFirstFrame, so testing that method tests the first-frame hook
// without driving a real OS window. app.Window.Option is safe headless — with
// no driver it only records the options — so a real NewWindow backs every
// test.

// TestOptionNotifiesRegistrantsInOrder asserts that Window.Option forwards and
// then runs every registered func, in registration order, once per call — and
// that registration alone does not fire before the first frame.
func TestOptionNotifiesRegistrantsInOrder(t *testing.T) {
	w := NewWindow(app.Title("seam"))

	var order []string
	w.OnConfigure(func() { order = append(order, "first") })
	w.OnConfigure(func() { order = append(order, "second") })

	if len(order) != 0 {
		t.Fatalf("registration before the first frame fired %v; want no calls", order)
	}

	w.Option(app.Title("changed"))

	want := []string{"first", "second"}
	if len(order) != len(want) || order[0] != want[0] || order[1] != want[1] {
		t.Fatalf("after one Option call, notifications = %v; want %v", order, want)
	}

	w.Option(app.Title("changed again"))

	if len(order) != 4 {
		t.Fatalf("after two Option calls, %d notifications; want 4 (every call notifies)", len(order))
	}
}

// TestFirstFrameNotifiesExactlyOnce asserts the first-frame hook that Render's
// FrameEvent arm invokes: registrants fire once on the first frame — covering
// construction-time options applied before anyone could register — and never
// again from that path, while later Option calls still notify.
func TestFirstFrameNotifiesExactlyOnce(t *testing.T) {
	w := NewWindow(app.Title("seam"))

	calls := 0
	w.OnConfigure(func() { calls++ })

	w.notifyFirstFrame()
	if calls != 1 {
		t.Fatalf("first frame fired %d notifications; want 1", calls)
	}

	w.notifyFirstFrame() // every subsequent frame is a no-op on this path
	w.notifyFirstFrame()
	if calls != 1 {
		t.Fatalf("repeated frames fired %d notifications; want still 1 (first frame only)", calls)
	}

	w.Option(app.Title("changed"))
	if calls != 2 {
		t.Fatalf("Option after first frame fired %d notifications; want 2", calls)
	}
}

// TestLateRegistrationFiresImmediately asserts that a func registered after
// the first frame has already been delivered runs once at registration, so a
// late registrant never misses the initial configuration.
func TestLateRegistrationFiresImmediately(t *testing.T) {
	w := NewWindow(app.Title("seam"))
	w.notifyFirstFrame()

	calls := 0
	w.OnConfigure(func() { calls++ })
	if calls != 1 {
		t.Fatalf("late registration fired %d notifications; want 1 immediate call", calls)
	}

	w.Option(app.Title("changed"))
	if calls != 2 {
		t.Fatalf("Option after late registration fired %d notifications; want 2", calls)
	}
}

// TestRawHandleBypassesNotification pins the documented meaning of Window():
// the raw handle stays reachable, but options applied through it do not
// notify — the exact bypass the doc comment warns about.
func TestRawHandleBypassesNotification(t *testing.T) {
	w := NewWindow(app.Title("seam"))

	calls := 0
	w.OnConfigure(func() { calls++ })

	raw := w.Window()
	if raw == nil {
		t.Fatal("Window() returned nil; the raw handle must stay reachable")
	}
	raw.Option(app.Title("changed behind the seam's back"))

	if calls != 0 {
		t.Fatalf("raw-handle Option fired %d notifications; want 0 (documented bypass)", calls)
	}
}

// TestRegistrantMayRegisterDuringNotification asserts a registrant can itself
// call OnConfigure while being notified — the registrant snapshot is taken
// before any func runs, so the notification neither deadlocks nor mutates the
// slice it is iterating.
func TestRegistrantMayRegisterDuringNotification(t *testing.T) {
	w := NewWindow(app.Title("seam"))

	lateCalls := 0
	registered := false
	w.OnConfigure(func() {
		if !registered {
			registered = true
			w.OnConfigure(func() { lateCalls++ })
		}
	})

	w.Option(app.Title("changed")) // must not deadlock; new registrant not in this round
	if lateCalls != 0 {
		t.Fatalf("registrant added during notification ran %d times in the same round; want 0", lateCalls)
	}

	w.Option(app.Title("changed again"))
	if lateCalls != 1 {
		t.Fatalf("registrant added during a previous notification ran %d times; want 1", lateCalls)
	}
}
