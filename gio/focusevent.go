package gio

import (
	"log"

	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/op"
)

type FocusEvent struct {
	event.Tag
	key.FocusEvent
}

type FocusEventHandler struct {
	Tag  event.Tag
	Chan chan FocusEvent
	Fail int
}

func (h *FocusEventHandler) Dispatch(frame system.FrameEvent) bool {
	for _, e := range frame.Queue.Events(h.Tag) {
		if event, ok := e.(key.FocusEvent); ok {
			if h.Chan != nil {
				select {
				case h.Chan <- FocusEvent{Tag: h.Tag, FocusEvent: event}:
					h.Fail = 0
				default:
					log.Println("Dropping Key Focus Event", event)
					if h.Fail++; h.Fail >= 3 {
						return false
					}
				}
			}
		}
	}
	return true
}

func (h *FocusEventHandler) Register(ops *op.Ops) {
}

func (h *Handlers) FocusEvents() ObservableFocusEvent {
	observable := DeferFocusEvent(func() ObservableFocusEvent {
		c := make(chan FocusEvent, 2)
		h.Append(&FocusEventHandler{Tag: tag(), Chan: c})
		return FromChanFocusEvent(c)
	})
	return observable
}
