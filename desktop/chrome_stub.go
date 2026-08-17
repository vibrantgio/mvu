//go:build !darwin

package desktop

import (
	"gioui.org/app"
	"gioui.org/unit"

	"github.com/vibrantgio/mvu"
)

// The stubs keep the package compiling and callable on every platform. The
// treatment is macOS-only, and away from macOS the exported API returns no
// options — never a borderless window on Linux or Windows — registers
// nothing and reports no insets.

func fullSizeContent() []app.Option { return nil }

func showWindowButtons(*mvu.Window) {}

func placeWindowButtons(unit.Dp) {}

func placeWindowButtonsAt(_, _ unit.Dp) {}

func topInset() unit.Dp { return 0 }

func leadingInset() unit.Dp { return 0 }
