package desktop_test

import (
	"testing"

	"gioui.org/app"

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
}
