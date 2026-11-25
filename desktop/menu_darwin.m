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

#include <stdlib.h>
#include <string.h>

#include "menu_darwin.h"

// The Go-exported handler (menu_darwin.go). tag identifies the chosen item.
extern void vgioMenuChosen(int tag);

// One declared item. Immutable once made, so the main thread may read it
// while the declaring goroutine holds the lock over the array alone. A new
// title or disabled state replaces the object in the array.
@interface VGioMenuDecl : NSObject
@property(nonatomic, copy) NSString *menu;
@property(nonatomic, copy) NSString *title;
@property(nonatomic, copy) NSString *key;
@property(nonatomic, assign) int itemTag;
@property(nonatomic, assign) int parentTag;
@property(nonatomic, assign) BOOL disabled;
@property(nonatomic, assign) BOOL control;
@property(nonatomic, assign) BOOL submenu;
@end

@implementation VGioMenuDecl
@end

// The declared item with tag, or nil. Any thread.
static VGioMenuDecl *vgio_menu_decl_for(int tag);

// Gives each declared item of menu its key equivalent, or none for a
// disabled item when forKey is YES, and does the same in each submenu of
// menu. A submenu whose declared items changed gets the new items first.
// Main thread only.
static void vgio_menu_sync_keys(NSMenu *menu, BOOL forKey);

// The target every added item points at. An item whose target is nil is
// offered to the responder chain and disabled when nobody answers; naming a
// target that answers the selector is what keeps these items live regardless
// of what has focus. validateMenuItem: gives the item its declared title
// and enables it unless the declaration disables it. AppKit calls it on the
// main thread when the item's menu opens and when its key equivalent is
// pressed, so a new state reaches the bar with no dispatch of its own.
//
// The target is also the delegate of each menu that this package creates.
// AppKit searches a menu's items for a key equivalent after its delegate's
// menuHasKeyEquivalent: answers NO, and a disabled item takes a key that
// it matches, with no action. A headless probe measured both on macOS 26.
// So menuHasKeyEquivalent: takes the key equivalent from each disabled
// item before the search, and answers NO. The search then finds no
// disabled item, and AppKit gives the key to the key window.
// menuNeedsUpdate: gives each item its key equivalent again before the
// menu shows, so a disabled item shows its key. The probe also measured
// that AppKit asks the delegate of the menu before the delegate of each
// submenu in it. Each menu still syncs its submenus itself, so the route
// does not depend on that order.
@interface VGioMenuTarget : NSObject <NSMenuDelegate, NSMenuItemValidation>
- (void)vgioMenuChosen:(id)sender;
@end

@implementation VGioMenuTarget
- (void)vgioMenuChosen:(id)sender {
	if ([sender isKindOfClass:[NSMenuItem class]]) {
		vgioMenuChosen((int)[(NSMenuItem *)sender tag]);
	}
}
- (BOOL)validateMenuItem:(NSMenuItem *)item {
	VGioMenuDecl *d = vgio_menu_decl_for((int)[item tag]);
	if (d == nil) {
		return YES;
	}
	if (![[item title] isEqualToString:d.title]) {
		[item setTitle:d.title];
	}
	return !d.disabled;
}
- (BOOL)menuHasKeyEquivalent:(NSMenu *)menu
                    forEvent:(NSEvent *)event
                      target:(id _Nullable *)target
                      action:(SEL _Nullable *)action {
	vgio_menu_sync_keys(menu, YES);
	return NO;
}
- (void)menuNeedsUpdate:(NSMenu *)menu {
	vgio_menu_sync_keys(menu, NO);
}
@end

// The declaration, written by the declaring goroutine and read on the main
// thread; vgio_menu_decls is the lock token for both.
static NSMutableArray<VGioMenuDecl *> *vgio_menu_decls = nil;

// The items that the next vgio_menu_replace takes. vgio_menu_decls is the
// lock token.
static NSMutableArray<VGioMenuDecl *> *vgio_menu_staged = nil;

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

static VGioMenuDecl *vgio_menu_new_decl(const char *menu, const char *title, const char *key, int tag, int parent, int flags) {
	VGioMenuDecl *d = [VGioMenuDecl new];
	d.menu = menu != NULL ? @(menu) : @"";
	d.title = title != NULL ? @(title) : @"";
	d.key = key != NULL ? @(key) : @"";
	d.itemTag = tag;
	d.parentTag = parent;
	d.disabled = (flags & VGIO_MENU_DISABLED) != 0;
	d.control = (flags & VGIO_MENU_CONTROL) != 0;
	d.submenu = (flags & VGIO_MENU_SUBMENU) != 0;
	return d;
}

void vgio_menu_declare(const char *menu, const char *title, const char *key, int tag, int parent, int flags) {
	@autoreleasepool {
		VGioMenuDecl *d = vgio_menu_new_decl(menu, title, key, tag, parent, flags);
		NSMutableArray<VGioMenuDecl *> *decls = vgio_menu_declarations();
		@synchronized(decls) {
			[decls addObject:d];
		}
	}
}

void vgio_menu_stage(const char *title, const char *key, int tag, int flags) {
	@autoreleasepool {
		VGioMenuDecl *d = vgio_menu_new_decl(NULL, title, key, tag, 0, flags);
		NSMutableArray<VGioMenuDecl *> *decls = vgio_menu_declarations();
		@synchronized(decls) {
			if (vgio_menu_staged == nil) {
				vgio_menu_staged = [NSMutableArray array];
			}
			[vgio_menu_staged addObject:d];
		}
	}
}

void vgio_menu_replace(int parent) {
	@autoreleasepool {
		NSMutableArray<VGioMenuDecl *> *decls = vgio_menu_declarations();
		@synchronized(decls) {
			NSArray<VGioMenuDecl *> *staged = vgio_menu_staged != nil ? [vgio_menu_staged copy] : @[];
			[vgio_menu_staged removeAllObjects];
			VGioMenuDecl *holder = nil;
			for (VGioMenuDecl *d in decls) {
				if (d.itemTag == parent) {
					holder = d;
				}
			}
			if (holder == nil) {
				return;
			}
			NSIndexSet *old = [decls indexesOfObjectsPassingTest:^BOOL(VGioMenuDecl *d, NSUInteger i, BOOL *stop) {
				return d.parentTag == parent;
			}];
			[decls removeObjectsAtIndexes:old];
			for (VGioMenuDecl *d in staged) {
				d.menu = holder.menu;
				d.parentTag = parent;
				[decls addObject:d];
			}
		}
	}
}

// The declared items of the submenu item with tag parent, in order. Any
// thread.
static NSArray<VGioMenuDecl *> *vgio_menu_children(int parent) {
	NSMutableArray<VGioMenuDecl *> *out = [NSMutableArray array];
	NSMutableArray<VGioMenuDecl *> *decls = vgio_menu_declarations();
	@synchronized(decls) {
		for (VGioMenuDecl *d in decls) {
			if (d.parentTag == parent) {
				[out addObject:d];
			}
		}
	}
	return out;
}

static VGioMenuDecl *vgio_menu_decl_for(int tag) {
	NSMutableArray<VGioMenuDecl *> *decls = vgio_menu_declarations();
	@synchronized(decls) {
		for (VGioMenuDecl *d in decls) {
			if (d.itemTag == tag) {
				return d;
			}
		}
	}
	return nil;
}

void vgio_menu_set(int tag, const char *title, int disabled) {
	@autoreleasepool {
		NSMutableArray<VGioMenuDecl *> *decls = vgio_menu_declarations();
		@synchronized(decls) {
			for (NSUInteger i = 0; i < [decls count]; i++) {
				VGioMenuDecl *old = decls[i];
				if (old.itemTag != tag) {
					continue;
				}
				VGioMenuDecl *d = [VGioMenuDecl new];
				d.menu = old.menu;
				d.title = title != NULL ? @(title) : @"";
				d.key = old.key;
				d.itemTag = tag;
				d.parentTag = old.parentTag;
				d.disabled = disabled != 0;
				d.control = old.control;
				d.submenu = old.submenu;
				decls[i] = d;
				return;
			}
		}
	}
}

static NSMenuItem *vgio_menu_item(VGioMenuDecl *d, VGioMenuTarget *target);

// Gives submenu the items that the submenu item d declares, when its
// items are not those. Main thread only.
static void vgio_menu_sync_items(NSMenu *submenu, VGioMenuDecl *d) {
	NSArray<VGioMenuDecl *> *kids = vgio_menu_children(d.itemTag);
	BOOL same = [submenu numberOfItems] == (NSInteger)[kids count];
	for (NSUInteger i = 0; same && i < [kids count]; i++) {
		same = [[submenu itemAtIndex:i] tag] == kids[i].itemTag;
	}
	if (same) {
		return;
	}
	VGioMenuTarget *target = (VGioMenuTarget *)[submenu delegate];
	[submenu removeAllItems];
	for (VGioMenuDecl *k in kids) {
		[submenu addItem:vgio_menu_item(k, target)];
	}
}

static void vgio_menu_sync_keys(NSMenu *menu, BOOL forKey) {
	for (NSMenuItem *item in [[menu itemArray] copy]) {
		VGioMenuDecl *d = vgio_menu_decl_for((int)[item tag]);
		if (d == nil) {
			continue;
		}
		if (d.submenu) {
			// AppKit validates no submenu item, so its title is given here.
			if (![[item title] isEqualToString:d.title]) {
				[item setTitle:d.title];
			}
			vgio_menu_sync_items([item submenu], d);
			vgio_menu_sync_keys([item submenu], forKey);
			continue;
		}
		NSString *key = forKey && d.disabled ? @"" : d.key;
		if (![[item keyEquivalent] isEqualToString:key]) {
			[item setKeyEquivalent:key];
		}
	}
}

// The menu item that shows declaration d and posts its choice to target.
// A submenu item holds a submenu of its declared items, and target is the
// submenu's delegate. AppKit enables a submenu item and validates none.
static NSMenuItem *vgio_menu_item(VGioMenuDecl *d, VGioMenuTarget *target) {
	if (d.submenu) {
		NSMenuItem *item = [[NSMenuItem alloc] initWithTitle:d.title action:NULL keyEquivalent:@""];
		NSMenu *sub = [[NSMenu alloc] initWithTitle:d.title];
		[sub setDelegate:target];
		for (VGioMenuDecl *k in vgio_menu_children(d.itemTag)) {
			[sub addItem:vgio_menu_item(k, target)];
		}
		[item setSubmenu:sub];
		[item setTag:d.itemTag];
		return item;
	}
	NSMenuItem *item = [[NSMenuItem alloc] initWithTitle:d.title
	                                              action:@selector(vgioMenuChosen:)
	                                       keyEquivalent:d.key];
	NSEventModifierFlags mask = NSEventModifierFlagCommand;
	if (d.control) {
		mask |= NSEventModifierFlagControl;
	}
	[item setKeyEquivalentModifierMask:mask];
	[item setEnabled:!d.disabled];
	[item setTarget:target];
	[item setTag:d.itemTag];
	return item;
}

// The bar, its one menu and the target that vgio_menu_test_build made.
// Test only.
static NSMenu *vgio_menu_test_bar = nil;
static NSMenu *vgio_menu_test_menu = nil;
static VGioMenuTarget *vgio_menu_test_target = nil;

void vgio_menu_test_build(const char *menu) {
	@autoreleasepool {
		NSString *title = menu != NULL ? @(menu) : @"";
		vgio_menu_test_target = [VGioMenuTarget new];
		vgio_menu_test_bar = [[NSMenu alloc] initWithTitle:@""];
		vgio_menu_test_menu = [[NSMenu alloc] initWithTitle:title];
		[vgio_menu_test_menu setDelegate:vgio_menu_test_target];
		NSMenuItem *top = [[NSMenuItem alloc] initWithTitle:title action:NULL keyEquivalent:@""];
		[top setSubmenu:vgio_menu_test_menu];
		[vgio_menu_test_bar addItem:top];
		for (VGioMenuDecl *d in vgio_menu_children(0)) {
			if ([d.menu isEqualToString:title]) {
				[vgio_menu_test_menu addItem:vgio_menu_item(d, vgio_menu_test_target)];
			}
		}
	}
}

int vgio_menu_test_press(const char *chars, int shift, int control) {
	@autoreleasepool {
		NSString *ignoring = chars != NULL ? @(chars) : @"";
		NSString *typed = [ignoring lowercaseString];
		NSEventModifierFlags flags = NSEventModifierFlagCommand;
		if (shift != 0) {
			flags |= NSEventModifierFlagShift;
		}
		if (control != 0) {
			flags |= NSEventModifierFlagControl;
		}
		unsigned short code = 0;
		if ([ignoring length] == 1) {
			unichar c = [ignoring characterAtIndex:0];
			if (c == NSUpArrowFunctionKey || c == NSDownArrowFunctionKey) {
				code = c == NSUpArrowFunctionKey ? 126 : 125;
				flags |= NSEventModifierFlagFunction | NSEventModifierFlagNumericPad;
				typed = ignoring;
			}
		}
		NSEvent *event = [NSEvent keyEventWithType:NSEventTypeKeyDown
		                                  location:NSZeroPoint
		                             modifierFlags:flags
		                                 timestamp:0
		                              windowNumber:0
		                                   context:nil
		                                characters:typed
		                   charactersIgnoringModifiers:ignoring
		                                 isARepeat:NO
		                                   keyCode:code];
		return [vgio_menu_test_bar performKeyEquivalent:event] ? 1 : 0;
	}
}

// The submenu titled title in menu or in a submenu of menu, or nil.
static NSMenu *vgio_menu_find(NSMenu *menu, NSString *title) {
	for (NSMenuItem *item in [menu itemArray]) {
		NSMenu *sub = [item submenu];
		if (sub == nil) {
			continue;
		}
		if ([[sub title] isEqualToString:title]) {
			return sub;
		}
		NSMenu *found = vgio_menu_find(sub, title);
		if (found != nil) {
			return found;
		}
	}
	return nil;
}

char *vgio_menu_test_shown(const char *submenu) {
	@autoreleasepool {
		[vgio_menu_test_target menuNeedsUpdate:vgio_menu_test_menu];
		NSMenu *sub = vgio_menu_find(vgio_menu_test_menu, submenu != NULL ? @(submenu) : @"");
		if (sub == nil) {
			return NULL;
		}
		[vgio_menu_test_target menuNeedsUpdate:sub];
		// Without an application, NSMenu's update validates no item, so
		// each item is validated here, as AppKit validates it.
		NSMutableArray<NSString *> *lines = [NSMutableArray array];
		for (NSMenuItem *item in [sub itemArray]) {
			BOOL enabled = [vgio_menu_test_target validateMenuItem:item];
			[lines addObject:enabled ? [item title] : [[item title] stringByAppendingString:@" off"]];
		}
		return strdup([[lines componentsJoinedByString:@"\n"] UTF8String]);
	}
}

char *vgio_menu_lookup(int tag, int *disabled) {
	@autoreleasepool {
		VGioMenuDecl *d = vgio_menu_decl_for(tag);
		if (d == nil) {
			return NULL;
		}
		if (disabled != NULL) {
			*disabled = d.disabled ? 1 : 0;
		}
		return strdup([d.title UTF8String]);
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
	[menu setDelegate:vgio_menu_target];
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
	decls = vgio_menu_children(0);

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
		NSMenuItem *item = vgio_menu_item(d, vgio_menu_target);
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
