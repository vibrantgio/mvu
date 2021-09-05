package gio

import (
	"fmt"
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

func NewFrameEventHandler() *FrameEventHandler {
	return &FrameEventHandler{
		Tag:  tag(),
		Chan: make(chan FrameEvent, 2),
	}
}

func (h FrameEventHandler) Name() string {
	return fmt.Sprint("Frame", h.Tag)
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

func (h *Handlers) FrameEvents() ObservableFrameEvent {
	observable := func(observe FrameEventObserver, scheduler Scheduler, subscriber Subscriber) {
		if subscriber.Subscribed() {
			handler := NewFrameEventHandler()
			FromChanFrameEvent(handler.Chan)(observe, scheduler, subscriber)
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
