//go:build darwin

package desktop_test

import (
	"testing"

	"gioui.org/app"
	"gioui.org/unit"

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
