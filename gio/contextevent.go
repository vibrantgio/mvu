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
	Tag event.Tag
	ContextEventObserver
}

func NewContextEventHandler() *ContextEventHandler {
	return &ContextEventHandler{
		Tag: tag(),
	}
}

func (h ContextEventHandler) Name() string {
	return fmt.Sprint("Context", h.Tag)
}

func (h ContextEventHandler) Dispatch(frame system.FrameEvent) bool {
	if h.ContextEventObserver == nil {
		return false
	}
	observe := h.ContextEventObserver
	context := layout.NewContext(new(op.Ops), frame)
	observe(ContextEvent{Tag: h.Tag, Context: context}, nil, false)
	return true
}

func (h ContextEventHandler) Register(ops *op.Ops) {
}

func (h *Handlers) ContextEvents() ObservableContextEvent {
	observable := func(observe ContextEventObserver, scheduler Scheduler, subscriber Subscriber) {
		if subscriber.Subscribed() {
			handler := NewContextEventHandler()
			handler.ContextEventObserver = func(next ContextEvent, err error, done bool) {
				if subscriber.Subscribed() {
					observe(next, err, done)
				}
			}
			if subscriber.Subscribed() {
				h.Append(handler)
				subscriber.OnUnsubscribe(func() {
					h.Delete(handler)
				})
			}
		}
	}
	return observable
}
