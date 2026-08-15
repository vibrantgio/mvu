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

	configMu    sync.Mutex
	onConfigure []func()
	configured  bool // the first FrameEvent has been delivered
}

func NewWindow(options ...app.Option) *Window {
	w := new(app.Window)
	w.Option(options...)
	return &Window{window: w, messageOps: make(chan MessageOp, 1)}
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

// Render drives the Gio event loop. The returned subscription terminates
// when the window emits an `app.DestroyEvent`.
//
// Gio has enforced a synchronous protocol since v0.9: after delivering a `FrameEvent`,
// the OS-side `deliverEvent` enters a select that can either receive the
// rendered frame on `e.frames` or send `theFlushEvent` on `e.events`. Whoever
// completes first wins, and if the flush is delivered before `Frame()` is
// called, `deliverEvent` returns and the next `Frame()` deadlocks. The fix is
// to read events and call `Frame()` on the *same* goroutine. Layer state is
// updated concurrently via an atomic snapshot.
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
			}
			again()
		})
	})

	return loop.Subscribe(context.Background(), func(struct{}, error, bool) {})
}
