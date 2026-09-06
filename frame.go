package mvu

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gioui.org/app"
	"gioui.org/unit"
)

// Frame is a window's size and position together, in device-independent
// pixels. X and Y are the position: the top-left corner of the window,
// measured from the top-left corner of the primary screen, y growing
// downwards, so a window on a screen above or left of the primary one has a
// negative coordinate. Width and Height are the size.
//
// The unit is dp on every platform, which is what makes a remembered frame
// survive a move between screens of different pixel densities: the same dp
// frame is the same physical rectangle on both.
type Frame struct {
	X      unit.Dp `json:"x"`
	Y      unit.Dp `json:"y"`
	Width  unit.Dp `json:"width"`
	Height unit.Dp `json:"height"`
}

// FrameOptions adjusts [RememberFrame]. The zero value is the ordinary case
// and every field may be left out.
type FrameOptions struct {
	// Path is the file the frame is kept in. Empty resolves the OS-appropriate
	// path from the application name, which is what an application wants; a
	// test points this at a temporary directory instead.
	Path string
	// Delay is how long a change must stand still before it is written. Zero
	// is the default delay. It exists so a test does not have to wait out the
	// real one.
	Delay time.Duration
}

// frameDelay is how long the window's frame must stand still before it is
// written. A drag of a window's edge produces a change per frame; half a
// second is past the end of a drag and still short enough that a user who
// resizes and immediately quits keeps the new frame — and a quit that beats
// it still writes, because the window's destruction flushes what is pending.
const frameDelay = 500 * time.Millisecond

// frameFile is the name the frame is kept under, inside the application's own
// directory in the OS config directory.
const frameFile = "window.json"

// frameMinVisible is how much of the window must land on a screen, on both
// axes, for a remembered frame to be worth restoring: enough to see it and to
// grab it with the pointer. It is also the smallest size a remembered frame
// may claim, so a file recording a collapsed or zero window is treated as
// damaged rather than obeyed.
const frameMinVisible = unit.Dp(64)

// RememberFrame makes w remember its frame — its size and, where the platform
// can tell, its position — across launches, under appName. Call it once at
// window construction, before the window renders:
//
//	w := mvu.NewWindow(app.Title("Vaultview"), app.Size(1200, 800))
//	if err := mvu.RememberFrame(w, "vaultview"); err != nil { ... }
//
// The options passed to [NewWindow] stay the application's defaults. A
// remembered frame is applied over them before the first frame; with no file,
// an unreadable or damaged file, or a remembered frame that no longer lands
// on any screen, the defaults stand untouched and the file is replaced by the
// next change the user makes.
//
// Nothing is written until then: the frame the window opens with is the
// baseline, and only a change away from it starts the clock. Changes are
// written after they stand still (see [FrameOptions].Delay), so dragging an
// edge writes once at the end of the drag rather than once per frame, and
// whatever is still pending is written when the window is destroyed.
//
// The file is JSON in the OS config directory, under the application's name,
// the same directory a preferences file resolves to:
//
//   - darwin:  ~/Library/Application Support/<appName>/window.json
//   - linux:   $XDG_CONFIG_HOME/<appName>/window.json (or ~/.config/...)
//   - windows: %AppData%\<appName>\window.json
//
// Position is remembered on macOS, where it is read from and applied to the
// native window; every other platform remembers the size alone, because
// neither Gio's [app.Config] nor any option carries a window position there.
// A frame written on macOS and read on another platform is not wrong, it is
// simply half used: the size is applied and the position ignored.
//
// At most one FrameOptions may be given; more than one is a programming error
// and is reported as such.
func RememberFrame(w *Window, appName string, options ...FrameOptions) error {
	if w == nil {
		return errors.New("mvu: RememberFrame needs a window")
	}
	if len(options) > 1 {
		return fmt.Errorf("mvu: RememberFrame takes at most one FrameOptions, got %d", len(options))
	}
	var opts FrameOptions
	if len(options) == 1 {
		opts = options[0]
	}
	path := opts.Path
	if path == "" {
		var err error
		if path, err = framePath(appName); err != nil {
			return err
		}
	}
	delay := opts.Delay
	if delay <= 0 {
		delay = frameDelay
	}

	memory := &frameMemory{path: path, delay: delay}
	memory.write = func(f Frame) error { return saveFrame(memory.path, f) }

	remembered, restore := rememberedFrame(path, platformScreens())
	if restore {
		// Applied before the event loop starts, so Gio folds it into the
		// window's initial configuration and the first frame is already the
		// remembered size — no visible jump from the default size to this one.
		w.Option(app.Size(remembered.Width, remembered.Height))
	}
	w.rememberFrame(memory, remembered, restore)
	return nil
}

// framePath resolves the frame file for appName. It creates nothing.
func framePath(appName string) (string, error) {
	if appName == "" {
		return "", errors.New("mvu: RememberFrame needs an application name")
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, appName, frameFile), nil
}

// rememberedFrame reads path and answers whether what it found is worth
// applying. Every way of having nothing to apply — no file, an unreadable
// one, damaged JSON, an implausible frame, a frame that lands on none of the
// screens — answers the same false, because they all mean the same thing to
// the caller: leave the application's own defaults alone.
//
// An empty screens list means the platform cannot say where the screens are,
// and the position is then taken on trust; the platforms that cannot say are
// the ones that never restore a position anyway.
func rememberedFrame(path string, screens []Frame) (Frame, bool) {
	f, ok := loadFrame(path)
	if !ok {
		return Frame{}, false
	}
	if len(screens) > 0 && !frameOnScreen(f, screens) {
		return Frame{}, false
	}
	return f, true
}

// loadFrame reads one frame from path. A frame whose size is below
// frameMinVisible on either axis is damaged, not small: no window is opened
// that size on purpose, and obeying it would hand the user a window they
// cannot see or grab.
func loadFrame(path string) (Frame, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Frame{}, false
	}
	var f Frame
	if err := json.Unmarshal(data, &f); err != nil {
		return Frame{}, false
	}
	if !f.plausible() {
		return Frame{}, false
	}
	return f, true
}

// saveFrame writes f to path, creating the intermediate directories.
func saveFrame(path string, f Frame) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// plausible reports whether f could describe a real window: a finite
// rectangle at least frameMinVisible across on both axes.
func (f Frame) plausible() bool {
	for _, v := range []unit.Dp{f.X, f.Y, f.Width, f.Height} {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return false
		}
	}
	return f.Width >= frameMinVisible && f.Height >= frameMinVisible
}

// frameOnScreen reports whether f still lands on one of screens: it overlaps
// that screen by at least frameMinVisible on both axes. Overlap rather than
// containment, because a window the user deliberately hangs off an edge is a
// window they still want back where they left it; the threshold is what keeps
// a window that a departed screen took with it from coming back invisible.
func frameOnScreen(f Frame, screens []Frame) bool {
	if !f.plausible() {
		return false
	}
	for _, s := range screens {
		wide := min(f.X+f.Width, s.X+s.Width) - max(f.X, s.X)
		high := min(f.Y+f.Height, s.Y+s.Height) - max(f.Y, s.Y)
		if wide >= frameMinVisible && high >= frameMinVisible {
			return true
		}
	}
	return false
}

// frameMemory holds one window's remembered frame between the changes that
// arrive and the file they eventually reach.
//
// Changes arrive from two directions and neither is on a schedule of its own:
// on macOS from the native window's own move and resize notifications, on the
// other platforms from the size carried by each frame event. Both call
// [frameMemory.record], which is why the debounce lives here rather than at
// either source.
type frameMemory struct {
	path  string
	delay time.Duration
	// write is the one seam this type has. It is a field so a test can count
	// the writes a burst produces without going near a file system.
	write func(Frame) error

	mu sync.Mutex
	// current is the last frame handed in, baseline included; pending says
	// whether it still has to reach the file. haveCurrent separates the
	// baseline — the frame the window opened with, which is recorded and
	// never written — from every later change.
	current     Frame
	haveCurrent bool
	pending     bool
	timer       *time.Timer
}

// record takes in one observed frame. The first one is the baseline and
// writes nothing: it is the frame the window opened with, which is either
// what the file already holds or the application's own default, and neither
// is news. A frame equal to the last one is likewise nothing to write —
// frames arrive per rendered frame on the platforms that read the size from
// one, so most of them repeat.
//
// Anything else restarts the delay, so a burst — a drag of an edge, a slide
// across the desktop — collapses into the one write that follows it.
func (m *frameMemory) record(f Frame) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.haveCurrent {
		m.current, m.haveCurrent = f, true
		return
	}
	if f == m.current {
		return
	}
	m.current = f
	m.pending = true
	if m.timer == nil {
		m.timer = time.AfterFunc(m.delay, m.flush)
		return
	}
	m.timer.Reset(m.delay)
}

// flush writes what is pending, if anything. It runs on the timer's goroutine
// and, through [frameMemory.stop], on the render goroutine as the window is
// destroyed; a failed write is dropped rather than retried, because a frame
// nobody could store is not worth a second window's worth of attention.
func (m *frameMemory) flush() {
	m.mu.Lock()
	pending, f := m.pending, m.current
	m.pending = false
	m.mu.Unlock()
	if !pending {
		return
	}
	_ = m.write(f)
}

// stop writes whatever is pending and stops the timer. The window is going
// away, and a resize the user made inside the delay is exactly the change
// they most expect to find again.
func (m *frameMemory) stop() {
	m.mu.Lock()
	if m.timer != nil {
		m.timer.Stop()
	}
	m.mu.Unlock()
	m.flush()
}

// frameFromPixels restates a window size given in device pixels as dp, which
// is what a frame is stored in. Sub-dp differences are rounded away: they are
// a pixel grid showing through, not a change the user made, and writing on
// them would defeat the debounce.
func frameFromPixels(width, height int, metric unit.Metric) Frame {
	scale := metric.PxPerDp
	if scale <= 0 {
		scale = 1
	}
	return Frame{
		Width:  roundDp(float64(width) / float64(scale)),
		Height: roundDp(float64(height) / float64(scale)),
	}
}

// roundDp rounds a measured length to whole dp.
func roundDp(v float64) unit.Dp {
	return unit.Dp(math.Round(v))
}

// rememberFrame attaches memory to the window and, where the platform has a
// position to restore and report, starts that half once the native window
// exists.
//
// The first configuration notification is the earliest moment the native
// window is there to be moved; the goroutine is not an optimisation but a
// requirement. The platform half reaches the native window through the
// window's own main-thread door, and during the first-frame notification the
// main thread is parked in Gio's frame handoff — a synchronous hop from the
// notifying goroutine would deadlock against it, and the hop from a goroutine
// of its own simply waits for the frame to finish.
func (w *Window) rememberFrame(memory *frameMemory, remembered Frame, restore bool) {
	w.frame.Store(memory)
	if !platformOwnsPosition() {
		return
	}
	var once sync.Once
	w.OnConfigure(func() {
		once.Do(func() { go platformRememberFrame(w, memory, remembered, restore) })
	})
}

// recordFrame takes the size out of one frame event, in device pixels, and
// hands it to the window's frame memory. It runs on every frame of every
// window, so the nil check ahead of everything else is what keeps a window
// that never asked to be remembered from paying for the feature.
//
// Where the platform reports the whole frame itself this does nothing: a size
// with no position would overwrite the position that memory holds with a
// zero, and the platform's own reports carry both.
func (w *Window) recordFrame(width, height int, metric unit.Metric) {
	memory := w.frame.Load()
	if memory == nil || platformOwnsPosition() {
		return
	}
	memory.record(frameFromPixels(width, height, metric))
}

// stopFrameMemory writes whatever change is still pending as the window is
// destroyed, and detaches the memory so a late report finds nothing.
func (w *Window) stopFrameMemory() {
	platformForgetFrame()
	if memory := w.frame.Swap(nil); memory != nil {
		memory.stop()
	}
}
