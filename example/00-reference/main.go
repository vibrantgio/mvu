// 00-reference is a minimal Gio v0.9 app that uses the canonical event loop
// directly (no mvu, no rx). It serves as a known-good baseline for diagnosing
// freezes in the mvu loop. If this app renders and 01-minimal does not, the
// fault is in the mvu/rx machinery, not in Gio or the OS.
package main

import (
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/op"
)

func main() {
	go run()
	app.Main()
}

func run() {
	w := new(app.Window)
	w.Option(app.Title("00 - Reference"))
	var ops op.Ops
	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			if e.Err != nil {
				log.Fatal(e.Err)
			}
			os.Exit(0)
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			e.Frame(gtx.Ops)
		}
	}
}
