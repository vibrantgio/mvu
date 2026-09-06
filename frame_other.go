//go:build !darwin

package mvu

// The platforms that remember a size and no position. Neither Gio's
// [app.Config] nor any window option carries a window's position on them, and
// this package holds no native code of its own beyond macOS's — so the size
// is read from each frame event, the position is not read at all, and a
// remembered position written elsewhere is loaded and ignored.

// platformOwnsPosition reports whether the platform reads and applies the
// window's position itself.
func platformOwnsPosition() bool { return false }

// platformScreens reports where the screens are, empty where that is unknown.
func platformScreens() []Frame { return nil }

// platformRememberFrame is unreachable here: [Window.rememberFrame] starts it
// only where the platform owns the position.
func platformRememberFrame(*Window, *frameMemory, Frame, bool) {}

// platformForgetFrame has nothing to forget: nothing platform-side reports.
func platformForgetFrame() {}
