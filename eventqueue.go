package vibrant

import "gioui.org/io/event"

type EventQueue []event.Event

func (q EventQueue) Events(k event.Tag) []event.Event {
	n := len(q)
	return q[:n:n]
}
