package main

import (
	"os"

	"gioui.org/app"

	"github.com/vibrantgio/mvu"
)

func main() {
	go Minimal()
	app.Main()
}

func Minimal() {
	window := mvu.NewWindow(app.Title("MVU - Minimal"))
	window.Render().Wait()
	os.Exit(0)
}
