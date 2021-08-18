package gio

import "gioui.org/io/event"

var tags int

func tag() event.Tag {
	tags++
	t := tags
	return &t
}
