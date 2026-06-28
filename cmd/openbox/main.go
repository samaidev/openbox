// Command openbox launches the OpenBox GUI.
package main

import (
	"fyne.io/fyne/v2/app"

	"github.com/samaidev/openbox/internal/ui"
)

// Version is set at build time via -ldflags "-X main.Version=...".
var Version = "0.1.0"

func main() {
	a := app.NewWithID("io.github.samaidev.openbox")
	a.SetIcon(nil) // TODO: ship an icon in assets/ and load it here.

	w := ui.New(a)
	w.ShowAndRun()
}
