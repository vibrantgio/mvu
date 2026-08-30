package mvu

import (
	"testing"

	"gioui.org/op"
)

// TestCollectorRoundTripsMessageOpsWithinFrame exercises the collector seam
// that Window.Render's FrameEvent path depends on: while a collector is
// registered for a given *op.Ops, every MessageOp.Add(ops) made during layout
// must append to that frame's slice, so the messages can be forwarded to
// Messages() after Frame() returns — the mechanism that lets an interactive
// callback emit mvu.MessageOp{...}.Add(gtx.Ops) and have the model update on
// the same frame.
func TestCollectorRoundTripsMessageOpsWithinFrame(t *testing.T) {
	ops := new(op.Ops)
	var frame []MessageOp
	registerCollector(ops, &frame)

	// Simulate three callbacks firing during one layout pass.
	MessageOp{Message: "a"}.Add(ops)
	MessageOp{Message: "b"}.Add(ops)
	MessageOp{Message: "c"}.Add(ops)

	unregisterCollector(ops)

	if len(frame) != 3 {
		t.Fatalf("collector captured %d MessageOps; want 3", len(frame))
	}
	want := []string{"a", "b", "c"}
	for i, w := range want {
		got, ok := frame[i].Message.(string)
		if !ok || got != w {
			t.Errorf("frame[%d] = %v; want %q (order must be preserved)", i, frame[i].Message, w)
		}
	}
}

// TestCollectorDropsAddsAfterUnregister confirms that MessageOp.Add is a safe
// no-op once the frame's collector has been removed — an Add arriving outside a
// frame (or against a stale *op.Ops) must neither panic nor leak into the next
// frame's slice.
func TestCollectorDropsAddsAfterUnregister(t *testing.T) {
	ops := new(op.Ops)
	var frame []MessageOp
	registerCollector(ops, &frame)
	unregisterCollector(ops)

	MessageOp{Message: "late"}.Add(ops) // must be dropped, not panic.

	if len(frame) != 0 {
		t.Fatalf("collector captured %d MessageOps after unregister; want 0", len(frame))
	}
}

// TestCollectorIsolatesPerOps confirms two concurrently-registered *op.Ops
// keys collect independently — a MessageOp added against one frame's ops must
// not leak into another's. Window.Render allocates one *op.Ops per loop, but
// the map-keyed design must not cross-contaminate if that ever changes.
func TestCollectorIsolatesPerOps(t *testing.T) {
	opsA, opsB := new(op.Ops), new(op.Ops)
	var frameA, frameB []MessageOp
	registerCollector(opsA, &frameA)
	registerCollector(opsB, &frameB)

	MessageOp{Message: "a1"}.Add(opsA)
	MessageOp{Message: "b1"}.Add(opsB)
	MessageOp{Message: "a2"}.Add(opsA)

	unregisterCollector(opsA)
	unregisterCollector(opsB)

	if len(frameA) != 2 || len(frameB) != 1 {
		t.Fatalf("isolation broken: frameA=%d (want 2), frameB=%d (want 1)", len(frameA), len(frameB))
	}
}
