package main

import (
	"image"
	"image/color"
	"os"

	"golang.org/x/exp/shiny/materialdesign/colornames"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/backdrop"
	"github.com/vibrantgio/mvu"
	"github.com/vibrantgio/spectrum/tokens"
)

func main() {
	go Edit()
	app.Main()
}

func Edit() {
	window := mvu.NewWindow(app.Title("MVU - Edit"))

	backdrops := rx.Of(backdrop.Widget(colornames.Grey800))

	edit1 := Input(0.5, 0.0, "What's Up?", colornames.Blue50, colornames.Blue700, colornames.Blue900)
	edit2 := Input(0.5, 1.0, "What's Up?", colornames.Yellow50, colornames.Yellow700, colornames.Yellow900)

	window.Render(backdrops, edit1, edit2).Wait()
	os.Exit(0)
}

func Input(ax, ay float32, initial string, textColor, selectColor, backColor color.Color) rx.Observable[layout.Widget] {
	// The theme's Typography supplies both the shaper (Roboto, system fonts
	// excluded) and the editor's type role; BodyLarge matches the 16 sp body
	// text this example always used.
	typ := tokens.DefaultTypography
	shaper := typ.Shaper()
	editFont := font.Font{Typeface: font.Typeface(typ.BodyLarge.Typeface)}
	if typ.BodyLarge.Weight != 0 {
		editFont.Weight = tokens.FontWeight(typ.BodyLarge.Weight)
	}
	return rx.Defer(func() rx.Observable[layout.Widget] {
		edit := widget.Editor{}
		edit.SetText(initial)
		return rx.Of[layout.Widget](func(gtx layout.Context) layout.Dimensions {
			size := gtx.Constraints.Max

			if ay == 0.0 {
				defer op.Offset(image.Pt(0, 0)).Push(gtx.Ops).Pop()
				defer clip.Rect{Max: image.Pt(size.X, size.Y/2)}.Op().Push(gtx.Ops).Pop()
			} else {
				defer op.Offset(image.Pt(0, size.Y/2)).Push(gtx.Ops).Pop()
				defer clip.Rect{Max: image.Pt(size.X, size.Y/2)}.Op().Push(gtx.Ops).Pop()
			}

			backdrop.Fill(gtx.Ops, backColor)

			m := op.Record(gtx.Ops)
			c := color.NRGBAModel.Convert(textColor).(color.NRGBA)
			paint.ColorOp{Color: c}.Add(gtx.Ops)
			textMaterial := m.Stop()

			m = op.Record(gtx.Ops)
			c = color.NRGBAModel.Convert(selectColor).(color.NRGBA)
			paint.ColorOp{Color: c}.Add(gtx.Ops)
			selectMaterial := m.Stop()

			edit.Layout(gtx, shaper, editFont, unit.Sp(typ.BodyLarge.Size), textMaterial, selectMaterial)

			return layout.Dimensions{Size: size}
		})
	})
}
