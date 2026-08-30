package desktop

// The title band is the strip at the top of a full-size-content window that
// the application draws itself, standing where the native title bar would
// otherwise be. The arithmetic here answers three questions about it: where
// the band's own content may start given that the platform's control buttons
// stand in its leading run, where those buttons go given the band's height,
// and how far down a layer must start where the band is still the platform's.
//
// Geometry only, answered in dp. What a band is painted with belongs to
// packages above this one.

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// Window button geometry that belongs to the platform rather than to any one
// window, in dp. Both are measured from live captures on current macOS rather
// than assumed: the buttons are the system's to draw, and so is their size.
//
// They are untyped so that they compose with dp expressions of either sort —
// the arithmetic a band does around them is as often in layout pixels as in
// [unit.Dp].
const (
	// WindowButtonDiameter is the drawn diameter of one control button.
	// Without it an edge inset cannot be turned into the centre line a
	// placement call wants.
	WindowButtonDiameter = 14

	// WindowButtonPitch is the distance from one button's leading edge to
	// the next one's — equivalently, between two neighbouring centres. The
	// group is three circles at this spacing, and nothing but the leading
	// inset moves when the group moves.
	WindowButtonPitch = 23
)

// ButtonRun is where a window's three standard control buttons — close,
// miniaturize and zoom, which move as one group — stand: the whole row of
// the platform's measurement, in dp from the window's top-leading corner.
//
// Derive one with [ButtonRunIn] from the height of the band the buttons are
// to sit in, or with [ButtonRunAt] from a leading inset already decided.
type ButtonRun struct {
	// Leading is the leading edge of the first circle, in from the window's
	// leading edge: [PlaceWindowButtonsAt]'s first argument.
	Leading unit.Dp

	// Center is the horizontal line the three circles are centred on, below
	// the window's top edge: [PlaceWindowButtonsAt]'s second argument, and
	// the line anything else standing in the band centres on if the band is
	// to read as one row of furniture.
	Center unit.Dp

	// Diameter is the drawn diameter of one circle — [WindowButtonDiameter],
	// carried along so that a caller sizing a strip around the run has the
	// whole geometry in one value.
	Diameter unit.Dp

	// Trailing is the trailing edge of the third circle, in from the
	// window's leading edge: the run's whole width, and the number
	// [LeadingInset] reports once there is a window with these buttons on it
	// to measure. It is not a substitute for that measurement: a window that
	// has not drawn its first frame reports 0 there, and so does every
	// platform whose windows carry no such buttons.
	Trailing unit.Dp
}

// ButtonRunIn derives the button run for a band of the given height.
//
// The rule the platform's own windows follow is that the buttons are centred
// in whatever band the window has, and that their leading inset equals their
// top inset — so the band's height is the only input. A 32 dp plain title bar
// puts them 9 dp in from both edges; a 52 dp unified toolbar band puts them
// 19 dp in, which is where the platform's toolbar windows were measured:
// 19 leading, 19 top, 14 across, 23 between centres, 79 to the far edge of
// the third circle.
//
// The result is exact rather than rounded, so a band of odd height centres
// the run on a half-dp line. That is what centring it means, and the
// platform's own coordinates hold halves without complaint.
func ButtonRunIn(band unit.Dp) ButtonRun {
	return ButtonRunAt((band - WindowButtonDiameter) / 2)
}

// ButtonRunAt derives the button run from a leading inset already decided:
// since the leading inset equals the top inset, the inset alone fixes the
// centre line, and the rest of the run follows from the platform's own
// diameter and pitch.
//
// ButtonRunAt(19) and ButtonRunIn(52) are the same run.
func ButtonRunAt(inset unit.Dp) ButtonRun {
	return ButtonRun{
		Leading:  inset,
		Center:   inset + WindowButtonDiameter/2,
		Diameter: WindowButtonDiameter,
		Trailing: inset + 2*WindowButtonPitch + WindowButtonDiameter,
	}
}

// BandLead reports where a band's own content may start, in dp in from the
// leading edge of the window: past the trailing edge of the window's control
// buttons, plus gap, where the platform draws them inside the content area —
// and at gutter where it does not.
//
// The measurement is [LeadingInset]'s, read afresh, so it is the buttons' own
// frames rather than an assumed geometry and it follows a placement that
// moved them. It is 0 — the gutter case — until a window has drawn its first
// frame, in headless rendering, and on every platform that keeps its own
// decorations and so puts no buttons in the application's content at all.
//
// gap is the air the band owes the last button. What [LeadingInset] reports
// is the bare glass the third circle ends at and carries no breathing room of
// its own, so a band that spaces the things standing in it passes its own
// spacing step here; a band whose leading run holds nothing but the buttons
// passes 0 and starts flush with their trailing edge.
//
// gutter is what the band falls back to with no buttons to clear: its own
// leading margin, whatever the band has decided that is.
func BandLead(gap, gutter unit.Dp) unit.Dp {
	return BandLeadFrom(LeadingInset(), gap, gutter)
}

// BandLeadFrom is [BandLead] over a stated measurement rather than the
// window's own: buttonsEnd is the trailing edge of the control buttons, or 0
// where the window has none. It is the form for a caller whose measurement
// arrives from somewhere else — a test stating an edge it has no window to
// take, a band that has already read [LeadingInset] this frame — and it is
// where the arithmetic lives.
func BandLeadFrom(buttonsEnd, gap, gutter unit.Dp) unit.Dp {
	if buttonsEnd > 0 {
		return buttonsEnd + gap
	}
	return gutter
}

// InsetTop wraps w in a widget that offsets it down by height and reports the
// full size it was given rather than the inset one — so the layer it wraps
// still measures as the whole window, and the strip above it keeps whatever
// is painted underneath.
//
// height is a function rather than a value because the numbers a band works
// in are measured from the native window and move under the application:
// [TopInset] reports 0 until the window's first frame and changes when the
// window's own geometry does, so the height is read afresh every frame.
// Where it reports 0 — headless rendering and every platform other than
// macOS included — the wrapper is an exact no-op, w laid out in the context
// it was handed and nothing recorded around it.
//
// The wrapped widget's constraints lose the inset from their maximum height,
// and their minimum follows the maximum down rather than exceeding it, so a
// layer that fills what it is given ends flush with the bottom of the window
// instead of overhanging it by the inset.
func InsetTop(height func() unit.Dp, w layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		inset := gtx.Dp(height())
		if inset <= 0 {
			return w(gtx)
		}
		size := gtx.Constraints.Max
		defer op.Offset(image.Pt(0, inset)).Push(gtx.Ops).Pop()
		gtx.Constraints.Max.Y -= inset
		if gtx.Constraints.Min.Y > gtx.Constraints.Max.Y {
			gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
		}
		w(gtx)
		return layout.Dimensions{Size: size}
	}
}
