// The C surface of the package's Objective-C (chrome_darwin.m). The Go side
// (chrome_darwin.go) explains why re-assertion is asynchronous.

#ifndef VGIO_MVU_DESKTOP_CHROME_DARWIN_H
#define VGIO_MVU_DESKTOP_CHROME_DARWIN_H

// vgio_desktop_reassert re-shows the three standard window buttons on the
// application's window and refreshes the cached insets, on the AppKit
// main thread. Asynchronous unless already on the main thread; a no-op while
// no application or no titled window exists.
extern void vgio_desktop_reassert(void);

// vgio_desktop_top_inset returns the most recently measured top inset in
// AppKit points (which equal Gio dp), 0 before any measurement. Safe to call
// from any thread; it never dispatches.
extern double vgio_desktop_top_inset(void);

// vgio_desktop_leading_inset returns the most recently measured trailing edge
// of the standard window buttons, in AppKit points (which equal Gio dp),
// relative to the leading edge of the content layout rect; 0 before any
// measurement and 0 where the window has no such buttons. Safe to call from
// any thread; it never dispatches.
extern double vgio_desktop_leading_inset(void);

#endif
