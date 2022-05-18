package vibrant

import (
	"gioui.org/io/event"

	"github.com/reactivego/x"
)

type EventHandler struct{ observe x.Observer[event.Event] }

func (handler EventHandler) Handle(event event.Event) {
	handler.observe(event, nil, false)
}

func (handler EventHandler) Error(err error) {
	handler.observe(nil, err, true)
}

func (handler EventHandler) Complete() {
	handler.observe(nil, nil, true)
}
