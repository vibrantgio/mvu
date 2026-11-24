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

// vgio_menu_declare appends one item to the declaration: menu is the title of
// the menu it sits in, or "" for the application's own menu; title is the
// item's label; key is its key equivalent, "" for none, with the command
// modifier implied; tag is the integer the choice is reported back under.
// disabled is 1 for an item that a person cannot choose. The strings are
// copied.
extern void vgio_menu_declare(const char *menu, const char *title, const char *key, int tag, int disabled);

// vgio_menu_set gives the declared item with tag a new title and disabled
// state. The bar reads them when AppKit validates the item, on the main
// thread, so this changes no menu item itself. The title is copied. A tag
// that the declaration does not hold changes nothing.
extern void vgio_menu_set(int tag, const char *title, int disabled);

// vgio_menu_lookup gives the declared item's title as a new C string that
// the caller frees, and its disabled state in *disabled. It gives NULL when
// the declaration holds no item with tag.
extern char *vgio_menu_lookup(int tag, int *disabled);

// vgio_menu_apply makes the bar match the declaration, on the AppKit main
// thread: every item this package added before is removed, and the current
// declaration is added in its place. Asynchronous unless already on the main
// thread; a no-op while no application or no main menu exists.
extern void vgio_menu_apply(void);

#endif
