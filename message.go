package mvu

import (
	"gioui.org/io/event"
	"gioui.org/op"
	"gioui.org/op/clip"
)

type Message = any

type MessageOp struct{ Message }

func (op MessageOp) Add(o *op.Ops) {
	defer clip.Rect{}.Push(o).Pop()
	event.Op(o, op)
}
