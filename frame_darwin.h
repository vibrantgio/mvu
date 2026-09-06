// The C surface of the frame memory's Objective-C (frame_darwin.m). macOS is
// the one platform this package reads and applies a window position on: Gio's
// app.Config carries a size and no origin, so the origin is taken from, and
// given to, the native window.
//
// AppKit points equal Gio dp, so no scale conversion crosses this boundary.
// What does get converted is the vertical direction: AppKit measures a
// window's origin from the bottom of the primary screen upwards, and every
// frame here is stated the way mvu.Frame is — from the top-left of the
// primary screen, y downwards, which is also Quartz's own display space.

#ifndef VGIO_MVU_FRAME_DARWIN_H
#define VGIO_MVU_FRAME_DARWIN_H

// VgioFrame is one rectangle in that top-left-origin space, in points.
typedef struct {
	double x;
	double y;
	double width;
	double height;
} VgioFrame;

// vgio_frame_displays fills out with the bounds of up to max active displays
// and returns how many it wrote. It asks Quartz Display Services rather than
// AppKit precisely so that it may be called from any thread, which is what
// lets the restore decision be made before the event loop exists.
extern int vgio_frame_displays(VgioFrame *out, int max);

// vgio_frame_set moves and resizes the application's window. Main thread
// only, and a no-op while no application or no titled window exists.
extern void vgio_frame_set(double x, double y, double width, double height);

// vgio_frame_observe reports the application's window's frame to the Go side
// now and on every later move and resize of it. Main thread only, and a no-op
// while no application or no titled window exists. Calling it again for a
// window already being watched changes nothing; calling it for a different
// window moves the watch to that one, so one watch exists at a time.
extern void vgio_frame_observe(void);

#endif
