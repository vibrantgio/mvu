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
	observable := DeferClickEvent(func() ObservableClickEvent {
		c := make(chan ClickEvent, 2)
		h.Append(&ClickEventHandler{Click: &gesture.Click{}, Chan: c})
		return FromChanClickEvent(c)
	})
	return observable
}
