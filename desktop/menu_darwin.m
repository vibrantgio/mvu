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
@property(nonatomic, assign) BOOL disabled;
@end

@implementation VGioMenuDecl
@end

// The declared item with tag, or nil. Any thread.
static VGioMenuDecl *vgio_menu_decl_for(int tag);

// The tag of the enabled item in the menu titled menu whose key equivalent
// chars and shift press, or 0. Any thread.
static int vgio_menu_key_tag_for(NSString *menu, NSString *chars, BOOL shift);

// The target and action that a key equivalent of a declared item resolves
// to. It carries the item's tag, so its action reads no sender.
@interface VGioMenuKey : NSObject
@property(nonatomic, assign) int itemTag;
- (void)vgioMenuKey:(id)sender;
@end

@implementation VGioMenuKey
- (void)vgioMenuKey:(id)sender {
	vgioMenuChosen(self.itemTag);
}
@end

// The target every added item points at. An item whose target is nil is
// offered to the responder chain and disabled when nobody answers; naming a
// target that answers the selector is what keeps these items live regardless
// of what has focus. validateMenuItem: gives the item its declared title
// and enables it unless the declaration disables it. AppKit calls it on the
// main thread when the item's menu opens and when its key equivalent is
// pressed, so a new state reaches the bar with no dispatch of its own.
//
// The target is also the delegate of each menu that this package creates.
// menuHasKeyEquivalent: answers a key equivalent from the declaration: an
// enabled item takes its key, and a disabled item does not. AppKit then
// gives the key to the key window, so a disabled item never takes a key
// that the window reads.
@interface VGioMenuTarget : NSObject <NSMenuDelegate>
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
	NSEventModifierFlags flags = [event modifierFlags];
	NSEventModifierFlags held = flags & (NSEventModifierFlagCommand | NSEventModifierFlagOption |
	                                     NSEventModifierFlagControl);
	if (held != NSEventModifierFlagCommand) {
		return NO;
	}
	int tag = vgio_menu_key_tag_for([menu title], [event charactersIgnoringModifiers],
	                                (flags & NSEventModifierFlagShift) != 0);
	if (tag == 0) {
		return NO;
	}
	VGioMenuKey *k = [VGioMenuKey new];
	k.itemTag = tag;
	*target = k;
	*action = @selector(vgioMenuKey:);
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

void vgio_menu_declare(const char *menu, const char *title, const char *key, int tag, int disabled) {
	@autoreleasepool {
		VGioMenuDecl *d = [VGioMenuDecl new];
		d.menu = menu != NULL ? @(menu) : @"";
		d.title = title != NULL ? @(title) : @"";
		d.key = key != NULL ? @(key) : @"";
		d.itemTag = tag;
		d.disabled = disabled != 0;
		NSMutableArray<VGioMenuDecl *> *decls = vgio_menu_declarations();
		@synchronized(decls) {
			[decls addObject:d];
		}
	}
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
				d.disabled = disabled != 0;
				decls[i] = d;
				return;
			}
		}
	}
}

// A key equivalent with a cased letter matches that letter, and an
// uppercase letter needs Shift, as AppKit draws it. Any other key
// equivalent matches the characters exactly.
static int vgio_menu_key_tag_for(NSString *menu, NSString *chars, BOOL shift) {
	if (menu == nil || chars == nil || [chars length] == 0) {
		return 0;
	}
	NSMutableArray<VGioMenuDecl *> *decls = vgio_menu_declarations();
	@synchronized(decls) {
		for (VGioMenuDecl *d in decls) {
			NSString *k = d.key;
			if (d.disabled || [k length] == 0 || ![d.menu isEqualToString:menu]) {
				continue;
			}
			NSString *lower = [k lowercaseString];
			BOOL match;
			if ([lower isEqualToString:[k uppercaseString]]) {
				match = [k isEqualToString:chars];
			} else {
				match = [lower isEqualToString:[chars lowercaseString]] &&
				        shift == ![k isEqualToString:lower];
			}
			if (match) {
				return d.itemTag;
			}
		}
	}
	return 0;
}

int vgio_menu_key_tag(const char *menu, const char *chars, int shift) {
	@autoreleasepool {
		return vgio_menu_key_tag_for(menu != NULL ? @(menu) : @"", chars != NULL ? @(chars) : @"",
		                             shift != 0);
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
		[item setEnabled:!d.disabled];
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
