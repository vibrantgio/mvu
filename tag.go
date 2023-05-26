package vibrant

import (
	"strconv"

	"gioui.org/io/event"
)

var _uid uint64

func Tag() event.Tag {
	_uid++
	tag := tag(_uid)
	return &tag
}

type tag uint64

func (t tag) String() string {
	return strconv.FormatUint(uint64(t), 10)
}
