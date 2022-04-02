package main

import (
	"image/color"
	"os"
	"time"

	"golang.org/x/exp/shiny/materialdesign/colornames"
	"golang.org/x/exp/shiny/materialdesign/icons"

	"gioui.org/app"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	ivg "github.com/reactivego/ivg/raster/gio"
	rx "github.com/reactivego/observable"
	"github.com/reactivego/vibrant"
)

func main() {
	go Icon()
	app.Main()
}

func Icon() {
	window := vibrant.NewWindow(app.Title("Vibrant - Icon"))
	frame := window.Frame()

	icons := rx.Map(rx.Timer[int](0, time.Second), func(i int) []byte {
		return [...][]byte{icons.ActionEuroSymbol, icons.AVArtTrack, icons.ActionAlarm}[i%3]
	})

	drawing := rx.SwitchMap(icons, func(data []byte) rx.Observable[op.CallOp] {
		icon, _ := ivg.NewIcon(data)
		bg := color.NRGBAModel.Convert(colornames.Grey900).(color.NRGBA)
		fg := colornames.Orange400
		return rx.Map(frame, func(frame vibrant.Frame) op.CallOp {
			rect := frame.SafeRect().Inset(frame.Px(unit.Dp(12)))
			ops := new(op.Ops)
			macro := op.Record(ops)
			paint.FillShape(ops, bg, clip.Rect(rect).Op())
			rect = icon.AspectMeet(rect.Size(), 0.5, 0.5).Add(rect.Min)
			ivg.Draw(ops, icon, rect, fg)
			return macro.Stop()
		})
	})

	backdrop := vibrant.Backdrop(colornames.Grey600)

	window.Render(backdrop, rx.Of(backdrop), drawing).Wait()
	os.Exit(0)
}
