package vibrant

import (
	"sync"

	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/io/system"
	"gioui.org/op"
	rx "github.com/reactivego/observable"
)

type Hover struct {
	*gesture.Hover
	event.Queue
}

// Add the gesture to detect hovering over the current pointer area.
func (h *Hover) Add(ops *op.Ops) {
	h.Hover.Add(ops)
}

// Hovered returns whether a pointer is inside the area.
func (c Hover) Hovered() bool {
	return c.Hover.Hovered(c.Queue)
}

func (window *Window) Hover() rx.Observable[Hover] {
	hover := struct {
		sync.Mutex
		Map map[*gesture.Hover][]event.Event
	}{Map: make(map[*gesture.Hover][]event.Event)}
	return func(observe rx.Observer[Hover], scheduler rx.Scheduler, subscriber rx.Subscriber) {
		channel := make(chan any, 5)
		rx.AsObservable[Hover](rx.FromChan(channel))(observe, scheduler, subscriber)
		tag := new(gesture.Hover)
		channel <- Hover{Hover: tag, Queue: EventQueue([]event.Event{})}
		hover.Lock()
		hover.Map[tag] = nil
		hover.Unlock()
		observer := func(next event.Event, err error, done bool) {
			switch {
			case !done:
				if frame, ok := next.(system.FrameEvent); ok {
					var events []event.Event
					for k := range hover.Map {
						events = append(events, frame.Queue.Events(k)...)
					}
					if n := len(events); n > 0 {
						for k := range hover.Map {
							hover.Map[k] = events
						}
					}
					if subscriber.Subscribed() {
						if events := hover.Map[tag]; events != nil {
							select {
							case channel <- Hover{Hover: tag, Queue: EventQueue(events)}:
								hover.Map[tag] = nil
							default:
								panic("Hover: Channel Overflow")
							}
						}
					}
				}
			case err != nil:
				select {
				case channel <- err:
					// OK
				default:
					panic("Hover: Channel Overflow")
				}
				close(channel) // currently unable to forward an error
			case err == nil:
				close(channel)
			}
		}
		handler := &EventHandler{observer}
		window.Append(handler)
		subscriber.OnUnsubscribe(func() { window.Delete(handler) })
	}
}
