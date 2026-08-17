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

// These windows carry no buttons inside their content, so nothing is ever
// displaced: the leading inset is zero unconditionally, not merely until some
// measurement lands, and an application may lay out from the leading edge.
func TestLeadingInsetIsZero(t *testing.T) {
	if got := desktop.LeadingInset(); got != 0 {
		t.Fatalf("LeadingInset() = %v on a platform without window buttons, want 0", got)
	}
}
