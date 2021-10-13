package gio

import (
	"log"

	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/op"
)

type KeyEvent struct {
	event.Tag
	key.Event
}

type KeyEventHandler struct {
	Tag  event.Tag
	Chan chan KeyEvent
	Fail int
}

func NewKeyEventHandler() *KeyEventHandler {
	return &KeyEventHandler{
		Tag:  tag(),
		Chan: make(chan KeyEvent, 2),
	}
}

func (h KeyEventHandler) Name() string {
	return "Key"
}

func (h *KeyEventHandler) Dispatch(frame system.FrameEvent) bool {
	for _, e := range frame.Queue.Events(h.Tag) {
		if event, ok := e.(key.Event); ok {
			if h.Chan != nil {
				select {
				case h.Chan <- KeyEvent{Tag: h.Tag, Event: event}:
					h.Fail = 0
				default:
					log.Println("Dropping Key Event", event)
					if h.Fail++; h.Fail >= 3 {
						return false
					}
				}
			}
		}
	}
	return true
}

func (h *KeyEventHandler) Register(ops *op.Ops) {
}

func (h *Handlers) KeyEvents() ObservableKeyEvent {
	observable := func(observe KeyEventObserver, scheduler Scheduler, subscriber Subscriber) {
		if subscriber.Subscribed() {
			handler := NewKeyEventHandler()
			FromChanKeyEvent(handler.Chan)(observe, scheduler, subscriber)
			h.Append(handler)
			subscriber.OnUnsubscribe(func() { h.Delete(handler) })
		}
	}
	return observable
}
