package mvu

import (
	"log"
	"unsafe"

	"gioui.org/app"
	"gioui.org/io/event"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/reactivego/x"
)

// Viewer handles the events of a single gioui app window.
type Viewer interface {
	Messages() x.Observable[any]
	View(layers ...x.Observable[layout.Widget]) x.Subscription
}

func NewViewer(options ...app.Option) Viewer {
	return &view{options: options, messageOps: make(chan MessageOp, 1)}
}

type view struct {
	options    []app.Option
	messageOps chan MessageOp
}

func (r *view) Messages() x.Observable[any] {
	f := func(msgOp MessageOp) any { return msgOp.Message }
	return x.Map(x.Recv(r.messageOps), f)
}

func (r *view) View(layers ...x.Observable[layout.Widget]) x.Subscription {
	window := app.NewWindow(r.options...)

	events := x.Recv(window.Events()).Filter(func(next event.Event) bool {
		if kLogEvents {
			log.Printf("event: %[1]T %[1]v\n", next)
		}
		return next != nil
	})

	// Slow loading layers should not block the event loop.
	blank := func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}
	for i := range layers {
		layers[i] = layers[i].StartWith(blank)
	}

	// Whenever the layers change, invalidate the window.
	invalidate := func(layers []layout.Widget) []layout.Widget {
		window.Invalidate()
		return layers
	}

	pairs := x.WithLatestFromPair(events, x.Map(x.Combine(layers...), invalidate).SubscribeOn(x.Goroutine))

	var ops op.Ops
	observer := func(next x.Pair[event.Event, []layout.Widget], err error, done bool) {
		switch {
		case !done:
			if frame, ok := next.First.(system.FrameEvent); ok {
				gtx := layout.NewContext(&ops, frame)
				for _, widget := range next.Second {
					widget(gtx)
				}
				frame.Frame(gtx.Ops)

				type internalOps struct {
					version int // int gioui v0.8 this has become a uint32
					data    []byte
					refs    []interface{}
				}
				for _, op := range (*internalOps)(unsafe.Pointer(&ops.Internal)).refs {
					if msgOp, matches := op.(MessageOp); matches {
						r.messageOps <- msgOp
					}
				}
			}
		case err != nil:
			// log.Printf("error: %v\n", err)
		default:
			// log.Println("complete")
			if r.messageOps != nil {
				close(r.messageOps)
				r.messageOps = nil
			}
		}
	}
	return pairs.Subscribe(observer, x.NewScheduler())
}
