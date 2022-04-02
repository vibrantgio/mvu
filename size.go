package vibrant

import (
	"image"

	"golang.org/x/image/math/fixed"

	"gioui.org/text"
)

func Size(shaper text.Shaper, font text.Font, size, maxWidth int, txt string) image.Point {
	var dx, dy int
	for _, line := range shaper.LayoutString(font, fixed.I(size), maxWidth, txt) {
		dy += line.Ascent.Ceil()
		if dx < line.Width.Ceil() {
			dx = line.Width.Ceil()
		}
		dy += line.Descent.Ceil()
	}
	return image.Pt(dx, dy)
}
