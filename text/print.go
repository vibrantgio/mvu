package text

import (
	"image/color"

	"gioui.org/f32"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"golang.org/x/image/math/fixed"
)

func Print(shaper text.Shaper, txt string, r f32.Rectangle, ax, ay, maxWidth float32, style Style, col color.Color, ops *op.Ops) (dx, dy float32) {
	lines := shaper.LayoutString(style.Font, fixed.I(style.Size), int(maxWidth), txt)
	for _, line := range lines {
		dy += float32(line.Ascent.Ceil() + line.Descent.Ceil())
		lineWidth := float32(line.Width.Ceil())
		if dx < lineWidth {
			dx = lineWidth
		}
	}
	nrgba := color.NRGBAModel.Convert(col).(color.NRGBA)
	offset := f32.Pt(r.Min.X+ax*(r.Dx()-dx), r.Min.Y+ay*(r.Dy()-dy))
	for _, line := range lines {
		offset.Y += float32(line.Ascent.Ceil())
		tstack := op.Offset(offset).Push(ops)
		offset.Y += float32(line.Descent.Ceil())
		cstack := clip.Outline{Path: shaper.Shape(style.Font, fixed.I(style.Size), line.Layout)}.Op().Push(ops)
		paint.ColorOp{Color: nrgba}.Add(ops)
		paint.PaintOp{}.Add(ops)
		cstack.Pop()
		tstack.Pop()
	}
	return dx, dy
}
