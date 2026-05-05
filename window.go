package mvu

import (
	"log"
	"unsafe"

	"gioui.org/app"
	"gioui.org/io/event"
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

// windowEvents wraps window.Event() (blocking) in a channel for rx.Recv.
func windowEvents(win *app.Window) <-chan event.Event {
	ch := make(chan event.Event)
	go func() {
		for {
			e := win.Event()
			ch <- e
			if _, ok := e.(app.DestroyEvent); ok {
				close(ch)
				return
			}
		}
	}()
	return ch
}

func (w *Window) Render(layers ...rx.Observable[layout.Widget]) rx.Subscription {
	events := rx.Recv(windowEvents(w.window)).Filter(func(next event.Event) bool {
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
		w.window.Invalidate()
		return layers
	}

	pairs := rx.WithLatestFrom2(events, rx.Map(rx.CombineLatest(layers...), invalidate).SubscribeOn(rx.Goroutine))

	ops := new(op.Ops)
	observer := func(next rx.Tuple2[event.Event, []layout.Widget], err error, done bool) {
		switch {
		case !done:
			if frameEvent, ok := next.First.(app.FrameEvent); ok {
				gtx := app.NewContext(ops, frameEvent)
				for _, widget := range next.Second {
					widget(gtx)
				}
				frameEvent.Frame(gtx.Ops)

				// Scan ops refs for MessageOps emitted during layout.
				// version is uint32 in gioui.org/internal/ops since v0.8+.
				type unsafeOps struct {
					version uint32
					data    []byte
					refs    []any
				}
				for _, op := range (*unsafeOps)(unsafe.Pointer(&ops.Internal)).refs {
					if msgOp, matches := op.(MessageOp); matches {
						w.messageOps <- msgOp
					}
				}
			}
		case err != nil:
			// log.Printf("error: %v\n", err)
		default:
			// log.Println("complete")
			if w.messageOps != nil {
				close(w.messageOps)
				w.messageOps = nil
			}
		}
	}
	return pairs.Subscribe(observer, rx.NewScheduler())
}
