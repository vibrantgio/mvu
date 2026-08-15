//go:build darwin

// The native half of file drops, kept as thin as the seam allows: the
// Objective-C side does only what cannot leave AppKit — pasteboard checks,
// pasteboard reads, and the one geometry call [view convertPoint:fromView:nil]
// — and ships the RAW transform components (view-point x/y, bounds height,
// backing scale) to Go, where the flip-and-scale math lives as a tested pure
// function (transform.go). The scale is re-read from the window on every
// event, never cached: a window can move between displays of different scale
// mid-drag.
//
// Every function here runs on the AppKit main thread. The drag callbacks
// must never block — blocking freezes the compositor mid-drag — and the Go
// handlers they call hand off through a non-blocking channel send. The
// callbacks identify their window by passing the view's own pointer through;
// the Go side routes on it, which is what keeps several windows' drops from
// cross-talking.

#import <AppKit/AppKit.h>
#import <objc/runtime.h>

#include "drop_darwin.h"

// Go-exported handlers (drop_darwin.go). kind is the Go dragKind ordinal:
// 0 enter, 1 move, 2 exit; drops travel via vgioDropPaths. paths is a buffer
// of count NUL-terminated UTF-8 paths laid end to end.
extern void vgioDropUpdate(uintptr_t view, int kind, double x, double y,
                           double viewHeight, double scale);
extern void vgioDropPaths(uintptr_t view, const char *paths, int length, int count,
                          double x, double y, double viewHeight, double scale);

// vgio_drop_handle is the identity a view crosses the boundary under: the
// same integer Gio published in its view event, which is what the Go side
// keys its routing on.
static uintptr_t vgio_drop_handle(id view) {
	return (uintptr_t)(__bridge void *)view;
}

// vgio_drop_components extracts the raw transform components for the current
// drag location: view-point x/y (the one conversion that must stay in
// AppKit), the view bounds height, and the backing scale factor — re-read on
// every event.
static void vgio_drop_components(NSView *view, id<NSDraggingInfo> sender,
                                 double *x, double *y, double *height, double *scale) {
	NSPoint p = [view convertPoint:[sender draggingLocation] fromView:nil];
	*x = p.x;
	*y = p.y;
	*height = view.bounds.size.height;
	*scale = view.window ? view.window.backingScaleFactor : 1.0;
}

static BOOL vgio_drop_has_file_urls(id<NSDraggingInfo> sender) {
	NSPasteboard *pb = [sender draggingPasteboard];
	return [pb canReadObjectForClasses:@[ NSURL.class ]
	                           options:@{NSPasteboardURLReadingFileURLsOnlyKey : @YES}];
}

// draggingEntered: — encoding "Q@:@". The cursor answer is deliberately
// coarse: Copy whenever file URLs are present at all, refined per zone by
// the messages rather than the cursor. A drag without file URLs (dragged
// text, say) is refused with None — AppKit animates the refusal, no drop
// callback ever fires, and nothing crosses into Go at all.
static NSDragOperation vgio_drop_entered(id self, SEL _cmd, id<NSDraggingInfo> sender) {
	if (!vgio_drop_has_file_urls(sender)) {
		return NSDragOperationNone;
	}
	double x, y, h, s;
	vgio_drop_components((NSView *)self, sender, &x, &y, &h, &s);
	vgioDropUpdate(vgio_drop_handle(self), 0 /* dragEnter */, x, y, h, s);
	return NSDragOperationCopy;
}

// draggingUpdated: — encoding "Q@:@". Fires on every pointer motion.
static NSDragOperation vgio_drop_updated(id self, SEL _cmd, id<NSDraggingInfo> sender) {
	if (!vgio_drop_has_file_urls(sender)) {
		return NSDragOperationNone;
	}
	double x, y, h, s;
	vgio_drop_components((NSView *)self, sender, &x, &y, &h, &s);
	vgioDropUpdate(vgio_drop_handle(self), 1 /* dragMove */, x, y, h, s);
	return NSDragOperationCopy;
}

// draggingExited: — encoding "v@:@". The drag left the window (or was
// cancelled); position is irrelevant, only the fact crosses.
static void vgio_drop_exited(id self, SEL _cmd, id<NSDraggingInfo> sender) {
	vgioDropUpdate(vgio_drop_handle(self), 2 /* dragExit */, 0, 0, 0, 1);
}

// performDragOperation: — encoding "c@:@". Returns YES the moment the
// pasteboard read succeeds; YES means "the data was accepted", never "the
// application finished handling it" — zone resolution and message delivery
// are asynchronous and this return never waits for them. A drop outside
// every zone still returns YES here: the refinement is Go-side silence, the
// OS-level answer stays coarse.
static BOOL vgio_drop_perform(id self, SEL _cmd, id<NSDraggingInfo> sender) {
	NSPasteboard *pb = [sender draggingPasteboard];
	NSArray<NSURL *> *urls =
	    [pb readObjectsForClasses:@[ NSURL.class ]
	                      options:@{NSPasteboardURLReadingFileURLsOnlyKey : @YES}];
	if (urls == nil || urls.count == 0) {
		return NO;
	}

	double x, y, h, s;
	vgio_drop_components((NSView *)self, sender, &x, &y, &h, &s);

	NSMutableData *buf = [NSMutableData data];
	for (NSURL *u in urls) {
		const char *path = u.path.UTF8String;
		[buf appendBytes:path length:strlen(path) + 1]; // include the NUL
	}
	vgioDropPaths(vgio_drop_handle(self), buf.bytes, (int)buf.length,
	              (int)urls.count, x, y, h, s);
	return YES;
}

// vgio_drop_owns_method reports whether cls ITSELF defines sel, ignoring
// inherited implementations. This precise form matters: NSView inherits
// inert default drag selectors, so respondsToSelector-style checks answer
// YES for them and would veto the whole augmentation, while the actual
// intent of the guard is "never clobber an implementation the view class
// grows itself". Own-method detection is also exactly the condition under
// which class_addMethod refuses to act, so guard and contract agree.
static BOOL vgio_drop_owns_method(Class cls, SEL sel) {
	unsigned int n = 0;
	Method *ms = class_copyMethodList(cls, &n);
	BOOL found = NO;
	for (unsigned int i = 0; i < n; i++) {
		if (method_getName(ms[i]) == sel) {
			found = YES;
			break;
		}
	}
	free(ms);
	return found;
}

int vgio_drop_add_methods(uintptr_t view) {
	Class cls = object_getClass((__bridge id)(void *)view);
	int added = 0;
	SEL sels[4] = {
	    @selector(draggingEntered:),
	    @selector(draggingUpdated:),
	    @selector(draggingExited:),
	    @selector(performDragOperation:),
	};
	IMP imps[4] = {
	    (IMP)vgio_drop_entered,
	    (IMP)vgio_drop_updated,
	    (IMP)vgio_drop_exited,
	    (IMP)vgio_drop_perform,
	};
	const char *encs[4] = {"Q@:@", "Q@:@", "v@:@", "c@:@"};

	for (int i = 0; i < 4; i++) {
		if (vgio_drop_owns_method(cls, sels[i])) {
			// The class defines this selector itself — upstream grew its
			// own drag support. Leave it in charge.
			NSLog(@"mvu/desktop: %s defines %s itself; leaving it untouched",
			      class_getName(cls), sel_getName(sels[i]));
		} else if (class_addMethod(cls, sels[i], imps[i], encs[i])) {
			added++;
		} else {
			NSLog(@"mvu/desktop: class_addMethod(%s) failed", sel_getName(sels[i]));
		}
	}
	return added;
}

void vgio_drop_register(uintptr_t view) {
	NSView *v = (__bridge NSView *)(void *)view;
	[v registerForDraggedTypes:@[ NSPasteboardTypeFileURL ]];
}
