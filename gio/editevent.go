package gio

import (
	"log"

	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/op"
)

type EditEvent struct {
	event.Tag
	key.EditEvent
}

type EditEventHandler struct {
	Tag  event.Tag
	Chan chan EditEvent
	Fail int
}

func (h *EditEventHandler) Dispatch(frame system.FrameEvent) bool {
	for _, e := range frame.Queue.Events(h.Tag) {
		if event, ok := e.(key.EditEvent); ok {
			if h.Chan != nil {
				select {
				case h.Chan <- EditEvent{Tag: h.Tag, EditEvent: event}:
					h.Fail = 0
				default:
					log.Println("Dropping Key Edit Event", event)
					if h.Fail++; h.Fail >= 3 {
						return false
					}
				}
			}
		}
	}
	return true
}

func (h *EditEventHandler) Register(ops *op.Ops) {
}

func (h *Handlers) EditEvents() ObservableEditEvent {
	observable := DeferEditEvent(func() ObservableEditEvent {
		c := make(chan EditEvent, 2)
		h.Append(&EditEventHandler{Tag: tag(), Chan: c})
		return FromChanEditEvent(c)
	})
	return observable
}
