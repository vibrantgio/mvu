//go:build darwin

// The native half of the application menu. Gio builds a fixed menu bar — one
// application menu holding Hide and Quit — in its own darwin glue, before it
// starts the run loop, and exposes no way to add to it. NSApp's main menu is
// an ordinary mutable NSMenu all the same, so this amends it in place: items
// are inserted at the top of the application's own menu, and menus named by
// the declaration are appended to the bar beside it. Nothing here forks,
// vendors or patches Gio.
//
// The declaration is held here rather than in Go because the amendment must
// happen on the AppKit main thread and the hop is asynchronous — Gio's frame
// handoff parks the main thread, so a synchronous hop from the notification
// this rides on would deadlock there, exactly as the window buttons'
// re-assertion documents. A block that ran over Go memory after the hop would
// be reading it from another thread; copies of the strings cross once, when
// the declaration is made, and the main thread reads only its own.

#import <AppKit/AppKit.h>

#include "menu_darwin.h"

// The Go-exported handler (menu_darwin.go). tag identifies the chosen item.
extern void vgioMenuChosen(int tag);

// One declared item. Immutable once made, so the main thread may read it
// while the declaring goroutine holds the lock over the array alone.
@interface VGioMenuDecl : NSObject
@property(nonatomic, copy) NSString *menu;
@property(nonatomic, copy) NSString *title;
@property(nonatomic, copy) NSString *key;
@property(nonatomic, assign) int itemTag;
@end

@implementation VGioMenuDecl
@end

// The target every added item points at. An item whose target is nil is
// offered to the responder chain and disabled when nobody answers; naming a
// target that answers the selector is what keeps these items live regardless
// of what has focus. validateMenuItem: says so explicitly rather than leaving
// it to AppKit's automatic enabling — a message an application declared is
// always sendable.
@interface VGioMenuTarget : NSObject
- (void)vgioMenuChosen:(id)sender;
@end

@implementation VGioMenuTarget
- (void)vgioMenuChosen:(id)sender {
	if ([sender isKindOfClass:[NSMenuItem class]]) {
		vgioMenuChosen((int)[(NSMenuItem *)sender tag]);
	}
}
- (BOOL)validateMenuItem:(NSMenuItem *)item {
	return YES;
}
@end

// The declaration, written by the declaring goroutine and read on the main
// thread; vgio_menu_decls is the lock token for both.
static NSMutableArray<VGioMenuDecl *> *vgio_menu_decls = nil;

// What the last apply put into the bar, so the next one can take it out
// again: the added items, plus the separator that fences them off from Gio's
// Hide and Quit, plus the top-level menu items whose menus this created.
// Main thread only.
static NSMutableArray<NSMenuItem *> *vgio_menu_added = nil;

// The one target instance, retained for the life of the process because the
// items hold it weakly the way AppKit holds every menu target. Main thread
// only.
static VGioMenuTarget *vgio_menu_target = nil;

static NSMutableArray<VGioMenuDecl *> *vgio_menu_declarations(void) {
	if (vgio_menu_decls == nil) {
		vgio_menu_decls = [NSMutableArray array];
	}
	return vgio_menu_decls;
}

void vgio_menu_reset(void) {
	@autoreleasepool {
		NSMutableArray<VGioMenuDecl *> *decls = vgio_menu_declarations();
		@synchronized(decls) {
			[decls removeAllObjects];
		}
	}
}

void vgio_menu_declare(const char *menu, const char *title, const char *key, int tag) {
	@autoreleasepool {
		VGioMenuDecl *d = [VGioMenuDecl new];
		d.menu = menu != NULL ? @(menu) : @"";
		d.title = title != NULL ? @(title) : @"";
		d.key = key != NULL ? @(key) : @"";
		d.itemTag = tag;
		NSMutableArray<VGioMenuDecl *> *decls = vgio_menu_declarations();
		@synchronized(decls) {
			[decls addObject:d];
		}
	}
}

// The application's own menu: the submenu of the first item of the bar, which
// is where macOS puts it and where Gio's Hide and Quit already are.
static NSMenu *vgio_menu_app_menu(NSMenu *bar) {
	if ([bar numberOfItems] == 0) {
		return nil;
	}
	return [[bar itemAtIndex:0] submenu];
}

// The menu a declaration names, creating it on first use. "" is the
// application's own; anything else is a menu of that title in the bar, added
// after whatever is already there. created maps a title to the menu made for
// it during this apply, so two items naming one menu land in one menu.
static NSMenu *vgio_menu_for(NSMenu *bar, NSString *title,
                             NSMutableDictionary<NSString *, NSMenu *> *created) {
	if ([title length] == 0) {
		return vgio_menu_app_menu(bar);
	}
	NSMenu *menu = created[title];
	if (menu != nil) {
		return menu;
	}
	// The bar shows the SUBMENU's title, not the item's, so both are set.
	menu = [[NSMenu alloc] initWithTitle:title];
	NSMenuItem *item = [[NSMenuItem alloc] initWithTitle:title action:NULL keyEquivalent:@""];
	[item setSubmenu:menu];
	[bar addItem:item];
	[vgio_menu_added addObject:item];
	created[title] = menu;
	return menu;
}

// Runs on the main thread only.
static void vgio_menu_apply_main(void) {
	NSMenu *bar = [NSApp mainMenu];
	if (bar == nil) {
		return;
	}
	if (vgio_menu_target == nil) {
		vgio_menu_target = [VGioMenuTarget new];
	}
	if (vgio_menu_added == nil) {
		vgio_menu_added = [NSMutableArray array];
	}

	// Out with the previous amendment, item by item and from whatever menu
	// each one ended up in — a top-level item is removed from the bar, and
	// removing it takes its whole menu with it.
	for (NSMenuItem *item in vgio_menu_added) {
		[[item menu] removeItem:item];
	}
	[vgio_menu_added removeAllObjects];

	NSArray<VGioMenuDecl *> *decls = nil;
	NSMutableArray<VGioMenuDecl *> *live = vgio_menu_declarations();
	@synchronized(live) {
		decls = [live copy];
	}

	NSMutableDictionary<NSString *, NSMenu *> *created = [NSMutableDictionary dictionary];
	// Items destined for the application's own menu go in at the top, in
	// declaration order, above Gio's Hide and Quit — which is where this
	// platform keeps Settings and everything else an application adds there.
	NSInteger appIndex = 0;
	for (VGioMenuDecl *d in decls) {
		NSMenu *menu = vgio_menu_for(bar, d.menu, created);
		if (menu == nil) {
			continue;
		}
		NSMenuItem *item = [[NSMenuItem alloc] initWithTitle:d.title
		                                              action:@selector(vgioMenuChosen:)
		                                       keyEquivalent:d.key];
		[item setKeyEquivalentModifierMask:NSEventModifierFlagCommand];
		[item setTarget:vgio_menu_target];
		[item setTag:d.itemTag];
		if ([d.menu length] == 0) {
			[menu insertItem:item atIndex:appIndex++];
		} else {
			[menu addItem:item];
		}
		[vgio_menu_added addObject:item];
	}
	// A rule under the application's own additions, so they read as this
	// application's rather than as more of the system's Hide and Quit.
	if (appIndex > 0) {
		NSMenu *appMenu = vgio_menu_app_menu(bar);
		NSMenuItem *sep = [NSMenuItem separatorItem];
		[appMenu insertItem:sep atIndex:appIndex];
		[vgio_menu_added addObject:sep];
	}
}

void vgio_menu_apply(void) {
	if (NSApp == nil) {
		// No NSApplication: a test binary, or a call before app.Main. There
		// is no bar to amend, and dispatching to a main queue nobody drains
		// would wedge the caller.
		return;
	}
	if ([NSThread isMainThread]) {
		@autoreleasepool {
			vgio_menu_apply_main();
		}
		return;
	}
	dispatch_async(dispatch_get_main_queue(), ^{
		@autoreleasepool {
			vgio_menu_apply_main();
		}
	});
}
