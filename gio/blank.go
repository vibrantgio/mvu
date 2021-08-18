package gio

import (
	"image/color"

	"gioui.org/op"
	"gioui.org/op/paint"
)

func BlankScreen(bg color.Color) op.CallOp {
	ops := &op.Ops{}
	m := op.Record(ops)
	paint.Fill(ops, color.NRGBAModel.Convert(bg).(color.NRGBA))
	return m.Stop()
}
