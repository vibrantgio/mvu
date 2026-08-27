package desktop

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
)

// capProbe is a page that records the constraints it was handed and fills
// them, so a test can read off both halves of the cap: what the inset did to
// the page, and what the claim did to the frame.
func capProbe(seen *layout.Constraints) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		*seen = gtx.Constraints
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}
}

// The cap is both halves at once, over one height: the strip is the window's
// drag band and the page starts below it. Either half missing is a window
// that cannot be moved by its top edge, or a page under the platform's own
// buttons.
func TestCapTopClaimsTheStripAndInsetsThePage(t *testing.T) {
	var seen layout.Constraints
	gtx := bandGtx(400, 300)
	dims := CapTop(func() unit.Dp { return 32 }, capProbe(&seen))(gtx)

	if seen.Max.Y != 268 {
		t.Errorf("page's max height = %d, want 268 — the window less the strip", seen.Max.Y)
	}
	if seen.Min.Y != 268 {
		t.Errorf("page's min height = %d, want 268 — the minimum must follow the maximum down", seen.Min.Y)
	}
	if want := (image.Pt(400, 300)); dims.Size != want {
		t.Errorf("size = %v, want %v — a capped layer still measures as the whole window", dims.Size, want)
	}

	moves := movesAt(t, gtx.Ops)
	// With no window behind the test there are no control buttons to clear,
	// so the strip is claimed from the leading edge — the headless case
	// TestDragTopClaimsTheStripAndNothingBelow documents for the claim alone.
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

// No strip is no cap: the page is laid out in the context it was handed and
// nothing is claimed over it. This is every platform but macOS, and macOS
// itself until the window's first frame.
func TestCapTopIsANoOpWithNoStrip(t *testing.T) {
	var seen layout.Constraints
	gtx := bandGtx(400, 300)
	dims := CapTop(func() unit.Dp { return 0 }, capProbe(&seen))(gtx)

	if seen != gtx.Constraints {
		t.Errorf("constraints handed on = %+v, want the context's own %+v", seen, gtx.Constraints)
	}
	if want := (image.Pt(400, 300)); dims.Size != want {
		t.Errorf("size = %v, want %v", dims.Size, want)
	}
	if movesAt(t, gtx.Ops)(200, 0) {
		t.Error("window-move action with no strip above the page; there is no band to claim")
	}
}

// The strip is measured from the live window, so it is 0 until the first
// frame and moves afterwards. Both halves have to follow it, and follow it
// together: a claim reading one number and an inset reading another is a band
// that overlaps the page or a gap nothing claims.
func TestCapTopReadsTheHeightEveryFrame(t *testing.T) {
	strip := unit.Dp(0)
	var seen layout.Constraints
	w := CapTop(func() unit.Dp { return strip }, capProbe(&seen))

	gtx := bandGtx(400, 300)
	w(gtx)
	if seen.Max.Y != 300 {
		t.Fatalf("first frame handed the page max height %d, want 300", seen.Max.Y)
	}
	if movesAt(t, gtx.Ops)(200, 0) {
		t.Fatal("window-move action before the strip was measured")
	}

	strip = 32
	gtx = bandGtx(400, 300)
	w(gtx)
	if seen.Max.Y != 268 {
		t.Errorf("second frame handed the page max height %d, want 268 — the measurement was not read again", seen.Max.Y)
	}
	if !movesAt(t, gtx.Ops)(200, 16) {
		t.Error("no window-move action after the strip was measured; the claim did not read the height again")
	}
}

// The claim is recorded before the page, so a page that reaches back up into
// the strip keeps its own presses there rather than handing them to the
// window. The inset means no ordinary page does reach up; the order is what
// holds when one does.
func TestCapTopClaimsBeforeThePage(t *testing.T) {
	gtx := bandGtx(400, 300)
	page := func(gtx layout.Context) layout.Dimensions {
		// Back up over the strip the inset just pushed this page past, and
		// take a control's worth of it.
		defer op.Offset(image.Pt(0, -32)).Push(gtx.Ops).Pop()
		var click widget.Clickable
		return click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: image.Pt(100, 32)}
		})
	}
	CapTop(func() unit.Dp { return 32 }, page)(gtx)

	moves := movesAt(t, gtx.Ops)
	if moves(50, 16) {
		t.Error("window-move action over a control standing in the strip; the claim was recorded after the page and swallowed its press")
	}
	if !moves(200, 16) {
		t.Error("no window-move action beside that control; the rest of the strip is still the band")
	}
}
