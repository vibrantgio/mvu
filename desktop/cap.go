package desktop

// The plainest thing a full-size-content window does with the strip at its
// top: nothing. No row of its own stands up there, the page simply begins
// below the strip, and whatever the window is painted with shows through it.
//
// That arrangement is still two calls rather than none — the strip has to be
// handed back as a drag band, and the page has to be held down past it — and
// they are the same two calls with the same height every time. This file is
// where the pair is said once. It is the one place in the package that spans
// both sides of the seam, the geometry beside it and the input claim: the
// composition is exactly the claim over the strip the inset holds open, and
// it cannot be stated from either side alone.
//
// Still nothing about colour. A strip this helper caps carries no fill of its
// own; a window that wants one paints it around the call, in the package that
// knows what its region's ground is.

import (
	"gioui.org/layout"
	"gioui.org/unit"
)

// CapTop caps a window whose page begins below the native title-bar strip: it
// declares the strip a drag band with [DragTop] and lays w out below it with
// [InsetTop], both over the same height, so that the two halves of the
// arrangement cannot disagree about where the strip ends.
//
//	CapTop(TopInset, page)
//
// is the whole of it for a window under the platform's own strip. height is a
// function for [InsetTop]'s reason — the measurement arrives after the
// window's first frame and moves with the window, so it is read afresh every
// frame — and both calls here read the one function, once per frame.
//
// The claim is recorded before the page, and that order is the rule rather
// than a detail: a move action swallows the press before any control under it
// sees one, so a band recorded over a control would make that control's press
// the window's. Declaring the claim first and the page second leaves every
// region the page goes on to record — its editors, its scrolls, its focus
// catchers — holding its own presses. The two do not overlap in any case,
// since the page starts where the band ends; the order is what keeps that
// true when a page reaches back up.
//
// A region recorded after this call, over the whole window, is a different
// matter: it shadows the claim wherever it lies, and the strip goes with it.
// A window that lays such a region over its page — a modal's key catcher
// spanning the window is the usual one — says [DragTop] again on top of it,
// over the same height, to give the strip back.
//
// The returned widget reports the size it was given rather than the inset one,
// [InsetTop]'s contract, so a layer capped here still measures as the whole
// window and anything anchored to the window's foot stays there. Where height
// reports 0 — before the first frame, in headless rendering, and on every
// platform that keeps its own decorations — nothing is claimed and nothing is
// inset: the wrapper is an exact no-op and w is laid out in the context it was
// handed.
//
// This is the plain shape only. A window whose own regions stand in the strip
// rather than starting below it insets no page and has no cap to make: it
// claims what is left of the strip with [DragTop] or [DragRun] directly, at
// the point in its own layout where the run is known.
func CapTop(height func() unit.Dp, w layout.Widget) layout.Widget {
	inset := InsetTop(height, w)
	return func(gtx layout.Context) layout.Dimensions {
		DragTop(gtx, height)
		return inset(gtx)
	}
}
