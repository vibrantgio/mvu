package desktop

import (
	"image"
	"testing"
)

// The macOS drop-point transform against known inputs at backing scale 1 and
// 2. View coordinates are AppKit points with a lower-left origin; the
// expected values are Gio pixels with an upper-left origin. The transform is
// pure arithmetic, so the table runs on every platform.
func TestGioPoint(t *testing.T) {
	tests := []struct {
		name       string
		x, y       float64 // view points, lower-left origin
		viewHeight float64 // view bounds height, points
		scale      float64 // backing scale factor
		want       image.Point
	}{
		// Scale 1: flip only.
		{"scale1 interior", 100, 150, 600, 1, image.Pt(100, 450)},
		{"scale1 lower-left corner", 0, 0, 600, 1, image.Pt(0, 600)},
		{"scale1 top-left corner", 0, 600, 600, 1, image.Pt(0, 0)},
		{"scale1 top-right corner", 800, 600, 600, 1, image.Pt(800, 0)},

		// Scale 2 (Retina): flip in points first, THEN scale. A transform
		// that scaled y before flipping would yield 600-300=300 here, not
		// (600-150)*2=900 — this case pins the order.
		{"scale2 interior", 100, 150, 600, 2, image.Pt(200, 900)},
		{"scale2 lower-left corner", 0, 0, 600, 2, image.Pt(0, 1200)},
		{"scale2 top-left corner", 0, 600, 600, 2, image.Pt(0, 0)},

		// Fractional points exist on screen (AppKit reports sub-point drag
		// locations); quantization is truncation to the containing pixel.
		{"scale2 fractional", 12.5, 0.25, 600, 2, image.Pt(25, 1199)},
		{"scale1 fractional", 12.75, 100.5, 600, 1, image.Pt(12, 499)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := gioPoint(tc.x, tc.y, tc.viewHeight, tc.scale)
			if got != tc.want {
				t.Errorf("gioPoint(%v, %v, h=%v, s=%v) = %v, want %v",
					tc.x, tc.y, tc.viewHeight, tc.scale, got, tc.want)
			}
		})
	}
}
