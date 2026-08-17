// The C surface of the package's Objective-C (chrome_darwin.m). The Go side
// (chrome_darwin.go) explains why re-assertion is asynchronous.

#ifndef VGIO_MVU_DESKTOP_CHROME_DARWIN_H
#define VGIO_MVU_DESKTOP_CHROME_DARWIN_H

// vgio_desktop_reassert re-shows the three standard window buttons on the
// application's window and refreshes the cached insets, on the AppKit
// main thread. Asynchronous unless already on the main thread; a no-op while
// no application or no titled window exists.
extern void vgio_desktop_reassert(void);

// vgio_desktop_place_buttons records the placement the standard window
// buttons are to take — leading is where the group's leading edge sits, in
// AppKit points in from the leading edge of the window frame, and center is
// the line the buttons are centred on, in points below its top — and asks
// for a re-assertion so the request takes effect at once. Zero on either
// axis gives that axis back to AppKit's own placement, independently of the
// other; zero on both restores the system geometry exactly. Safe to call
// from any thread.
extern void vgio_desktop_place_buttons(double leading, double center);

// vgio_desktop_top_inset returns the most recently measured top inset in
// AppKit points (which equal Gio dp), 0 before any measurement and 0 while a
// placement from vgio_desktop_place_buttons is in force. Safe to call from any
// thread; it never dispatches.
extern double vgio_desktop_top_inset(void);

// vgio_desktop_leading_inset returns the most recently measured trailing edge
// of the standard window buttons, in AppKit points (which equal Gio dp),
// relative to the leading edge of the content layout rect; 0 before any
// measurement and 0 where the window has no such buttons. It is measured from
// the buttons' own frames, so it follows wherever a placement puts them. Safe
// to call from any thread; it never dispatches.
extern double vgio_desktop_leading_inset(void);

#endif
