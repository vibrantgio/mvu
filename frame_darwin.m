//go:build darwin

#import <AppKit/AppKit.h>

#include "frame_darwin.h"

// The Go side of the watch (frame_darwin.go), declared the way the other
// Objective-C in the organization declares its Go exports rather than through
// the generated header.
extern void vgioFrameChanged(double x, double y, double width, double height);

// The move-and-resize watch. One window at a time: an application that
// remembers its frame has one window whose frame is worth remembering, and a
// second call for a second window would need a second file to write it to.
static id vgio_frame_token_move = nil;
static id vgio_frame_token_size = nil;
static __weak NSWindow *vgio_frame_observed = nil;

// The application's window: its first titled window, the same window the
// chrome treatment finds, since the full-size-content treatment keeps
// NSWindowStyleMaskTitled.
static NSWindow *vgio_frame_window(void) {
	for (NSWindow *w in [NSApp windows]) {
		if ([w styleMask] & NSWindowStyleMaskTitled) {
			return w;
		}
	}
	return nil;
}

// The line every vertical conversion is measured from: the top of the primary
// screen, in AppKit's own upward coordinates. AppKit puts the primary screen's
// origin at zero, so this is simply its height, and both conversions below are
// the same subtraction read in opposite directions.
static double vgio_frame_top(void) {
	NSScreen *primary = [[NSScreen screens] firstObject];
	if (primary == nil) {
		return 0;
	}
	return NSMaxY([primary frame]);
}

int vgio_frame_displays(VgioFrame *out, int max) {
	if (out == NULL || max <= 0) {
		return 0;
	}
	uint32_t count = 0;
	CGDirectDisplayID ids[32];
	uint32_t room = (uint32_t)(max < 32 ? max : 32);
	if (CGGetActiveDisplayList(room, ids, &count) != kCGErrorSuccess) {
		return 0;
	}
	if (count > room) {
		count = room;
	}
	for (uint32_t i = 0; i < count; i++) {
		// Quartz already states display bounds from the top-left of the main
		// display with y downwards, in points — the space mvu.Frame uses — so
		// nothing here needs converting.
		CGRect b = CGDisplayBounds(ids[i]);
		out[i].x = b.origin.x;
		out[i].y = b.origin.y;
		out[i].width = b.size.width;
		out[i].height = b.size.height;
	}
	return (int)count;
}

void vgio_frame_set(double x, double y, double width, double height) {
	NSWindow *w = vgio_frame_window();
	if (w == nil || width <= 0 || height <= 0) {
		return;
	}
	NSRect f = NSMakeRect(x, vgio_frame_top() - (y + height), width, height);
	// display:NO because the move happens while the window is already showing
	// its first frame; Gio redraws on the resize notification that follows.
	[w setFrame:f display:NO];
}

// Hands the window's current frame to the Go side, converted out of AppKit's
// upward coordinates. Main thread only, which is where every caller is.
static void vgio_frame_report(NSWindow *w) {
	NSRect f = [w frame];
	vgioFrameChanged(NSMinX(f), vgio_frame_top() - NSMaxY(f), NSWidth(f), NSHeight(f));
}

void vgio_frame_observe(void) {
	NSWindow *w = vgio_frame_window();
	if (w == nil) {
		return;
	}
	if (vgio_frame_observed != w) {
		NSNotificationCenter *center = [NSNotificationCenter defaultCenter];
		if (vgio_frame_token_move != nil) {
			[center removeObserver:vgio_frame_token_move];
			[center removeObserver:vgio_frame_token_size];
		}
		// Both notifications, because neither implies the other: a drag of the
		// title bar moves without resizing, a drag of the bottom-right corner
		// resizes without moving, and a drag of the top-left does both. Gio's
		// own configuration notification fires for neither.
		vgio_frame_token_move = [center
			addObserverForName:NSWindowDidMoveNotification
			            object:w
			             queue:[NSOperationQueue mainQueue]
			        usingBlock:^(NSNotification *n) {
				vgio_frame_report([n object]);
			}];
		vgio_frame_token_size = [center
			addObserverForName:NSWindowDidResizeNotification
			            object:w
			             queue:[NSOperationQueue mainQueue]
			        usingBlock:^(NSNotification *n) {
				vgio_frame_report([n object]);
			}];
		vgio_frame_observed = w;
	}
	// The frame the window opens with, which the Go side keeps as the baseline
	// it writes nothing for.
	vgio_frame_report(w);
}
