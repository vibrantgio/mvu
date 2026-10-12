//go:build darwin

// The stand-in dragging source: the three Objective-C objects a released drag
// presents to a drop target, built by hand.
//
// A file promise needs a live delegate — the receiver calls back into it to
// have the file written — so the delegates are kept for the life of the
// process rather than released with the call that made one. A test process
// makes a handful.

#import <AppKit/AppKit.h>

#include "perform_darwin.h"

// VGIODropSourcePromise writes the promised bytes when the receiving side
// asks for them, which is the whole of what a promise source does.
@interface VGIODropSourcePromise : NSObject <NSFilePromiseProviderDelegate>
@property (nonatomic, copy) NSString *name;
@property (nonatomic, copy) NSData *bytes;
@end

@implementation VGIODropSourcePromise

- (NSString *)filePromiseProvider:(NSFilePromiseProvider *)provider
                   fileNameForType:(NSString *)type {
	return self.name;
}

- (void)filePromiseProvider:(NSFilePromiseProvider *)provider
          writePromiseToURL:(NSURL *)url
          completionHandler:(void (^)(NSError *))done {
	NSError *err = nil;
	[self.bytes writeToURL:url options:NSDataWritingAtomic error:&err];
	done(err);
}

// The queue the writing runs on. Named explicitly rather than left to the
// default, which is the main queue: the main thread belongs to the window's
// event loop while a test drives a drop into it.
- (NSOperationQueue *)operationQueueForFilePromiseProvider:(NSFilePromiseProvider *)provider {
	static NSOperationQueue *queue;
	static dispatch_once_t once;
	dispatch_once(&once, ^{
		queue = [[NSOperationQueue alloc] init];
		queue.name = @"mvu-desktop drop source promises";
	});
	return queue;
}

@end

// VGIODropSourceInfo is the dragging info a released drag hands the view. The
// drop path asks it for the pasteboard and the drag location and nothing
// else, so those are what it answers; it is passed as an untyped object
// because the full NSDraggingInfo protocol is the drag session's to implement.
@interface VGIODropSourceInfo : NSObject
@property (nonatomic, strong) NSPasteboard *pasteboard;
@property (nonatomic, assign) NSPoint where;
@end

@implementation VGIODropSourceInfo
- (NSPasteboard *)draggingPasteboard { return self.pasteboard; }
- (NSPoint)draggingLocation { return self.where; }
- (id)draggingSource { return nil; }
- (NSDragOperation)draggingSourceOperationMask { return NSDragOperationCopy; }
@end

// vgio_dropsource_delegates holds every promise delegate the process made.
static NSMutableArray *vgio_dropsource_delegates(void) {
	static NSMutableArray *kept;
	static dispatch_once_t once;
	dispatch_once(&once, ^{ kept = [NSMutableArray array]; });
	return kept;
}

// vgio_dropsource_view finds the window a test opened: the key window when
// there is one, else the first window holding a content view.
static NSView *vgio_dropsource_view(void) {
	NSWindow *window = NSApp.keyWindow;
	if (window.contentView != nil) {
		return window.contentView;
	}
	for (NSWindow *w in NSApp.windows) {
		if (w.contentView != nil) {
			return w.contentView;
		}
	}
	return nil;
}

// vgio_dropsource_standing is the drag that stands over the window: the
// dragging info the view is handed on every call until the next begin.
static VGIODropSourceInfo *vgio_dropsource_standing;

int vgio_dropsource_begin(const char *url, const char *promise_name,
                          const void *promise_bytes, int promise_length) {
	// The drag pasteboard, not a pasteboard of this call's own: a promise
	// receiver read off any other pasteboard has no drag to belong to.
	NSPasteboard *pb = [NSPasteboard pasteboardWithName:NSPasteboardNameDrag];
	[pb clearContents];

	NSMutableArray *items = [NSMutableArray array];
	if (url != NULL) {
		[items addObject:[NSURL fileURLWithPath:@(url)]];
	}
	if (promise_name != NULL) {
		VGIODropSourcePromise *delegate = [[VGIODropSourcePromise alloc] init];
		delegate.name = @(promise_name);
		delegate.bytes = [NSData dataWithBytes:promise_bytes length:(NSUInteger)promise_length];
		[vgio_dropsource_delegates() addObject:delegate];
		[items addObject:[[NSFilePromiseProvider alloc] initWithFileType:@"public.plain-text"
		                                                       delegate:delegate]];
	}
	if (![pb writeObjects:items]) {
		return -2;
	}

	VGIODropSourceInfo *info = [[VGIODropSourceInfo alloc] init];
	info.pasteboard = pb;
	vgio_dropsource_standing = info;
	return 0;
}

// vgio_dropsource_send puts the standing drag at x, y and sends it to the
// view. The selector is sent to an untyped object so no conformance to the
// dragging-info protocol is claimed for the stand-in.
static int vgio_dropsource_send(SEL selector, double x, double y) {
	NSView *view = vgio_dropsource_view();
	if (view == nil) {
		return -1;
	}
	if (vgio_dropsource_standing == nil) {
		return -3;
	}
	vgio_dropsource_standing.where = NSMakePoint(x, y);

	id target = view;
	id info = vgio_dropsource_standing;
	if (selector == @selector(draggingEntered:)) {
		return [target draggingEntered:info] == NSDragOperationNone ? 0 : 1;
	}
	if (selector == @selector(draggingExited:)) {
		[target draggingExited:info];
		return 0;
	}
	return [target performDragOperation:info] ? 1 : 0;
}

int vgio_dropsource_enter(double x, double y) {
	return vgio_dropsource_send(@selector(draggingEntered:), x, y);
}

int vgio_dropsource_exit(void) {
	return vgio_dropsource_send(@selector(draggingExited:), 0, 0);
}

int vgio_dropsource_drop(double x, double y) {
	return vgio_dropsource_send(@selector(performDragOperation:), x, y);
}
