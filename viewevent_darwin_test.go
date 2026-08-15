//go:build darwin

package mvu

import "gioui.org/app"

// The view-event tests drive forwardViewEvent with real platform events, since
// app.ViewEvent's unexported method keeps test fakes out of the interface. This
// file picks the concrete type for the platform the tests compile on; the
// contract under test is the same everywhere.

// makeViewEvent returns a valid view event whose identity encodes id (id must
// be nonzero, or the event would be the invalid zero value).
func makeViewEvent(id uintptr) app.ViewEvent {
	return app.AppKitViewEvent{View: id, Layer: 0x2}
}

// invalidViewEvent returns the platform's zero event — the detach signal.
func invalidViewEvent() app.ViewEvent {
	return app.AppKitViewEvent{}
}

// viewIDOf recovers the identity that makeViewEvent encoded.
func viewIDOf(ev app.ViewEvent) uintptr {
	return ev.(app.AppKitViewEvent).View
}
