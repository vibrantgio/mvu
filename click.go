package vibrant

import (
	"sync"

	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/io/system"

	"github.com/reactivego/x"
)

func ClickEvents(observable x.Observable[Click]) x.Observable[gesture.ClickEvent] {
	return x.SwitchMap(observable, func(click Click) x.Observable[gesture.ClickEvent] {
		return x.From(click.Events()...)
	})
}

type ClickState struct {
	Hovered bool
	Pressed bool
	Events  []gesture.ClickEvent
}

func ClickStates(observable x.Observable[Click]) x.Observable[ClickState] {
	return x.Map(observable, func(click Click) ClickState {
		events := click.Click.Events(click.Queue)
		return ClickState{click.Hovered(), click.Pressed(), events}
	})
}

type Click struct {
	*gesture.Click
	event.Queue
}

func (c Click) Events() []gesture.ClickEvent {
	return c.Click.Events(c.Queue)
}

func (window *Window) Click() x.Observable[Click] {
	click := struct {
		sync.Mutex
		Map map[*gesture.Click][]event.Event
	}{Map: make(map[*gesture.Click][]event.Event)}
	return func(observe x.Observer[Click], scheduler x.Scheduler, subscriber x.Subscriber) {
		channel := make(chan any, 5)
		x.AsObservable[Click](x.FromChan(channel))(observe, scheduler, subscriber)
		tag := new(gesture.Click)
		channel <- Click{Click: tag, Queue: EventQueue([]event.Event{})}
		click.Lock()
		click.Map[tag] = nil
		click.Unlock()
		handler := NewHandler(func(next event.Event, done bool) {
			if done {
				close(channel)
				return
			}
			if frame, ok := next.(system.FrameEvent); ok {
				var all []event.Event
				for k := range click.Map {
					all = append(all, frame.Queue.Events(k)...)
				}
				if n := len(all); n > 0 {
					for k := range click.Map {
						click.Map[k] = all
					}
				}
				if subscriber.Subscribed() {
					if events := click.Map[tag]; events != nil {
						select {
						case channel <- Click{Click: tag, Queue: EventQueue(events)}:
							click.Map[tag] = nil
						default:
							panic("Click: Channel Overflow")
						}
					}
				}
			}
		})
		window.Append(handler)
		subscriber.OnUnsubscribe(func() { window.Delete(handler) })
	}
}
