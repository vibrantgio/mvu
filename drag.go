package vibrant

import (
	"sync"

	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/unit"

	"github.com/reactivego/x"
)

func DragEvents(observable x.Observable[Drag], cfg unit.Metric, axis gesture.Axis) x.Observable[pointer.Event] {
	return x.SwitchMap(observable, func(drag Drag) x.Observable[pointer.Event] {
		return x.From(drag.Events(cfg, axis)...)
	})
}

type DragState struct {
	Dragging bool
	Pressed  bool
	Events   []pointer.Event
}

func DragStates(observable x.Observable[Drag], cfg unit.Metric, axis gesture.Axis) x.Observable[DragState] {
	return x.Map(observable, func(drag Drag) DragState {
		events := drag.Events(cfg, axis)
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

func (window *Window) Drag() x.Observable[Drag] {
	drag := struct {
		sync.Mutex
		Map map[*gesture.Drag][]event.Event
	}{Map: make(map[*gesture.Drag][]event.Event)}
	return func(observe x.Observer[Drag], scheduler x.Scheduler, subscriber x.Subscriber) {
		channel := make(chan any, 5)
		x.AsObservable[Drag](x.FromChan(channel))(observe, scheduler, subscriber)
		tag := new(gesture.Drag)
		channel <- Drag{Drag: tag, Queue: EventQueue([]event.Event{})}
		drag.Lock()
		drag.Map[tag] = nil
		drag.Unlock()
		handler := NewHandler(func(next event.Event, done bool) {
			if done {
				close(channel)
				return
			}
			if frame, ok := next.(system.FrameEvent); ok {
				var all []event.Event
				for k := range drag.Map {
					all = append(all, frame.Queue.Events(k)...)
				}
				if n := len(all); n > 0 {
					for k := range drag.Map {
						drag.Map[k] = all
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
		})
		window.Append(handler)
		subscriber.OnUnsubscribe(func() { window.Delete(handler) })
	}
}
