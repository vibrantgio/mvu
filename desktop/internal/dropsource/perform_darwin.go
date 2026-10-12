//go:build darwin

package dropsource

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AppKit

#include <stdlib.h>
#include "perform_darwin.h"
*/
import "C"

import (
	"fmt"
	"unsafe"
)

func begin(url string, promise *Promise) error {
	var curl *C.char
	if url != "" {
		curl = C.CString(url)
		defer C.free(unsafe.Pointer(curl))
	}
	var cname *C.char
	var bytes unsafe.Pointer
	var length C.int
	if promise != nil {
		cname = C.CString(promise.Name)
		defer C.free(unsafe.Pointer(cname))
		if len(promise.Bytes) > 0 {
			bytes = C.CBytes(promise.Bytes)
			defer C.free(bytes)
			length = C.int(len(promise.Bytes))
		}
	}
	if code := int(C.vgio_dropsource_begin(curl, cname, bytes, length)); code != 0 {
		return reported(code)
	}
	return nil
}

func send(which call, x, y float64) (bool, error) {
	var code int
	switch which {
	case enterDrag:
		code = int(C.vgio_dropsource_enter(C.double(x), C.double(y)))
	case exitDrag:
		code = int(C.vgio_dropsource_exit())
	case dropDrag:
		code = int(C.vgio_dropsource_drop(C.double(x), C.double(y)))
	}
	switch code {
	case 1:
		return true, nil
	case 0:
		return false, nil
	default:
		return false, reported(code)
	}
}

// reported turns a native code into the condition it stands for.
func reported(code int) error {
	switch code {
	case -1:
		return fmt.Errorf("dropsource: no window with a content view to drag over")
	case -2:
		return fmt.Errorf("dropsource: the drag pasteboard refused the items")
	case -3:
		return fmt.Errorf("dropsource: no drag stands over the window")
	default:
		return fmt.Errorf("dropsource: the drag reported %d", code)
	}
}
