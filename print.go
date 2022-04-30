package vibrant

import (
	"image"
	"image/color"

	"golang.org/x/image/math/fixed"

	"gioui.org/f32"
	"gioui.org/io/system"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
)

const (
	Min float32 = 0.0
	Mid float32 = 0.5
	Max float32 = 1.0
)

var Locale = system.Locale{Language: "en-US", Direction: system.LTR}

func Print(shaper text.Shaper, font text.Font, size, maxWidth int, txt string, rect image.Rectangle, ax, ay float32, textColor color.Color, ops *op.Ops) image.Point {
	var dx, dy int
	lines := shaper.LayoutString(font, fixed.I(size), maxWidth, Locale, txt)
	for _, line := range lines {
		dy += line.Ascent.Ceil()
		if dx < line.Width.Ceil() {
			dx = line.Width.Ceil()
		}
		dy += line.Descent.Ceil()
	}
	fill := color.NRGBAModel.Convert(textColor).(color.NRGBA)
	px, py := rect.Min.X+int(ax*float32(rect.Dx()-dx)), rect.Min.Y+int(ay*float32(rect.Dy()-dy))
	for _, line := range lines {
		shape := clip.Outline{Path: shaper.Shape(font, fixed.I(size), line.Layout)}.Op()
		py += line.Ascent.Ceil()
		tstack := op.Offset(f32.Pt(float32(px), float32(py))).Push(ops)
		paint.FillShape(ops, fill, shape)
		tstack.Pop()
		py += line.Descent.Ceil()
	}
	return image.Pt(dx, dy)
}
