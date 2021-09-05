package gio

import (
	"sync"

	"gioui.org/io/system"
	"gioui.org/op"
)

type EventHandler interface {
	Name() string
	Dispatch(system.FrameEvent) bool
	Register(*op.Ops)
}

type Handlers struct {
	sync.Mutex
	Items []EventHandler
}

func (h *Handlers) Append(e EventHandler) {
	h.Lock()
	defer h.Unlock()
	h.Items = append(h.Items, e)
}

func (h *Handlers) Delete(e EventHandler) {
	h.Lock()
	defer h.Unlock()
	for i, handler := range h.Items {
		if handler == e {
			copy(h.Items[i:], h.Items[i+1:])
			h.Items[len(h.Items)-1] = nil
			h.Items = h.Items[:len(h.Items)-1]
			break
		}
	}
}

func (m *Handlers) Dispatch(ops *op.Ops, frame system.FrameEvent) {
	m.Lock()
	defer m.Unlock()
	items := m.Items[:0]
	for i, handler := range m.Items {
		if !handler.Dispatch(frame) {
			m.Items[i] = nil
		} else {
			items = append(items, handler)
			handler.Register(ops)
		}
	}
	m.Items = items
}
