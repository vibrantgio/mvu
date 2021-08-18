package gio

import (
	"image"
	"log"
	"time"

	"gioui.org/io/event"
	"gioui.org/io/system"
	"gioui.org/op"
	"gioui.org/unit"
)

type FrameEvent struct {
	event.Tag

	// Now is the current animation. Use Now instead of time.Now to
	// synchronize animation and to avoid the time.Now call overhead.
	Now time.Time
	// Metric converts device independent dp and sp to device pixels.
	Metric unit.Metric
	// Size is the dimensions of the window.
	Size image.Point
	// Insets is the insets to apply.
	Insets system.Insets
}

type FrameEventHandler struct {
	Tag  event.Tag
	Chan chan FrameEvent
	Fail int
}

func (h *FrameEventHandler) Dispatch(frame system.FrameEvent) bool {
	if h.Chan != nil {
		select {
		case h.Chan <- FrameEvent{Tag: h.Tag, Now: frame.Now, Metric: frame.Metric, Size: frame.Size, Insets: frame.Insets}:
			h.Fail = 0
		default:
			log.Println("Dropping System Frame Event", frame)
			if h.Fail++; h.Fail >= 3 {
				return false
			}
		}
	}
	return true
}

func (h *FrameEventHandler) Register(ops *op.Ops) {
}

func (h *Handlers) FrameEvents() (EventHandler, ObservableFrameEvent) {
	c := make(chan FrameEvent, 2)
	handler := &FrameEventHandler{Tag: tag(), Chan: c}
	h.Append(handler)
	return handler, FromChanFrameEvent(c)
}
