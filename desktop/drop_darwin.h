// The C surface of the drop half's Objective-C (drop_darwin.m). The Go side
// (drop_darwin.go) owns the lifecycle: both functions below are AppKit calls
// and run on the main thread only, reached through the window's Run.

#ifndef VGIO_MVU_DESKTOP_DROP_DARWIN_H
#define VGIO_MVU_DESKTOP_DROP_DARWIN_H

#include <stdint.h>

// Views cross this boundary as uintptr_t, the integer form Gio hands them
// out in; the Objective-C side bridges back to a view reference. Passing the
// integer keeps Go's unsafe.Pointer rules out of the picture entirely.

// vgio_drop_add_methods adds the drag-destination selectors to the class of
// the given view. Per-class and therefore process-global and permanent; the
// Go side guards it with sync.Once. Returns the number of methods actually
// added. Selectors the class already defines ITSELF are left untouched —
// inherited default implementations do not count, which is deliberate: the
// system view class inherits inert drag selectors, and a guard that honored
// inherited ones would veto the augmentation entirely, while a view class
// that grows its own upstream drag support must win over this package's.
extern int vgio_drop_add_methods(uintptr_t view);

// vgio_drop_register registers one view instance for file-URL drags.
// Per-instance: it runs for every view, every time it attaches to a window.
extern void vgio_drop_register(uintptr_t view);

#endif
