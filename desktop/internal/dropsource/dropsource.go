// Package dropsource stands in for a dragging source so the drop half of
// mvu/desktop can be exercised on a real window by its own tests.
//
// It exists because a drag cannot be driven from a Go test. Everything a drag
// source does — putting a file URL on the drag pasteboard, putting a file
// promise on it with a delegate that writes the file when asked, and handing
// the view the dragging info a drag presents — is Objective-C, and Go does not
// support cgo in test files. Without this package the choice would be between
// shipping that scaffolding in the package's own native code and not driving
// the native path at all; an internal package is neither, and nothing outside
// this module can see it.
//
// What a stand-in can and cannot do, measured on macOS 26: the view's drag
// destination calls — draggingEntered:, draggingExited: and
// performDragOperation: — answer a stand-in exactly as they answer a real
// drag, so a drop can be driven end to end. The fulfilment of a file promise
// cannot: a promise written to the drag pasteboard in the receiving process is
// read back as a promise receiver, which asks the provider for its file name
// and then never asks for the file, because the writing belongs to a live drag
// session no stand-in has. So a promised file's bytes arrive only from a real
// drag out of a real application.
//
// Away from macOS every call reports that it did nothing.
package dropsource

// Promise is a file a source does not hand over but offers to write: the name
// it will write under and the bytes it will write. It is what an application
// keeping its files in a library of its own drags.
type Promise struct {
	Name  string
	Bytes []byte
}

// Begin puts url (a file path, empty for none) and promise (nil for none) on
// the drag pasteboard and makes that the drag standing over the window. One
// drag stands at a time: a second Begin replaces the first.
//
// Call this and the three calls below on the platform's main thread: they
// touch AppKit throughout.
func Begin(url string, promise *Promise) error { return begin(url, promise) }

// Enter sends the standing drag's draggingEntered: to the frontmost window's
// content view, at x, y in window coordinates, and reports whether the view
// answered with a drag operation rather than refusing the drag.
func Enter(x, y float64) (bool, error) { return send(enterDrag, x, y) }

// Exit sends the standing drag's draggingExited: to that view.
func Exit() error { _, err := send(exitDrag, 0, 0); return err }

// Drop sends the standing drag's performDragOperation: to that view, at x, y
// in window coordinates, and reports whether the view accepted the drop.
func Drop(x, y float64) (bool, error) { return send(dropDrag, x, y) }

// The three calls a drag makes on a drop target.
type call int

const (
	enterDrag call = iota
	exitDrag
	dropDrag
)
