//go:build darwin

package desktop

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AppKit

#include "chrome_darwin.h"
*/
import "C"

import (
	"gioui.org/app"
	"gioui.org/unit"

	"github.com/vibrantgio/mvu"
)

func fullSizeContent() []app.Option {
	return []app.Option{app.Decorated(false)}
}

// showWindowButtons registers the re-assertion on the window's configuration
// notification. The registered func runs on whatever goroutine triggered the
// notification — the render goroutine for the first frame, an arbitrary
// caller of Option otherwise — and during the first-frame notification the
// main thread is parked inside Gio's frame handoff, so a synchronous hop to
// the main queue would deadlock there. vgio_desktop_reassert therefore
// dispatches asynchronously and returns immediately.
//
// The asynchronous hop cannot lose the race against Gio's own re-hide:
// Gio's Window.Option runs Configure on the main thread and waits for it to
// finish before returning, so by the time mvu notifies, the re-hide has
// already happened and the re-assert lands after it — never before.
func showWindowButtons(w *mvu.Window) {
	w.OnConfigure(func() {
		C.vgio_desktop_reassert()
	})
}

func placeWindowButtons(center unit.Dp) {
	C.vgio_desktop_place_buttons(C.double(center))
}

func topInset() unit.Dp {
	return unit.Dp(C.vgio_desktop_top_inset())
}

func leadingInset() unit.Dp {
	return unit.Dp(C.vgio_desktop_leading_inset())
}
