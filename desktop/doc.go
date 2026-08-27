// Package desktop adjusts the native desktop window that Gio creates for a
// [github.com/vibrantgio/mvu.Window] — in place, without forking Gio. It has
// two tenants behind the same seam: the macOS full-size-content window
// chrome (content extending behind a transparent title bar, the three
// standard window buttons floating over it), and OS file drops, delivered
// into the application's message loop as ordinary messages.
//
// The package is safe to call from platform-neutral code: every entry point
// compiles everywhere and quietly does nothing away from macOS.
//
//	w := mvu.NewWindow(append(desktop.FullSizeContent(),
//		app.Title("Notes"),
//		app.Size(unit.Dp(800), unit.Dp(600)),
//	)...)
//	desktop.ShowWindowButtons(w)
//
// Note the title: keep passing app.Title even though the treatment hides the
// title text — Mission Control, the Dock and VoiceOver read it all the same.
//
// # Re-assertion, not configuration
//
// Gio rebuilds the native window's configuration on every option change, and
// each rebuild re-hides the standard window buttons, so a one-shot unhide is
// wrong by construction. [ShowWindowButtons] therefore registers with
// [github.com/vibrantgio/mvu.Window.OnConfigure] and re-asserts the buttons
// after the window's first frame and after every
// [github.com/vibrantgio/mvu.Window.Option] call. That only covers options
// routed through the mvu window: options applied to the raw Gio handle, as in
// w.Window().Option(...), bypass the notification, and whatever Gio's rebuild
// undid stays undone until the next routed call. Route post-construction
// option changes through the mvu window's Option method, always.
//
// # The title-bar strip
//
// Gio reports no top inset on macOS, so content runs under the title-bar
// strip. Query the strip's height with [TopInset] and pad content down by it —
// the value is measured from the native window, never assumed — and keep the
// leading run of the row clear for the window buttons, which [LeadingInset]
// measures rather than guesses.
//
// The strip does not have to stay the system's. [PlaceWindowButtons] centres
// the buttons on a line the caller picks, which lets one row the application
// draws itself carry the buttons and the application's own controls together;
// with a placement in force [TopInset] reports 0, because the row at the top
// of the window is then the caller's. [PlaceWindowButtonsAt] states the
// buttons' leading edge as well as their line, for a surface whose own edge
// stands somewhere the system's x was never chosen for; zero on either axis
// keeps the system's placement on that axis.
//
// The arithmetic around such a row is the same wherever it is drawn, so it
// lives here rather than in each window that draws one. [ButtonRunIn] derives
// where the buttons go from the height of the band they are to sit in, by the
// platform's own rule — centred in whatever band the window has, leading inset
// equal to top inset — and [ButtonRunAt] derives the same run from an inset
// already decided. [BandLead] answers where a band's own content may start,
// past the buttons where the platform draws them in the content area and at
// the band's own gutter where it does not. [InsetTop] is the other side of
// that coin: it pads a layer down past a strip the application has not
// claimed, reading the height afresh each frame. All four are geometry in dp
// and nothing else — a band's ground and its type belong to the packages that
// know about colour, which are above this one.
//
// Two things measured about that row, both easy to assume wrongly. The Gio
// view spans the whole window frame and wins the hit test throughout the
// strip, so widgets drawn there do receive clicks — everywhere except over the
// buttons, which keep a few dp of slop around them. And precisely because the
// native title-bar view never sees the press, the window can no longer be
// dragged by its top edge: the drag leaves with the strip, and the region that
// caps the window hands a run of itself back to get it. [DragBand] is that
// claim over a rectangle, [DragRun] over a run of a row being laid out, and
// [DragTop] over the strip a page starts below — the same
// system.ActionMove an undecorated window would use anywhere else, said once
// for the three shapes the top of a window comes in.
//
// The last of those three shapes is the plain one, and it is always the same
// two calls over the same height: the strip claimed, the page held down past
// it. [CapTop] is that pair, so a window with no row of its own at the top
// states its strip once and gets both halves of it.
//
// The chrome treatment addresses the application's one window — the native
// window is found as the application's first titled window, which the
// treatment's borderless option still is on macOS. Applications with several
// windows are beyond the chrome's contract. File drops carry no such limit:
// a [DropTarget] is per window, and every window gets its own.
//
// # File drops
//
// A window accepts files dragged from the Finder by pairing a [ZoneGroup] —
// the registry of drop-target rectangles, recorded each frame during layout —
// with a [DropTarget], which performs the native registration and delivers
// [FilesEntered], [FilesExited] and [FilesDropped] messages resolved against
// those zones:
//
//	zones := &desktop.ZoneGroup{}
//	drops := desktop.NewDropTarget(w, zones)
//	models, runner := mvu.Loop(rx.Merge(w.Messages(), drops.Messages()), Init, Update)
//
// and in the view, each frame: zones.Update(gtx), then zones.Zone(gtx, i,
// origin, widget) for every target. Payload kinds are MIME-shaped, with
// [FileURLs] the one kind registered today; drops of anything else are
// refused at the window edge by the OS itself.
//
// Construct the DropTarget before the window starts rendering. It subscribes
// the window's ViewEvents stream — which mvu documents as single-subscriber —
// so constructing a target claims that stream for the window.
//
// # How the two tenants share the seam
//
// The chrome and the drop target repair different kinds of native state, and
// they deliberately listen on different notifications. The window buttons are
// window-level state that Gio's own configuration rebuild un-does, so the
// chrome re-asserts on the OnConfigure notification — after the first frame
// and after every routed Option call. Drop registration is per-view-instance
// state on the native view object itself: Gio's rebuild does not touch it,
// but the view can leave its window and a replacement can appear, so the
// drop target re-registers on the window's view events — registering on
// every valid event (attach) and dropping every native reference on the
// invalid one (detach). Neither tenant registers on the other's
// notification, and neither needs to: window-level state re-asserts on
// configuration, view-instance state re-registers on attachment. The
// raw-handle warning above is the chrome's alone — drop registration does
// not depend on how options are applied.
//
// # Threading and lifecycle, for anyone touching the native half
//
// These rules are collected here because each one is silent when violated.
//
//   - AppKit is called on its main thread only, reached through the
//     window's Run; the package does this itself, and callbacks arriving
//     FROM AppKit run on that thread too.
//   - A drag callback never blocks — blocking freezes the compositor
//     mid-drag. Events cross to the pipeline through a buffered channel with
//     a non-blocking send; on overflow the oldest event is evicted, never
//     the newest, because stale hover positions are superseded and the
//     newest event may be the drop itself.
//   - The drop is accepted before the application has processed it: the
//     native callback's affirmative answer means "the payload was read",
//     never "the application handled it".
//   - The class augmentation that installs the drag callbacks is per-class,
//     process-global and permanent; it runs once however many windows
//     exist. Per-view registration is the repeated half.
//   - The invalid view event is a real event, not an error: the native view
//     left its window, and every handle from the previous event is dead.
//     The package drops all of its references when it arrives; so must any
//     other code holding one.
//   - The display's backing scale is re-read on every drag event, because a
//     window can move between displays of different scale mid-drag.
//   - Messages reach the loop through the channel path, never through a
//     frame operation: a frame op recorded outside its own frame is dropped
//     silently, and a drag callback has no frame at all.
//
// # What is verified where
//
// An OS drag cannot be automated. The unit tests cover everything around it
// — the coordinate transform at both display scales, zone hit-testing and
// its frame double-buffering, hover tracking, per-view routing and teardown
// — and continuous integration compiles what it can, on every platform it
// can, but never fakes a drag. The drag path itself is verified by a manual
// script run on a Mac; treat any change to the native half as unverified
// until that script has been run again.
//
// # cgo
//
// The macOS side is Objective-C reached through cgo — the first cgo in the
// Vibrant Gio organization beyond what Gio itself carries. That is why this
// is a nested module rather than part of the mvu root: the root stays
// cgo-free and platform-neutral, and only applications that want native
// window chrome or file drops take on this module's AppKit build path.
package desktop
