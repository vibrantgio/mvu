package desktop

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/unit"
)

// roomProbe is a page that records the available room its context holds.
func roomProbe(room *image.Rectangle, set *bool) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		*room, *set = AvailableRoom(gtx)
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}
}

// The cap pushes the page down by the strip, so the page reads the window's
// content area one strip higher in its own coordinates.
func TestCapTopMovesTheAvailableRoomByItsInset(t *testing.T) {
	var (
		room image.Rectangle
		set  bool
	)
	gtx := WithAvailableRoom(bandGtx(400, 300), image.Rect(0, 0, 400, 300))
	CapTop(func() unit.Dp { return 32 }, roomProbe(&room, &set))(gtx)
	if !set {
		t.Fatal("the page's context holds no available room; the cap dropped it")
	}
	if want := image.Rect(0, -32, 400, 268); room != want {
		t.Errorf("the page's available room = %v, want %v: the window's area moved up by the 32 dp inset", room, want)
	}
	if r, _ := AvailableRoom(gtx); r != image.Rect(0, 0, 400, 300) {
		t.Errorf("the caller's available room = %v after the cap, want it unchanged", r)
	}
}

// No strip moves nothing: the page reads the room the window set.
func TestCapTopWithNoStripKeepsTheAvailableRoom(t *testing.T) {
	var (
		room image.Rectangle
		set  bool
	)
	gtx := WithAvailableRoom(bandGtx(400, 300), image.Rect(0, 0, 400, 300))
	CapTop(func() unit.Dp { return 0 }, roomProbe(&room, &set))(gtx)
	if want := image.Rect(0, 0, 400, 300); !set || room != want {
		t.Errorf("the page's available room = %v (set %v), want %v", room, set, want)
	}
}

// A window that sets no room gets none from the cap.
func TestInsetTopSetsNoRoomWhereNoneWasSet(t *testing.T) {
	var (
		room image.Rectangle
		set  bool
	)
	InsetTop(func() unit.Dp { return 32 }, roomProbe(&room, &set))(bandGtx(400, 300))
	if set {
		t.Errorf("the page's context holds the available room %v; no window set one", room)
	}
}

func TestWithoutAvailableRoomRemovesOnlyTheRoom(t *testing.T) {
	gtx := WithAvailableRoom(bandGtx(400, 300), image.Rect(0, 0, 400, 300))
	gtx.Values["other"] = 1
	cleared := WithoutAvailableRoom(gtx)
	if _, ok := AvailableRoom(cleared); ok {
		t.Error("the available room is still set after WithoutAvailableRoom")
	}
	if cleared.Values["other"] != 1 {
		t.Error("WithoutAvailableRoom removed a value that is not the room")
	}
	if _, ok := AvailableRoom(gtx); !ok {
		t.Error("WithoutAvailableRoom wrote the caller's context")
	}
}
