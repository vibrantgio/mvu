package desktop

import (
	"image"

	"github.com/vibrantgio/mvu"
)

// dragKind discriminates the raw events crossing from the native drag
// callbacks.
type dragKind int

const (
	dragEnter dragKind = iota // the drag entered the window with acceptable payload
	dragMove                  // the drag moved within the window
	dragExit                  // the drag left the window, or was cancelled
	dragDrop                  // the drop happened and its payload was read
)

// dragEvent is the raw, zone-agnostic event as it crosses the channel from
// the native side; pos is already in Gio pixels (the transform is applied at
// the callback boundary). paths is set for dragDrop only.
type dragEvent struct {
	kind  dragKind
	pos   image.Point
	paths []string
}

// hoverTracker turns raw drag events into zone-resolved messages. It is the
// per-session state machine the window-level native callbacks do not
// provide: the OS says "the drag moved", the tracker says "zone 0 was
// exited, zone 1 was entered". It lives outside the application's Update —
// the model stays a pure fold over messages — and runs on the drop
// pipeline's single goroutine, so it needs no lock. hit is the resolver, in
// practice [ZoneGroup.Hit] over the last completed frame's rects.
type hoverTracker struct {
	hit func(image.Point) int
	cur int // zone currently hovered, -1 for none
}

func newHoverTracker(hit func(image.Point) int) *hoverTracker {
	return &hoverTracker{hit: hit, cur: -1}
}

// step consumes one raw event and returns the messages it implies, in
// order. Zero messages is the common case: moves within one zone, moves
// within dead space, and dead-space drops — which are silence by design.
func (t *hoverTracker) step(ev dragEvent) []mvu.Message {
	var msgs []mvu.Message
	switch ev.kind {
	case dragEnter, dragMove:
		z := t.hit(ev.pos)
		if z != t.cur {
			if t.cur >= 0 {
				msgs = append(msgs, FilesExited{Zone: t.cur})
			}
			if z >= 0 {
				msgs = append(msgs, FilesEntered{Zone: z})
			}
			t.cur = z
		}
	case dragExit:
		if t.cur >= 0 {
			msgs = append(msgs, FilesExited{Zone: t.cur})
			t.cur = -1
		}
	case dragDrop:
		// A drop ends the hover; the exit precedes the delivery so a model
		// tracking highlight purely from Entered/Exited is left clean. The
		// drop is resolved at the DROP point, not at the last hovered zone —
		// the two agree unless the pointer moved in the final instant, and
		// the drop point is the truth.
		if t.cur >= 0 {
			msgs = append(msgs, FilesExited{Zone: t.cur})
			t.cur = -1
		}
		if z := t.hit(ev.pos); z >= 0 {
			msgs = append(msgs, FilesDropped{Zone: z, Paths: ev.paths, Pos: ev.pos})
		}
	}
	return msgs
}
