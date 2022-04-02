package vibrant

import (
	"image/color"

	"gioui.org/f32"
	"gioui.org/op"
	"gioui.org/op/paint"
)

func LinearGradient(stop1 f32.Point, color1 color.Color, stop2 f32.Point, color2 color.Color) op.CallOp {
	ops := &op.Ops{}
	macro := op.Record(ops)
	paint.LinearGradientOp{
		Stop1:  stop1,
		Color1: color.NRGBAModel.Convert(color1).(color.NRGBA),
		Stop2:  stop2,
		Color2: color.NRGBAModel.Convert(color2).(color.NRGBA),
	}.Add(ops)
	paint.PaintOp{}.Add(ops)
	return macro.Stop()
}
