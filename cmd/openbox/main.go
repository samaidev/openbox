// Command openbox launches the OpenBox GUI.
//
// Usage:
//
//	openbox                  # launch GUI with no pre-filled inputs
//	openbox <archive>        # launch GUI and pre-fill Extract tab with the archive
//	openbox -c <file...>     # launch GUI and pre-fill Compress tab with the given files
//	openbox -x <archive>     # alias for plain `<archive>`
//	openbox -version         # print version and exit
//
// On Windows, file associations and shell context-menu entries invoke
// these forms (see build/windows/openbox.iss).
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2/app"

	"github.com/samaidev/openbox/internal/assets"
	"github.com/samaidev/openbox/internal/ui"
)

// Version is set at build time via -ldflags "-X main.Version=...".
var Version = "0.1.0"

func main() {
	var (
		showVersion bool
		compress    bool
		extract     bool
	)
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.BoolVar(&compress, "c", false, "pre-fill Compress tab with the given files")
	flag.BoolVar(&extract, "x", false, "pre-fill Extract tab with the given archive")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "OpenBox — open-source cross-platform archiver")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  openbox                       launch GUI")
		fmt.Fprintln(os.Stderr, "  openbox <archive>             open archive in Extract tab")
		fmt.Fprintln(os.Stderr, "  openbox -c <file|dir>...      pre-fill Compress tab")
		fmt.Fprintln(os.Stderr, "  openbox -x <archive>          same as plain <archive>")
		fmt.Fprintln(os.Stderr, "  openbox -version              print version")
	}
	flag.Parse()

	if showVersion {
		fmt.Println("OpenBox", Version)
		return
	}

	args := flag.Args()

	// Decide the initial tab + pre-filled inputs.
	var initial *ui.InitialState
	if compress && len(args) > 0 {
		initial = &ui.InitialState{
			Tab:     ui.TabCompress,
			Sources: cleanPaths(args),
		}
	} else if len(args) > 0 {
		// First positional arg is treated as an archive to extract.
		initial = &ui.InitialState{
			Tab:     ui.TabExtract,
			Archive: cleanPath(args[0]),
		}
	}
	_ = extract // accepted for symmetry; plain arg path already covers it

	a := app.NewWithID("io.github.samaidev.openbox")
	a.SetIcon(assets.AppIcon())

	w := ui.NewWithState(a, initial)
	w.ShowAndRun()
}

// cleanPath expands ~ and returns an absolute path when possible.
func cleanPath(p string) string {
	if p == "" {
		return ""
	}
	if strings.HasPrefix(p, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			p = filepath.Join(home, p[1:])
		}
	}
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return p
}

func cleanPaths(ps []string) []string {
	out := make([]string, 0, len(ps))
	for _, p := range ps {
		if c := cleanPath(p); c != "" {
			out = append(out, c)
		}
	}
	return out
}
