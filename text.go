package vibrant

import (
	"image"
	"image/color"

	"golang.org/x/image/math/fixed"

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

func Text(ops *op.Ops, rect image.Rectangle, ax, ay float32, shaper text.Shaper, font text.Font, size, maxWidth int, textColor color.Color, txt string) (dx, dy int) {
	lines := shaper.LayoutString(font, fixed.I(size), maxWidth, Locale, txt)
	for _, line := range lines {
		dy += line.Ascent.Ceil()
		if dx < line.Width.Ceil() {
			dx = line.Width.Ceil()
		}
		dy += line.Descent.Ceil()
	}
	c := color.NRGBAModel.Convert(textColor).(color.NRGBA)
	offset := rect.Min.Add(image.Pt(int(ax*float32(rect.Dx()-dx)), int(ay*float32(rect.Dy()-dy))))
	for _, line := range lines {
		shape := clip.Outline{Path: shaper.Shape(font, fixed.I(size), line.Layout)}.Op()
		offset.Y += line.Ascent.Ceil()
		tstack := op.Offset(offset).Push(ops)
		paint.FillShape(ops, c, shape)
		tstack.Pop()
		offset.Y += line.Descent.Ceil()
	}
	return
}
