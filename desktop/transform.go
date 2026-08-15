package desktop

import "image"

// gioPoint is the pure half of the macOS drop-point transform: it converts a
// drag location that has already been converted to VIEW coordinates (AppKit
// points, lower-left origin) into Gio pixels (upper-left origin).
//
// The seam with the Objective-C side is deliberate: the one AppKit call that
// cannot leave Objective-C is the window-to-view point conversion, so the
// native side ships the raw components — view-point x/y, the view bounds
// height, the backing scale factor — across, and this function owns the math
// where a table test can see it. The scale is re-read from the window on
// every single event, native-side, because a window can move between
// displays of different scale mid-drag; it is a parameter here, never
// cached.
//
// Order matters and is what the scale-2 tests pin down: the flip happens in
// points, the scale multiplies last — (viewHeight - y) * scale — matching
// Gio's own mouse-event path bit for bit, so drops land exactly where clicks
// do.
//
// The result is quantized to whole pixels by truncation: the returned point
// is the pixel cell containing the drop location, which is what integer-rect
// zone hit-testing wants; sub-pixel accuracy is meaningless for a
// cursor-sized gesture.
//
// The function itself is pure arithmetic and compiles everywhere; only its
// caller is platform-gated.
func gioPoint(x, y, viewHeight, scale float64) image.Point {
	return image.Point{
		X: int(x * scale),
		Y: int((viewHeight - y) * scale),
	}
}
