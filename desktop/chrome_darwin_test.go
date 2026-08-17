//go:build darwin

package desktop_test

import (
	"testing"

	"gioui.org/app"
	"gioui.org/unit"

	"github.com/vibrantgio/mvu"
	"github.com/vibrantgio/mvu/desktop"
)

// On macOS the options must ask Gio for the undecorated window — that toggle
// is the whole full-size-content treatment on this platform — and nothing
// else.
func TestFullSizeContentRequestsUndecorated(t *testing.T) {
	opts := desktop.FullSizeContent()
	if len(opts) != 1 {
		t.Fatalf("FullSizeContent() returned %d options, want exactly 1", len(opts))
	}

	metric := unit.Metric{PxPerDp: 1, PxPerSp: 1}
	cnf := app.Config{Decorated: true}
	for _, opt := range opts {
		opt(metric, &cnf)
	}
	if cnf.Decorated {
		t.Fatal("FullSizeContent() left Config.Decorated true, want false")
	}
}

// What the leading inset may report on this platform. A test binary has no
// NSApplication, so the re-assertion returns before measuring anything and
// the answer is the unmeasured 0 — the same headless case the cross-platform
// test pins. Once a real window exists the answer is the trailing edge of the
// three window buttons, which no macOS release has put anywhere near the
// edges of this band; anything outside it is a coordinate-space error rather
// than a system that moved the buttons, and this catches that without
// nailing the test to one release's metrics.
func TestLeadingInsetIsUnmeasuredOrPlausible(t *testing.T) {
	w := mvu.NewWindow(append(desktop.FullSizeContent(), app.Title("desktop test"))...)
	desktop.ShowWindowButtons(w)
	w.Option(app.Title("desktop test retitled"))

	const (
		minPlausible = 40
		maxPlausible = 160
	)
	got := desktop.LeadingInset()
	if got == 0 {
		return
	}
	if got < minPlausible || got > maxPlausible {
		t.Fatalf("LeadingInset() = %v, want 0 (unmeasured) or within [%v, %v]",
			got, unit.Dp(minPlausible), unit.Dp(maxPlausible))
	}
}
