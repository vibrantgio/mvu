package gio

import (
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

func (h *Handlers) ContextEvents() (EventHandler, ObservableContextEvent) {
	handler := &ContextEventHandler{Tag: tag()}
	h.Append(handler)
	observable := func(observe ContextEventObserver, subscribeOn Scheduler, subscriber Subscriber) {
		observer := func(next ContextEvent, err error, done bool) {
			if subscriber.Subscribed() {
				observe(next, err, done)
			}
		}
		handler.ContextEventObserver = observer
	}
	return handler, observable
}
