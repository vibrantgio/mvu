//go:build darwin

package desktop

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Foundation

#include <stdlib.h>

#include "scrollbars_darwin.h"
*/
import "C"

import "unsafe"

// scrollBars reads the platform's own preference. An unwritten key is the
// platform's Automatic, and so is any value this build does not know: the
// setting gains a name no earlier release had more readily than a window
// gains a way to refuse one.
func scrollBars() ScrollBarSetting {
	s := C.vgio_desktop_scroll_bars()
	if s == nil {
		return ScrollBarsAutomatic
	}
	defer C.free(unsafe.Pointer(s))
	switch C.GoString(s) {
	case "Always":
		return ScrollBarsAlways
	case "WhenScrolling":
		return ScrollBarsWhenScrolling
	default:
		return ScrollBarsAutomatic
	}
}
