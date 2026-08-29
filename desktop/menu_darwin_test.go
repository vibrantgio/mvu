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
