package vibrant

import (
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/reactivego/x"
)

type Pointer struct {
	event.Tag
	Event pointer.Event
}

func (p Pointer) Add(ops *op.Ops, types pointer.Type) {
	pointer.InputOp{Tag: p.Tag, Types: types}.Add(ops)
}

func (window *Window) PointerEvents() x.Observable[Pointer] {
	return x.SwitchMap(window.PointerBurst(), func(burst PointerBurst) x.Observable[Pointer] {
		if len(burst.Events) == 0 {
			return x.Of(Pointer{Tag: burst.Tag})
		}
		var events []Pointer
		for _, b := range burst.Events {
			events = append(events, Pointer{Tag: burst.Tag, Event: b})
		}
		return x.From(events...)
	})
}

type PointerBurst struct {
	event.Tag
	Events []pointer.Event
}

func (window *Window) PointerBurst() x.Observable[PointerBurst] {
	return func(observe x.Observer[PointerBurst], scheduler x.Scheduler, subscriber x.Subscriber) {
		channel := make(chan PointerBurst, 5)
		x.FromChan(channel)(observe, scheduler, subscriber)
		tag := new(struct{})
		channel <- PointerBurst{Tag: tag}
		handler := NewHandler(
			func(gtx layout.Context) {
				var events []pointer.Event
				for _, event := range gtx.Events(tag) {
					if event, ok := event.(pointer.Event); ok {
						events = append(events, event)
					}
				}
				if subscriber.Subscribed() && len(events) > 0 {
					select {
					case channel <- PointerBurst{Tag: tag, Events: events}:
					default:
						panic("Pointer: Channel Overflow")
					}
				}
			}, func() {
				close(channel)
			})
		window.Append(handler)
		subscriber.OnUnsubscribe(func() { window.Delete(handler) })
	}
}
