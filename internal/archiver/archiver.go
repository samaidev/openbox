// Package archiver provides a unified Compress/Extract API for the formats
// supported by OpenBox: zip, tar, tar.gz, 7z (extract-only), rar (extract-only),
// iso (extract-only).
package archiver

import (
	"archive/tar"
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bodgit/sevenzip"
	"github.com/kdomanski/iso9660"
	"github.com/nwaples/rardecode"
)

// Format enumerates supported archive formats.
type Format int

const (
	FormatAuto Format = iota
	FormatZip
	FormatTar
	FormatTarGz
	Format7z
	FormatRar // extract-only (proprietary format, no open-source writer)
	FormatIso // extract-only in v1
)

// String returns the canonical short name.
func (f Format) String() string {
	switch f {
	case FormatZip:
		return "zip"
	case FormatTar:
		return "tar"
	case FormatTarGz:
		return "tar.gz"
	case Format7z:
		return "7z"
	case FormatRar:
		return "rar"
	case FormatIso:
		return "iso"
	}
	return "auto"
}

// Detect guesses the format from a filename extension.
func Detect(name string) Format {
	low := strings.ToLower(name)
	switch {
	case strings.HasSuffix(low, ".zip"):
		return FormatZip
	case strings.HasSuffix(low, ".tar.gz") || strings.HasSuffix(low, ".tgz"):
		return FormatTarGz
	case strings.HasSuffix(low, ".tar"):
		return FormatTar
	case strings.HasSuffix(low, ".7z"):
		return Format7z
	case strings.HasSuffix(low, ".rar"):
		return FormatRar
	case strings.HasSuffix(low, ".iso"):
		return FormatIso
	}
	return FormatAuto
}

// CanCompress reports whether OpenBox can create archives in this format.
//   - zip / tar / tar.gz: pure-Go writers, always available
//   - 7z: requires the 7-Zip CLI to be installed (see SevenZipWriterAvailable)
//   - rar: proprietary format, no open-source writer exists
//   - iso: out of scope for v1
func CanCompress(f Format) bool {
	switch f {
	case FormatZip, FormatTar, FormatTarGz, Format7z:
		return true
	}
	return false
}

// Level controls the compression strength (0=store, 9=max).
type Level int

const (
	LevelStore   Level = 0
	LevelFastest Level = 1
	LevelFast    Level = 3
	LevelNormal  Level = 6
	LevelMaximum Level = 9
)

// Options controls a Compress or Extract operation.
type Options struct {
	Format   Format
	Level    Level
	Password string // currently used for 7z + rar extraction
	RootDir  string // when set, paths inside the archive are made relative to this dir
}

// Progress is called periodically with counts. done/total refer to file count.
// When total is unknown (streaming tar/rar), it is passed as -1.
type Progress struct {
	OnAdvance func(done, total int, current string)
	OnDone    func()
}

// Compress creates a new archive at dest containing every file/dir under sources.
func Compress(sources []string, dest string, opts Options, p *Progress) error {
	if opts.Format == FormatAuto {
		opts.Format = Detect(dest)
		if opts.Format == FormatAuto {
			return fmt.Errorf("cannot detect format from %s", dest)
		}
	}
	if !CanCompress(opts.Format) {
		return fmt.Errorf("format %s is not supported for compression", opts.Format)
	}

	// 7z write shells out to the 7-Zip CLI and walks directories itself,
	// so it doesn't need the collected file list. Handle it before collect()
	// to avoid an unnecessary directory walk.
	if opts.Format == Format7z {
		if err := write7zViaBinary(sources, dest, opts, p); err != nil {
			return err
		}
		if p != nil && p.OnDone != nil {
			p.OnDone()
		}
		return nil
	}

	files, err := collect(sources, opts.RootDir)
	if err != nil {
		return err
	}
	reportProgress(p, 0, len(files), "")

	switch opts.Format {
	case FormatZip:
		err = writeZip(files, dest, opts, p)
	case FormatTar:
		err = writeTar(files, dest, opts, false, p)
	case FormatTarGz:
		err = writeTar(files, dest, opts, true, p)
	}
	if err != nil {
		return err
	}
	if p != nil && p.OnDone != nil {
		p.OnDone()
	}
	return nil
}

// Extract unpacks archive at src into destDir (created if missing).
func Extract(src, destDir string, opts Options, p *Progress) error {
	f := opts.Format
	if f == FormatAuto {
		f = Detect(src)
		if f == FormatAuto {
			return fmt.Errorf("cannot detect format from %s", src)
		}
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	var err error
	switch f {
	case FormatZip:
		err = extractZip(src, destDir, p)
	case FormatTar:
		err = extractTar(src, destDir, false, p)
	case FormatTarGz:
		err = extractTar(src, destDir, true, p)
	case Format7z:
		err = extract7z(src, destDir, opts, p)
	case FormatRar:
		err = extractRar(src, destDir, opts, p)
	case FormatIso:
		err = extractIso(src, destDir, p)
	default:
		return fmt.Errorf("unsupported format %s", f)
	}
	if err != nil {
		return err
	}
	if p != nil && p.OnDone != nil {
		p.OnDone()
	}
	return nil
}

// ---------- helpers ----------

// fileEntry is a file we plan to write into an archive.
type fileEntry struct {
	absPath string // absolute path on disk (source)
	arcPath string // path stored inside the archive
	info    os.FileInfo
}

// collect walks every source, producing a list of fileEntry.
func collect(sources []string, rootDir string) ([]fileEntry, error) {
	if len(sources) == 0 {
		return nil, fmt.Errorf("no sources")
	}
	var out []fileEntry
	for _, src := range sources {
		abs, err := filepath.Abs(src)
		if err != nil {
			return nil, err
		}
		base := filepath.Dir(abs)
		if rootDir != "" {
			if rb, e := filepath.Abs(rootDir); e == nil {
				base = rb
			}
		}
		err = filepath.WalkDir(abs, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, e := filepath.Rel(base, path)
			if e != nil {
				return e
			}
			info, e := d.Info()
			if e != nil {
				return e
			}
			arc := filepath.ToSlash(rel)
			if d.IsDir() {
				arc += "/"
			}
			out = append(out, fileEntry{absPath: path, arcPath: arc, info: info})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

// safeJoin prevents path traversal when extracting (e.g. "../../etc/passwd").
func safeJoin(base, rel string) (string, error) {
	cleaned := filepath.Clean("/" + rel) // forces leading slash, removes ..
	final := filepath.Join(base, cleaned)
	bc := filepath.Clean(base)
	fc := filepath.Clean(final)
	if !strings.HasPrefix(fc, bc+string(os.PathSeparator)) && fc != bc {
		return "", fmt.Errorf("path escapes target dir: %s", rel)
	}
	return final, nil
}

// copyFile is the workhorse for extraction.
func copyFile(dst string, src io.Reader, mode fs.FileMode, mtime time.Time) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, src); err != nil {
		return err
	}
	return os.Chtimes(dst, mtime, mtime)
}

func reportProgress(p *Progress, done, total int, current string) {
	if p != nil && p.OnAdvance != nil {
		p.OnAdvance(done, total, current)
	}
}

// ---------- ZIP ----------

func writeZip(files []fileEntry, dest string, opts Options, p *Progress) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	defer zw.Close()

	for i, fe := range files {
		reportProgress(p, i, len(files), fe.arcPath)
		if strings.HasSuffix(fe.arcPath, "/") {
			if _, err := zw.CreateHeader(&zip.FileHeader{Name: fe.arcPath, Method: zip.Store}); err != nil {
				return err
			}
			continue
		}
		header, err := zip.FileInfoHeader(fe.info)
		if err != nil {
			return err
		}
		header.Name = fe.arcPath
		header.Method = zip.Deflate
		if opts.Level == LevelStore {
			header.Method = zip.Store
		}
		w, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		r, err := os.Open(fe.absPath)
		if err != nil {
			return err
		}
		_, err = io.Copy(w, r)
		r.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func extractZip(src, dest string, p *Progress) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()
	total := len(r.File)
	for i, f := range r.File {
		// Decode the name now (may be GBK-encoded on Windows-created zips).
		// We pass the decoded name to safeJoin/copyFile so the on-disk
		// filename uses the correct characters.
		decodedName := decodeZipName(f.Name)
		reportProgress(p, i, total, decodedName)
		target, err := safeJoin(dest, decodedName)
		if err != nil {
			return err
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		// Make sure parent dirs exist even when the zip omits explicit
		// directory entries (common on macOS-created zips).
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		err = copyFile(target, rc, f.Mode(), f.ModTime())
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// ---------- TAR / TAR.GZ ----------

func writeTar(files []fileEntry, dest string, opts Options, gzip bool, p *Progress) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	var w io.Writer = f
	var gw *gzipWriter
	if gzip {
		gw = newGzipWriter(f, opts.Level)
		defer gw.Close()
		w = gw
	}
	tw := tar.NewWriter(w)
	defer tw.Close()

	for i, fe := range files {
		reportProgress(p, i, len(files), fe.arcPath)
		hdr, err := tar.FileInfoHeader(fe.info, "")
		if err != nil {
			return err
		}
		hdr.Name = fe.arcPath
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if fe.info.IsDir() {
			continue
		}
		r, err := os.Open(fe.absPath)
		if err != nil {
			return err
		}
		_, err = io.Copy(tw, r)
		r.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func extractTar(src, dest string, gzip bool, p *Progress) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	var r io.Reader = f
	if gzip {
		gr, err := newGzipReader(f)
		if err != nil {
			return err
		}
		defer gr.Close()
		r = gr
	}
	tr := tar.NewReader(r)
	i := 0
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		reportProgress(p, i, -1, hdr.Name)
		i++
		target, err := safeJoin(dest, hdr.Name)
		if err != nil {
			return err
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(hdr.Mode)); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := copyFile(target, tr, os.FileMode(hdr.Mode), hdr.ModTime); err != nil {
				return err
			}
		case tar.TypeSymlink:
			// Skipped to avoid path-traversal escapes. A future release can
			// add a per-entry prompt.
		}
	}
	return nil
}

// ---------- 7z (extract-only) ----------

func extract7z(src, dest string, opts Options, p *Progress) error {
	r, err := sevenzip.OpenReaderWithPassword(src, opts.Password)
	if err != nil {
		return err
	}
	defer r.Close()
	total := len(r.File)
	for i, f := range r.File {
		reportProgress(p, i, total, f.Name)
		target, err := safeJoin(dest, f.Name)
		if err != nil {
			return err
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		err = copyFile(target, rc, f.Mode(), f.Modified)
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// ---------- RAR (extract-only) ----------

func extractRar(src, dest string, opts Options, p *Progress) error {
	rr, err := rardecode.OpenReader(src, opts.Password)
	if err != nil {
		return err
	}
	defer rr.Close()
	i := 0
	for {
		hdr, err := rr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		reportProgress(p, i, -1, hdr.Name)
		i++
		target, err := safeJoin(dest, hdr.Name)
		if err != nil {
			return err
		}
		if hdr.IsDir {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := copyFile(target, rr, hdr.Mode(), hdr.ModificationTime); err != nil {
			return err
		}
	}
	return nil
}

// ---------- ISO (extract-only) ----------

func extractIso(src, dest string, p *Progress) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	img, err := iso9660.OpenImage(f)
	if err != nil {
		return err
	}
	root, err := img.RootDir()
	if err != nil {
		return err
	}
	walked := 0
	return walkIso(root, dest, &walked, p)
}

// walkIso recursively extracts an ISO9660 directory tree.
func walkIso(d *iso9660.File, dest string, walked *int, p *Progress) error {
	children, err := d.GetChildren()
	if err != nil {
		return err
	}
	for _, c := range children {
		*walked++
		reportProgress(p, *walked, -1, c.Name())
		target := filepath.Join(dest, c.Name())
		if c.IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			if err := walkIso(c, target, walked, p); err != nil {
				return err
			}
			continue
		}
		if err := copyFile(target, c.Reader(), c.Mode(), c.ModTime()); err != nil {
			return err
		}
	}
	return nil
}
