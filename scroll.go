package vibrant

import (
	"image"
	"sync"
	"time"

	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/io/system"
	"gioui.org/op"
	"gioui.org/unit"

	rx "github.com/reactivego/observable"
)

type Scroll struct {
	*gesture.Scroll
	event.Queue
	unit.Metric
	system.Insets
}

// Add the handler to the operation list to receive scroll events.
// The bounds variable refers to the scrolling boundaries
// as defined in io/pointer.InputOp.
func (s *Scroll) Add(ops *op.Ops, bounds image.Rectangle) {
	s.Scroll.Add(ops, bounds)
}

// Stop any remaining fling movement.
func (s *Scroll) Stop() {
	s.Scroll.Stop()
}

// Scroll detects the scrolling distance from the available events and
// ongoing fling gestures.
func (s Scroll) Distance(cfg unit.Metric, t time.Time, axis gesture.Axis) int {
	return s.Scroll.Scroll(cfg, s.Queue, t, axis)
}

// State reports the scroll state.
func (s *Scroll) State() gesture.ScrollState {
	return s.Scroll.State()
}

func (window *Window) Scroll() rx.Observable[Scroll] {
	scroll := struct {
		sync.Mutex
		Map map[*gesture.Scroll][]event.Event
	}{Map: make(map[*gesture.Scroll][]event.Event)}
	return func(observe rx.Observer[Scroll], scheduler rx.Scheduler, subscriber rx.Subscriber) {
		channel := make(chan any, 5)
		rx.AsObservable[Scroll](rx.FromChan(channel))(observe, scheduler, subscriber)
		tag := new(gesture.Scroll)
		channel <- Scroll{Scroll: tag, Queue: EventQueue([]event.Event{})}
		scroll.Lock()
		scroll.Map[tag] = nil
		scroll.Unlock()
		observer := func(next event.Event, err error, done bool) {
			switch {
			case !done:
				if frame, ok := next.(system.FrameEvent); ok {
					var events []event.Event
					for k := range scroll.Map {
						events = append(events, frame.Queue.Events(k)...)
					}
					if n := len(events); n > 0 {
						for k := range scroll.Map {
							scroll.Map[k] = events
						}
					}
					if subscriber.Subscribed() {
						if events := scroll.Map[tag]; events != nil {
							select {
							case channel <- Scroll{Scroll: tag, Queue: EventQueue(events), Metric: frame.Metric, Insets: frame.Insets}:
								scroll.Map[tag] = nil
							default:
								panic("Scroll: Channel Overflow")
							}
						}
					}
				}
			case err != nil:
				select {
				case channel <- err:
					// OK
				default:
					panic("Scroll: Channel Overflow")
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
