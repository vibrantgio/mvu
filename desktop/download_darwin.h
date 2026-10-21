// The C surface of the download half's Objective-C (download_darwin.m).
// Both calls are Foundation rather than AppKit, so neither is made on the
// main queue: the Go side (download_darwin.go) runs them on the command's
// own goroutine, where a wait costs the window nothing.

#ifndef VGIO_MVU_DESKTOP_DOWNLOAD_DARWIN_H
#define VGIO_MVU_DESKTOP_DOWNLOAD_DARWIN_H

// Every string answered here is malloc'd and the caller frees it. An
// autoreleased buffer would be gone before Go has copied out of it.

// The downloading states an item can be in, the platform's own set.
#define VGIO_DOWNLOAD_UNKNOWN 0        // the item answers no downloading status
#define VGIO_DOWNLOAD_NOT_DOWNLOADED 1 // no copy of the contents on this machine
#define VGIO_DOWNLOAD_DOWNLOADED 2     // a copy is here; the cloud may hold a newer one
#define VGIO_DOWNLOAD_CURRENT 3        // the copy here is the newest there is

// vgio_download_start asks the platform to fetch the item at path out of
// the cloud. It answers NULL when the ask was taken, and a description of
// the refusal when it was not: a path under no file provider — nothing
// there is in the cloud — and a path naming no item both refuse here, so
// the caller never waits for a download that was never going to happen.
extern char *vgio_download_start(const char *path);

// vgio_download_state reads one item's downloading state: the return is
// one of the states above, and *failure is set to the error the platform
// has recorded against the item when it has recorded one.
//
// The URL is built afresh on every call. NSURL caches the resource values
// it has read, so a URL kept across polls answers the first poll's state
// for the rest of the download.
extern int vgio_download_state(const char *path, char **failure);

#endif
