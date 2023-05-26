package vibrant

import "gioui.org/io/event"

type EventQueue []event.Event

func (q EventQueue) Events(k event.Tag) []event.Event {
	if n := len(q); n > 0 {
		return q[:n:n]
	}
	return nil
}
