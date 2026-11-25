//go:build darwin

package desktop

import (
	"testing"

	"gioui.org/app"
)

// What this can prove, and what it cannot. A test binary has no
// NSApplication, so nothing here amends a real menu bar: the native entry
// points find no application and return. What it does prove is that the whole
// darwin path is real — the Objective-C compiles and links into the test
// binary, the declaration crosses the boundary for every item, the amendment
// is asked for at declaration time and again on the window's configuration
// notification, and none of it blocks or crashes without an application
// behind it. That the bar actually grows the items is verified by running the
// application, which is the only place a menu bar exists.
func TestTheDarwinMenuPathIsSafeWithoutAnApplication(t *testing.T) {
	w := testWindow(t)
	m := NewMenuBar(w,
		MenuItem{Menu: "File", Title: "New Chat", Key: "n", Msg: menuNew{}},
		MenuItem{Title: "Settings…", Key: ",", Msg: menuSettings{}},
		MenuItem{Menu: "View", Title: "Hide/Show Conversations", Key: "\\", Msg: menuToggle{}},
	)

	// The configuration notification is the seam the amendment rides on, and
	// an Option call is what raises it outside a running application.
	w.Option(app.Title("menu darwin test retitled"))

	// Re-declaring is the supersede path: reset, declare, apply, all again.
	m.install()

	if got := len(m.tags); got != 3 {
		t.Fatalf("the bar carries %d tags, want one per declared item (3)", got)
	}
	select {
	case msg := <-m.msgs:
		t.Fatalf("the native path posted %T with no application and nobody clicking", msg)
	default:
	}
}

// The declaration's native copy follows Set: the title and the disabled
// state that validateMenuItem: gives the bar are the ones Set gave last. A
// bar that a later bar superseded has no native items, so its Set changes
// nothing there.
func TestTheNativeDeclarationFollowsSet(t *testing.T) {
	m := NewMenuBar(testWindow(t),
		MenuItem{Menu: "Edit", Title: "Undo", Key: "z", Disabled: true, Msg: menuNew{}},
		MenuItem{Menu: "Edit", Title: "Redo", Key: "Z", Msg: menuSettings{}},
	)
	check := func(i int, title string, disabled bool) {
		t.Helper()
		got, d, ok := nativeItem(m.tags[i])
		if !ok {
			t.Fatalf("the native declaration holds no item %d", i)
		}
		if got != title || d != disabled {
			t.Fatalf("native item %d is %q disabled=%v, want %q disabled=%v", i, got, d, title, disabled)
		}
	}
	check(0, "Undo", true)
	check(1, "Redo", false)
	m.Set(0, "Undo Delete Card", false)
	m.Set(1, "Redo", true)
	check(0, "Undo Delete Card", false)
	check(1, "Redo", true)

	next := NewMenuBar(testWindow(t), MenuItem{Menu: "Edit", Title: "Undo", Msg: menuNew{}})
	m.Set(0, "Undo Add Card", false)
	if _, _, ok := nativeItem(m.tags[0]); ok {
		t.Fatal("the superseded bar's item is still in the native declaration")
	}
	if got, _, _ := nativeItem(next.tags[0]); got != "Undo" {
		t.Fatalf("the superseded bar's Set changed the next bar's item to %q", got)
	}
}

// menuKey is one key press on a menu, and whether the bar takes it.
type menuKey struct {
	menu, chars string
	shift       bool
	takes       bool
}

// A menu that this package creates takes the key equivalent of an enabled
// item, with Shift for an uppercase letter. A disabled item does not take
// its key, so the key goes to the window.
func TestADisabledItemGivesItsKeyToTheWindow(t *testing.T) {
	m := NewMenuBar(testWindow(t),
		MenuItem{Menu: "Edit", Title: "Undo", Key: "z", Disabled: true, Msg: menuNew{}},
		MenuItem{Menu: "Edit", Title: "Redo", Key: "Z", Disabled: true, Msg: menuSettings{}},
		MenuItem{Menu: "View", Title: "Toggle", Key: "\\", Msg: menuToggle{}},
	)
	check := func(keys []menuKey) {
		t.Helper()
		for _, k := range keys {
			if got := nativeTakesKey(k.menu, k.chars, k.shift); got != k.takes {
				t.Errorf("%s %q shift=%v: the bar takes the key %v, want %v", k.menu, k.chars, k.shift, got, k.takes)
			}
		}
	}
	check([]menuKey{
		{"Edit", "z", false, false},
		{"Edit", "Z", true, false},
		{"View", "\\", false, true},
	})
	m.Set(0, "Undo Delete Card", false)
	check([]menuKey{
		{"Edit", "z", false, true},
		{"Edit", "Z", true, false},
	})
	m.Set(0, "Undo", true)
	m.Set(1, "Redo Add Card", false)
	check([]menuKey{
		{"Edit", "z", false, false},
		{"Edit", "Z", true, true},
	})
}
