// The C surface of the menu half's Objective-C (menu_darwin.m). The Go side
// (menu_darwin.go) owns the declaration and the timing; the functions
// below are safe to call from any thread — the AppKit work is dispatched.

#ifndef VGIO_MVU_DESKTOP_MENU_DARWIN_H
#define VGIO_MVU_DESKTOP_MENU_DARWIN_H

// vgio_menu_reset drops the whole declaration. The bar itself is not touched
// until the next vgio_menu_apply, which is what removes the items already in
// it — so reset then declare then apply replaces the amendment atomically as
// far as the bar is concerned.
extern void vgio_menu_reset(void);

// The flags of a declared item.
enum {
	VGIO_MENU_DISABLED = 1, // a person cannot choose the item
	VGIO_MENU_CONTROL = 2,  // the key equivalent adds Control to Command
	VGIO_MENU_SUBMENU = 4,  // the item opens a submenu of the items that name it as parent
};

// vgio_menu_declare appends one item to the declaration: menu is the title of
// the menu it sits in, or "" for the application's own menu; title is the
// item's label; key is its key equivalent, "" for none, with the command
// modifier implied and Shift for an uppercase letter; tag is the integer the
// choice is reported back under; parent is the tag of the submenu item that
// holds the item, or 0; flags holds the VGIO_MENU flags. The strings are
// copied.
extern void vgio_menu_declare(const char *menu, const char *title, const char *key, int tag, int parent, int flags);

// vgio_menu_stage appends one item to the staged items, which the next
// vgio_menu_replace takes. The arguments are those of vgio_menu_declare.
extern void vgio_menu_stage(const char *title, const char *key, int tag, int flags);

// vgio_menu_replace makes the staged items the items of the submenu item
// with tag parent, in one step, and empties the stage. The bar shows them
// when AppKit next updates the submenu or searches it for a key
// equivalent, on the main thread.
extern void vgio_menu_replace(int parent);

// vgio_menu_set gives the declared item with tag a new title and disabled
// state. The bar reads them when AppKit validates the item, on the main
// thread, so this changes no menu item itself. The title is copied. A tag
// that the declaration does not hold changes nothing.
extern void vgio_menu_set(int tag, const char *title, int disabled);

// vgio_menu_lookup gives the declared item's title as a new C string that
// the caller frees, and its disabled state in *disabled. It gives NULL when
// the declaration holds no item with tag.
extern char *vgio_menu_lookup(int tag, int *disabled);

// vgio_menu_test_build builds a menu bar with one menu titled menu, which
// holds the declared items of that menu, and keeps it for the next
// vgio_menu_test_press and vgio_menu_test_shown. It needs no application.
extern void vgio_menu_test_build(const char *menu);

// vgio_menu_test_press presses one key equivalent on the kept bar: the
// characters chars with Command, and with Shift and Control when asked. An
// arrow key carries its key code and the function and numeric pad flags,
// as a keyboard sends it. It gives 1 when the bar takes the key, and 0
// when the key goes on to the window. No action runs.
extern int vgio_menu_test_press(const char *chars, int shift, int control);

// vgio_menu_test_shown updates and validates the submenu titled submenu of
// the kept bar, as AppKit does before it shows the submenu, and gives one
// line per item:
// its title, and " off" after a disabled item. The caller frees the
// string. It gives NULL when the bar holds no such submenu.
extern char *vgio_menu_test_shown(const char *submenu);

// vgio_menu_apply makes the bar match the declaration, on the AppKit main
// thread: every item this package added before is removed, and the current
// declaration is added in its place. Asynchronous unless already on the main
// thread; a no-op while no application or no main menu exists.
extern void vgio_menu_apply(void);

#endif
