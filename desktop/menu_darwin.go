//go:build darwin

package desktop

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AppKit

#include <stdlib.h>
#include "menu_darwin.h"
*/
import "C"

import "unsafe"

// install hands the bar's declaration to the native side and asks for the
// amendment. The declaration is reset first, so a second [MenuBar] replaces
// the first rather than adding to it — one bar, one claim.
//
// The amendment is asked for twice, and both asks are the same idempotent
// rebuild. Once now, because by the time an application declares its menu the
// run loop may already be up and there is no reason to make the bar wait for
// a frame; and once from the window's configuration notification, because at
// declaration time there is often no NSApplication yet — the application
// goroutine runs ahead of app.Main — and the notification is the first moment
// this package is told there certainly is one.
//
// Nothing here calls AppKit: vgio_menu_apply dispatches to the main thread
// itself and returns immediately. That is deliberate rather than incidental.
// The notification fires during Gio's first-frame handoff with the main
// thread parked inside it, so a synchronous hop from here would deadlock
// there — the same hazard the window buttons' re-assertion documents, and the
// same answer.
func (m *MenuBar) install() {
	C.vgio_menu_reset()
	for i, it := range m.items {
		m.declare(it, m.tags[i])
	}
	C.vgio_menu_apply()
	m.w.OnConfigure(func() {
		C.vgio_menu_apply()
	})
	debugf("desktop: %d menu item(s) declared on the application's bar", len(m.items))
}

// declare passes one item across the boundary. The three strings are copied
// on the far side, so the C strings are freed as soon as the call returns and
// no Go memory is reachable from the native declaration.
func (m *MenuBar) declare(it MenuItem, tag int) {
	menu := C.CString(it.Menu)
	title := C.CString(it.Title)
	key := C.CString(it.Key)
	defer C.free(unsafe.Pointer(menu))
	defer C.free(unsafe.Pointer(title))
	defer C.free(unsafe.Pointer(key))
	C.vgio_menu_declare(menu, title, key, C.int(tag))
}

// vgioMenuChosen is called from the menu target's action method on the AppKit
// main thread when a person chooses one of the declared items. It carries the
// item's tag and nothing else — which is the whole reason the bar's routing
// is a package-level registry (see menuActions).
//
//export vgioMenuChosen
func vgioMenuChosen(tag C.int) {
	chooseMenuTag(int(tag))
}
