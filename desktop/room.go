package desktop

import (
	"image"
	"maps"

	"gioui.org/layout"
)

// availableRoomKey is the key the available room is set under in
// layout.Context.Values. This file is the one place that names it.
const availableRoomKey = "github.com/vibrantgio/mvu/desktop.AvailableRoom"

// WithAvailableRoom returns gtx with r set as the available room, in gtx's
// own coordinates: the rectangle a floating surface laid out under gtx may be
// placed in. A window sets it to its content area before it lays out what it
// holds.
//
// The room is passed on the context because Gio offers no readback of the
// transform an ancestor pushed. Every layout between the window and a
// floating surface moves the room by the offset it pushes, with
// [AvailableRoomAt]. [InsetTop] and [CapTop] move it by the inset they push.
// gtx.Values is cloned rather than written, so the caller's context keeps
// the room it had.
func WithAvailableRoom(gtx layout.Context, r image.Rectangle) layout.Context {
	values := maps.Clone(gtx.Values)
	if values == nil {
		values = make(map[string]any, 1)
	}
	values[availableRoomKey] = r
	gtx.Values = values
	return gtx
}

// AvailableRoom reads the available room in gtx's own coordinates. It reports
// false when no window set the room on gtx.
func AvailableRoom(gtx layout.Context) (image.Rectangle, bool) {
	r, ok := gtx.Values[availableRoomKey].(image.Rectangle)
	return r, ok
}

// AvailableRoomAt returns gtx with the available room moved into the
// coordinates of a child placed at off: the room less off. A layout calls it
// with the offset it pushes before it lays the child out. Where gtx has no
// room set, or off is zero, it returns gtx unchanged.
func AvailableRoomAt(gtx layout.Context, off image.Point) layout.Context {
	if off == (image.Point{}) {
		return gtx
	}
	if r, ok := AvailableRoom(gtx); ok {
		return WithAvailableRoom(gtx, r.Sub(off))
	}
	return gtx
}

// WithoutAvailableRoom returns gtx with no available room set. A layout
// calls it for a child whose offset it decides only after it measures the
// child: the room cannot be moved by that offset yet. gtx.Values is cloned
// rather than written.
func WithoutAvailableRoom(gtx layout.Context) layout.Context {
	if _, ok := gtx.Values[availableRoomKey]; !ok {
		return gtx
	}
	values := maps.Clone(gtx.Values)
	delete(values, availableRoomKey)
	gtx.Values = values
	return gtx
}
