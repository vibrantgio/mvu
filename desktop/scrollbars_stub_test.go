//go:build !darwin

package desktop

import "testing"

// TestScrollBarsStub asserts the answer away from macOS: the overlay bar
// that shows while the content scrolls, which is what the library draws by
// default.
func TestScrollBarsStub(t *testing.T) {
	if got := ScrollBars(); got != ScrollBarsWhenScrolling {
		t.Errorf("ScrollBars() = %v, want %v", got, ScrollBarsWhenScrolling)
	}
}
