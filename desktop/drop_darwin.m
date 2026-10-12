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
// Every drag callback here runs on the AppKit main thread. The callbacks
// must never block — blocking freezes the compositor mid-drag — and the Go
// handlers they call hand off through a non-blocking channel send. The
// callbacks identify their window by passing the view's own pointer through;
// the Go side routes on it, which is what keeps several windows' drops from
// cross-talking.
//
// A promised file is the one thing here that outlives its callback. A source
// that keeps its files in a library of its own puts no file URL on the
// pasteboard: it puts a promise, and the file exists only once the destination
// asks the promise receiver to write it into a folder the destination names.
// So the drop callback starts the receiving and returns, the promise readers
// run later on the queue below, and the counting and assembly that turn them
// back into one drop live on the Go side — the same split the transform
// follows, for the same reason: neither needs AppKit.

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
// A drop carrying promises opens with vgioDropPromiseBegin — which takes how
// many files the receivers promise between them and returns the id they are
// reported under — and reports each one with vgioDropPromiseFile: path set and
// failure NULL when the file was written, path NULL and failure set when it
// was not. The Go side delivers the one drop when the last file is in.
extern uint64_t vgioDropPromiseBegin(uintptr_t view, const char *paths, int length,
                                     int count, int promised, double x, double y,
                                     double viewHeight, double scale);
extern void vgioDropPromiseFile(uint64_t drop, const char *path, const char *failure);

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

// vgio_drop_has_files reports whether the drag carries files in either of the
// two shapes a source can offer them: file URLs, which name files that exist,
// and file promises, which name files the source will write when asked. Both
// answer the same way to the cursor and produce the same messages, so the
// hover callbacks ask this one question.
static BOOL vgio_drop_has_files(id<NSDraggingInfo> sender) {
	NSPasteboard *pb = [sender draggingPasteboard];
	if ([pb canReadObjectForClasses:@[ NSURL.class ]
	                        options:@{NSPasteboardURLReadingFileURLsOnlyKey : @YES}]) {
		return YES;
	}
	return [pb canReadObjectForClasses:@[ NSFilePromiseReceiver.class ] options:@{}];
}

// vgio_drop_queue is the queue the package hands every promise receiver, kept
// for the life of the process. It runs one reader at a time, so the files of a
// drop arrive one by one however many receivers are writing them, and the Go
// side can simply append each one as it comes.
static NSOperationQueue *vgio_drop_queue(void) {
	static NSOperationQueue *queue;
	static dispatch_once_t once;
	dispatch_once(&once, ^{
		queue = [[NSOperationQueue alloc] init];
		queue.name = @"mvu-desktop promised files";
		queue.maxConcurrentOperationCount = 1;
	});
	return queue;
}

// vgio_drop_promised is how many files one receiver will call its reader for.
// A receiver names its files up front only for a promise written the legacy
// way; a promise from a file-promise provider leaves fileNames empty and
// promises exactly one file, which is the provider's own shape — it is
// constructed around one file type and asked for one file name. Measured on
// macOS 26: fileNames is nil for a provider-written promise both before and
// during receiving, so the count has to come from the shape, not from the
// receiver.
static int vgio_drop_promised(NSFilePromiseReceiver *receiver) {
	NSUInteger named = receiver.fileNames.count;
	return named > 0 ? (int)named : 1;
}

// vgio_drop_destination makes the folder one drop's promised files are written
// into: a folder named for the package under the system's temporary directory,
// with a fresh subfolder per drop so two drops of the same file name cannot
// collide. The package deletes nothing it creates here — the files are the
// application's to read, and only the application knows when it is done with
// them. Returns nil when the folder cannot be made, which leaves the drop to
// deliver whatever plain URLs it also carried.
static NSString *vgio_drop_destination(void) {
	NSString *root = [NSTemporaryDirectory() stringByAppendingPathComponent:@"mvu-desktop"];
	NSString *dir = [root stringByAppendingPathComponent:[[NSUUID UUID] UUIDString]];
	NSError *err = nil;
	if (![NSFileManager.defaultManager createDirectoryAtPath:dir
	                            withIntermediateDirectories:YES
	                                             attributes:nil
	                                                  error:&err]) {
		return nil;
	}
	return dir;
}

// draggingEntered: — encoding "Q@:@". The cursor answer is deliberately
// coarse: Copy whenever files are present at all, refined per zone by the
// messages rather than the cursor. A drag carrying no files (dragged text,
// say) is refused with None — AppKit animates the refusal, no drop callback
// ever fires, and nothing crosses into Go at all.
static NSDragOperation vgio_drop_entered(id self, SEL _cmd, id<NSDraggingInfo> sender) {
	if (!vgio_drop_has_files(sender)) {
		return NSDragOperationNone;
	}
	double x, y, h, s;
	vgio_drop_components((NSView *)self, sender, &x, &y, &h, &s);
	vgioDropUpdate(vgio_drop_handle(self), 0 /* dragEnter */, x, y, h, s);
	return NSDragOperationCopy;
}

// draggingUpdated: — encoding "Q@:@". Fires on every pointer motion.
static NSDragOperation vgio_drop_updated(id self, SEL _cmd, id<NSDraggingInfo> sender) {
	if (!vgio_drop_has_files(sender)) {
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
// are asynchronous and this return never waits for them, and for a promise it
// returns long before the file exists. A drop outside every zone still returns
// YES here: the refinement is Go-side silence, the OS-level answer stays
// coarse.
//
// The drag location must be read here, inside the callback: the dragging info
// is only valid for the duration of the call, so the components travel to Go
// with the drop and the promise readers never touch the sender again.
static BOOL vgio_drop_perform(id self, SEL _cmd, id<NSDraggingInfo> sender) {
	NSPasteboard *pb = [sender draggingPasteboard];
	NSArray<NSURL *> *urls =
	    [pb readObjectsForClasses:@[ NSURL.class ]
	                      options:@{NSPasteboardURLReadingFileURLsOnlyKey : @YES}];
	NSArray<NSFilePromiseReceiver *> *receivers =
	    [pb readObjectsForClasses:@[ NSFilePromiseReceiver.class ] options:@{}];
	if (urls.count == 0 && receivers.count == 0) {
		return NO;
	}

	double x, y, h, s;
	vgio_drop_components((NSView *)self, sender, &x, &y, &h, &s);

	NSMutableData *buf = [NSMutableData data];
	for (NSURL *u in urls) {
		const char *path = u.path.UTF8String;
		[buf appendBytes:path length:strlen(path) + 1]; // include the NUL
	}

	uintptr_t handle = vgio_drop_handle(self);
	if (receivers.count == 0) {
		vgioDropPaths(handle, buf.bytes, (int)buf.length, (int)urls.count, x, y, h, s);
		return YES;
	}

	// Without a folder to write into there is nothing to ask the promises for,
	// so the plain URLs are the whole drop.
	NSString *dir = vgio_drop_destination();
	if (dir == nil) {
		if (urls.count > 0) {
			vgioDropPaths(handle, buf.bytes, (int)buf.length, (int)urls.count, x, y, h, s);
		}
		return YES;
	}

	int promised = 0;
	for (NSFilePromiseReceiver *r in receivers) {
		promised += vgio_drop_promised(r);
	}
	uint64_t drop = vgioDropPromiseBegin(handle, buf.bytes, (int)buf.length,
	                                     (int)urls.count, promised, x, y, h, s);
	NSURL *dest = [NSURL fileURLWithPath:dir isDirectory:YES];
	NSOperationQueue *queue = vgio_drop_queue();
	for (NSFilePromiseReceiver *r in receivers) {
		[r receivePromisedFilesAtDestination:dest
		                            options:@{}
		                     operationQueue:queue
		                             reader:^(NSURL *file, NSError *err) {
			if (err != nil) {
				vgioDropPromiseFile(drop, NULL,
				                    err.localizedDescription.UTF8String);
				return;
			}
			vgioDropPromiseFile(drop, file.path.UTF8String, NULL);
		}];
	}
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
	NSMutableArray<NSPasteboardType> *types =
	    [NSMutableArray arrayWithObject:NSPasteboardTypeFileURL];
	// The promise receiver names the types a promised file arrives under; a
	// view that does not register them is never offered a promise drag at all.
	[types addObjectsFromArray:NSFilePromiseReceiver.readableDraggedTypes];
	[v registerForDraggedTypes:types];
}
