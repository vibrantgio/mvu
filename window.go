package mvu

import (
	"context"
	"log"
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
}

func NewWindow(options ...app.Option) *Window {
	w := new(app.Window)
	w.Option(options...)
	return &Window{window: w, messageOps: make(chan MessageOp, 1)}
}

func (w *Window) Window() *app.Window {
	return w.window
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
