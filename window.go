package mvu

import (
	"context"
	"log"
	"slices"
	"sync"
	"sync/atomic"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/reactivego/rx"
)

// Window handles the events of a single gioui app Window.
type Window struct {
	window     *app.Window
	messageOps chan MessageOp
	viewEvents chan app.ViewEvent

	configMu    sync.Mutex
	onConfigure []func()
	configured  bool // the first FrameEvent has been delivered
}

// viewEventBuffer is the capacity of the per-window view-event channel. View
// events come in attach/detach pairs — one valid event when the native view
// joins a window, one invalid event when it leaves — so four holds two full
// cycles, more than accumulate in practice before a subscriber attaches.
// [Window.forwardViewEvent] documents what happens when it fills anyway.
const viewEventBuffer = 4

func NewWindow(options ...app.Option) *Window {
	w := new(app.Window)
	w.Option(options...)
	return &Window{
		window:     w,
		messageOps: make(chan MessageOp, 1),
		viewEvents: make(chan app.ViewEvent, viewEventBuffer),
	}
}

// Window returns the underlying Gio window. The raw handle stays reachable —
// platform adapters need it to find and adjust the native window — but
// options applied directly to it, as in w.Window().Option(...), bypass the
// [Window.OnConfigure] notification: nothing registered gets a chance to
// re-assert its native-window adjustments, so they can silently disappear.
// Apply post-construction options through [Window.Option] instead.
func (w *Window) Window() *app.Window {
	return w.window
}

// Option applies options to the underlying window and then notifies every
// func registered with [Window.OnConfigure], in registration order. Route
// every post-construction option change through this method rather than
// through the raw [Window.Window] handle: applying options makes Gio rebuild
// the native window's configuration, which can silently undo adjustments made
// directly to the native window, and the notification is how registrants get
// to re-assert them.
func (w *Window) Option(options ...app.Option) {
	w.window.Option(options...)
	w.notifyConfigure()
}

// OnConfigure registers f to run after the window's configuration may have
// changed. The notification carries no information about what changed — its
// meaning is "the native window's configuration may have been rebuilt;
// re-assert whatever you asserted directly against the native window".
//
// f runs once after the window delivers its first frame — covering the
// options passed to [NewWindow] and Gio's own initial configuration, both of
// which happen before any caller could have registered — and again after
// every [Window.Option] call. If the first frame has already been delivered
// when OnConfigure is called, f additionally runs once immediately, so a late
// registrant never misses the initial configuration. Options applied through
// the raw [Window.Window] handle do not notify.
//
// Any number of funcs may be registered; each notification runs them all in
// registration order, and there is no unregistration — a registrant lives as
// long as the window. f must be idempotent: it re-asserts state rather than
// toggling it, and it may run when there is nothing to do — including before
// the native window exists, when [Window.Option] is called ahead of the
// event loop. f is invoked from whatever goroutine triggered the
// notification — the render goroutine for the first frame, the caller of
// [Window.Option] otherwise — so it must not assume any particular thread; a
// registrant that touches a platform API with thread affinity dispatches to
// that thread itself.
func (w *Window) OnConfigure(f func()) {
	w.configMu.Lock()
	w.onConfigure = append(w.onConfigure, f)
	configured := w.configured
	w.configMu.Unlock()
	if configured {
		f()
	}
}

// notifyConfigure runs every registered func in registration order. The
// registrant slice is snapshotted under the mutex and the funcs run outside
// it, so a registrant may itself call OnConfigure or Option without
// deadlocking.
func (w *Window) notifyConfigure() {
	w.configMu.Lock()
	registered := slices.Clone(w.onConfigure)
	w.configMu.Unlock()
	for _, f := range registered {
		f()
	}
}

// notifyFirstFrame delivers the one-time first-frame notification. Gio's
// first Configure of the native window happens before any caller could have
// registered, so Render calls this from its FrameEvent arm; only the first
// call notifies.
func (w *Window) notifyFirstFrame() {
	w.configMu.Lock()
	if w.configured {
		w.configMu.Unlock()
		return
	}
	w.configured = true
	registered := slices.Clone(w.onConfigure)
	w.configMu.Unlock()
	for _, f := range registered {
		f()
	}
}

func (w *Window) Messages() rx.Observable[Message] {
	return rx.Map(rx.Recv(w.messageOps), func(msgOp MessageOp) Message { return msgOp.Message })
}

// ViewEvents returns the window's platform view events — the [app.ViewEvent]
// values (app.AppKitViewEvent on macOS, app.X11ViewEvent on X11, and so on)
// through which Gio hands out the native window handles. Platform adapters
// subscribe here to augment the native view — installing a drop target, for
// example — and everything else can ignore this method entirely. It is
// deliberately narrow: view events are the one Gio event class mvu forwards
// beyond its own two, and no general unhandled-events stream exists or will.
//
// The stream is fed by [Window.Render] and backed by a buffered per-window
// channel in the same idiom as [Window.Messages]: subscribe it once — two
// subscriptions would compete for the same channel — and expect completion
// when the window is destroyed.
//
// The first view event is delivered before the window's first frame — before
// any ConfigEvent or FrameEvent, and therefore almost certainly before the
// application subscribes. It is not lost: events sent before (or between)
// subscriptions sit in the channel's buffer, so a late subscriber receives
// everything still buffered, the initial attach event included. Only if more
// than four view events accumulate with no subscriber draining them does the
// buffer overflow, and then the oldest buffered event is evicted for the new
// one (see [Window.forwardViewEvent] for why keep-latest is the only safe
// eviction). Subscribing before [Window.Render] starts — the natural order in
// an application, since Render blocks its goroutine — makes overflow
// unreachable.
//
// An invalid event (Valid() == false) is a real event, not an error: the
// native view left its window, and every handle taken from the previous event
// is dead. Subscribers must drop their references when it arrives.
func (w *Window) ViewEvents() rx.Observable[app.ViewEvent] {
	return rx.Recv(w.viewEvents)
}

// forwardViewEvent hands a view event from Render's event loop to the
// ViewEvents channel without ever blocking the render goroutine — a
// subscriber may not exist, and stalling the event loop on one would hang
// every application that never calls ViewEvents.
//
// The policy on a full buffer is drop-OLDEST, never drop-newest: Gio retains
// the handles in a view event only until the next view event is sent, so once
// a newer event exists the buffered older ones describe dead handles. Keeping
// the latest event means a subscriber always ends on the current state of the
// native view; dropping the newest instead would preserve a stale handle —
// the use-after-free shape — and lose the live one. Eviction is only reachable
// when nothing is subscribed (see ViewEvents), so evicting cannot race a
// delivery that already happened.
func (w *Window) forwardViewEvent(e app.ViewEvent) {
	for {
		select {
		case w.viewEvents <- e:
			return
		default:
		}
		select {
		case <-w.viewEvents: // full: evict the oldest, then retry the send
		default:
		}
	}
}

// Render drives the Gio event loop. The returned subscription terminates
// when the window emits an app.DestroyEvent.
//
// Gio's frame protocol is synchronous: after delivering a FrameEvent, the
// OS side selects between receiving the rendered frame and sending its flush
// event, and if the flush is delivered before Frame is called the next Frame
// deadlocks. Events must therefore be read and Frame called on the same
// goroutine. Layer state reaches that goroutine as an atomic snapshot.
func (w *Window) Render(layers ...rx.Observable[layout.Widget]) rx.Subscription {
	blank := func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}
	initial := make([]layout.Widget, len(layers))
	for i := range layers {
		layers[i] = layers[i].StartWith(blank)
		initial[i] = blank
	}

	var current atomic.Pointer[[]layout.Widget]
	current.Store(&initial)

	var layersSub rx.Subscription
	if len(layers) > 0 {
		layersSub = rx.CombineLatest(layers...).Subscribe(rx.GoroutineContext(), func(next []layout.Widget, err error, done bool) {
			if !done && next != nil {
				cp := make([]layout.Widget, len(next))
				copy(cp, next)
				current.Store(&cp)
				w.window.Invalidate()
			}
		})
	}

	loop := rx.Observable[struct{}](func(observe rx.Observer[struct{}], scheduler rx.Scheduler, subscriber rx.Subscriber) {
		ops := new(op.Ops)
		scheduler.ScheduleRecursive(func(again func()) {
			if !subscriber.Subscribed() {
				return
			}
			e := w.window.Event()
			if kLogEvents {
				log.Printf("event: %[1]T %[1]v\n", e)
			}
			switch e := e.(type) {
			case app.DestroyEvent:
				if layersSub != nil {
					layersSub.Unsubscribe()
				}
				if w.messageOps != nil {
					close(w.messageOps)
					w.messageOps = nil
				}
				if w.viewEvents != nil {
					close(w.viewEvents)
					w.viewEvents = nil
				}
				observe(struct{}{}, e.Err, true)
				return
			case app.FrameEvent:
				w.notifyFirstFrame()
				gtx := app.NewContext(ops, e)
				var frameMessages []MessageOp
				registerCollector(ops, &frameMessages)
				snapshot := *current.Load()
				for _, layer := range snapshot {
					layer(gtx)
				}
				unregisterCollector(ops)
				e.Frame(gtx.Ops)
				for _, msgOp := range frameMessages {
					w.messageOps <- msgOp
				}
			case app.ViewEvent:
				// app.ViewEvent is an interface, so this arm matches every
				// platform's concrete type. It is the one event class
				// forwarded beyond the two above; all other events are
				// dropped.
				w.forwardViewEvent(e)
			}
			again()
		})
	})

	return loop.Subscribe(context.Background(), func(struct{}, error, bool) {})
}
