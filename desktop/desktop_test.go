package desktop_test

import (
	"testing"

	"gioui.org/app"
	"gioui.org/unit"

	"github.com/vibrantgio/mvu"
	"github.com/vibrantgio/mvu/desktop"
)

// Without a live native window there is nothing to assert against, so the
// whole seam must be inert: registration, the notification fired by a later
// Option call, and the inset query all run headless, on every platform,
// without touching AppKit state that does not exist. This is the most a test
// can prove without a real window; what the re-assertion does to actual
// window chrome needs one, and is proven by an adopting application.
func TestSeamInertWithoutNativeWindow(t *testing.T) {
	w := mvu.NewWindow(append(desktop.FullSizeContent(), app.Title("desktop test"))...)
	desktop.ShowWindowButtons(w)

	// Fires the OnConfigure notification, and with it the re-assertion, on
	// this goroutine.
	w.Option(app.Title("desktop test retitled"))

	if got := desktop.TopInset(); got != 0 {
		t.Fatalf("TopInset() = %v before any native window exists, want 0", got)
	}
	if got := desktop.LeadingInset(); got != 0 {
		t.Fatalf("LeadingInset() = %v before any native window exists, want 0", got)
	}
}

// A placement asked for with no window to place anything in must be recorded
// and forgotten about, not crash and not invent an inset: an application is
// free to state where its row is before the first frame has drawn.
func TestPlaceWindowButtonsInertWithoutNativeWindow(t *testing.T) {
	w := mvu.NewWindow(append(desktop.FullSizeContent(), app.Title("desktop test"))...)
	desktop.ShowWindowButtons(w)

	desktop.PlaceWindowButtons(14)
	defer desktop.PlaceWindowButtons(0)

	if got := desktop.TopInset(); got != 0 {
		t.Fatalf("TopInset() = %v after a placement with no native window, want 0", got)
	}
	if got := desktop.LeadingInset(); got != 0 {
		t.Fatalf("LeadingInset() = %v after a placement with no native window, want 0", got)
	}
}

// Dropping a placement is a supported state, not merely the absence of one:
// zero hands the buttons back, and asking for zero when nothing was ever
// placed is legal too.
func TestPlaceWindowButtonsZeroIsAccepted(t *testing.T) {
	desktop.PlaceWindowButtons(0)

	if got := desktop.TopInset(); got != 0 {
		t.Fatalf("TopInset() = %v with no placement and no native window, want 0", got)
	}
}

// The two-axis placement is under the same contract as the vertical one: a
// request with no window to place anything in is recorded and forgotten
// about, zero per axis is legal in every combination, and no combination
// invents an inset. What the request does to real window chrome needs a real
// window, and is proven by an adopting application.
func TestPlaceWindowButtonsAtInertWithoutNativeWindow(t *testing.T) {
	w := mvu.NewWindow(append(desktop.FullSizeContent(), app.Title("desktop test"))...)
	desktop.ShowWindowButtons(w)

	for _, c := range []struct{ leading, center int }{
		{25, 22}, // both axes stated
		{25, 0},  // leading alone, the line left to the system
		{0, 22},  // the vertical call's own placement, spelled here
		{0, 0},   // the full restore
	} {
		desktop.PlaceWindowButtonsAt(unit.Dp(c.leading), unit.Dp(c.center))
		if got := desktop.TopInset(); got != 0 {
			t.Fatalf("TopInset() = %v after PlaceWindowButtonsAt(%d, %d) with no native window, want 0",
				got, c.leading, c.center)
		}
		if got := desktop.LeadingInset(); got != 0 {
			t.Fatalf("LeadingInset() = %v after PlaceWindowButtonsAt(%d, %d) with no native window, want 0",
				got, c.leading, c.center)
		}
	}
}
