//go:build darwin

#import <AppKit/AppKit.h>
#include <stdatomic.h>
#include <stdbool.h>

#include "chrome_darwin.h"

// The three buttons, in the one order every walk over them uses.
static const NSWindowButton vgio_desktop_buttons[] = {
	NSWindowCloseButton,
	NSWindowMiniaturizeButton,
	NSWindowZoomButton,
};

// The one measured value the Go side may read from any goroutine without a
// hop to the main thread. Layout code queries the inset mid-frame, when the
// main thread is parked in Gio's frame handoff and a dispatch_sync would
// deadlock — so the query reads this cache and the re-assertion refreshes it.
static _Atomic double vgio_desktop_inset = 0;

// The horizontal companion to vgio_desktop_inset, cached and read on the same
// terms and for the same reason.
static _Atomic double vgio_desktop_leading = 0;

// The requested centre line for the window buttons, in points below the top of
// the window frame; 0 asks for the system's own placement. Written from any
// thread, read on the main thread by the re-assertion.
static _Atomic double vgio_desktop_center = 0;

// The requested leading edge for the buttons as a group, in points in from
// the leading edge of the window frame; 0 asks for the system's own x.
// Written from any thread, read on the main thread by the re-assertion.
static _Atomic double vgio_desktop_lead = 0;

// Whether the buttons currently stand where a request put them. Main thread
// only. It exists so that dropping a request restores the system placement
// exactly once, and so that nothing is touched at all in the overwhelmingly
// common case of a window that never asked.
static bool vgio_desktop_placed = false;

// The horizontal counterpart of vgio_desktop_placed, and the record that
// makes its restore exact. The vertical restore recomputes the system's own
// centring from the strip and the button height; the x positions have no
// such formula — they are AppKit's to choose — so the first horizontal move
// records them, per button in its superview's space plus the group's leading
// edge in window coordinates, and both the move and the restore are stated
// against that record: absolute, idempotent, exactly reversible. Main thread
// only.
static bool vgio_desktop_moved_x = false;
static double vgio_desktop_home_x[3];
static double vgio_desktop_home_lead = 0;

// The resize subscription. AppKit re-lays the title-bar container on every
// size change, which undoes a placement, and Gio's configuration notification
// does not fire for a user drag of the window's corner — so the placement
// subscribes to the window's own resize notification and re-applies there.
static id vgio_desktop_resize_token = nil;
static __weak NSWindow *vgio_desktop_observed = nil;

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

// Places the buttons where the last request asked for them, on the main
// thread, and answers whether a request is in force. strip is the measured
// height of the native title-bar strip.
//
// The placement is absolute, never incremental: each application recomputes
// every frame from the window's own metrics, so applying it twice with no
// reconfigure in between lands on exactly the same geometry, and dropping the
// request restores what AppKit itself would have drawn.
//
// The buttons are moved inside their own superview, and that superview and the
// container above it are grown downwards to hold them, because a view only
// hit-tests points inside its own bounds: a button hanging below its
// container's edge would still draw and would no longer be clickable. Growing
// the container does not take the enlarged band away from the application —
// the container's own views are transparent here and decline the hit — and it
// does not change what the window reports as its content layout rect.
static bool vgio_desktop_place_main(NSWindow *w, double strip) {
	double center = atomic_load(&vgio_desktop_center);
	double lead = atomic_load(&vgio_desktop_lead);
	if (center <= 0 && lead <= 0 && !vgio_desktop_placed && !vgio_desktop_moved_x) {
		return false;
	}
	NSView *close = [w standardWindowButton:NSWindowCloseButton];
	if (close == nil || strip <= 0) {
		return false;
	}
	NSView *bar = [close superview];
	NSView *container = [bar superview];
	double button = NSHeight([close frame]);
	if (bar == nil || container == nil || button <= 0) {
		return false;
	}

	// Where the top edge of a button goes, in points below the top of the
	// window frame: from the requested centre line, or centred in the strip
	// the way the system centres it.
	double top = (strip - button) / 2;
	if (center > 0) {
		top = center - button / 2;
		if (top < 0) {
			top = 0;
		}
	}
	double height = strip;
	if (top + button > height) {
		height = top + button;
	}

	NSRect c = [container frame];
	c.size.height = height;
	c.origin.y = NSHeight([w frame]) - height;
	[container setFrame:c];
	NSRect b = [bar frame];
	b.origin.y = 0;
	b.size.height = height;
	[bar setFrame:b];

	// The first horizontal move records where AppKit itself had the buttons —
	// they stand untouched on that axis until a move happens, so the record is
	// the system's own geometry. No sideways growth is needed to keep them
	// hit-testing: the title-bar views span the window's whole width, so a
	// button stays inside its container's bounds wherever a sane leading edge
	// puts it, where the vertical move really can push one below the edge.
	if (lead > 0 && !vgio_desktop_moved_x) {
		bool have = false;
		for (size_t i = 0; i < sizeof(vgio_desktop_buttons) / sizeof(vgio_desktop_buttons[0]); i++) {
			NSView *v = [w standardWindowButton:vgio_desktop_buttons[i]];
			if (v == nil) {
				continue;
			}
			vgio_desktop_home_x[i] = [v frame].origin.x;
			double minx = NSMinX([v convertRect:[v bounds] toView:nil]);
			if (!have || minx < vgio_desktop_home_lead) {
				vgio_desktop_home_lead = minx;
				have = true;
			}
		}
	}

	for (size_t i = 0; i < sizeof(vgio_desktop_buttons) / sizeof(vgio_desktop_buttons[0]); i++) {
		NSView *v = [w standardWindowButton:vgio_desktop_buttons[i]];
		if (v == nil) {
			continue;
		}
		NSRect f = [v frame];
		// The superview's y grows upwards from its own bottom edge, and its
		// top edge is pinned to the window's.
		f.origin.y = height - top - NSHeight(f);
		// Each button shifts by the same delta, so the group keeps AppKit's
		// own spacing; the delta is a difference of window-frame x positions,
		// which a translation-only view hierarchy keeps equal to a superview-
		// space delta. With the request dropped, the recorded x is put back
		// verbatim.
		if (lead > 0) {
			f.origin.x = vgio_desktop_home_x[i] + (lead - vgio_desktop_home_lead);
		} else if (vgio_desktop_moved_x) {
			f.origin.x = vgio_desktop_home_x[i];
		}
		[v setFrame:f];
	}
	vgio_desktop_moved_x = lead > 0;
	// Only the vertical request takes the row: TopInset answers who owns the
	// strip, and a horizontal move leaves the buttons on the system's own
	// line inside a strip that is still the system's.
	vgio_desktop_placed = center > 0;
	return vgio_desktop_placed;
}

// Runs on the main thread only.
static void vgio_desktop_reassert_main(void) {
	NSWindow *w = vgio_desktop_window();
	if (w == nil) {
		return;
	}
	if (vgio_desktop_observed != w) {
		if (vgio_desktop_resize_token != nil) {
			[[NSNotificationCenter defaultCenter] removeObserver:vgio_desktop_resize_token];
		}
		vgio_desktop_resize_token = [[NSNotificationCenter defaultCenter]
			addObserverForName:NSWindowDidResizeNotification
			            object:w
			             queue:[NSOperationQueue mainQueue]
			        usingBlock:^(NSNotification *n) {
				vgio_desktop_reassert_main();
			}];
		vgio_desktop_observed = w;
		// A different window means the recorded x positions belong to a
		// window that is gone; the next horizontal move re-records them
		// from the new window's own geometry.
		vgio_desktop_moved_x = false;
	}
	[[w standardWindowButton:NSWindowCloseButton] setHidden:NO];
	[[w standardWindowButton:NSWindowMiniaturizeButton] setHidden:NO];
	[[w standardWindowButton:NSWindowZoomButton] setHidden:NO];

	// The strip AppKit reserves for itself. It is measured before the buttons
	// are placed and is unaffected by placing them: moving views inside the
	// title bar does not change what the window calls its content layout rect.
	double strip = NSHeight([w frame]) - NSHeight([w contentLayoutRect]);
	bool placed = vgio_desktop_place_main(w, strip);

	// With the buttons placed by a caller, the row is the caller's and there
	// is nothing above its content left to pad for; without one, the strip
	// stands where AppKit put it.
	double inset = placed ? 0 : strip;

	// The buttons' frames are in their superview's coordinate space — the
	// title-bar view, not the content view — so they cannot be compared with
	// a content-relative measurement as they stand. convertRect:toView:nil
	// puts each one into the window's own base coordinates, and subtracting
	// the content layout rect's origin restates it in the space the content
	// is laid out in. Under the full-size-content treatment that origin is
	// zero and the subtraction changes nothing; it is written out anyway so
	// the value stays correct if the window ever stops spanning its frame.
	double leading = 0;
	for (size_t i = 0; i < sizeof(vgio_desktop_buttons) / sizeof(vgio_desktop_buttons[0]); i++) {
		NSView *b = [w standardWindowButton:vgio_desktop_buttons[i]];
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

void vgio_desktop_place_buttons(double leading, double center) {
	atomic_store(&vgio_desktop_lead, leading);
	atomic_store(&vgio_desktop_center, center);
	// The request only becomes geometry under the re-assertion, which is the
	// same path a reconfigure takes; asking for one now applies it at once
	// without a second code path that could drift from the first.
	vgio_desktop_reassert();
}

double vgio_desktop_top_inset(void) {
	return atomic_load(&vgio_desktop_inset);
}

double vgio_desktop_leading_inset(void) {
	return atomic_load(&vgio_desktop_leading);
}
