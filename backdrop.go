package vibrant

import (
	"image/color"

	"gioui.org/op"
	"gioui.org/op/paint"
)

func Backdrop(fill color.Color) op.CallOp {
	ops := &op.Ops{}
	macro := op.Record(ops)
	paint.ColorOp{Color: color.NRGBAModel.Convert(fill).(color.NRGBA)}.Add(ops)
	paint.PaintOp{}.Add(ops)
	return macro.Stop()
}
