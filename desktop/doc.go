// Package desktop adjusts the native desktop window that Gio creates for a
// [github.com/vibrantgio/mvu.Window] — in place, without forking Gio. Its
// first treatment is the macOS full-size-content window: content extending
// behind a transparent title bar, the three standard window buttons floating
// over it, the way current macOS applications look.
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
// strip. Query the strip's height with [TopInset] and pad interactive content
// down by it — the value is measured from the native window, never assumed.
// Clicks in the strip go to the native title-bar view, not to widgets drawn
// underneath; that is exactly what makes native window dragging and
// double-click zoom work, so the strip is paint-only for the application:
// draw background there, put nothing interactive in it, and keep roughly the
// leading 80 dp clear for the window buttons.
//
// The package addresses the application's one window — the native window is
// found as the application's first titled window, which the treatment's
// borderless option still is on macOS. Applications with several windows are
// beyond its contract.
//
// # cgo
//
// The macOS side is Objective-C reached through cgo — the first cgo in the
// Vibrant Gio organization beyond what Gio itself carries. That is why this
// is a nested module rather than part of the mvu root: the root stays
// cgo-free and platform-neutral, and only applications that want native
// window chrome take on this module's AppKit build path.
package desktop
