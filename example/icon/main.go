package main

import (
	"os"

	"gioui.org/app"
	"gioui.org/f32"
	"golang.org/x/exp/shiny/materialdesign/colornames"
	"golang.org/x/exp/shiny/materialdesign/icons"

	icon "github.com/reactivego/ivg/raster/gio"
	vibrant "github.com/reactivego/vibrant/gio"

	_ "github.com/reactivego/rx"
	_ "github.com/reactivego/vibrant/gio/generic"
)

func main() {
	go Icon()
	app.Main()
}

//jig:type Bytes = []byte

func Icon() {
	window := vibrant.NewWindow(app.Title("Vibrant - Icon"))
	frames := ExtendGioObservableFrameEvent(window.FrameEvents())
	AliasGioObservableCallOp()

	grey600 := vibrant.BlankScreen(colornames.Grey600)

	icos := FromBytes(icons.ActionEuroSymbol, icons.AVArtTrack).SwitchMapCallOp(func(b []byte) vibrant.ObservableCallOp {
		ico, _ := icon.NewIcon(b)
		return frames.MapCallOp(func(fe FrameEvent) CallOp {
			rect := ico.AspectMeet(f32.Rect(0, 0, float32(fe.Size.X), float32(fe.Size.Y)), 0.5, 0.5)
			callop, _ := icon.Rasterize(ico, rect, icon.WithColors(colornames.Orange400))
			return callop
		})
	})

	window.Frame(grey600, vibrant.FromCallOp(grey600), icos).Wait()
	os.Exit(0)
}
