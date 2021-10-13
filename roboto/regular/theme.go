package regular

import "github.com/reactivego/vibrant/text"

const (
	Roboto = "Roboto"
)

var (
	RobotoThin   = text.Font{Typeface: Roboto, Variant: "", Style: text.Regular, Weight: text.Thin}
	RobotoLight  = text.Font{Typeface: Roboto, Variant: "", Style: text.Regular, Weight: text.Light}
	RobotoNormal = text.Font{Typeface: Roboto, Variant: "", Style: text.Regular, Weight: text.Normal}
	RobotoMedium = text.Font{Typeface: Roboto, Variant: "", Style: text.Regular, Weight: text.Medium}
	RobotoBold   = text.Font{Typeface: Roboto, Variant: "", Style: text.Regular, Weight: text.Bold}
	RobotoBlack  = text.Font{Typeface: Roboto, Variant: "", Style: text.Regular, Weight: text.Black}
)

var (
	H1        = text.Style{Font: RobotoThin, Size: 96}   // w300
	H2        = text.Style{Font: RobotoLight, Size: 60}  // w300
	H3        = text.Style{Font: RobotoNormal, Size: 48} // w400
	H4        = text.Style{Font: RobotoNormal, Size: 34} // w400
	H5        = text.Style{Font: RobotoNormal, Size: 24} // w400
	H6        = text.Style{Font: RobotoMedium, Size: 20} // w500
	Subtitle1 = text.Style{Font: RobotoNormal, Size: 16} // w400
	Subtitle2 = text.Style{Font: RobotoMedium, Size: 14} // w500
	BodyText1 = text.Style{Font: RobotoNormal, Size: 16} // w400
	BodyText2 = text.Style{Font: RobotoNormal, Size: 14} // w400
	Button    = text.Style{Font: RobotoMedium, Size: 14} // w500
	Caption   = text.Style{Font: RobotoNormal, Size: 12} // w400
	Overline  = text.Style{Font: RobotoNormal, Size: 10} // w400
)
