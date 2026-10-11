//go:build !darwin

package desktop

import (
	"testing"
	"time"
)

// What this can prove, and what it cannot. Away from macOS the close is one
// line — Gio's own close action on the window — so there is no package
// behaviour left to assert: the window leaving the screen and the destroy
// event that ends the run are Gio's own, verified where Gio is. What is
// worth holding is that this package adds nothing of its own on the way:
// the ask is handed over at once, from the caller's own goroutine, with no
// native hop to wait on and nothing to go wrong before there is a window —
// Gio keeps an action asked for that early and performs it when the window
// arrives.
func TestTheCloseIsHandedToGio(t *testing.T) {
	w := testWindow(t)

	done := make(chan struct{})
	go func() {
		closeWindow(w)
		closeWindow(w) // asking twice is the application's right
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("closeWindow blocked; the stub must hand the ask over and return")
	}
}
