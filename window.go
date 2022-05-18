package vibrant

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"gioui.org/app"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/op"

	"github.com/reactivego/x"
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

func (window *Window) Render(launchscreen op.CallOp, layers ...x.Observable[op.CallOp]) x.Subscription {
	events := x.FromChan(window.Events()).Filter(func(next event.Event) bool {
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
		layers = append(layers, x.Of(launchscreen))
	} else {
		for i := range layers {
			layers[i] = layers[i].StartWith(launchscreen)
			launchscreen = op.CallOp{}
		}
	}
	callops := x.Map(x.Combine(layers...), invalidate).SubscribeOn(x.Goroutine)
	pairs := x.WithLatestFromPair(events, callops)

	var last struct {
		sync.Mutex
		Enter time.Time
		Leave time.Time
	}
	ticker := time.NewTicker(5 * time.Second)
	poison := make(chan struct{})
	var ops op.Ops
	main := func(next x.Pair[event.Event, []op.CallOp], err error, done bool) {
		switch {
		case !done:
			window.Handle(next.First)
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
				last.Lock()
				last.Enter = time.Now()
				last.Unlock()
				ops.Reset()
				for _, callop := range next.Second {
					callop.Add(&ops)
				}
				event.Frame(&ops)
				last.Lock()
				last.Leave = time.Now()
				last.Unlock()
			case system.DestroyEvent:
				// if event.Err != nil {
				log.Printf("destroy: %v\n", event.Err)
				// }
			case pointer.Event:
				// log.Printf("pointer: %v\n", event)
			default:
				log.Printf("event: %#v\n", event)
			}
		case err != nil:
			log.Printf("error: %v\n", err)
			ticker.Stop()
			close(poison)
		default:
			log.Println("complete")
			window.Handle(nil)
			ticker.Stop()
			close(poison)
		}
	}

	go func() {
		counter := 0
		for {
			select {
			case <-poison:
				fmt.Println("Poison!")
				return
			case <-ticker.C:
				counter++
				last.Lock()
				if last.Enter.After(last.Leave) {
					fmt.Println("#", counter, "dead at", last.Enter.Format("15:04:05"))
				} else {
					fmt.Println("#", counter, "live at", last.Leave.Format("15:04:05"))
				}
				last.Unlock()
			}
		}
	}()

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
