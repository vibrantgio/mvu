//go:build darwin

package mvu

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AppKit -framework CoreGraphics

#include "frame_darwin.h"
*/
import "C"

import (
	"sync/atomic"
	"unsafe"

	"gioui.org/unit"
)

// macOS is the one platform this package reads and applies a window position
// on. Gio's app.Config carries a size and no origin, so the origin comes from
// the native window: it is applied once, through the window's own main-thread
// door, and reported back on every move and resize of the window afterwards.
// The Objective-C half (frame_darwin.m) states both directions of the
// conversion out of AppKit's upward coordinates.

// frameDisplays is how many screens are asked about at once. A machine with
// more is not misread, only read to this depth, and the restore decision only
// needs one screen to land on.
const frameDisplays = 16

// reporter is the memory the native window's move and resize notifications
// reach. It has to be package-level state: the notification block re-enters Go
// through a plain C function, which carries no instance and no closure, so a
// package-level pointer is the only bridge. One window at a time, matching the
// one watch the Objective-C side keeps.
var reporter atomic.Pointer[frameMemory]

// platformOwnsPosition reports whether the platform reads and applies the
// window's position itself.
func platformOwnsPosition() bool { return true }

// platformScreens reports where the screens are. Quartz answers this from any
// thread, which is what lets [RememberFrame] decide whether a remembered frame
// still lands on a screen before the event loop exists.
func platformScreens() []Frame {
	var buf [frameDisplays]C.VgioFrame
	n := int(C.vgio_frame_displays((*C.VgioFrame)(unsafe.Pointer(&buf[0])), C.int(len(buf))))
	if n <= 0 {
		return nil
	}
	screens := make([]Frame, 0, n)
	for _, d := range buf[:n] {
		screens = append(screens, Frame{
			X:      dpOf(d.x),
			Y:      dpOf(d.y),
			Width:  dpOf(d.width),
			Height: dpOf(d.height),
		})
	}
	return screens
}

// platformRememberFrame applies the remembered position and starts the watch,
// on the main thread — the only thread AppKit may be touched from. It runs on
// a goroutine of its own (see [Window.rememberFrame]) because the window's Run
// blocks until the main thread has taken the work, and at the moment this is
// called the main thread is still finishing the first frame.
func platformRememberFrame(w *Window, memory *frameMemory, remembered Frame, restore bool) {
	reporter.Store(memory)
	w.Window().Run(func() {
		if restore {
			C.vgio_frame_set(
				C.double(remembered.X),
				C.double(remembered.Y),
				C.double(remembered.Width),
				C.double(remembered.Height),
			)
		}
		C.vgio_frame_observe()
	})
}

// platformForgetFrame drops the watch's destination as the window goes away,
// so a notification arriving during teardown writes nothing.
func platformForgetFrame() { reporter.Store(nil) }

// vgioFrameChanged is called from the window's move and resize notifications
// on the AppKit main thread, with the frame already in mvu's own top-left
// space and in points, which equal dp. The debounce is on the other side of
// this call, so a drag of the window's edge lands here per notification and
// reaches the file once.
//
//export vgioFrameChanged
func vgioFrameChanged(x, y, width, height C.double) {
	if memory := reporter.Load(); memory != nil {
		memory.record(Frame{
			X:      roundDp(float64(x)),
			Y:      roundDp(float64(y)),
			Width:  roundDp(float64(width)),
			Height: roundDp(float64(height)),
		})
	}
}

// dpOf restates one measured C length as whole dp.
func dpOf(v C.double) unit.Dp { return roundDp(float64(v)) }
