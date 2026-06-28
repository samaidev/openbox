// Package assets embeds static resources (icon, etc.) into the binary.
package assets

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed icon.png
var iconPNG []byte

// AppIcon returns the embedded application icon as a Fyne resource.
func AppIcon() fyne.Resource {
	return fyne.NewStaticResource("icon.png", iconPNG)
}
