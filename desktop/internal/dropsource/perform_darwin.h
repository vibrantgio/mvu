// The C surface of the stand-in dragging source (perform_darwin.m). Every
// function here is an AppKit call and runs on the main thread only. One drag
// stands at a time: begin replaces whatever stood before it.

#ifndef VGIO_MVU_DESKTOP_DROPSOURCE_PERFORM_DARWIN_H
#define VGIO_MVU_DESKTOP_DROPSOURCE_PERFORM_DARWIN_H

// vgio_dropsource_begin writes the given file URL (NULL for none) and file
// promise (a NULL name for none) to the drag pasteboard and builds the
// dragging info the drag will hand the view. Returns 0, or -2 when the
// pasteboard refused the items.
extern int vgio_dropsource_begin(const char *url, const char *promise_name,
                                 const void *promise_bytes, int promise_length);

// vgio_dropsource_enter, _exit and _drop send draggingEntered:,
// draggingExited: and performDragOperation: to the frontmost window's content
// view, with the drag at x, y in window coordinates.
//
// Enter returns 1 when the view answered with a drag operation and 0 when it
// refused the drag; drop returns 1 when the view accepted the drop. Both
// return -1 when there is no window with a content view and -3 when no drag
// stands.
extern int vgio_dropsource_enter(double x, double y);
extern int vgio_dropsource_exit(void);
extern int vgio_dropsource_drop(double x, double y);

#endif
