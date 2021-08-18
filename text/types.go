package text

import (
	"gioui.org/text"
	"gioui.org/unit"
)

const (
	Thin   text.Weight = 100 - 400
	Light  text.Weight = 200 - 400
	Normal             = text.Normal // 400 - 400
	Medium             = text.Medium // 500 - 400
	Bold               = text.Bold   // 600 - 400
	Black  text.Weight = 800 - 400
)

const (
	Regular = text.Regular
	Italic  = text.Italic
)

type Style struct {
	Font text.Font
	Size int
}

func (s Style) Scale(metric unit.Metric) Style {
	return Style{Font: s.Font, Size: metric.Px(unit.Sp(float32(s.Size)))}
}

type Font = text.Font
type FontFace = text.FontFace
type Cache = text.Cache
type Shaper = text.Shaper

func NewCache(collection []text.FontFace) *Cache {
	return text.NewCache(collection)
}
