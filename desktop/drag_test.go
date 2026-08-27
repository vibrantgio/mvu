package desktop

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/system"
	"gioui.org/op"
	"gioui.org/unit"
)

// movesAt replays a frame through the same hit test the window makes on a
// press — asked what action stands at a point — so the tests measure the area
// a band actually claimed rather than the ops it recorded.
func movesAt(t *testing.T, ops *op.Ops) func(x, y int) bool {
	t.Helper()
	var r input.Router
	r.Frame(ops)
	return func(x, y int) bool {
		a, ok := r.ActionAt(f32.Pt(float32(x), float32(y)))
		return ok && a == system.ActionMove
	}
}

func TestDragBandClaimsItsRectangle(t *testing.T) {
	gtx := bandGtx(400, 300)
	DragBand(gtx, image.Rect(50, 0, 200, 40))
	moves := movesAt(t, gtx.Ops)

	for _, p := range []image.Point{{X: 50, Y: 0}, {X: 120, Y: 20}, {X: 199, Y: 39}} {
		if !moves(p.X, p.Y) {
			t.Errorf("no window-move action at %v; the band was declared over it", p)
		}
	}
	for _, p := range []image.Point{{X: 49, Y: 20}, {X: 200, Y: 20}, {X: 120, Y: 40}, {X: 300, Y: 200}} {
		if moves(p.X, p.Y) {
			t.Errorf("window-move action at %v; the band ends before it, and what stands outside a band keeps its own presses", p)
		}
	}
}

// A band asked for a run it has no room for records no area at all: a
// degenerate one would be an invisible claim nothing could ever hit, and the
// callers that compute a run from a width they may not have rely on being able
// to hand it over as it comes out.
func TestDragBandClaimsNothingWithoutRoom(t *testing.T) {
	for _, tc := range []struct {
		name string
		r    image.Rectangle
	}{
		{"no width", image.Rect(10, 0, 10, 40)},
		{"no height", image.Rect(10, 0, 200, 0)},
		{"inverted", image.Rectangle{Min: image.Pt(200, 0), Max: image.Pt(10, 40)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gtx := bandGtx(400, 300)
			DragBand(gtx, tc.r)
			moves := movesAt(t, gtx.Ops)
			for _, p := range []image.Point{{X: 0, Y: 0}, {X: 10, Y: 20}, {X: 100, Y: 20}, {X: 199, Y: 39}} {
				if moves(p.X, p.Y) {
					t.Errorf("window-move action at %v for %v; a band with no room claims nothing", p, tc.r)
				}
			}
		})
	}
}

func TestDragRunIsAsDeepAsTheRow(t *testing.T) {
	gtx := bandGtx(400, 52)
	dims := DragRun(gtx, 120)
	if want := (image.Pt(120, 52)); dims.Size != want {
		t.Errorf("size = %v, want %v — the run is as deep as the row it stands in", dims.Size, want)
	}
	moves := movesAt(t, gtx.Ops)
	for _, p := range []image.Point{{X: 0, Y: 0}, {X: 60, Y: 26}, {X: 119, Y: 51}} {
		if !moves(p.X, p.Y) {
			t.Errorf("no window-move action at %v; the run reaches the row's top and foot", p)
		}
	}
	if moves(120, 26) {
		t.Error("window-move action past the run's trailing edge")
	}
}

// The degenerate runs, which a row hands over as they come out of its own
// arithmetic: a width it has nothing left for, and a row measured at nothing.
func TestDragRunWithoutRoom(t *testing.T) {
	for _, tc := range []struct {
		name    string
		rowH, w int
		want    image.Point
	}{
		{"no width left in the row", 52, 0, image.Pt(0, 0)},
		{"a width the row overran", 52, -5, image.Pt(0, 0)},
		{"a row measured at nothing", 0, 120, image.Pt(120, 0)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gtx := bandGtx(400, tc.rowH)
			if got := DragRun(gtx, tc.w).Size; got != tc.want {
				t.Errorf("size = %v, want %v", got, tc.want)
			}
			moves := movesAt(t, gtx.Ops)
			for _, p := range []image.Point{{X: 0, Y: 0}, {X: 60, Y: 0}, {X: 60, Y: 26}} {
				if moves(p.X, p.Y) {
					t.Errorf("window-move action at %v; a run with no room claims nothing", p)
				}
			}
		})
	}
}

func TestDragTopClaimsTheStripAndNothingBelow(t *testing.T) {
	gtx := bandGtx(400, 300)
	DragTop(gtx, func() unit.Dp { return 32 })
	moves := movesAt(t, gtx.Ops)

	// With no window behind the test there are no control buttons to clear,
	// so the strip is claimed from the leading edge — which is also what
	// every platform that keeps its own decorations would get, if it had a
	// strip to claim at all.
	for _, p := range []image.Point{{X: 0, Y: 0}, {X: 200, Y: 16}, {X: 399, Y: 31}} {
		if !moves(p.X, p.Y) {
			t.Errorf("no window-move action at %v; the strip above the page is the band", p)
		}
	}
	for _, p := range []image.Point{{X: 200, Y: 32}, {X: 200, Y: 150}, {X: 200, Y: 299}} {
		if moves(p.X, p.Y) {
			t.Errorf("window-move action at %v; the page below the strip is the page's, not the band's", p)
		}
	}
}

// The strip's height is measured from the window, so it is 0 until the first
// frame and moves afterwards. Nothing is claimed while it reports nothing, and
// the claim follows it when it changes.
func TestDragTopReadsTheHeightEveryFrame(t *testing.T) {
	strip := unit.Dp(0)
	height := func() unit.Dp { return strip }

	gtx := bandGtx(400, 300)
	DragTop(gtx, height)
	if movesAt(t, gtx.Ops)(200, 0) {
		t.Error("window-move action with no strip above the page; there is no band to claim")
	}

	strip = 32
	gtx = bandGtx(400, 300)
	DragTop(gtx, height)
	if !movesAt(t, gtx.Ops)(200, 16) {
		t.Error("no window-move action after the strip was measured; the height was not read again")
	}
}

// The control buttons stand in the leading run of the very strip this band
// claims, and a move action over them would fight them for the press. The run
// they occupy is not the strip's to give away.
func TestDragTopLeavesTheButtonsTheirRun(t *testing.T) {
	gtx := bandGtx(400, 300)
	got := dragTopRect(gtx, 79, 32)
	if want := (image.Rect(79, 0, 400, 32)); got != want {
		t.Fatalf("dragTopRect(79, 32) = %v, want %v", got, want)
	}
	if bare := dragTopRect(gtx, 0, 32); bare != image.Rect(0, 0, 400, 32) {
		t.Errorf("with no buttons to clear the strip is claimed whole, got %v", bare)
	}

	DragBand(gtx, got)
	moves := movesAt(t, gtx.Ops)
	if moves(40, 16) {
		t.Error("window-move action over the window buttons' own run")
	}
	for _, p := range []image.Point{{X: 79, Y: 0}, {X: 200, Y: 16}, {X: 399, Y: 31}} {
		if !moves(p.X, p.Y) {
			t.Errorf("no window-move action at %v; the strip past the buttons is the band", p)
		}
	}
}
