//go:build darwin

#import <AppKit/AppKit.h>
#include <stdatomic.h>

#include "chrome_darwin.h"

// The one measured value the Go side may read from any goroutine without a
// hop to the main thread. Layout code queries the inset mid-frame, when the
// main thread is parked in Gio's frame handoff and a dispatch_sync would
// deadlock — so the query reads this cache and the re-assertion refreshes it.
static _Atomic double vgio_desktop_inset = 0;

// The horizontal companion to vgio_desktop_inset, cached and read on the same
// terms and for the same reason.
static _Atomic double vgio_desktop_leading = 0;

// The application's window: its first titled window. The full-size-content
// treatment keeps NSWindowStyleMaskTitled on macOS, so the treated window is
// still found by this test.
static NSWindow *vgio_desktop_window(void) {
	for (NSWindow *w in [NSApp windows]) {
		if ([w styleMask] & NSWindowStyleMaskTitled) {
			return w;
		}
	}
	return nil;
}

// Runs on the main thread only.
static void vgio_desktop_reassert_main(void) {
	NSWindow *w = vgio_desktop_window();
	if (w == nil) {
		return;
	}
	[[w standardWindowButton:NSWindowCloseButton] setHidden:NO];
	[[w standardWindowButton:NSWindowMiniaturizeButton] setHidden:NO];
	[[w standardWindowButton:NSWindowZoomButton] setHidden:NO];
	double inset = NSHeight([w frame]) - NSHeight([w contentLayoutRect]);

	// The buttons' frames are in their superview's coordinate space — the
	// title-bar view, not the content view — so they cannot be compared with
	// a content-relative measurement as they stand. convertRect:toView:nil
	// puts each one into the window's own base coordinates, and subtracting
	// the content layout rect's origin restates it in the space the content
	// is laid out in. Under the full-size-content treatment that origin is
	// zero and the subtraction changes nothing; it is written out anyway so
	// the value stays correct if the window ever stops spanning its frame.
	const NSWindowButton kButtons[] = {
		NSWindowCloseButton,
		NSWindowMiniaturizeButton,
		NSWindowZoomButton,
	};
	double leading = 0;
	for (size_t i = 0; i < sizeof(kButtons) / sizeof(kButtons[0]); i++) {
		NSView *b = [w standardWindowButton:kButtons[i]];
		if (b == nil) {
			continue;
		}
		double edge = NSMaxX([b convertRect:[b bounds] toView:nil]);
		if (edge > leading) {
			leading = edge;
		}
	}
	if (leading > 0) {
		leading -= NSMinX([w contentLayoutRect]);
	}
	if (leading < 0) {
		leading = 0;
	}

	double prevLeading = atomic_exchange(&vgio_desktop_leading, leading);
	double prev = atomic_exchange(&vgio_desktop_inset, inset);
	if (prev != inset || prevLeading != leading) {
		// The frame that triggered this re-assertion laid out with the stale
		// inset. Redraw the way Gio's own Invalidate does, so the next frame
		// picks the fresh value up even in an otherwise idle application.
		[[w contentView] setNeedsDisplay:YES];
	}
}

void vgio_desktop_reassert(void) {
	if (NSApp == nil) {
		// No NSApplication: a test binary, or a notification fired before
		// app.Main. Nothing to assert against, and dispatching to a main
		// queue nobody drains would wedge the caller.
		return;
	}
	if ([NSThread isMainThread]) {
		vgio_desktop_reassert_main();
		return;
	}
	dispatch_async(dispatch_get_main_queue(), ^{
		vgio_desktop_reassert_main();
	});
}

double vgio_desktop_top_inset(void) {
	return atomic_load(&vgio_desktop_inset);
}

double vgio_desktop_leading_inset(void) {
	return atomic_load(&vgio_desktop_leading);
}
