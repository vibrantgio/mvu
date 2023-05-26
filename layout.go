package vibrant

import (
	"log"

	"gioui.org/app"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"github.com/reactivego/x"
)

const kLogEvents = false

func (window *Window) Layout(layers ...x.Observable[layout.Widget]) x.Subscription {
	// events
	events := x.FromChan(window.Events()).Filter(func(next event.Event) bool {
		if kLogEvents {
			log.Printf("event: %[1]T %[1]v\n", next)
		}
		return next != nil
	})

	// callops
	invalidate := func(widgets []layout.Widget) []layout.Widget {
		window.Invalidate()
		return widgets
	}
	empty := func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}
	for i := range layers {
		// Slow loading layers should not block the event loop.
		layers[i] = layers[i].StartWith(empty)
	}
	widgets := x.Map(x.Combine(layers...), invalidate).SubscribeOn(x.Goroutine)

	pairs := x.WithLatestFromPair(events, widgets)
	var ops op.Ops
	main := func(next x.Pair[event.Event, []layout.Widget], err error, done bool) {
		switch {
		case !done:
			// fmt.Printf("event:%v\nhandlers:%v\n", event, window)
			for _, handler := range window.handlers {
				handler.Next(next.First)
			}

			switch event := next.First.(type) {
			case app.ConfigEvent:
				// log.Printf("config: %v\n", event.Config)
			case app.ViewEvent:
				// log.Printf("view: %v\n", event)
			case system.StageEvent:
				// log.Printf("stage: %v\n", event.Stage)
			case key.FocusEvent:
				// log.Printf("focus: %v\n", event.Focus)
			case system.FrameEvent:
				gtx := layout.NewContext(&ops, event)
				for _, widget := range next.Second {
					widget(gtx)
				}
				event.Frame(gtx.Ops)
			case system.DestroyEvent:
				if event.Err != nil {
					log.Printf("destroy: %v\n", event.Err)
				}
			case pointer.Event:
				// log.Printf("pointer: %v\n", event)
			default:
				// log.Printf("event: %#v\n", event)
			}
		case err != nil:
			log.Printf("error: %v\n", err)
		default:
			log.Println("complete")
			for _, handler := range window.handlers {
				handler.Done()
			}
			window.handlers = nil
		}
	}
	return pairs.Subscribe(main)
}
