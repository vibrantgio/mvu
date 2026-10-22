// The C surface of the scroll bar setting's Objective-C
// (scrollbars_darwin.m). The call is Foundation rather than AppKit and reads
// a preference, so it needs no hop to the main queue.

#ifndef VGIO_MVU_DESKTOP_SCROLLBARS_DARWIN_H
#define VGIO_MVU_DESKTOP_SCROLLBARS_DARWIN_H

// vgio_desktop_scroll_bars answers the user's "Show scroll bars" setting as
// the platform's own string — "Automatic", "WhenScrolling" or "Always" — and
// NULL where the setting has never been written, which is the platform's
// Automatic. The string is malloc'd and the caller frees it: an autoreleased
// buffer would be gone before Go has copied out of it.
extern char *vgio_desktop_scroll_bars(void);

#endif
