package desktop

import (
	"image"
	"reflect"
	"sync"
	"testing"
	"time"

	"gioui.org/app"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/mvu"
)

// The tests below exercise the drop pipeline without a native window, the
// way the chrome's seam tests do: everything from the raw-event channel to
// the message stream is platform-neutral, so a fake raw event driven through
// sendRaw proves the goroutine plumbing on every platform. What the native
// callbacks themselves do needs a live window and a real drag, and is proven
// by the manual script.

// collectMessages subscribes d.Messages and returns a snapshot func plus a
// func reporting whether the stream completed.
func collectMessages(t *testing.T, d *DropTarget) (func() []mvu.Message, func() bool) {
	t.Helper()
	var mu sync.Mutex
	var seen []mvu.Message
	var done bool
	d.Messages().Subscribe(rx.GoroutineContext(), func(next mvu.Message, err error, complete bool) {
		mu.Lock()
		defer mu.Unlock()
		if complete {
			done = true
			return
		}
		seen = append(seen, next)
	})
	snapshot := func() []mvu.Message {
		mu.Lock()
		defer mu.Unlock()
		return append([]mvu.Message(nil), seen...)
	}
	completed := func() bool {
		mu.Lock()
		defer mu.Unlock()
		return done
	}
	return snapshot, completed
}

// await polls cond until it holds or the timeout expires.
func await(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(time.Millisecond)
	}
}

// TestDropPipelineDeliversResolvedMessages drives fake raw events through
// the target's non-blocking send — the same door the native callbacks use —
// and asserts the zone-resolved messages arrive on the stream in order, and
// that closing the target completes the stream so a consuming loop can end.
func TestDropPipelineDeliversResolvedMessages(t *testing.T) {
	w := mvu.NewWindow(app.Title("drop pipeline test"))
	zones := &ZoneGroup{}
	zones.Record(0, image.Rect(0, 0, 100, 100))
	zones.Record(1, image.Rect(200, 0, 300, 100))
	zones.Update(zeroGtx()) // promote to the completed frame Hit resolves against

	d := NewDropTarget(w, zones)
	snapshot, completed := collectMessages(t, d)

	d.sendRaw(dragEvent{kind: dragEnter, pos: image.Pt(50, 50)})
	d.sendRaw(dragEvent{kind: dragMove, pos: image.Pt(250, 50)})
	d.sendRaw(dragEvent{kind: dragDrop, pos: image.Pt(250, 50), paths: []string{"/tmp/a", "/tmp/b"}})

	want := []mvu.Message{
		FilesEntered{Zone: 0},
		FilesExited{Zone: 0},
		FilesEntered{Zone: 1},
		FilesExited{Zone: 1},
		FilesDropped{Zone: 1, Paths: []string{"/tmp/a", "/tmp/b"}, Pos: image.Pt(250, 50)},
	}
	await(t, "the resolved messages", func() bool { return len(snapshot()) >= len(want) })
	if got := snapshot(); !reflect.DeepEqual(got, want) {
		t.Fatalf("messages = %#v, want %#v", got, want)
	}

	// A dead-space drop after the sequence produces nothing further.
	d.sendRaw(dragEvent{kind: dragDrop, pos: image.Pt(150, 50), paths: []string{"/tmp/c"}})

	// Destroying the window ends the pipeline: the stream completes.
	d.close()
	await(t, "stream completion", completed)
	if got := snapshot(); !reflect.DeepEqual(got, want) {
		t.Fatalf("messages after dead-space drop = %#v, want unchanged %#v", got, want)
	}
}

// TestDropTargetCloseIsIdempotent pins that the teardown path can run twice —
// the detach event and the window's destruction both reach it — without a
// double-close panic.
func TestDropTargetCloseIsIdempotent(t *testing.T) {
	w := mvu.NewWindow(app.Title("drop close test"))
	d := NewDropTarget(w, &ZoneGroup{})
	_, completed := collectMessages(t, d)
	d.close()
	d.close()
	await(t, "stream completion", completed)
}

// TestNewDropTargetKinds pins the construction contract: no kinds means the
// file-URL kind, and a kind this package does not implement fails loudly at
// construction rather than as silently refused drags.
func TestNewDropTargetKinds(t *testing.T) {
	w := mvu.NewWindow(app.Title("drop kinds test"))

	d := NewDropTarget(w, &ZoneGroup{})
	if len(d.kinds) != 1 || d.kinds[0] != FileURLs {
		t.Fatalf("default kinds = %v, want [%s]", d.kinds, FileURLs)
	}

	defer func() {
		if recover() == nil {
			t.Fatal("NewDropTarget accepted an unimplemented kind; want panic")
		}
	}()
	NewDropTarget(w, &ZoneGroup{}, "image/png")
}

// TestNewDropTargetRequiresZones pins the nil-zones panic: without a
// registry every drop would resolve to dead space and the target would be
// silent forever, which is a construction error, not a runtime condition.
func TestNewDropTargetRequiresZones(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("NewDropTarget accepted a nil ZoneGroup; want panic")
		}
	}()
	NewDropTarget(mvu.NewWindow(app.Title("drop zones test")), nil)
}

// TestSendRawEvictsOldestWhenFull pins the overflow policy at the native
// boundary: the send never blocks, and when the buffer is full the OLDEST
// event is evicted — stale hover positions are superseded by every later
// event, and the newest event may be the drop itself, the one that must not
// be lost behind a stalled pipeline.
func TestSendRawEvictsOldestWhenFull(t *testing.T) {
	// A bare target whose tracker goroutine is deliberately not running, so
	// the channel actually fills.
	d := &DropTarget{raw: make(chan dragEvent, rawBuffer)}
	for i := 0; i < rawBuffer; i++ {
		d.sendRaw(dragEvent{kind: dragMove, pos: image.Pt(i, 0)})
	}
	drop := dragEvent{kind: dragDrop, pos: image.Pt(999, 0), paths: []string{"/tmp/kept"}}
	done := make(chan struct{})
	go func() {
		d.sendRaw(drop) // must not block despite the full buffer
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("sendRaw blocked on a full buffer")
	}

	// Drain: the drop must still be there; the evicted event is the oldest.
	var got []dragEvent
	for {
		select {
		case ev := <-d.raw:
			got = append(got, ev)
			continue
		default:
		}
		break
	}
	if len(got) != rawBuffer {
		t.Fatalf("buffer held %d events, want %d", len(got), rawBuffer)
	}
	if got[0].pos.X != 1 {
		t.Fatalf("oldest surviving event at x=%d, want x=1 (event 0 evicted)", got[0].pos.X)
	}
	if last := got[len(got)-1]; last.kind != dragDrop || len(last.paths) != 1 {
		t.Fatalf("newest event = %#v, want the drop", last)
	}
}
