//go:build darwin

package desktop

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Foundation

#include <stdlib.h>

#include "download_darwin.h"
*/
import "C"

import (
	"os"
	"path/filepath"
	"time"
	"unsafe"
)

// icloudPlaceholderSuffix is what the platform appends to the hidden
// stand-in it leaves beside a file whose contents it has not downloaded:
// `note.md` is kept as `.note.md.icloud`, and the item has no other name
// on disk until the download writes it.
const icloudPlaceholderSuffix = ".icloud"

// download is the darwin watch: ask the platform for the item, then read
// its state at the poll interval until the file stands at its own path
// with contents behind it, the platform records a failure against it, or
// the deadline passes.
//
// Polling is what the platform leaves: the resource values carry the state
// and nothing notifies a plain Foundation caller of a change to them. An
// NSMetadataQuery would notify, but it needs a run loop of its own and
// reports over a whole scope rather than about one item, which is more
// machinery than one file wants.
//
// Two shapes of "not downloaded" exist and this handles both. The platform
// may leave the file at its own path with no contents behind it, where the
// downloading status is the only thing that distinguishes it from a note
// that is fully here; or it may leave the hidden placeholder alone, where
// the file's own path holds nothing at all until the download writes it.
// So the watch ends on the status having moved off not-downloaded AND the
// file existing, which is the one condition both shapes reach.
func download(path string, poll, deadline time.Duration) Downloaded {
	if refusal := startDownload(path); refusal != "" {
		return Downloaded{Path: path, Err: refusal}
	}
	until := time.Now().Add(deadline)
	for {
		state, failure := downloadState(path)
		if failure != "" {
			return Downloaded{Path: path, Err: failure}
		}
		if state != C.VGIO_DOWNLOAD_NOT_DOWNLOADED && fileExists(path) {
			debugf("desktop: %s is downloaded", path)
			return Downloaded{Path: path}
		}
		if !time.Now().Add(poll).Before(until) {
			return Downloaded{Path: path, Err: "the download did not finish in time"}
		}
		time.Sleep(poll)
	}
}

// startDownload asks the platform for one item and answers the refusal,
// empty when the ask was taken. A file the platform keeps as a hidden
// placeholder has nothing at its own path, so a refusal there is asked
// again at the placeholder's path, that being the only name the item has
// on disk until the download writes the file. The first refusal is the one
// reported, since it names the path the caller asked about.
func startDownload(path string) string {
	refusal := startDownloadAt(path)
	if refusal == "" {
		return ""
	}
	if hidden := placeholderPath(path); hidden != "" {
		if again := startDownloadAt(hidden); again == "" {
			return ""
		}
	}
	return refusal
}

// startDownloadAt makes the platform's ask for exactly the path given.
func startDownloadAt(path string) string {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	refusal := C.vgio_download_start(cpath)
	if refusal == nil {
		return ""
	}
	defer C.free(unsafe.Pointer(refusal))
	return C.GoString(refusal)
}

// downloadState reads the item's downloading state and the error the
// platform has recorded against it, if any.
func downloadState(path string) (state int, failure string) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	var cfailure *C.char
	state = int(C.vgio_download_state(cpath, &cfailure))
	if cfailure != nil {
		failure = C.GoString(cfailure)
		C.free(unsafe.Pointer(cfailure))
	}
	return state, failure
}

// placeholderPath is the hidden stand-in beside path, and empty when no
// such file is there — including when path names the placeholder itself,
// which must not grow a second suffix.
func placeholderPath(path string) string {
	dir, name := filepath.Split(path)
	if name == "" || name[0] == '.' {
		return ""
	}
	hidden := filepath.Join(dir, "."+name+icloudPlaceholderSuffix)
	if !fileExists(hidden) {
		return ""
	}
	return hidden
}

// fileExists reports whether anything stands at path. A stat that fails
// for any other reason than absence answers false too: the watch wants
// the file readable, and a path it cannot stat is not that.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
