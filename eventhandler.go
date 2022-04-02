package vibrant

import (
	"gioui.org/io/event"
	rx "github.com/reactivego/observable"
)

type EventHandler struct{ observe rx.Observer[event.Event] }

func (handler EventHandler) Handle(event event.Event) {
	handler.observe(event, nil, false)
}

func (handler EventHandler) Error(err error) {
	handler.observe(nil, err, true)
}

func (handler EventHandler) Complete() {
	handler.observe(nil, nil, true)
}
