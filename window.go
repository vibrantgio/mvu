package vibrant

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"gioui.org/app"
	"gioui.org/io/event"
	"gioui.org/io/system"
	"gioui.org/op"

	rx "github.com/reactivego/observable"
)

const kLogEvents = false

// Window handles the events of a single gioui app window.
type Window struct {
	sync.Mutex
	*app.Window
	handlers []Handler
}

func NewWindow(options ...app.Option) *Window {
	return &Window{
		Window: app.NewWindow(options...),
	}
}

func (window *Window) Append(handler Handler) {
	window.Lock()
	defer window.Unlock()
	window.handlers = append(window.handlers, handler)
}

func (window *Window) Delete(handler Handler) {
	window.Lock()
	defer window.Unlock()
	for i, h := range window.handlers {
		if h == handler {
			copy(window.handlers[i:], window.handlers[i+1:])
			window.handlers[len(window.handlers)-1] = nil
			window.handlers = window.handlers[:len(window.handlers)-1]
			break
		}
	}
}

func (window *Window) Handle(event event.Event) {
	// fmt.Printf("event:%v\nhandlers:%v\n", event, window)
	if event != nil {
		for _, handler := range window.handlers {
			handler.Handle(event)
		}
	} else {
		for _, handler := range window.handlers {
			handler.Complete()
		}
		window.handlers = nil
	}
}

func (window *Window) Render(launchscreen op.CallOp, layers ...rx.Observable[op.CallOp]) rx.Subscription {
	events := rx.FromChan(window.Events()).Filter(func(next event.Event) bool {
		if kLogEvents {
			log.Printf("event: %[1]T %[1]v\n", next)
		}
		return next != nil
	})
	invalidate := func(callops []op.CallOp) []op.CallOp {
		window.Invalidate()
		return callops
	}
	if len(layers) == 0 {
		layers = append(layers, rx.Of(launchscreen))
	} else {
		for i := range layers {
			layers[i] = layers[i].StartWith(launchscreen)
			launchscreen = op.CallOp{}
		}
	}
	callops := rx.Map(rx.Combine(layers...), invalidate).SubscribeOn(rx.Goroutine)
	pairs := rx.WithLatestFromPair(events, callops)
	var ops op.Ops
	main := func(next rx.Pair[event.Event, []op.CallOp], err error, done bool) {
		ops.Reset()
		switch {
		case !done:
			window.Handle(next.First)
			switch event := next.First.(type) {
			case system.FrameEvent:
				for _, callop := range next.Second {
					callop.Add(&ops)
				}
				event.Frame(&ops)
			case system.DestroyEvent:
				if event.Err != nil {
					log.Printf("error: %v\n", event.Err)
				}
			}
		case err != nil:
			log.Printf("error: %v\n", err)
		default:
			window.Handle(nil)
		}
	}
	return pairs.Subscribe(main)
}

func (w *Window) String() string {
	sb := new(strings.Builder)
	sb.WriteString("Window{\n")
	for _, h := range w.handlers {
		fmt.Fprintf(sb, "  (%[1]T) %[1]v\n", h)
	}
	sb.WriteString("}")
	return sb.String()
}
