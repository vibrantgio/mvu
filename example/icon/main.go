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

	"github.com/reactivego/ivg/raster/gio"
	"github.com/reactivego/vibrant"
	"github.com/reactivego/x"
)

func main() {
	go Icon()
	app.Main()
}

func Icon() {
	window := vibrant.NewWindow(app.Title("Vibrant - Icon"))
	frame := window.Frame()

	icons := x.Map(x.Timer[int](0, time.Second), func(i int) []byte {
		return [...][]byte{icons.ActionEuroSymbol, icons.AVArtTrack, icons.ActionAlarm}[i%3]
	})

	drawing := x.SwitchMap(icons, func(data []byte) x.Observable[op.CallOp] {
		icon, _ := gio.NewIcon(data)
		bg := color.NRGBAModel.Convert(colornames.Grey900).(color.NRGBA)
		fg := colornames.Orange400
		return x.Map(frame, func(frame vibrant.Frame) op.CallOp {
			rect := frame.SafeRect().Inset(frame.Dp(unit.Dp(12)))
			ops := new(op.Ops)
			macro := op.Record(ops)
			paint.FillShape(ops, bg, clip.Rect(rect).Op())
			rect = icon.AspectMeet(rect.Size(), 0.5, 0.5).Add(rect.Min)
			gio.Draw(ops, icon, rect, fg)
			return macro.Stop()
		})
	})

	backdrop := vibrant.Backdrop(colornames.Grey600)

	window.Render(backdrop, x.Of(backdrop), drawing).Wait()
	os.Exit(0)
}
