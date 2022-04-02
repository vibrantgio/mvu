package roboto

import (
	"fmt"
	"sync"

	"eliasnaur.com/font/roboto/robotoblack"
	"eliasnaur.com/font/roboto/robotobold"
	"eliasnaur.com/font/roboto/robotolight"
	"eliasnaur.com/font/roboto/robotomedium"
	"eliasnaur.com/font/roboto/robotoregular"
	"eliasnaur.com/font/roboto/robotothin"

	"eliasnaur.com/font/roboto/robotoblackitalic"
	"eliasnaur.com/font/roboto/robotobolditalic"
	"eliasnaur.com/font/roboto/robotoitalic"
	"eliasnaur.com/font/roboto/robotolightitalic"
	"eliasnaur.com/font/roboto/robotomediumitalic"
	"eliasnaur.com/font/roboto/robotothinitalic"

	// "eliasnaur.com/font/roboto/robotocondensedbold"
	// "eliasnaur.com/font/roboto/robotocondensedlight"
	// "eliasnaur.com/font/roboto/robotocondensedmedium"
	// "eliasnaur.com/font/roboto/robotocondensedregular"

	// "eliasnaur.com/font/roboto/robotocondensedbolditalic"
	// "eliasnaur.com/font/roboto/robotocondenseditalic"
	// "eliasnaur.com/font/roboto/robotocondensedlightitalic"
	// "eliasnaur.com/font/roboto/robotocondensedmediumitalic"

	"gioui.org/font/opentype"
	"gioui.org/text"
)

var (
	once   sync.Once
	roboto []text.FontFace
)

var (
	Typeface text.Typeface = "Roboto"

	RegularThin   = text.Font{Typeface: Typeface, Variant: "", Style: text.Regular, Weight: text.Thin}
	RegularLight  = text.Font{Typeface: Typeface, Variant: "", Style: text.Regular, Weight: text.Light}
	RegularNormal = text.Font{Typeface: Typeface, Variant: "", Style: text.Regular, Weight: text.Normal}
	RegularMedium = text.Font{Typeface: Typeface, Variant: "", Style: text.Regular, Weight: text.Medium}
	RegularBold   = text.Font{Typeface: Typeface, Variant: "", Style: text.Regular, Weight: text.Bold}
	RegularBlack  = text.Font{Typeface: Typeface, Variant: "", Style: text.Regular, Weight: text.Black}

	ItalicThin   = text.Font{Typeface: Typeface, Variant: "", Style: text.Italic, Weight: text.Thin}
	ItalicLight  = text.Font{Typeface: Typeface, Variant: "", Style: text.Italic, Weight: text.Light}
	ItalicNormal = text.Font{Typeface: Typeface, Variant: "", Style: text.Italic, Weight: text.Normal}
	ItalicMedium = text.Font{Typeface: Typeface, Variant: "", Style: text.Italic, Weight: text.Medium}
	ItalicBold   = text.Font{Typeface: Typeface, Variant: "", Style: text.Italic, Weight: text.Bold}
	ItalicBlack  = text.Font{Typeface: Typeface, Variant: "", Style: text.Italic, Weight: text.Black}

	// ItalicCondensedLight  = text.Font{Typeface: Typeface, Variant: "Condensed", Style: text.Italic, Weight: text.Light}
	// ItalicCondensedNormal = text.Font{Typeface: Typeface, Variant: "Condensed", Style: text.Italic, Weight: text.Normal}
	// ItalicCondensedMedium = text.Font{Typeface: Typeface, Variant: "Condensed", Style: text.Italic, Weight: text.Medium}
	// ItalicCondensedBold   = text.Font{Typeface: Typeface, Variant: "Condensed", Style: text.Italic, Weight: text.Bold}

	// RegularCondensedLight  = text.Font{Typeface: Typeface, Variant: "Condensed", Style: text.Italic, Weight: text.Light}
	// RegularCondensedNormal = text.Font{Typeface: Typeface, Variant: "Condensed", Style: text.Italic, Weight: text.Normal}
	// RegularCondensedMedium = text.Font{Typeface: Typeface, Variant: "Condensed", Style: text.Italic, Weight: text.Medium}
	// RegularCondensedBold   = text.Font{Typeface: Typeface, Variant: "Condensed", Style: text.Italic, Weight: text.Bold}
)

func Collection() []text.FontFace {
	register := func(fnt text.Font, ttf []byte) {
		face, err := opentype.Parse(ttf)
		if err != nil {
			panic(fmt.Sprintf("failed to parse font: %v", err))
		}
		roboto = append(roboto, text.FontFace{Font: fnt, Face: face})
	}
	once.Do(func() {
		register(RegularNormal, robotoregular.TTF)
		register(RegularThin, robotothin.TTF)
		register(RegularLight, robotolight.TTF)
		register(RegularMedium, robotomedium.TTF)
		register(RegularBold, robotobold.TTF)
		register(RegularBlack, robotoblack.TTF)

		register(ItalicNormal, robotoitalic.TTF)
		register(ItalicThin, robotothinitalic.TTF)
		register(ItalicLight, robotolightitalic.TTF)
		register(ItalicMedium, robotomediumitalic.TTF)
		register(ItalicBold, robotobolditalic.TTF)
		register(ItalicBlack, robotoblackitalic.TTF)

		// register(RegularCondensedNormal, robotocondensedregular.TTF)
		// register(RegularCondensedLight, robotocondensedlight.TTF)
		// register(RegularCondensedMedium, robotocondensedmedium.TTF)
		// register(RegularCondensedBold, robotocondensedbold.TTF)

		// register(ItalicCondensedNormal, robotocondenseditalic.TTF)
		// register(ItalicCondensedLight, robotocondensedlightitalic.TTF)
		// register(ItalicCondensedMedium, robotocondensedmediumitalic.TTF)
		// register(ItalicCondensedBold, robotocondensedbolditalic.TTF)
	})
	n := len(roboto)
	return roboto[0:n:n]
}

func Shaper() text.Shaper {
	return text.NewCache(Collection())
}
