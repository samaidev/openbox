package archiver

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// SevenZipWriterAvailable reports whether the system has a 7-Zip CLI binary
// that OpenBox can shell out to for creating .7z archives. The official
// 7-Zip (Windows), p7zip (Linux), and 7-zip via Homebrew (macOS) are all
// supported.
//
// Why subprocess instead of pure Go? The open-source Go ecosystem does not
// have a stable 7z writer. The official 7-Zip codebase is C++ and complex;
// nobody has ported the writer side yet (the reader side — bodgit/sevenzip —
// works fine for extraction). Shelling out to the well-known CLI is the
// pragmatic choice and keeps OpenBox 100% free / open-source.
func SevenZipWriterAvailable() bool {
	return sevenzipBinaryPath() != ""
}

// sevenzipBinaryPath returns the absolute path to the first available 7z
// CLI binary, or "" if none is found. Probes well-known names:
//
//	7z    — Linux p7zip-full, Windows 7-Zip
//	7za   — Linux p7zip standalone
//	7zz   — macOS Homebrew 7zip formula
//
// On Windows it also falls back to the standard install paths under
// Program Files, since 7-Zip doesn't always add itself to PATH.
func sevenzipBinaryPath() string {
	candidates := []string{"7z", "7za", "7zz"}
	if runtime.GOOS == "windows" {
		candidates = append(candidates,
			`C:\Program Files\7-Zip\7z.exe`,
			`C:\Program Files (x86)\7-Zip\7z.exe`,
		)
	}
	for _, c := range candidates {
		if path, err := exec.LookPath(c); err == nil {
			return path
		}
	}
	return ""
}

// write7zViaBinary creates a .7z archive by shelling out to the system
// 7-Zip CLI. We invoke it once per source so the in-archive paths match
// what the pure-Go writers produce (each source becomes a top-level entry,
// relative to its own parent directory).
func write7zViaBinary(sources []string, dest string, opts Options, p *Progress) error {
	binary := sevenzipBinaryPath()
	if binary == "" {
		return fmt.Errorf("creating .7z archives requires the 7-Zip CLI to be installed.\n" +
			"Install it from one of:\n" +
			"  • Windows:  https://www.7-zip.org/download.html\n" +
			"  • macOS:    brew install 7zip\n" +
			"  • Linux:    sudo apt install p7zip-full   (Debian/Ubuntu)\n" +
			"              sudo dnf install p7zip        (Fedora)\n" +
			"              sudo pacman -S p7zip          (Arch)")
	}

	// 7z 'a' (add) mode appends to an existing archive. We always want a
	// fresh archive, so remove any pre-existing destination first.
	_ = os.Remove(dest)

	// Map our internal Level enum to 7z's -mx flag (0=store, 9=max).
	mxArg := fmt.Sprintf("-mx=%d", int(opts.Level))

	total := len(sources)
	for i, src := range sources {
		reportProgress(p, i, total, src)

		abs, err := filepath.Abs(src)
		if err != nil {
			return fmt.Errorf("resolve %s: %w", src, err)
		}
		parent := filepath.Dir(abs)
		base := filepath.Base(abs)

		args := []string{"a", "-t7z", "-y", dest, mxArg}
		if opts.Password != "" {
			// -mhe=on encrypts archive headers (file names + sizes) too,
			// so the archive is fully opaque without the password.
			args = append(args, "-p"+opts.Password, "-mhe=on")
		}
		args = append(args, base)

		cmd := exec.Command(binary, args...)
		cmd.Dir = parent
		// Capture stderr for diagnostics; 7z writes its main progress output
		// to stdout which we deliberately ignore — per-source progress is
		// good enough for v1 and avoids fragile stdout parsing.
		var stderr strings.Builder
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("7z failed on %s: %w\n%s", src, err, stderr.String())
		}
	}
	reportProgress(p, total, total, dest)
	return nil
}
