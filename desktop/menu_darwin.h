// The C surface of the menu half's Objective-C (menu_darwin.m). The Go side
// (menu_darwin.go) owns the declaration and the timing; the three functions
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
// The strings are copied.
extern void vgio_menu_declare(const char *menu, const char *title, const char *key, int tag);

// vgio_menu_apply makes the bar match the declaration, on the AppKit main
// thread: every item this package added before is removed, and the current
// declaration is added in its place. Asynchronous unless already on the main
// thread; a no-op while no application or no main menu exists.
extern void vgio_menu_apply(void);

#endif
