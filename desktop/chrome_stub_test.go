//go:build !darwin

package desktop_test

import (
	"testing"

	"github.com/vibrantgio/mvu/desktop"
)

// Away from macOS the treatment must vanish entirely: no options, because
// the borderless-window option behind it would remove the whole window frame
// on Linux or Windows.
func TestFullSizeContentReturnsNothing(t *testing.T) {
	if opts := desktop.FullSizeContent(); len(opts) != 0 {
		t.Fatalf("FullSizeContent() returned %d options, want none", len(opts))
	}
}
