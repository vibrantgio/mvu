package vibrant

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"gioui.org/app"
	"gioui.org/io/event"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"github.com/reactivego/gio"
	"github.com/reactivego/x"
)

type Handle func(layout.Context)

type Done func()

type Handler interface {
	Handle(layout.Context)
	Done()
}

func NewHandler(handle Handle, done Done) Handler {
	return &handler{handle, done}
}

type handler struct {
	handle Handle
	done   Done
}

func (h handler) Handle(gtx layout.Context) {
	h.handle(gtx)
}

func (h handler) Done() {
	h.done()
}

type EventQueue []event.Event

func (q EventQueue) Events(k event.Tag) []event.Event {
	if n := len(q); n > 0 {
		return q[:n:n]
	}
	return nil
}

// Window handles the events of a single gioui app window.
type Window struct {
	*app.Window

	sync.Mutex
	handlers []Handler
}

func NewWindow(options ...app.Option) *Window {
	return &Window{Window: app.NewWindow(options...)}
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

func (window *Window) MsgOps(size int) x.Observable[MsgOp] {
	return x.FromChan(MsgOps(window.Window, size))
}

func (window *Window) Msgs(size int) x.Observable[any] {
	return x.Map(x.FromChan(MsgOps(window.Window, size)), func(msg MsgOp) any { return msg.Msg })
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
			if frame, ok := next.First.(system.FrameEvent); ok {
				frame = gio.HookFrameEvent(window.Window, frame)
				gtx := layout.NewContext(&ops, frame)
				for _, handler := range window.handlers {
					handler.Handle(gtx)
				}
				for _, widget := range next.Second {
					widget(gtx)
				}
				frame.Frame(gtx.Ops)
			}
		case err != nil:
			// log.Printf("error: %v\n", err)
		default:
			// log.Println("complete")
			for _, handler := range window.handlers {
				handler.Done()
			}
			window.handlers = nil
		}
	}
	return pairs.Subscribe(main)
}
