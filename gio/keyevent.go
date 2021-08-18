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
	observable := DeferKeyEvent(func() ObservableKeyEvent {
		c := make(chan KeyEvent, 2)
		h.Append(&KeyEventHandler{Tag: tag(), Chan: c})
		return FromChanKeyEvent(c)
	})
	return observable
}
