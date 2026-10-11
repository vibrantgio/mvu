//go:build !darwin

package desktop

import (
	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/unit"

	"github.com/vibrantgio/mvu"
)

// The stubs keep the package compiling and callable on every platform. The
// treatment is macOS-only, and away from macOS the exported API returns no
// options — never a borderless window on Linux or Windows — registers no
// handler and reports no insets. The close is the one entry point that does
// something here, because closing a window is not a treatment: Gio's own
// close action closes the window and raises the destroy event on these
// platforms, so the close is simply handed to it.

func fullSizeContent() []app.Option { return nil }

func showWindowButtons(*mvu.Window) {}

func placeWindowButtons(unit.Dp) {}

func placeWindowButtonsAt(_, _ unit.Dp) {}

func topInset() unit.Dp { return 0 }

func leadingInset() unit.Dp { return 0 }

func closeWindow(w *mvu.Window) {
	w.Window().Perform(system.ActionClose)
}
