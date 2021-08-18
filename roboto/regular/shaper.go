package regular

import (
	"fmt"
	"sync"

	"eliasnaur.com/font/roboto/robotoblack"
	"eliasnaur.com/font/roboto/robotobold"
	"eliasnaur.com/font/roboto/robotolight"
	"eliasnaur.com/font/roboto/robotomedium"
	"eliasnaur.com/font/roboto/robotoregular"
	"eliasnaur.com/font/roboto/robotothin"
	"gioui.org/font/opentype"

	"github.com/reactivego/vibrant/text"
)

var (
	once   sync.Once
	roboto []text.FontFace
)

func FontFaces() []text.FontFace {
	register := func(fnt text.Font, ttf []byte) {
		face, err := opentype.Parse(ttf)
		if err != nil {
			panic(fmt.Sprintf("failed to parse font: %v", err))
		}
		fnt.Typeface = "Roboto"
		roboto = append(roboto, text.FontFace{Font: fnt, Face: face})
	}
	once.Do(func() {
		register(text.Font{Weight: text.Normal, Style: text.Regular}, robotoregular.TTF)
		register(text.Font{Weight: text.Thin, Style: text.Regular}, robotothin.TTF)
		register(text.Font{Weight: text.Light, Style: text.Regular}, robotolight.TTF)
		register(text.Font{Weight: text.Medium, Style: text.Regular}, robotomedium.TTF)
		register(text.Font{Weight: text.Bold, Style: text.Regular}, robotobold.TTF)
		register(text.Font{Weight: text.Black, Style: text.Regular}, robotoblack.TTF)
	})
	return roboto
}

func Shaper() text.Shaper {
	return text.NewCache(FontFaces())
}
