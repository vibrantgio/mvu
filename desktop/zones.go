package desktop

// The zone registry is deliberately platform-neutral: hit-testing is integer
// arithmetic over rectangles recorded during layout, with no AppKit in it,
// and an application written against zones keeps the same shape when drop
// support for further platforms arrives. Only the native half of the drop
// path is gated by build tags.

import (
	"image"
	"sync"

	"gioui.org/layout"
	"gioui.org/op"
)

// ZoneGroup tracks file-drop target zones among a fixed set of targets:
// allocate one per window, call [ZoneGroup.Update] every frame before the
// layout pass, register each zone during layout, and hand the group to
// [NewDropTarget] so drops resolve against it. Zones are identified by plain
// index — no address identity is needed, because nothing round-trips through
// Gio's event router.
//
// Usage pattern:
//
//	var z desktop.ZoneGroup
//
//	// each frame:
//	z.Update(gtx)
//	z.Zone(gtx, 0, originA, zoneWidgetA)
//	z.Zone(gtx, 1, originB, zoneWidgetB)
//
// Recording is double-buffered per frame: Zone appends into the current
// frame's slice, Update promotes it to "last completed frame" and starts a
// fresh one. [ZoneGroup.Hit] resolves against the last completed frame's
// set — a drop arrives out-of-band, between frames at best and mid-frame at
// worst, and must see a complete, consistent set; a target that moved within
// one frame of the drop is a non-problem in practice.
//
// The origin parameter on [ZoneGroup.Zone] is forced by Gio itself: a
// layout.Context carries no transform, and no public API answers "where am
// I?" mid-layout, so a widget cannot learn its own absolute rectangle. The
// absolute origin therefore comes from whoever laid the zone out — zones are
// registered from a root that does its own placement math, and that root is
// the one place absolute geometry exists. Callers that already position
// content by other means can skip the layout convenience and record a known
// rectangle directly with [ZoneGroup.Record].
//
// ZoneGroup carries a mutex because its two sides run on different
// goroutines: recording happens on the render goroutine while Hit is called
// from the drop pipeline — drops arriving out-of-band is the whole point of
// the registry. The critical sections are constant-time appends and one
// bounded scan, so no caller is meaningfully blocked.
type ZoneGroup struct {
	mu   sync.Mutex
	last []ZoneRect // last completed frame's zones, in recording order
	cur  []ZoneRect // zones recorded so far this frame
}

// ZoneRect is one recorded zone: its index and its absolute rectangle in Gio
// pixels, window coordinates (upper-left origin) — the same space a drop
// point arrives in, so [ZoneGroup.Hit] needs no further conversion.
type ZoneRect struct {
	Index int
	Rect  image.Rectangle
}

// Update marks the frame boundary: the rects recorded since the previous
// Update become the completed set that [ZoneGroup.Hit] resolves against, and
// recording starts fresh. Call it once per frame, before any Zone call. The
// context is accepted for call-site symmetry with the frame it belongs to;
// today the boundary itself is the only information consumed.
func (z *ZoneGroup) Update(layout.Context) {
	z.mu.Lock()
	z.last, z.cur = z.cur, z.last[:0]
	z.mu.Unlock()
}

// Zone lays out w at origin and records the resulting rectangle for zone i
// into the current frame's set. origin is the zone's absolute position in
// Gio pixels; call Zone from the root coordinate system (identity
// transform), because the offset it pushes composes with any transform
// already in effect while the recorded rectangle does not.
func (z *ZoneGroup) Zone(gtx layout.Context, i int, origin image.Point, w layout.Widget) layout.Dimensions {
	defer op.Offset(origin).Push(gtx.Ops).Pop()
	dims := w(gtx)
	z.Record(i, image.Rectangle{Min: origin, Max: origin.Add(dims.Size)})
	return dims
}

// Record appends one zone rectangle to the current frame's set: the
// primitive under [ZoneGroup.Zone], for callers that position their zone
// content by other means and already know its absolute rectangle in Gio
// pixels.
func (z *ZoneGroup) Record(i int, r image.Rectangle) {
	z.mu.Lock()
	z.cur = append(z.cur, ZoneRect{Index: i, Rect: r})
	z.mu.Unlock()
}

// Hit returns the index of the zone containing p in the last completed
// frame's set, or -1 for dead space. Safe to call from any goroutine.
func (z *ZoneGroup) Hit(p image.Point) int {
	z.mu.Lock()
	defer z.mu.Unlock()
	return resolve(z.last, p)
}

// resolve is the pure hit-test over a recorded rect slice: the TOPMOST zone
// containing p wins, and topmost means recorded last — painter's order.
// Zones are recorded in the order they are laid out, and in Gio later ops
// paint over earlier ones, so the zone drawn on top is exactly the one the
// user believes they are dropping on; the scan therefore runs from the end.
// Containment follows image.Rectangle: Min edges inclusive, Max edges
// exclusive, so two zones sharing an edge never both claim a point on it.
func resolve(rects []ZoneRect, p image.Point) int {
	for i := len(rects) - 1; i >= 0; i-- {
		if p.In(rects[i].Rect) {
			return rects[i].Index
		}
	}
	return -1
}
