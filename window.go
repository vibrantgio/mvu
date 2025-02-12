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

// Window handles the events of a single gioui app window.
type Window struct {
	*app.Window

	messageOps chan MessageOp
}

func NewWindow(options ...app.Option) *Window {
	return &Window{Window: app.NewWindow(options...), messageOps: make(chan MessageOp, 1)}
}

func (window *Window) MessageOps() x.Observable[MessageOp] {
	return x.Recv(window.messageOps)
}

func (window *Window) Layout(layers ...x.Observable[layout.Widget]) x.Subscription {
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
	main := func(next x.Pair[event.Event, []layout.Widget], err error, done bool) {
		switch {
		case !done:
			if frame, ok := next.First.(system.FrameEvent); ok {
				gtx := layout.NewContext(&ops, frame)
				for _, widget := range next.Second {
					widget(gtx)
				}
				frame.Frame(gtx.Ops)
				type internalOps struct {
					version int
					data    []byte
					refs    []interface{}
				}
				for _, op := range (*internalOps)(unsafe.Pointer(&ops.Internal)).refs {
					if mo, matches := op.(MessageOp); matches {
						window.messageOps <- mo
					}
				}
			}
		case err != nil:
			// log.Printf("error: %v\n", err)
		default:
			// log.Println("complete")
			if window.messageOps != nil {
				close(window.messageOps)
				window.messageOps = nil
			}
		}
	}
	return pairs.Subscribe(main, x.NewScheduler())
}
