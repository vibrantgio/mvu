package text

import (
	"gioui.org/text"
	"golang.org/x/image/math/fixed"
)

func Size(shaper text.Shaper, txt string, maxWidth float32, style Style) (dx, dy float32) {
	lines := shaper.LayoutString(style.Font, fixed.I(style.Size), int(maxWidth), txt)
	for _, line := range lines {
		dy += float32(line.Ascent.Ceil() + line.Descent.Ceil())
		lineWidth := float32(line.Width.Ceil())
		if dx < lineWidth {
			dx = lineWidth
		}
	}
	return
}
