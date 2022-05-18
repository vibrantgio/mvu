package vibrant

import (
	"gioui.org/io/event"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/reactivego/x"
)

type Context = layout.Context

func (window *Window) Layout() x.Observable[Context] {
	return func(observe x.Observer[Context], scheduler x.Scheduler, subscriber x.Subscriber) {
		ops := new(op.Ops)
		observer := func(next event.Event, err error, done bool) {
			switch {
			case !done:
				if frame, ok := next.(system.FrameEvent); ok {
					if subscriber.Subscribed() {
						observe(layout.NewContext(ops, frame), nil, false)
					}
				}
			case err != nil:
				var zero Context
				observe(zero, err, true)
			default:
				var zero Context
				observe(zero, nil, true)
			}
		}
		handler := &EventHandler{observer}
		window.Append(handler)
		subscriber.OnUnsubscribe(func() { window.Delete(handler) })
	}
}
