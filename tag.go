package vibrant

import (
	"strconv"

	"gioui.org/io/event"
)

type tag uint64

var _tag tag

func Tag() event.Tag {
	_tag++
	t := _tag
	return &t
}

func (t tag) String() string {
	return strconv.FormatUint(uint64(t), 10)
}
