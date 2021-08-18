package gio

import (
	"log"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/op"
)

type PointerEvent struct {
	event.Tag
	pointer.Event
}

type PointerEventHandler struct {
	Tag   event.Tag
	Types pointer.Type
	Chan  chan PointerEvent
	Fail  int
}

func (h *PointerEventHandler) Dispatch(frame system.FrameEvent) bool {
	if h.Chan != nil {
		for _, e := range frame.Queue.Events(h.Tag) {
			if event, ok := e.(pointer.Event); ok {
				if h.Chan != nil {
					select {
					case h.Chan <- PointerEvent{Tag: tag, Event: event}:
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
		// state := op.Save(ops)
		pointer.InputOp{Tag: h.Tag, Types: h.Types}.Add(ops)
		// state.Load()
	}
}

func (h *Handlers) PointerEvents(types pointer.Type) ObservablePointerEvent {
	observable := DeferPointerEvent(func() ObservablePointerEvent {
		c := make(chan PointerEvent, 2)
		h.Append(&PointerEventHandler{Tag: tag(), Types: types, Chan: c})
		return FromChanPointerEvent(c)
	})
	return observable
}
