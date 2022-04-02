package vibrant

import (
	"gioui.org/io/event"
)

type Handler interface {
	Handle(event.Event)
	Error(error)
	Complete()
}
