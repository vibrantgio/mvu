package vibrant

import (
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"github.com/reactivego/x"
)

type PointerBurst struct {
	event.Tag
	Events []pointer.Event
}

func (window *Window) PointerBurst() x.Observable[PointerBurst] {
	return func(observe x.Observer[PointerBurst], scheduler x.Scheduler, subscriber x.Subscriber) {
		channel := make(chan PointerBurst, 5)
		x.FromChan(channel)(observe, scheduler, subscriber)
		tag := Tag()
		channel <- PointerBurst{Tag: tag}
		handler := NewHandler(func(next event.Event, done bool) {
			if done {
				close(channel)
				return
			}
			if frame, ok := next.(system.FrameEvent); ok {
				var events []pointer.Event
				for _, event := range frame.Queue.Events(tag) {
					if event, ok := event.(pointer.Event); ok {
						events = append(events, event)
					}
				}
				if subscriber.Subscribed() {
					select {
					case channel <- PointerBurst{Tag: tag, Events: events}:
					default:
						panic("Pointer: Channel Overflow")
					}
				}
			}
		})
		window.Append(handler)
		subscriber.OnUnsubscribe(func() { window.Delete(handler) })
	}
}
