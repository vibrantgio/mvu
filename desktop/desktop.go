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
// ShowWindowButtons registers a handler with w's OnConfigure notification and
// re-asserts the buttons after the window's first frame and after every
// Option call on w. Call it once, right after constructing w; on platforms
// other than macOS it does nothing.
//
// Each re-assertion dispatches itself onto the AppKit main thread, applies any
// placement asked for by [PlaceWindowButtons], and refreshes the measurements
// reported by [TopInset] and [LeadingInset]. The first re-assertion also
// subscribes to the window's own resize notification, so every one of those
// three survives a resize the user drives with the mouse — which changes no
// window option and so raises no configuration notification of its own.
//
// Options applied through the raw Gio handle — w.Window().Option(...) —
// bypass the notification this relies on: Gio re-hides the buttons and
// nothing re-asserts them. Route post-construction options through w.Option
// instead.
func ShowWindowButtons(w *mvu.Window) {
	showWindowButtons(w)
}

// PlaceWindowButtons centres the three standard macOS window buttons on a
// horizontal line center dp below the top edge of the window, so that a row
// the application draws itself can hold them rather than sit below them. A
// center of 0 — the default — leaves the buttons where macOS puts them, and
// passing 0 after a placement gives them back. On platforms other than macOS
// it does nothing at all.
//
// Pass the vertical centre of the row the buttons are to sit in: a 28 dp row
// at the top of the window asks for 14. The buttons keep their own size and
// their horizontal positions, which are the system's to choose; only the line
// they sit on is the caller's. [LeadingInset] keeps reporting their trailing
// edge, and [TopInset] reports 0 while a placement is in force, since the row
// at the top of the window is then the application's to lay out from.
//
// The call states the whole placement: PlaceWindowButtons(center) is
// [PlaceWindowButtonsAt](0, center), the same vertical line with the system's
// own x. A caller that also states the leading edge uses PlaceWindowButtonsAt
// directly.
//
// The placement is re-applied by the same re-assertion that re-shows the
// buttons, because AppKit rebuilds the title bar's layout on a window resize
// and on every configuration change, and each rebuild puts the buttons back.
// Off macOS the call compiles to an empty function.
//
// Call it after [ShowWindowButtons]; a placement asked for before there is a
// window is remembered and applied to the first re-assertion that finds one.
func PlaceWindowButtons(center unit.Dp) {
	placeWindowButtons(center)
}

// PlaceWindowButtonsAt places the three standard macOS window buttons on both
// axes: leading is where the group's leading edge sits, in dp in from the
// leading edge of the window, and center is the horizontal line their centres
// sit on, in dp below its top — [PlaceWindowButtons]'s own parameter, meaning
// exactly what it means there. The buttons keep their own size and the
// system's own spacing; the group moves as one.
//
// Zero means the system's own placement, per axis and independently:
// PlaceWindowButtonsAt(0, 14) moves the line and leaves x alone — the same
// placement PlaceWindowButtons(14) states — PlaceWindowButtonsAt(25, 0)
// moves the leading edge and leaves the buttons on the system's own line,
// and PlaceWindowButtonsAt(0, 0), like PlaceWindowButtons(0), restores the
// system geometry exactly. Each call states the complete placement; the two
// functions set the same state, and the last call wins.
//
// [LeadingInset] keeps reporting the buttons' trailing edge at wherever the
// placement put them, because it is measured from their frames rather than
// assumed. [TopInset] reports 0 only while a vertical placement — a non-zero
// center — is in force: moving the leading edge alone claims no row, so the
// native strip stays what an application must clear.
//
// Everything else is [PlaceWindowButtons]'s contract unchanged: applied under
// the same re-assertion, callable before the window exists, an empty function
// off macOS.
func PlaceWindowButtonsAt(leading, center unit.Dp) {
	placeWindowButtonsAt(leading, center)
}

// TopInset reports how far below the top of the window an application must
// begin its own content: the height of the native title-bar strip that a
// full-size-content window's content extends behind — the window frame height
// minus the content layout height, measured from the native window — or 0
// once [PlaceWindowButtons] has placed the window buttons, because a caller
// that has taken the buttons into a row of its own has taken the row.
//
// AppKit points are Gio dp, so the value pads layout directly. It is measured,
// never a constant — the strip stands at 32 dp on current macOS where folklore
// says 28, and a hardcoded value fails in the direction that clips content.
//
// That zero is this package's answer, not AppKit's: a window's content layout
// rect does not shrink when the buttons move, and AppKit goes on reporting the
// same strip. The question answered here is how much is left above the caller
// that is not the caller's, which follows from who owns the row.
//
// The measurement is maintained by [ShowWindowButtons]'s re-assertion: until
// the window's first frame TopInset reports 0, and on platforms other than
// macOS it always reports 0. When a fresh measurement changes the value, the
// window is redrawn so the next frame lays out with it.
//
// The strip is not as dead as a native title bar looks. Under the
// full-size-content treatment the Gio view spans the whole window frame and
// wins the hit test throughout the strip — everywhere except over the window
// buttons themselves, which keep a few dp of slop around them — so components
// drawn up there do receive their clicks. What the strip does not give back is
// the native drag: the title-bar view never sees the press, so the window
// cannot be moved by its top edge until the application claims a region for it
// with Gio's own system.ActionMove.
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
// [PlaceWindowButtons] does not change it: a vertical placement moves the
// line the buttons sit on and nothing else, so their trailing edge stands
// where it stood. A [PlaceWindowButtonsAt] leading edge does move it, and
// the measurement follows: the value is read from the buttons' frames where
// the placement put them, not from where the system would have.
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
