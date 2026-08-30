package mvu

import (
	"slices"
	"sync"
	"testing"
	"time"

	"gioui.org/app"

	"github.com/reactivego/rx"
)

// The tests below exercise the Option/OnConfigure seam directly: Render's
// FrameEvent arm calls notifyFirstFrame, so testing that method tests the
// first-frame hook without driving a real OS window. app.Window.Option is safe
// headless — with no driver it only records the options — so a real NewWindow
// backs every test.

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

// TestRawHandleBypassesNotification pins the meaning of Window(): the raw
// handle stays reachable, but options applied through it do not notify.
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

// The tests below exercise the ViewEvents seam: Render's app.ViewEvent arm
// calls forwardViewEvent, so driving that method drives the delivery contract
// without a real OS window. The events fed in come from makeViewEvent in the
// per-platform viewevent_*_test.go files — app.ViewEvent's unexported method
// admits only gioui.org/app's own types, and each platform's build compiles
// only its own concrete type. The seam itself is platform-agnostic.

// collectViewEvents subscribes w.ViewEvents() and returns a snapshot func.
func collectViewEvents(t *testing.T, w *Window) (func() []app.ViewEvent, rx.Subscription) {
	t.Helper()
	var mu sync.Mutex
	var seen []app.ViewEvent
	sub := w.ViewEvents().Subscribe(rx.GoroutineContext(), func(next app.ViewEvent, err error, done bool) {
		if !done {
			mu.Lock()
			seen = append(seen, next)
			mu.Unlock()
		}
	})
	snapshot := func() []app.ViewEvent {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(seen)
	}
	return snapshot, sub
}

// awaitViewEvents polls snapshot until cond holds or the timeout expires.
func awaitViewEvents(t *testing.T, snapshot func() []app.ViewEvent, cond func([]app.ViewEvent) bool) []app.ViewEvent {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if seen := snapshot(); cond(seen) {
			return seen
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("condition not met before timeout; view events seen: %v", snapshot())
	return nil
}

// TestViewEventsBuffersInitialEventForLateSubscriber pins the crux of the
// delivery contract: the platform delivers the first view event before the
// first frame — before application code has typically subscribed — and it
// must not be lost. The buffered channel is the replay: an event forwarded
// with no subscriber waits in the buffer and reaches the subscriber that
// attaches later.
func TestViewEventsBuffersInitialEventForLateSubscriber(t *testing.T) {
	w := NewWindow(app.Title("seam"))
	initial := makeViewEvent(0x1)

	w.forwardViewEvent(initial) // Render's arm fires before anyone subscribes

	snapshot, sub := collectViewEvents(t, w)
	defer sub.Unsubscribe()

	seen := awaitViewEvents(t, snapshot, func(seen []app.ViewEvent) bool { return len(seen) > 0 })
	if seen[0] != initial {
		t.Fatalf("late subscriber received %v; want the buffered initial event %v", seen[0], initial)
	}
}

// TestViewEventsKeepsLatestOnOverflow pins the eviction policy: when the
// buffer is full and nothing is subscribed, the OLDEST event is dropped for
// the new one — never the reverse. Gio retains a view event's handles only
// until the next view event, so under overflow the stale events are the
// expendable ones and the latest must survive; drop-newest would hand a
// subscriber a dead handle and lose the live one.
func TestViewEventsKeepsLatestOnOverflow(t *testing.T) {
	w := NewWindow(app.Title("seam"))
	capacity := cap(w.viewEvents)
	total := capacity + 3
	for i := 1; i <= total; i++ {
		w.forwardViewEvent(makeViewEvent(uintptr(i)))
	}

	snapshot, sub := collectViewEvents(t, w)
	defer sub.Unsubscribe()

	seen := awaitViewEvents(t, snapshot, func(seen []app.ViewEvent) bool { return len(seen) >= capacity })
	first, lastEv := viewIDOf(seen[0]), viewIDOf(seen[len(seen)-1])
	if want := uintptr(total - capacity + 1); first != want {
		t.Fatalf("oldest surviving event has id %#x; want %#x (evict-oldest)", first, want)
	}
	if lastEv != uintptr(total) {
		t.Fatalf("latest event has id %#x; want %#x (the newest event must never be dropped)", lastEv, uintptr(total))
	}
}

// TestViewEventsDeliversInOrderWhileSubscribed asserts plain in-order flow
// when a subscriber is attached before events arrive — the ordinary case once
// an application subscribes ahead of starting Render.
func TestViewEventsDeliversInOrderWhileSubscribed(t *testing.T) {
	w := NewWindow(app.Title("seam"))
	snapshot, sub := collectViewEvents(t, w)
	defer sub.Unsubscribe()

	attach := makeViewEvent(0x1)
	detach := invalidViewEvent() // the invalid event is a real event
	w.forwardViewEvent(attach)
	w.forwardViewEvent(detach)

	seen := awaitViewEvents(t, snapshot, func(seen []app.ViewEvent) bool { return len(seen) >= 2 })
	if got := seen[0]; got != attach || !got.Valid() {
		t.Fatalf("first event = %v (Valid()=%t); want the valid attach event", got, got.Valid())
	}
	if got := seen[1]; got != detach || got.Valid() {
		t.Fatalf("second event = %v (Valid()=%t); want the invalid detach event", got, got.Valid())
	}
}
