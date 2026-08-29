//go:build !darwin

package desktop

import "testing"

// Away from macOS the menu seam must be inert: declaring a bar remembers the
// items, touches nothing native and posts nothing. The application declares
// its menu unconditionally and relies on its in-window chords here, so what
// matters is that the declaration costs nothing rather than that it does
// something.
func TestTheMenuSeamIsInertOffMacOS(t *testing.T) {
	m := NewMenuBar(testWindow(t),
		MenuItem{Menu: "File", Title: "New Chat", Key: "n", Msg: menuNew{}},
		MenuItem{Title: "Settings…", Key: ",", Msg: menuSettings{}},
	)
	m.install()

	if got := len(m.items); got != 2 {
		t.Fatalf("the bar remembered %d items, want the 2 declared", got)
	}
	select {
	case msg := <-m.msgs:
		t.Fatalf("the inert seam posted %T", msg)
	default:
	}
}
