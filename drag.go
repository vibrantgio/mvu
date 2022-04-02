package vibrant

import (
	"sync"

	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/unit"
	rx "github.com/reactivego/observable"
)

func DragEvents(observable rx.Observable[Drag], cfg unit.Metric, axis gesture.Axis) rx.Observable[pointer.Event] {
	return rx.SwitchMap(observable, func(drag Drag) rx.Observable[pointer.Event] {
		return rx.From(drag.Events(cfg, axis)...)
	})
}

type DragState struct {
	Dragging bool
	Pressed  bool
	Events   []pointer.Event
}

func DragStates(observable rx.Observable[Drag], cfg unit.Metric, axis gesture.Axis) rx.Observable[DragState] {
	return rx.Map(observable, func(drag Drag) DragState {
		events := drag.Drag.Events(cfg, drag.Queue, axis)
		return DragState{drag.Dragging(), drag.Pressed(), events}
	})
}

type Drag struct {
	*gesture.Drag
	event.Queue
}

func (c Drag) Events(cfg unit.Metric, axis gesture.Axis) []pointer.Event {
	return c.Drag.Events(cfg, c.Queue, axis)
}

func (window *Window) Drag() rx.Observable[Drag] {
	drag := struct {
		sync.Mutex
		Map map[*gesture.Drag][]event.Event
	}{Map: make(map[*gesture.Drag][]event.Event)}
	return func(observe rx.Observer[Drag], scheduler rx.Scheduler, subscriber rx.Subscriber) {
		channel := make(chan any, 5)
		rx.AsObservable[Drag](rx.FromChan(channel))(observe, scheduler, subscriber)
		tag := new(gesture.Drag)
		channel <- Drag{Drag: tag, Queue: EventQueue([]event.Event{})}
		drag.Lock()
		drag.Map[tag] = nil
		drag.Unlock()
		observer := func(next event.Event, err error, done bool) {
			switch {
			case !done:
				if frame, ok := next.(system.FrameEvent); ok {
					var events []event.Event
					for k := range drag.Map {
						events = append(events, frame.Queue.Events(k)...)
					}
					if n := len(events); n > 0 {
						for k := range drag.Map {
							drag.Map[k] = events
						}
					}
					if subscriber.Subscribed() {
						if events := drag.Map[tag]; events != nil {
							select {
							case channel <- Drag{Drag: tag, Queue: EventQueue(events)}:
								drag.Map[tag] = nil
							default:
								panic("Drag: Channel Overflow")
							}
						}
					}
				}
			case err != nil:
				select {
				case channel <- err:
					// OK
				default:
					panic("Drag: Channel Overflow")
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
