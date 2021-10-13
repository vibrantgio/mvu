package gio

import (
	"log"

	"gioui.org/gesture"
	"gioui.org/io/system"
	"gioui.org/op"
)

type ClickEvent struct {
	gesture.Click
	gesture.ClickEvent
}

type ClickEventHandler struct {
	*gesture.Click
	Chan chan ClickEvent
	Fail int
}

func NewClickEventHandler() *ClickEventHandler {
	return &ClickEventHandler{
		Click: &gesture.Click{},
		Chan:  make(chan ClickEvent, 2),
	}
}

func (h ClickEventHandler) Name() string {
	return "Click"
}

func (h *ClickEventHandler) Dispatch(frame system.FrameEvent) bool {
	for _, event := range h.Click.Events(frame.Queue) {
		if h.Chan != nil {
			select {
			case h.Chan <- ClickEvent{Click: *h.Click, ClickEvent: event}:
				h.Fail = 0
			default:
				log.Println("Dropping Gesture Click Event", event)
				if h.Fail++; h.Fail >= 3 {
					return false
				}
			}
		}
	}
	return true
}

func (h ClickEventHandler) Register(ops *op.Ops) {
}

func (h *Handlers) ClickEvents() ObservableClickEvent {
	observable := func(observe ClickEventObserver, scheduler Scheduler, subscriber Subscriber) {
		if subscriber.Subscribed() {
			handler := NewClickEventHandler()
			FromChanClickEvent(handler.Chan)(observe, scheduler, subscriber)
			h.Append(handler)
			subscriber.OnUnsubscribe(func() { h.Delete(handler) })
		}
	}
	return observable
}
