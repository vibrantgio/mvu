package theme

import (
	"gioui.org/text"
	"gioui.org/unit"
	"github.com/reactivego/vibrant/font/roboto"
)

type TextStyle struct {
	Font text.Font
	Size unit.Value
}

var (
	H1          = TextStyle{Font: roboto.RegularThin, Size: unit.Sp(96)}   // w300
	H2          = TextStyle{Font: roboto.RegularLight, Size: unit.Sp(60)}  // w300
	H3          = TextStyle{Font: roboto.RegularNormal, Size: unit.Sp(48)} // w400
	H4          = TextStyle{Font: roboto.RegularNormal, Size: unit.Sp(34)} // w400
	H5          = TextStyle{Font: roboto.RegularNormal, Size: unit.Sp(24)} // w400
	H6          = TextStyle{Font: roboto.RegularMedium, Size: unit.Sp(20)} // w500
	Subtitle1   = TextStyle{Font: roboto.RegularNormal, Size: unit.Sp(16)} // w400
	Subtitle2   = TextStyle{Font: roboto.RegularMedium, Size: unit.Sp(14)} // w500
	BodyText1   = TextStyle{Font: roboto.RegularNormal, Size: unit.Sp(16)} // w400
	BodyText2   = TextStyle{Font: roboto.RegularNormal, Size: unit.Sp(14)} // w400
	Button      = TextStyle{Font: roboto.RegularMedium, Size: unit.Sp(14)} // w500
	Caption     = TextStyle{Font: roboto.RegularNormal, Size: unit.Sp(12)} // w400
	SmallButton = TextStyle{Font: roboto.RegularBold, Size: unit.Sp(12)}   // w500
	Overline    = TextStyle{Font: roboto.RegularNormal, Size: unit.Sp(10)} // w400
)
