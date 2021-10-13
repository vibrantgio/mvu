package gio

import (
	"fmt"

	"gioui.org/io/event"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
)

type ContextEvent struct {
	event.Tag
	layout.Context
}

type ContextEventHandler struct {
	Tag     event.Tag
	Observe ContextEventObserver
}

func NewContextEventHandler(observe ContextEventObserver) *ContextEventHandler {
	return &ContextEventHandler{
		Tag:     tag(),
		Observe: observe,
	}
}

func (h ContextEventHandler) Name() string {
	return fmt.Sprintf("Context %v", *h.Tag.(*int))
}

func (h ContextEventHandler) Dispatch(frame system.FrameEvent) bool {
	if h.Observe == nil {
		return false
	}
	event := ContextEvent{
		Tag:     h.Tag,
		Context: layout.NewContext(new(op.Ops), frame),
	}
	h.Observe(event, nil, false)
	return true
}

func (h ContextEventHandler) Register(ops *op.Ops) {
}

func (h *Handlers) ContextEvents() ObservableContextEvent {
	observable := func(observe ContextEventObserver, scheduler Scheduler, subscriber Subscriber) {
		if subscriber.Subscribed() {
			handler := NewContextEventHandler(func(next ContextEvent, err error, done bool) {
				if subscriber.Subscribed() {
					observe(next, err, done)
				}
			})
			h.Append(handler)
			subscriber.OnUnsubscribe(func() { h.Delete(handler) })
		}
	}
	return observable
}
