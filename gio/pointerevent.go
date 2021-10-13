package gio

import (
	"fmt"
	"log"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/op"
)

type PointerEvent struct {
	event.Tag
	Types pointer.Type
	pointer.Event
}

type PointerEventHandler struct {
	Tag   event.Tag
	Types pointer.Type
	Chan  chan PointerEvent
	Fail  int
}

func NewPointerEventHandler(types pointer.Type) *PointerEventHandler {
	return &PointerEventHandler{
		Tag:   tag(),
		Types: types,
		Chan:  make(chan PointerEvent, 2),
	}
}

func (h PointerEventHandler) Name() string {
	return fmt.Sprintf("Pointer %v", *h.Tag.(*int))
}

func (h *PointerEventHandler) Dispatch(frame system.FrameEvent) bool {
	if h.Chan != nil {
		for _, e := range frame.Queue.Events(h.Tag) {
			if event, ok := e.(pointer.Event); ok {
				if h.Chan != nil {
					select {
					case h.Chan <- PointerEvent{Tag: h.Tag, Types: h.Types, Event: event}:
						h.Fail = 0
					default:
						log.Println("Dropping Pointer Event", event)
						if h.Fail++; h.Fail >= 3 {
							return false
						}
					}
				}
			}
		}
	}
	return true
}

func (h *PointerEventHandler) Register(ops *op.Ops) {
	if h.Chan != nil {
		pointer.InputOp{Tag: h.Tag, Types: h.Types}.Add(ops)
	}
}

func (h *Handlers) PointerEvents(types pointer.Type) ObservablePointerEvent {
	observable := func(observe PointerEventObserver, scheduler Scheduler, subscriber Subscriber) {
		if subscriber.Subscribed() {
			handler := NewPointerEventHandler(types)
			FromChanPointerEvent(handler.Chan)(observe, scheduler, subscriber)
			h.Append(handler)
			subscriber.OnUnsubscribe(func() { h.Delete(handler) })
		}
	}
	return observable
}
