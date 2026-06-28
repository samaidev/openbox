package archiver

import (
	"archive/tar"
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bodgit/sevenzip"
	"github.com/kdomanski/iso9660"
	"github.com/nwaples/rardecode"
)

// Entry describes a single file or directory inside an archive.
// Used by List() and the UI's browse view.
type Entry struct {
	Name      string    // path inside the archive, '/'-separated
	Size      int64     // uncompressed size in bytes (0 for dirs)
	IsDir     bool      // directory entry
	ModTime   time.Time // modification time (zero if not set)
	IsSymlink bool      // symlink entry (extractable, but skipped by default)
}

// List returns the entries inside an archive without extracting it.
// Format is auto-detected from the filename.
//
// For .7z and .rar archives that are password-protected, you must pass
// opts.Password — otherwise the call fails.
func List(src string, opts Options) ([]Entry, error) {
	f := opts.Format
	if f == FormatAuto {
		f = Detect(src)
		if f == FormatAuto {
			return nil, fmt.Errorf("cannot detect format from %s", src)
		}
	}

	switch f {
	case FormatZip:
		return listZip(src)
	case FormatTar:
		return listTar(src, false)
	case FormatTarGz:
		return listTar(src, true)
	case Format7z:
		return list7z(src, opts)
	case FormatRar:
		return listRar(src, opts)
	case FormatIso:
		return listIso(src)
	}
	return nil, fmt.Errorf("unsupported format %s", f)
}

// listZip reads the central directory of a .zip file.
func listZip(src string) ([]Entry, error) {
	r, err := zip.OpenReader(src)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	out := make([]Entry, 0, len(r.File))
	for _, f := range r.File {
		name := decodeZipName(f.Name)
		out = append(out, Entry{
			Name:    name,
			Size:    int64(f.UncompressedSize64),
			IsDir:   f.FileInfo().IsDir(),
			ModTime: f.Modified,
		})
	}
	return out, nil
}

// listTar streams through a .tar / .tar.gz archive and collects headers.
func listTar(src string, gzip bool) ([]Entry, error) {
	f, err := os.Open(src)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var r io.Reader = f
	if gzip {
		gr, err := newGzipReader(f)
		if err != nil {
			return nil, err
		}
		defer gr.Close()
		r = gr
	}
	tr := tar.NewReader(r)
	var out []Entry
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		entry := Entry{
			Name:    hdr.Name,
			Size:    hdr.Size,
			ModTime: hdr.ModTime,
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			entry.IsDir = true
		case tar.TypeSymlink, tar.TypeLink:
			entry.IsSymlink = true
		}
		out = append(out, entry)
	}
	return out, nil
}

// list7z reads the file table of a .7z archive.
func list7z(src string, opts Options) ([]Entry, error) {
	r, err := sevenzip.OpenReaderWithPassword(src, opts.Password)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	out := make([]Entry, 0, len(r.File))
	for _, f := range r.File {
		out = append(out, Entry{
			Name:    f.Name,
			Size:    int64(f.UncompressedSize),
			IsDir:   f.FileInfo().IsDir(),
			ModTime: f.Modified,
		})
	}
	return out, nil
}

// listRar streams through a .rar archive.
func listRar(src string, opts Options) ([]Entry, error) {
	rr, err := rardecode.OpenReader(src, opts.Password)
	if err != nil {
		return nil, err
	}
	defer rr.Close()
	var out []Entry
	for {
		hdr, err := rr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		out = append(out, Entry{
			Name:    hdr.Name,
			Size:    hdr.UnPackedSize,
			IsDir:   hdr.IsDir,
			ModTime: hdr.ModificationTime,
		})
	}
	return out, nil
}

// listIso walks an ISO 9660 image recursively.
func listIso(src string) ([]Entry, error) {
	f, err := os.Open(src)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, err := iso9660.OpenImage(f)
	if err != nil {
		return nil, err
	}
	root, err := img.RootDir()
	if err != nil {
		return nil, err
	}
	var out []Entry
	walkIsoList(root, "", &out)
	return out, nil
}

func walkIsoList(d *iso9660.File, prefix string, out *[]Entry) {
	children, err := d.GetChildren()
	if err != nil {
		return
	}
	for _, c := range children {
		name := prefix + c.Name()
		if c.IsDir() {
			*out = append(*out, Entry{Name: name + "/", IsDir: true, ModTime: c.ModTime()})
			walkIsoList(c, name+"/", out)
		} else {
			*out = append(*out, Entry{Name: name, Size: c.Size(), ModTime: c.ModTime()})
		}
	}
}

// ExtractEntry extracts a single entry from an archive into destDir.
// Used by the UI's "extract this file only" feature.
func ExtractEntry(src, entryName, destDir string, opts Options) error {
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
	switch f {
	case FormatZip:
		return extractZipEntry(src, entryName, destDir)
	case FormatTar, FormatTarGz:
		return extractTarEntry(src, entryName, destDir, f == FormatTarGz)
	case Format7z:
		return extract7zEntry(src, entryName, destDir, opts)
	case FormatRar:
		return extractRarEntry(src, entryName, destDir, opts)
	case FormatIso:
		return extractIsoEntry(src, entryName, destDir)
	}
	return fmt.Errorf("unsupported format %s", f)
}

func extractZipEntry(src, entryName, destDir string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		if decodeZipName(f.Name) != entryName {
			continue
		}
		target, err := safeJoin(destDir, f.Name)
		if err != nil {
			return err
		}
		if f.FileInfo().IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()
		return copyFile(target, rc, f.Mode(), f.ModTime())
	}
	return fmt.Errorf("entry %q not found in archive", entryName)
}

func extractTarEntry(src, entryName, destDir string, gzip bool) error {
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
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.Name != entryName {
			continue
		}
		target, err := safeJoin(destDir, hdr.Name)
		if err != nil {
			return err
		}
		if hdr.Typeflag == tar.TypeDir {
			return os.MkdirAll(target, os.FileMode(hdr.Mode))
		}
		return copyFile(target, tr, os.FileMode(hdr.Mode), hdr.ModTime)
	}
	return fmt.Errorf("entry %q not found in archive", entryName)
}

func extract7zEntry(src, entryName, destDir string, opts Options) error {
	r, err := sevenzip.OpenReaderWithPassword(src, opts.Password)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		if f.Name != entryName {
			continue
		}
		target, err := safeJoin(destDir, f.Name)
		if err != nil {
			return err
		}
		if f.FileInfo().IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()
		return copyFile(target, rc, f.Mode(), f.Modified)
	}
	return fmt.Errorf("entry %q not found in archive", entryName)
}

func extractRarEntry(src, entryName, destDir string, opts Options) error {
	rr, err := rardecode.OpenReader(src, opts.Password)
	if err != nil {
		return err
	}
	defer rr.Close()
	for {
		hdr, err := rr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.Name != entryName {
			continue
		}
		target, err := safeJoin(destDir, hdr.Name)
		if err != nil {
			return err
		}
		if hdr.IsDir {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(target, rr, hdr.Mode(), hdr.ModificationTime)
	}
	return fmt.Errorf("entry %q not found in archive", entryName)
}

func extractIsoEntry(src, entryName, destDir string) error {
	// ISO entries are full paths; we walk to find the matching name.
	// For simplicity, use the full-extract path but skip non-matching entries.
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
	// entryName like "subdir/file.txt" — split into segments.
	segs := strings.Split(strings.TrimSuffix(entryName, "/"), "/")
	return walkIsoExtract(root, segs, 0, destDir)
}

// walkIsoExtract walks the ISO tree following segs[] until it finds the
// target file/dir, then extracts it.
func walkIsoExtract(d *iso9660.File, segs []string, depth int, destDir string) error {
	if depth >= len(segs) {
		return nil
	}
	children, err := d.GetChildren()
	if err != nil {
		return err
	}
	for _, c := range children {
		if c.Name() != segs[depth] {
			continue
		}
		target := filepath.Join(destDir, c.Name())
		if depth == len(segs)-1 {
			// final segment — extract
			if c.IsDir() {
				return os.MkdirAll(target, 0o755)
			}
			// Note: iso9660 File.Reader() returns io.Reader (no Closer);
			// the underlying file is closed when we close src at the end
			// of ExtractEntry().
			return copyFile(target, c.Reader(), c.Mode(), c.ModTime())
		}
		if c.IsDir() {
			return walkIsoExtract(c, segs, depth+1, destDir)
		}
	}
	return fmt.Errorf("entry not found in ISO")
}
