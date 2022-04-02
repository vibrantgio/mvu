package vibrant

import (
	"sync"

	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/io/system"
	rx "github.com/reactivego/observable"
)

func ClickEvents(observable rx.Observable[Click]) rx.Observable[gesture.ClickEvent] {
	return rx.SwitchMap(observable, func(click Click) rx.Observable[gesture.ClickEvent] {
		return rx.From(click.Events()...)
	})
}

type ClickState struct {
	Hovered bool
	Pressed bool
	Events  []gesture.ClickEvent
}

func ClickStates(observable rx.Observable[Click]) rx.Observable[ClickState] {
	return rx.Map(observable, func(click Click) ClickState {
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

func (window *Window) Click() rx.Observable[Click] {
	click := struct {
		sync.Mutex
		Map map[*gesture.Click][]event.Event
	}{Map: make(map[*gesture.Click][]event.Event)}
	return func(observe rx.Observer[Click], scheduler rx.Scheduler, subscriber rx.Subscriber) {
		channel := make(chan any, 5)
		rx.AsObservable[Click](rx.FromChan(channel))(observe, scheduler, subscriber)
		tag := new(gesture.Click)
		channel <- Click{Click: tag, Queue: EventQueue([]event.Event{})}
		click.Lock()
		click.Map[tag] = nil
		click.Unlock()
		observer := func(next event.Event, err error, done bool) {
			switch {
			case !done:
				if frame, ok := next.(system.FrameEvent); ok {
					var events []event.Event
					for k := range click.Map {
						events = append(events, frame.Queue.Events(k)...)
					}
					if n := len(events); n > 0 {
						for k := range click.Map {
							click.Map[k] = events
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
			case err != nil:
				select {
				case channel <- err:
					// OK
				default:
					panic("Click: Channel Overflow")
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
