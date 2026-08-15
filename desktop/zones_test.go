package desktop

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
)

// zeroGtx builds the minimal layout.Context Update's signature asks for; the
// frame boundary itself is the only information Update consumes.
func zeroGtx() layout.Context {
	return layout.Context{Ops: new(op.Ops)}
}

// The resolver as a pure function over a rect slice: hits, misses, overlap
// (last recorded — topmost — wins), dead space, boundary semantics.
func TestResolve(t *testing.T) {
	left := ZoneRect{Index: 0, Rect: image.Rect(10, 10, 110, 210)}
	right := ZoneRect{Index: 1, Rect: image.Rect(150, 10, 250, 210)}
	sideBySide := []ZoneRect{left, right}

	tests := []struct {
		name  string
		rects []ZoneRect
		p     image.Point
		want  int
	}{
		{"hit left", sideBySide, image.Pt(50, 100), 0},
		{"hit right", sideBySide, image.Pt(200, 100), 1},
		{"dead space between", sideBySide, image.Pt(130, 100), -1},
		{"miss above", sideBySide, image.Pt(50, 5), -1},
		{"miss outside all", sideBySide, image.Pt(500, 500), -1},
		{"empty set", nil, image.Pt(50, 100), -1},

		// Boundary semantics: Min inclusive, Max exclusive (image.Point.In).
		{"min corner is inside", sideBySide, image.Pt(10, 10), 0},
		{"max corner is outside", sideBySide, image.Pt(110, 210), -1},
		{"right edge is outside", sideBySide, image.Pt(110, 100), -1},

		// Overlap: the zone recorded LAST is topmost and wins — painter's
		// order. Index values deliberately disagree with recording order so
		// the test cannot pass by accident.
		{
			"overlap, last recorded wins",
			[]ZoneRect{
				{Index: 7, Rect: image.Rect(0, 0, 100, 100)},
				{Index: 3, Rect: image.Rect(50, 50, 150, 150)},
			},
			image.Pt(75, 75),
			3,
		},
		{
			"overlap, point only in the earlier zone",
			[]ZoneRect{
				{Index: 7, Rect: image.Rect(0, 0, 100, 100)},
				{Index: 3, Rect: image.Rect(50, 50, 150, 150)},
			},
			image.Pt(25, 25),
			7,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolve(tc.rects, tc.p); got != tc.want {
				t.Errorf("resolve(%v) = %d, want %d", tc.p, got, tc.want)
			}
		})
	}
}

// Hit resolves against the LAST COMPLETED frame, not the frame currently
// being recorded: a drop arrives out-of-band, mid-frame at worst, and must
// see a complete, consistent set.
func TestZoneGroupFrameBuffering(t *testing.T) {
	var z ZoneGroup

	// Nothing completed yet: everything is dead space.
	if got := z.Hit(image.Pt(5, 5)); got != -1 {
		t.Fatalf("Hit before any completed frame = %d, want -1", got)
	}

	// Frame 1 records a zone at the origin corner... but until Update
	// promotes it, Hit must not see it.
	z.Record(0, image.Rect(0, 0, 10, 10))
	if got := z.Hit(image.Pt(5, 5)); got != -1 {
		t.Fatalf("Hit sees the in-progress frame: got %d, want -1", got)
	}

	// Frame boundary: frame 1 completes; frame 2 begins, moving the zone.
	z.Update(zeroGtx())
	if got := z.Hit(image.Pt(5, 5)); got != 0 {
		t.Fatalf("Hit after frame 1 completed = %d, want 0", got)
	}
	z.Record(0, image.Rect(100, 100, 110, 110))
	if got := z.Hit(image.Pt(5, 5)); got != 0 {
		t.Fatalf("Hit switched to the in-progress frame 2: got %d, want 0", got)
	}
	if got := z.Hit(image.Pt(105, 105)); got != -1 {
		t.Fatalf("Hit sees frame 2's unpromoted rect: got %d, want -1", got)
	}

	// Frame 2 completes: the old position is dead space, the new one live.
	z.Update(zeroGtx())
	if got := z.Hit(image.Pt(5, 5)); got != -1 {
		t.Fatalf("Hit still sees frame 1 after two Updates: got %d, want -1", got)
	}
	if got := z.Hit(image.Pt(105, 105)); got != 0 {
		t.Fatalf("Hit after frame 2 completed = %d, want 0", got)
	}

	// A frame that records nothing (zone unmounted) leaves only dead space.
	z.Update(zeroGtx())
	if got := z.Hit(image.Pt(105, 105)); got != -1 {
		t.Fatalf("Hit after an empty frame = %d, want -1", got)
	}
}
