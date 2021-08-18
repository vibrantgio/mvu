package italic

import (
	"fmt"
	"sync"

	"eliasnaur.com/font/roboto/robotoblackitalic"
	"eliasnaur.com/font/roboto/robotobolditalic"
	"eliasnaur.com/font/roboto/robotoitalic"
	"eliasnaur.com/font/roboto/robotolightitalic"
	"eliasnaur.com/font/roboto/robotomediumitalic"
	"eliasnaur.com/font/roboto/robotothinitalic"
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
		fnt.Typeface = Roboto
		roboto = append(roboto, text.FontFace{Font: fnt, Face: face})
	}
	once.Do(func() {
		register(text.Font{Weight: text.Normal, Style: text.Italic}, robotoitalic.TTF)
		register(text.Font{Weight: text.Thin, Style: text.Italic}, robotothinitalic.TTF)
		register(text.Font{Weight: text.Light, Style: text.Italic}, robotolightitalic.TTF)
		register(text.Font{Weight: text.Medium, Style: text.Italic}, robotomediumitalic.TTF)
		register(text.Font{Weight: text.Bold, Style: text.Italic}, robotobolditalic.TTF)
		register(text.Font{Weight: text.Black, Style: text.Italic}, robotoblackitalic.TTF)
	})
	return roboto
}

func Shaper() text.Shaper {
	return text.NewCache(FontFaces())
}
