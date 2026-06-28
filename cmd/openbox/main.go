// Command openbox is the OpenBox archiver.
//
// GUI mode (default):
//
//	openbox                  launch GUI with no pre-filled inputs
//	openbox <archive>        launch GUI with Extract tab pre-filled
//	openbox -c <file|dir>... launch GUI with Compress tab pre-filled
//	openbox -x <archive>     alias for plain <archive>
//
// CLI mode (-cli, no GUI; for scripting + automated testing):
//
//	openbox -cli c <src>... <dest>     compress sources into dest
//	openbox -cli x <archive> <dest>    extract archive into dest
//
// Other:
//
//	openbox -version         print version and exit
//	openbox -h               show help
//
// On Windows, file associations invoke `openbox <archive>` (double-click)
// and `openbox -c <file>` / `openbox -x <archive>` (right-click menu).
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2/app"

	"github.com/samaidev/openbox/internal/archiver"
	"github.com/samaidev/openbox/internal/assets"
	"github.com/samaidev/openbox/internal/i18n"
	"github.com/samaidev/openbox/internal/ui"
)

// Version is set at build time via -ldflags "-X main.Version=...".
var Version = "0.2.1"

func main() {
	var (
		showVersion bool
		cliMode     bool
		compress    bool
		extract     bool
		level       int
		password    string
		here        bool
	)
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.BoolVar(&cliMode, "cli", false, "non-interactive CLI mode (no GUI)")
	flag.BoolVar(&compress, "c", false, "GUI: pre-fill Compress tab with the given files")
	flag.BoolVar(&extract, "x", false, "GUI: pre-fill Extract tab with the given archive")
	flag.IntVar(&level, "level", 6, "compression level (0=store, 1=fastest, 3=fast, 6=normal, 9=max)")
	flag.StringVar(&password, "p", "", "password (for 7z/rar/zip with encryption)")
	flag.BoolVar(&here, "here", false, "with -cli x: extract into a subfolder of the archive's parent dir, named after the archive (matches 7-Zip / WinRAR 'Extract to <name>\\')")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "OpenBox — open-source cross-platform archiver")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "GUI mode (default):")
		fmt.Fprintln(os.Stderr, "  openbox                       launch GUI")
		fmt.Fprintln(os.Stderr, "  openbox <archive>             open archive in Extract tab")
		fmt.Fprintln(os.Stderr, "  openbox -c <file|dir>...      pre-fill Compress tab")
		fmt.Fprintln(os.Stderr, "  openbox -x <archive>          same as plain <archive>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "CLI mode (no GUI, for scripting/testing):")
		fmt.Fprintln(os.Stderr, "  openbox -cli c <src>... <dest>            compress")
		fmt.Fprintln(os.Stderr, "  openbox -cli x <archive> <dest>           extract")
		fmt.Fprintln(os.Stderr, "  openbox -cli -here x <archive>            extract to <parent>/<name>/")
		fmt.Fprintln(os.Stderr, "  openbox -cli l <archive>                  list contents")
		fmt.Fprintln(os.Stderr, "  (combine with -level N -p PASSWORD)")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Other:")
		fmt.Fprintln(os.Stderr, "  openbox -version              print version")
	}
	flag.Parse()

	if showVersion {
		fmt.Println("OpenBox", Version)
		return
	}

	args := flag.Args()

	if cliMode {
		os.Exit(runCLI(args, level, password, here))
	}

	// GUI mode: decide the initial tab + pre-filled inputs.
	var initial *ui.InitialState
	if compress && len(args) > 0 {
		initial = &ui.InitialState{
			Tab:     ui.TabCompress,
			Sources: cleanPaths(args),
		}
	} else if len(args) > 0 {
		initial = &ui.InitialState{
			Tab:     ui.TabExtract,
			Archive: cleanPath(args[0]),
		}
	}
	_ = extract

	a := app.NewWithID("io.github.samaidev.openbox")
	a.SetIcon(assets.AppIcon())

	w := ui.NewWithState(a, initial)
	w.ShowAndRun()
}

// runCLI executes the non-interactive compress/extract command.
// Returns the process exit code.
func runCLI(args []string, level int, password string, here bool) int {
	if len(args) < 1 {
		flag.Usage()
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "error: -cli requires a subcommand (c, x, or l)")
		return 2
	}

	cmd := args[0]
	rest := args[1:]

	prog := &archiver.Progress{
		OnAdvance: func(done, total int, current string) {
			if total > 0 {
				fmt.Fprintf(os.Stderr, "\r[%d/%d] %s", done, total, current)
			} else {
				fmt.Fprintf(os.Stderr, "\r[?] %s", current)
			}
		},
		OnDone: func() {
			fmt.Fprintln(os.Stderr, "\rdone.                    ")
		},
	}

	switch cmd {
	case "c", "compress":
		if len(rest) < 2 {
			fmt.Fprintln(os.Stderr, "error: compress requires <src>... <dest>")
			return 2
		}
		dest := cleanPath(rest[len(rest)-1])
		sources := cleanPaths(rest[:len(rest)-1])
		opts := archiver.Options{
			Format:   archiver.FormatAuto, // detect from dest extension
			Level:    archiver.Level(level),
			Password: password,
		}
		if err := archiver.Compress(sources, dest, opts, prog); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 1
		}
		fmt.Printf("OK: created %s\n", dest)
		return 0

	case "x", "extract":
		var src, dest string
		if here {
			// -here x <archive>
			if len(rest) != 1 {
				fmt.Fprintln(os.Stderr, "error: -here x requires exactly <archive>")
				return 2
			}
			src = cleanPath(rest[0])
			dir := filepath.Dir(src)
			base := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
			if strings.HasSuffix(base, ".tar") {
				base = strings.TrimSuffix(base, ".tar")
			}
			dest = filepath.Join(dir, base)
		} else {
			// x <archive> <dest>
			if len(rest) != 2 {
				fmt.Fprintln(os.Stderr, "error: extract requires <archive> <dest> (or use -here)")
				return 2
			}
			src = cleanPath(rest[0])
			dest = cleanPath(rest[1])
		}
		opts := archiver.Options{Password: password}
		if err := archiver.Extract(src, dest, opts, prog); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 1
		}
		fmt.Printf("OK: extracted to %s\n", dest)
		return 0

	case "l", "list":
		if len(rest) != 1 {
			fmt.Fprintln(os.Stderr, "error: list requires <archive>")
			return 2
		}
		src := cleanPath(rest[0])
		opts := archiver.Options{Password: password}
		entries, err := archiver.List(src, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 1
		}
		fmt.Printf("%-50s  %12s  %s\n", "NAME", "SIZE", "MODIFIED")
		for _, e := range entries {
			size := "-"
			if !e.IsDir {
				size = fmt.Sprintf("%d", e.Size)
			}
			mod := ""
			if !e.ModTime.IsZero() {
				mod = e.ModTime.Format("2006-01-02 15:04")
			}
			fmt.Printf("%-50s  %12s  %s\n", e.Name, size, mod)
		}
		fmt.Printf("\n%d entries\n", len(entries))
		return 0

	default:
		fmt.Fprintf(os.Stderr, "error: unknown subcommand %q (use 'c', 'x', or 'l')\n", cmd)
		return 2
	}
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

// keep i18n import alive even if not directly used in main (it's used
// implicitly by ui package which reads i18n state at startup).
var _ = i18n.English
