//go:build darwin

package desktop

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AppKit

#include <stdlib.h>
#include "menu_darwin.h"
*/
import "C"

import (
	"strings"
	"unsafe"
)

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
// itself and returns immediately. The notification fires during Gio's
// first-frame handoff with the main thread parked inside it, so a synchronous
// hop from here would deadlock.
func (m *MenuBar) install() {
	C.vgio_menu_reset()
	for i, it := range m.items {
		m.declare(it, m.tags[i], 0)
		for j, sub := range it.Items {
			m.declare(sub, m.subTags[i][j], m.tags[i])
		}
	}
	C.vgio_menu_apply()
	m.w.OnConfigure(func() {
		C.vgio_menu_apply()
	})
	debugf("desktop: %d menu item(s) declared on the application's bar", len(m.items))
}

// declare passes one item across the boundary. parent is the tag of the
// submenu item that holds it, or 0. The three strings are copied on the
// far side, so the C strings are freed as soon as the call returns and no
// Go memory is reachable from the native declaration.
func (m *MenuBar) declare(it MenuItem, tag, parent int) {
	menu := C.CString(it.Menu)
	title := C.CString(it.Title)
	key := C.CString(it.Key)
	defer C.free(unsafe.Pointer(menu))
	defer C.free(unsafe.Pointer(title))
	defer C.free(unsafe.Pointer(key))
	C.vgio_menu_declare(menu, title, key, C.int(tag), C.int(parent), itemFlags(it))
}

// showItems gives the submenu item with tag parent the items items, with
// the tags tags. The native side takes them in one step.
func (m *MenuBar) showItems(parent int, items []MenuItem, tags []int) {
	for j, it := range items {
		title := C.CString(it.Title)
		key := C.CString(it.Key)
		C.vgio_menu_stage(title, key, C.int(tags[j]), itemFlags(it))
		C.free(unsafe.Pointer(title))
		C.free(unsafe.Pointer(key))
	}
	C.vgio_menu_replace(C.int(parent))
}

// itemFlags is the native flags of it: disabled, Control and submenu.
func itemFlags(it MenuItem) C.int {
	var f C.int
	if it.Disabled {
		f |= C.VGIO_MENU_DISABLED
	}
	if it.Control {
		f |= C.VGIO_MENU_CONTROL
	}
	if it.Submenu {
		f |= C.VGIO_MENU_SUBMENU
	}
	return f
}

// show gives the declared item with tag its new title and disabled state.
// The native side copies the title, and the bar reads the state when it
// validates the item.
func (m *MenuBar) show(tag int, title string, disabled bool) {
	t := C.CString(title)
	defer C.free(unsafe.Pointer(t))
	C.vgio_menu_set(C.int(tag), t, cbool(disabled))
}

// nativeItem is the native declaration's state for the item with tag: its
// title and its disabled state. ok is false when the native declaration
// holds no item with tag.
func nativeItem(tag int) (title string, disabled, ok bool) {
	var d C.int
	t := C.vgio_menu_lookup(C.int(tag), &d)
	if t == nil {
		return "", false, false
	}
	defer C.free(unsafe.Pointer(t))
	return C.GoString(t), d != 0, true
}

// nativeTakesKey reports whether a menu bar that holds the declared items
// of the menu titled menu takes the key equivalent chars, with Command and
// with Shift when shift is true. A key that the bar does not take goes on
// to the window.
func nativeTakesKey(menu, chars string, shift bool) bool {
	nativeBuild(menu)
	return nativePress(chars, shift, false)
}

// nativeBuild builds a menu bar with one menu titled menu, which holds the
// declared items of that menu. The bar stays until the next nativeBuild,
// so a test presses keys on it after the declaration changes.
func nativeBuild(menu string) {
	m := C.CString(menu)
	defer C.free(unsafe.Pointer(m))
	C.vgio_menu_test_build(m)
}

// nativePress reports whether the bar of the last nativeBuild takes the key
// equivalent chars, with Command, and with Shift and Control when asked.
// An arrow key is pressed as a keyboard sends it.
func nativePress(chars string, shift, control bool) bool {
	c := C.CString(chars)
	defer C.free(unsafe.Pointer(c))
	return C.vgio_menu_test_press(c, cbool(shift), cbool(control)) != 0
}

// nativeShown is what the submenu titled submenu of the bar of the last
// nativeBuild shows when it opens, or the bar's one menu when submenu is
// its title: one line per item, its title and "off" for a disabled item.
func nativeShown(submenu string) []string {
	s := C.CString(submenu)
	defer C.free(unsafe.Pointer(s))
	out := C.vgio_menu_test_shown(s)
	if out == nil {
		return nil
	}
	defer C.free(unsafe.Pointer(out))
	shown := C.GoString(out)
	if shown == "" {
		return []string{}
	}
	return strings.Split(shown, "\n")
}

func cbool(b bool) C.int {
	if b {
		return 1
	}
	return 0
}

// vgioMenuChosen is called from the menu target's action method on the AppKit
// main thread when a person chooses one of the declared items. It carries the
// item's tag and nothing else, which is why the bar's routing is a
// package-level registry (see menuActions).
//
//export vgioMenuChosen
func vgioMenuChosen(tag C.int) {
	chooseMenuTag(int(tag))
}
