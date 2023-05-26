package vibrant

import (
	"fmt"
	"strings"

	"gioui.org/io/event"
)

type Handler interface {
	Next(event.Event)
	Done()
}

type Handle func(next event.Event, done bool)

func (h Handle) Next(event event.Event) {
	h(event, false)
}

func (h Handle) Done() {
	h(nil, true)
}

type handler struct {
	Handle
}

func NewHandler(handle Handle) Handler {
	return &handler{handle}
}

func (window *Window) Append(handler Handler) {
	window.Lock()
	defer window.Unlock()
	window.handlers = append(window.handlers, handler)
}

func (window *Window) Delete(handler Handler) {
	window.Lock()
	defer window.Unlock()
	for i, h := range window.handlers {
		if h == handler {
			copy(window.handlers[i:], window.handlers[i+1:])
			window.handlers[len(window.handlers)-1] = nil
			window.handlers = window.handlers[:len(window.handlers)-1]
			break
		}
	}
}

func (w *Window) String() string {
	sb := new(strings.Builder)
	sb.WriteString("Window{\n")
	for _, h := range w.handlers {
		fmt.Fprintf(sb, "  (%[1]T) %[1]v\n", h)
	}
	sb.WriteString("}")
	return sb.String()
}
