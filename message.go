package mvu

import (
	"gioui.org/io/pointer"
	"gioui.org/op"
	"gioui.org/op/clip"
)

type MessageOp struct{ Message any }

func (op MessageOp) Add(o *op.Ops) {
	defer clip.Rect{}.Push(o).Pop()
	pointer.InputOp{Tag: op}.Add(o)
}
