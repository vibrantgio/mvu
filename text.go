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

func Text(ops *op.Ops, shaper text.Shaper, font text.Font, size, maxWidth int, txt string, rect image.Rectangle, ax, ay float32, textColor color.Color) image.Point {
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
	px, py := float32(rect.Min.X)+ax*float32(rect.Dx()-dx), float32(rect.Min.Y)+ay*float32(rect.Dy()-dy)
	for _, line := range lines {
		shape := clip.Outline{Path: shaper.Shape(font, fixed.I(size), line.Layout)}.Op()
		py += float32(line.Ascent.Ceil())
		tstack := op.Offset(f32.Pt(px, py)).Push(ops)
		paint.FillShape(ops, fill, shape)
		tstack.Pop()
		py += float32(line.Descent.Ceil())
	}
	return image.Pt(dx, dy)
}
