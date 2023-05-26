package vibrant

import (
	"sync"
	"time"

	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/io/system"
	"gioui.org/unit"

	"github.com/reactivego/x"
)

type Scroll struct {
	*gesture.Scroll
	event.Queue
}

// Distance detects the scrolling distance from the available events and ongoing fling gestures.
func (s Scroll) Distance(cfg unit.Metric, t time.Time, axis gesture.Axis) int {
	return s.Scroll.Scroll(cfg, s.Queue, t, axis)
}

func (window *Window) Scroll() x.Observable[Scroll] {
	scroll := struct {
		sync.Mutex
		Map map[*gesture.Scroll][]event.Event
	}{Map: make(map[*gesture.Scroll][]event.Event)}
	return func(observe x.Observer[Scroll], scheduler x.Scheduler, subscriber x.Subscriber) {
		channel := make(chan any, 5)
		x.AsObservable[Scroll](x.FromChan(channel))(observe, scheduler, subscriber)
		tag := new(gesture.Scroll)
		channel <- Scroll{Scroll: tag, Queue: EventQueue([]event.Event{})}
		scroll.Lock()
		scroll.Map[tag] = nil
		scroll.Unlock()
		handler := NewHandler(func(next event.Event, done bool) {
			if done {
				close(channel)
				return
			}
			if frame, ok := next.(system.FrameEvent); ok {
				var all []event.Event
				for k := range scroll.Map {
					all = append(all, frame.Queue.Events(k)...)
				}
				if n := len(all); n > 0 {
					for k := range scroll.Map {
						scroll.Map[k] = all
					}
				}
				if subscriber.Subscribed() {
					if events := scroll.Map[tag]; events != nil {
						select {
						case channel <- Scroll{Scroll: tag, Queue: EventQueue(events)}:
							scroll.Map[tag] = nil
						default:
							panic("Scroll: Channel Overflow")
						}
					}
				}
			}
		})
		window.Append(handler)
		subscriber.OnUnsubscribe(func() { window.Delete(handler) })
	}
}
