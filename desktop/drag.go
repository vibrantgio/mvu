package desktop

// A window is dragged by the strip its native title bar stands in. The
// full-size-content treatment takes that strip, and the drag leaves with it:
// the press that would have reached the title bar reaches the application
// instead, and a window that claims nothing back cannot be moved by its top
// edge at all. So whatever region caps the window hands a run of itself back,
// and the three calls here are how it says so — the rectangle, the run of a
// row being laid out, and the strip a page simply starts below.
//
// Input only, like the geometry beside it. What the band is painted with is
// the business of the packages that know about colour; this one records an
// area and an action and nothing else.

import (
	"image"

	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/unit"
)

// DragBand declares r — in the coordinates the caller is laying out in — a
// region the window may be picked up and moved by, in place of the native
// title bar the full-size-content treatment removed.
//
// The claim covers the region's empty runs and only those. A move action
// swallows the press before any control under it sees one, so a band laid over
// a control makes that control's press the window's; declare the band first
// and draw the things that stand in it afterwards, and each of them keeps its
// own span. The platform's own window buttons are controls of this kind, and
// the run they stand in is nobody's to claim.
//
// An empty or inverted rectangle claims nothing, so a band asked for a run it
// has no room for records no area rather than a degenerate one. Away from the
// treatment — on every platform that keeps its own decorations included — the
// native strip is still up there moving the window, and a band declared here
// is one more handle rather than the only one.
func DragBand(gtx layout.Context, r image.Rectangle) {
	if r.Dx() <= 0 || r.Dy() <= 0 {
		return
	}
	defer clip.Rect(r).Push(gtx.Ops).Pop()
	system.ActionInputOp(system.ActionMove).Add(gtx.Ops)
}

// DragRun is [DragBand] for a run standing in a row that is being laid out: a
// w-wide claim at the current offset, as deep as the row, reported as the
// run's own dimensions so that it can stand in a flex as the gap it is.
//
// The depth is the row's whole height rather than a band around the line its
// labels sit on, because the run a hand aims for is the one it can see — the
// strip from the row's top edge to its foot.
//
// A run with no height claims nothing and still reports its width, so a row
// measured at nothing keeps its horizontal arrangement; a run with no width
// reports nothing at all.
func DragRun(gtx layout.Context, w int) layout.Dimensions {
	h := gtx.Constraints.Max.Y
	if w <= 0 || h <= 0 {
		return layout.Dimensions{Size: image.Pt(max(w, 0), 0)}
	}
	DragBand(gtx, image.Rect(0, 0, w, h))
	return layout.Dimensions{Size: image.Pt(w, h)}
}

// DragTop is [DragBand] for the window whose page simply begins below the
// strip rather than drawing a row of its own in it: it claims the width of the
// strip [InsetTop] holds open above the page, and nothing below it.
//
// height is the same function [InsetTop] is given — read afresh, since the
// measurement arrives after the window's first frame and moves with the
// window — so the two calls state one number once:
//
//	DragTop(gtx, TopInset)
//	InsetTop(TopInset, page)(gtx)
//
// The claim starts past the window's control buttons, at [LeadingInset], since
// the run they stand in is not the strip's to give away. Both measurements are
// 0 before the first frame and wherever the window keeps its own decorations,
// and a strip of no height claims nothing — so a page with no strip above it
// is a page with no band over it either.
func DragTop(gtx layout.Context, height func() unit.Dp) {
	DragBand(gtx, dragTopRect(gtx, LeadingInset(), height()))
}

// dragTopRect is [DragTop]'s rectangle over stated measurements rather than
// the window's own: it is where the arithmetic lives, so that a test can state
// a strip and a button run it has no window to take.
func dragTopRect(gtx layout.Context, buttonsEnd, height unit.Dp) image.Rectangle {
	return image.Rect(gtx.Dp(buttonsEnd), 0, gtx.Constraints.Max.X, gtx.Dp(height))
}
