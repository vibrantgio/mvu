package desktop

import (
	"gioui.org/app"
	"gioui.org/unit"

	"github.com/vibrantgio/mvu"
)

// FullSizeContent returns the construction-time window options for the
// full-size-content treatment. On macOS that is app.Decorated(false), which
// makes Gio extend the content behind a transparent title bar and hide the
// window title — and also hide the three standard window buttons, the part
// [ShowWindowButtons] brings back. On every other platform it returns no
// options at all: there Decorated(false) means a truly borderless window,
// and a missing macOS title bar must never degrade into that.
//
// Keep app.Title in the option list. The treatment hides the title text, but
// Mission Control, the Dock and VoiceOver still read it.
func FullSizeContent() []app.Option {
	return fullSizeContent()
}

// ShowWindowButtons keeps the standard macOS window buttons — close,
// miniaturize and zoom, the traffic lights — visible on a window opened with
// the [FullSizeContent] options. Gio re-hides them every time it rebuilds the
// native window's configuration, so a single unhide cannot hold:
// ShowWindowButtons registers with w's OnConfigure notification and
// re-asserts the buttons after the window's first frame and after every
// Option call on w. Call it once, right after constructing w; on platforms
// other than macOS it does nothing.
//
// Each re-assertion dispatches itself onto the AppKit main thread and also
// refreshes the measurements reported by [TopInset] and [LeadingInset].
//
// Options applied through the raw Gio handle — w.Window().Option(...) —
// bypass the notification this relies on: Gio re-hides the buttons and
// nothing re-asserts them. Route post-construction options through w.Option
// instead.
func ShowWindowButtons(w *mvu.Window) {
	showWindowButtons(w)
}

// TopInset reports the height of the native title-bar strip that a
// full-size-content window's content extends behind: the window frame height
// minus the content layout height, measured from the native window. AppKit
// points are Gio dp, so the value pads layout directly. It is measured, never
// a constant — the strip stands at 32 dp on current macOS where folklore says
// 28, and a hardcoded value fails in the direction that clips content.
//
// The measurement is maintained by [ShowWindowButtons]'s re-assertion: until
// the window's first frame TopInset reports 0, and on platforms other than
// macOS it always reports 0. When a fresh measurement changes the value, the
// window is redrawn so the next frame lays out with it.
//
// The strip itself is paint-only for the application: clicks there go to the
// native title-bar view — window dragging, double-click zoom — never to
// widgets underneath, and the window buttons occupy the leading run of it
// that [LeadingInset] measures.
func TopInset() unit.Dp {
	return topInset()
}

// LeadingInset reports how far in from the leading edge of the content a
// full-size-content window's own controls reach: the trailing edge of the
// rightmost standard window button — close, miniaturize and zoom together —
// measured from the native window. AppKit points are Gio dp, so the value
// pads layout directly. Like [TopInset] it is measured rather than assumed,
// because the button metrics are the system's to change.
//
// The value is the bare trailing edge of the buttons and nothing more: it
// includes no breathing room after them, so a caller placing content beside
// the controls adds its own spacing on top. It is expressed relative to the
// leading edge of the content area rather than of the window frame, which is
// the space layout works in; under the full-size-content treatment, where the
// content spans the frame, the two coincide.
//
// The measurement is maintained by [ShowWindowButtons]'s re-assertion on the
// same terms as [TopInset]'s: it reports 0 until the window's first frame,
// and when a fresh measurement changes it the window is redrawn so the next
// frame lays out with the new value. Platforms whose windows carry no such
// buttons — every platform other than macOS — always report 0, and content
// there starts at the leading edge.
func LeadingInset() unit.Dp {
	return leadingInset()
}
