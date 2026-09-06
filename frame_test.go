package mvu

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"gioui.org/app"
	"gioui.org/unit"
)

// The frame-memory tests exercise the parts a real window is not needed for:
// the file, the restore decision, and the debounce. What a real window adds is
// where the frames come from — a platform notification on macOS, a frame event
// elsewhere — and both funnel into frameMemory.record, which is what the tests
// below drive directly.

// countingWrites replaces a frameMemory's file with a counter, so a test can
// say how many writes a burst produced rather than inferring it from a file
// that looks the same after one write and after ten.
type countingWrites struct {
	mu    sync.Mutex
	calls int
	last  Frame
}

func (c *countingWrites) write(f Frame) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	c.last = f
	return nil
}

func (c *countingWrites) state() (int, Frame) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls, c.last
}

// TestFrameRoundTripsThroughAFile asserts that a frame written to a file comes
// back out of it unchanged, position included.
func TestFrameRoundTripsThroughAFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "window.json")
	want := Frame{X: -320, Y: 96, Width: 1200, Height: 800}

	if err := saveFrame(path, want); err != nil {
		t.Fatalf("saveFrame: %v", err)
	}
	got, ok := loadFrame(path)
	if !ok {
		t.Fatalf("loadFrame(%q) refused the frame it had just written", path)
	}
	if got != want {
		t.Fatalf("round trip = %+v; want %+v", got, want)
	}
}

// TestRememberedFrameFallsBackToDefaults asserts that every way of having
// nothing usable on disk answers the same "leave the defaults alone": no file,
// bytes that are not JSON, JSON that is not a frame, and a frame too small to
// be a window anyone opened on purpose.
func TestRememberedFrameFallsBackToDefaults(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name    string
		content string // "" means write no file at all
	}{
		{name: "missing"},
		{name: "corrupt", content: "{not json at all"},
		{name: "wrong shape", content: `["1200","800"]`},
		{name: "collapsed", content: `{"x":10,"y":10,"width":0,"height":0}`},
		{name: "too small to grab", content: `{"x":10,"y":10,"width":12,"height":12}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := filepath.Join(dir, c.name+".json")
			if c.content != "" {
				if err := os.WriteFile(path, []byte(c.content), 0o644); err != nil {
					t.Fatalf("writing the case file: %v", err)
				}
			}
			if got, ok := rememberedFrame(path, nil); ok {
				t.Fatalf("rememberedFrame accepted %+v; want the defaults left alone", got)
			}
		})
	}
}

// TestRememberedFrameRejectsAFrameOffEveryScreen asserts that a remembered
// frame is only restored while it still lands on a screen — the screen it was
// written on having been unplugged is exactly the case that would otherwise
// restore a window nobody can see.
func TestRememberedFrameRejectsAFrameOffEveryScreen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "window.json")
	frame := Frame{X: 2000, Y: 100, Width: 1200, Height: 800}
	if err := saveFrame(path, frame); err != nil {
		t.Fatalf("saveFrame: %v", err)
	}
	// The second screen is where the frame was written; only the first is
	// still attached.
	primary := Frame{X: 0, Y: 0, Width: 1920, Height: 1080}

	if _, ok := rememberedFrame(path, []Frame{primary}); ok {
		t.Fatalf("rememberedFrame restored %+v onto screens it no longer lands on", frame)
	}
	second := Frame{X: 1920, Y: 0, Width: 1920, Height: 1080}
	got, ok := rememberedFrame(path, []Frame{primary, second})
	if !ok {
		t.Fatalf("rememberedFrame refused %+v with the screen it lands on attached", frame)
	}
	if got != frame {
		t.Fatalf("rememberedFrame = %+v; want %+v", got, frame)
	}
}

// TestFrameOnScreenMeasuresOverlap pins the rule the restore decision rests
// on: enough of the window on a screen to see it and to grab it, not the whole
// window and not one pixel of it.
func TestFrameOnScreenMeasuresOverlap(t *testing.T) {
	screens := []Frame{{X: 0, Y: 0, Width: 1920, Height: 1080}}
	cases := []struct {
		name string
		f    Frame
		want bool
	}{
		{"wholly on", Frame{X: 100, Y: 100, Width: 800, Height: 600}, true},
		{"hanging off the right edge", Frame{X: 1800, Y: 100, Width: 800, Height: 600}, true},
		{"barely on", Frame{X: 1900, Y: 100, Width: 800, Height: 600}, false},
		{"above the top", Frame{X: 100, Y: -600, Width: 800, Height: 600}, false},
		{"wholly off", Frame{X: 4000, Y: 100, Width: 800, Height: 600}, false},
		{"no screens at all", Frame{X: 100, Y: 100, Width: 800, Height: 600}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			list := screens
			if c.name == "no screens at all" {
				list = nil
			}
			if got := frameOnScreen(c.f, list); got != c.want {
				t.Fatalf("frameOnScreen(%+v) = %v; want %v", c.f, got, c.want)
			}
		})
	}
}

// TestFrameMemoryWritesOnceForABurst asserts the debounce: a drag hands over a
// change per notification and reaches the file once, with the frame the drag
// ended on.
func TestFrameMemoryWritesOnceForABurst(t *testing.T) {
	writes := new(countingWrites)
	memory := &frameMemory{delay: 60 * time.Millisecond, write: writes.write}

	memory.record(Frame{X: 100, Y: 100, Width: 1200, Height: 800}) // the baseline
	last := Frame{}
	for i := 1; i <= 8; i++ {
		last = Frame{X: 100, Y: 100, Width: unit.Dp(1200 + i), Height: 800}
		memory.record(last)
	}

	if calls, _ := writes.state(); calls != 0 {
		t.Fatalf("the burst wrote %d times before the delay had passed; want 0", calls)
	}
	waitForWrites(t, writes, 1)
	calls, got := writes.state()
	if calls != 1 {
		t.Fatalf("the burst wrote %d times; want exactly 1", calls)
	}
	if got != last {
		t.Fatalf("the burst wrote %+v; want the frame it ended on, %+v", got, last)
	}
}

// TestFrameMemoryWritesNothingUntilTheFirstChange asserts that opening the
// window writes nothing: the first frame handed in is the baseline, and a
// frame equal to the baseline — which is most of them, since a frame event
// arrives per rendered frame — is not a change.
func TestFrameMemoryWritesNothingUntilTheFirstChange(t *testing.T) {
	writes := new(countingWrites)
	memory := &frameMemory{delay: 10 * time.Millisecond, write: writes.write}

	opened := Frame{X: 100, Y: 100, Width: 1200, Height: 800}
	for range 5 {
		memory.record(opened)
	}
	time.Sleep(80 * time.Millisecond)
	if calls, _ := writes.state(); calls != 0 {
		t.Fatalf("an unchanged window wrote %d times; want 0", calls)
	}

	moved := Frame{X: 240, Y: 100, Width: 1200, Height: 800}
	memory.record(moved)
	waitForWrites(t, writes, 1)
	if _, got := writes.state(); got != moved {
		t.Fatalf("the first change wrote %+v; want %+v", got, moved)
	}
}

// TestFrameMemoryWritesWhatIsPendingOnStop asserts that a change made inside
// the delay and followed straight away by a quit still reaches the file: the
// window's destruction flushes what is pending rather than dropping it.
func TestFrameMemoryWritesWhatIsPendingOnStop(t *testing.T) {
	writes := new(countingWrites)
	memory := &frameMemory{delay: time.Hour, write: writes.write}

	memory.record(Frame{X: 100, Y: 100, Width: 1200, Height: 800})
	resized := Frame{X: 100, Y: 100, Width: 1400, Height: 900}
	memory.record(resized)

	memory.stop()
	calls, got := writes.state()
	if calls != 1 {
		t.Fatalf("stop wrote %d times; want exactly 1", calls)
	}
	if got != resized {
		t.Fatalf("stop wrote %+v; want %+v", got, resized)
	}

	// Nothing is pending any more, so a second stop writes nothing.
	memory.stop()
	if calls, _ := writes.state(); calls != 1 {
		t.Fatalf("a second stop wrote again, %d writes in total; want 1", calls)
	}
}

// TestRememberFrameReachesTheFileItWasGiven walks the whole path a change
// takes — the options the call was given, the memory the window holds, the
// file the change lands in — without a native window, by handing the change to
// the memory the call attached.
func TestRememberFrameReachesTheFileItWasGiven(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vaultview", "window.json")
	w := NewWindow(app.Title("frame memory"))
	if err := RememberFrame(w, "", FrameOptions{Path: path, Delay: time.Hour}); err != nil {
		t.Fatalf("RememberFrame: %v", err)
	}
	memory := w.frame.Load()
	if memory == nil {
		t.Fatal("RememberFrame attached no frame memory to the window")
	}

	memory.record(Frame{X: 10, Y: 10, Width: 1200, Height: 800}) // the baseline
	moved := Frame{X: 300, Y: 220, Width: 1024, Height: 768}
	memory.record(moved)
	w.stopFrameMemory()

	got, ok := loadFrame(path)
	if !ok {
		t.Fatalf("nothing readable at %q after the window was destroyed", path)
	}
	if got != moved {
		t.Fatalf("the file holds %+v; want %+v", got, moved)
	}
	if w.frame.Load() != nil {
		t.Fatal("the destroyed window still holds its frame memory")
	}
}

// TestRememberFrameRefusesWhatItCannotResolve pins the two ways the call is
// wrong rather than merely unlucky.
func TestRememberFrameRefusesWhatItCannotResolve(t *testing.T) {
	w := NewWindow(app.Title("frame memory"))
	if err := RememberFrame(w, ""); err == nil {
		t.Fatal("RememberFrame with no application name and no path returned no error")
	}
	if err := RememberFrame(w, "vaultview", FrameOptions{}, FrameOptions{}); err == nil {
		t.Fatal("RememberFrame with two FrameOptions returned no error")
	}
	if err := RememberFrame(nil, "vaultview"); err == nil {
		t.Fatal("RememberFrame with no window returned no error")
	}
}

// TestFramePathSitsUnderTheApplicationName asserts the file lands beside the
// application's other configuration rather than anywhere of its own.
func TestFramePathSitsUnderTheApplicationName(t *testing.T) {
	got, err := framePath("vaultview")
	if err != nil {
		t.Fatalf("framePath: %v", err)
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		t.Skipf("no OS config directory here: %v", err)
	}
	want := filepath.Join(dir, "vaultview", "window.json")
	if got != want {
		t.Fatalf("framePath = %q; want %q", got, want)
	}
}

// TestFrameFromPixelsConvertsWithTheMetric asserts that a size read off a
// frame event is stored in dp, so the same window remembered on a dense screen
// and reopened on a coarse one is the same physical size.
func TestFrameFromPixelsConvertsWithTheMetric(t *testing.T) {
	got := frameFromPixels(2400, 1600, unit.Metric{PxPerDp: 2})
	want := Frame{Width: 1200, Height: 800}
	if got != want {
		t.Fatalf("frameFromPixels at 2x = %+v; want %+v", got, want)
	}
	// A metric that has not been filled in yet must not divide the size away.
	if got := frameFromPixels(1200, 800, unit.Metric{}); got != want {
		t.Fatalf("frameFromPixels with an empty metric = %+v; want %+v", got, want)
	}
}

// waitForWrites waits for the debounce to have produced want writes, rather
// than sleeping for a fixed stretch a loaded machine can overrun.
func waitForWrites(t *testing.T, writes *countingWrites, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if calls, _ := writes.state(); calls >= want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	calls, _ := writes.state()
	t.Fatalf("waited for %d writes, saw %d", want, calls)
}
