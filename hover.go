package vibrant

import (
	"sync"

	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/layout"

	"github.com/reactivego/x"
)

type Hover struct {
	*gesture.Hover
	event.Queue
}

// Hovered returns whether a pointer is inside the area.
func (c Hover) Hovered() bool {
	return c.Hover.Hovered(c.Queue)
}

func (window *Window) Hover() x.Observable[Hover] {
	hover := struct {
		sync.Mutex
		Map map[*gesture.Hover][]event.Event
	}{Map: make(map[*gesture.Hover][]event.Event)}
	return func(observe x.Observer[Hover], scheduler x.Scheduler, subscriber x.Subscriber) {
		channel := make(chan any, 5)
		x.AsObservable[Hover](x.FromChan(channel))(observe, scheduler, subscriber)
		tag := new(gesture.Hover)
		channel <- Hover{Hover: tag, Queue: EventQueue([]event.Event{})}
		hover.Lock()
		hover.Map[tag] = nil
		hover.Unlock()
		handler := NewHandler(
			func(gtx layout.Context) {
				var all []event.Event
				for k := range hover.Map {
					all = append(all, gtx.Events(k)...)
				}
				if n := len(all); n > 0 {
					for k := range hover.Map {
						hover.Map[k] = all
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
			}, func() {
				close(channel)
			})
		window.Append(handler)
		subscriber.OnUnsubscribe(func() { window.Delete(handler) })
	}
}
