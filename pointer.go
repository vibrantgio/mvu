package vibrant

import (
	"sync"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/op"

	"github.com/reactivego/x"
)

func PointerEvents(observable x.Observable[Pointer]) x.Observable[pointer.Event] {
	return x.SwitchMap(observable, func(pointer Pointer) x.Observable[pointer.Event] {
		return x.From(pointer.Events()...)
	})
}

type Pointer struct {
	event.Tag
	event.Queue
}

func (p Pointer) Add(ops *op.Ops, types pointer.Type) {
	pointer.InputOp{Tag: p.Tag, Types: types}.Add(ops)
}

func (p Pointer) Events() []pointer.Event {
	var events []pointer.Event
	for _, event := range p.Queue.Events(p.Tag) {
		if event, ok := event.(pointer.Event); ok {
			events = append(events, event)
		}
	}
	return events
}

func (window *Window) Pointer() x.Observable[Pointer] {
	pointer := struct {
		sync.Mutex
		Map map[event.Tag][]event.Event
	}{Map: make(map[event.Tag][]event.Event)}
	return func(observe x.Observer[Pointer], scheduler x.Scheduler, subscriber x.Subscriber) {
		channel := make(chan any, 5)
		x.AsObservable[Pointer](x.FromChan(channel))(observe, scheduler, subscriber)
		tag := Tag()
		channel <- Pointer{Tag: tag, Queue: EventQueue([]event.Event{})}
		pointer.Lock()
		pointer.Map[tag] = nil
		pointer.Unlock()
		observer := func(next event.Event, err error, done bool) {
			switch {
			case !done:
				if frame, ok := next.(system.FrameEvent); ok {
					var events []event.Event
					for k := range pointer.Map {
						events = append(events, frame.Queue.Events(k)...)
					}
					if n := len(events); n > 0 {
						for k := range pointer.Map {
							pointer.Map[k] = events
						}
					}
					if subscriber.Subscribed() {
						if events := pointer.Map[tag]; events != nil {
							select {
							case channel <- Pointer{Tag: tag, Queue: EventQueue(events)}:
								pointer.Map[tag] = nil
							default:
								panic("Pointer: Channel Overflow")
							}
						}
					}
				}
			case err != nil:
				select {
				case channel <- err:
					// OK
				default:
					panic("Pointer: Channel Overflow")
				}
				close(channel)
			default:
				close(channel)
			}
		}
		handler := &EventHandler{observer}
		window.Append(handler)
		subscriber.OnUnsubscribe(func() { window.Delete(handler) })
	}
}
