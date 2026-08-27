package desktop

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// The stored platform reference, as a table: the five toolbar windows that
// were measured all read 19/19/14/23/79, and the plain title bar reads
// 9/9/14/23/69. Both rows must fall out of the band alone.
func TestButtonRunFollowsTheStoredReference(t *testing.T) {
	tests := []struct {
		name string
		band unit.Dp
		want ButtonRun
	}{
		{"unified toolbar band", 52, ButtonRun{Leading: 19, Center: 26, Diameter: 14, Trailing: 79}},
		{"plain title bar", 32, ButtonRun{Leading: 9, Center: 16, Diameter: 14, Trailing: 69}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ButtonRunIn(tc.band); got != tc.want {
				t.Errorf("ButtonRunIn(%v) = %+v, want %+v — the stored measurement for this band", tc.band, got, tc.want)
			}
		})
	}
}

// The two derivations are one rule from opposite ends: a caller with a band
// and a caller with an inset must land on the same run, or the leading inset
// no longer equals the top inset.
func TestButtonRunAtAndInAgree(t *testing.T) {
	for _, band := range []unit.Dp{32, 38, 52, 64} {
		in := ButtonRunIn(band)
		at := ButtonRunAt(in.Leading)
		if in != at {
			t.Errorf("band %v derives %+v but its own inset derives %+v", band, in, at)
		}
		if in.Center != band/2 {
			t.Errorf("band %v centres its buttons on %v, not on its own middle %v", band, in.Center, band/2)
		}
	}
}

// A band of odd height has no whole-dp middle. Centring is still centring:
// the run lands on the half line rather than being rounded off it.
func TestButtonRunCentresOnAnOddBand(t *testing.T) {
	got := ButtonRunIn(51)
	want := ButtonRun{Leading: 18.5, Center: 25.5, Diameter: 14, Trailing: 78.5}
	if got != want {
		t.Errorf("ButtonRunIn(51) = %+v, want %+v", got, want)
	}
}

// Trailing is the run's whole width: three circles at the platform's pitch.
func TestButtonRunTrailingSpansTheGroup(t *testing.T) {
	run := ButtonRunAt(19)
	if want := run.Leading + 2*WindowButtonPitch + run.Diameter; run.Trailing != want {
		t.Errorf("Trailing = %v, want %v — three circles at pitch %v", run.Trailing, want, unit.Dp(WindowButtonPitch))
	}
}

func TestBandLeadFrom(t *testing.T) {
	tests := []struct {
		name                    string
		buttonsEnd, gap, gutter unit.Dp
		want                    unit.Dp
	}{
		{"buttons measured, band spaces them", 79, 12, 16, 91},
		{"buttons measured, band starts flush", 79, 0, 12, 79},
		{"no buttons in the content", 0, 12, 16, 16},
		{"no buttons, no gutter either", 0, 12, 0, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := BandLeadFrom(tc.buttonsEnd, tc.gap, tc.gutter); got != tc.want {
				t.Errorf("BandLeadFrom(%v, %v, %v) = %v, want %v", tc.buttonsEnd, tc.gap, tc.gutter, got, tc.want)
			}
		})
	}
}

// With no native window behind the test there are no buttons to clear, so
// the measured form must answer with the band's own gutter — which is also
// what every platform that keeps its decorations gets.
func TestBandLeadWithoutAWindow(t *testing.T) {
	if got := BandLead(12, 16); got != 16 {
		t.Errorf("BandLead(12, 16) = %v with no window, want the gutter 16", got)
	}
}

// bandGtx is a layout context at one pixel per dp, sized like a window.
func bandGtx(w, h int) layout.Context {
	return layout.Context{
		Ops:         new(op.Ops),
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(image.Pt(w, h)),
	}
}

func TestInsetTop(t *testing.T) {
	// The wrapped widget records the constraints it was handed, so the
	// test can assert on what the inset did to them rather than on pixels.
	var seen layout.Constraints
	probe := func(gtx layout.Context) layout.Dimensions {
		seen = gtx.Constraints
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}

	t.Run("no inset is an exact no-op", func(t *testing.T) {
		gtx := bandGtx(400, 300)
		dims := InsetTop(func() unit.Dp { return 0 }, probe)(gtx)
		if seen != gtx.Constraints {
			t.Errorf("constraints handed on = %+v, want the context's own %+v", seen, gtx.Constraints)
		}
		if want := (image.Pt(400, 300)); dims.Size != want {
			t.Errorf("size = %v, want %v", dims.Size, want)
		}
	})

	t.Run("an inset shortens the widget and not the layer", func(t *testing.T) {
		gtx := bandGtx(400, 300)
		dims := InsetTop(func() unit.Dp { return 32 }, probe)(gtx)
		if seen.Max.Y != 268 {
			t.Errorf("wrapped widget's max height = %d, want 268 — the window less the strip", seen.Max.Y)
		}
		if seen.Min.Y != 268 {
			t.Errorf("wrapped widget's min height = %d, want 268 — the minimum must follow the maximum down", seen.Min.Y)
		}
		if seen.Max.X != 400 {
			t.Errorf("wrapped widget's width = %d, want 400 — the inset is vertical only", seen.Max.X)
		}
		if want := (image.Pt(400, 300)); dims.Size != want {
			t.Errorf("size = %v, want %v — the layer still measures as the whole window", dims.Size, want)
		}
	})

	t.Run("the height is read every frame", func(t *testing.T) {
		inset := unit.Dp(0)
		w := InsetTop(func() unit.Dp { return inset }, probe)
		w(bandGtx(400, 300))
		if seen.Max.Y != 300 {
			t.Fatalf("first frame handed max height %d, want 300", seen.Max.Y)
		}
		inset = 32
		w(bandGtx(400, 300))
		if seen.Max.Y != 268 {
			t.Errorf("second frame handed max height %d, want 268 — a measurement that arrived after the first frame was not read", seen.Max.Y)
		}
	})
}
