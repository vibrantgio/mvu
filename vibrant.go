package vibrant

import (
	"image/color"
	"log"
	"unsafe"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"

	"github.com/reactivego/gio"
	"github.com/reactivego/x"
)

const kLogEvents = false

func Backdrops(fill color.Color) x.Observable[layout.Widget] {
	return x.Of(gio.Backdrop(fill))
}

func LinearGradients(stop1 f32.Point, color1 color.Color, stop2 f32.Point, color2 color.Color) x.Observable[layout.Widget] {
	return x.Of(gio.LinearGradient(stop1, color1, stop2, color2))
}

type MsgOp struct{ Msg any }

func (op MsgOp) Add(ops *op.Ops) {
	defer clip.Rect{}.Push(ops).Pop()
	pointer.InputOp{Tag: op}.Add(ops)
}

// Window handles the events of a single gioui app window.
type Window struct {
	*app.Window

	msgOps chan MsgOp
}

func NewWindow(options ...app.Option) *Window {
	return &Window{Window: app.NewWindow(options...), msgOps: make(chan MsgOp, 1)}
}

func (window *Window) MsgOps() x.Observable[MsgOp] {
	return x.Recv(window.msgOps)
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
					version     int
					data        []byte
					refs        []interface{}
					nextStateID int
					multipOp    bool
				}
				for _, op := range (*internalOps)(unsafe.Pointer(&ops.Internal)).refs {
					if msgop, matches := op.(MsgOp); matches {
						window.msgOps <- msgop
					}
				}
			}
		case err != nil:
			// log.Printf("error: %v\n", err)
		default:
			// log.Println("complete")
			if window.msgOps != nil {
				close(window.msgOps)
				window.msgOps = nil
			}
		}
	}
	return pairs.Subscribe(main, x.NewScheduler())
}
