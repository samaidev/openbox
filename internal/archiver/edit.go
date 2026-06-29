package archiver

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// DeleteEntries removes the named entries from an archive.
//
// Currently only .zip is supported (because Go's stdlib archive/zip gives us
// full read+write access). For .7z / .rar / .iso the function returns
// ErrDeleteNotSupported so the UI can show a clear message instead of
// silently failing.
//
// The archive is rewritten in place via a temp file + atomic rename, so a
// crash mid-rewrite leaves the original intact.
func DeleteEntries(src string, entries []string, opts Options) error {
	f := opts.Format
	if f == FormatAuto {
		f = Detect(src)
		if f == FormatAuto {
			return fmt.Errorf("cannot detect format from %s", src)
		}
	}
	switch f {
	case FormatZip:
		return deleteFromZip(src, entries)
	default:
		return fmt.Errorf("delete is only supported for .zip archives (got %s)", f)
	}
}

// deleteFromZip rewrites the zip at src without the named entries.
// entries are matched against the decoded in-archive name (so GBK-encoded
// names from Windows Explorer zips work too).
func deleteFromZip(src string, entries []string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	// Build a set of entries to drop, normalized to forward slashes and
	// without trailing slashes so both "foo/" and "foo" match a directory
	// entry.
	drop := make(map[string]struct{}, len(entries))
	for _, e := range entries {
		n := strings.TrimSuffix(filepath.ToSlash(e), "/")
		drop[n] = struct{}{}
		// Also drop with a trailing slash form, in case the stored name
		// has one.
		drop[n+"/"] = struct{}{}
	}

	tmp, err := os.CreateTemp(filepath.Dir(src), ".openbox-del-*.tmp")
	if err != nil {
		r.Close()
		return fmt.Errorf("create temp: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := func() {
		tmp.Close()
		os.Remove(tmpName)
	}

	zw := zip.NewWriter(tmp)
	for _, f := range r.File {
		name := decodeZipName(f.Name)
		if _, ok := drop[name]; ok {
			continue
		}
		if _, ok := drop[strings.TrimSuffix(name, "/")]; ok && strings.HasSuffix(name, "/") {
			continue
		}
		if err := copyZipEntry(zw, f); err != nil {
			r.Close()
			zw.Close()
			cleanup()
			return fmt.Errorf("copy entry %q: %w", f.Name, err)
		}
	}
	if err := zw.Close(); err != nil {
		r.Close()
		cleanup()
		return fmt.Errorf("close writer: %w", err)
	}
	if err := tmp.Close(); err != nil {
		r.Close()
		cleanup()
		return fmt.Errorf("close temp: %w", err)
	}
	r.Close()

	// Preserve the original file mode.
	if fi, err := os.Stat(src); err == nil {
		_ = os.Chmod(tmpName, fi.Mode())
	}
	if err := os.Rename(tmpName, src); err != nil {
		// On Windows, rename can fail if the target is open. Fall back
		// to copy+replace.
		if err := replaceViaCopy(tmpName, src); err != nil {
			os.Remove(tmpName)
			return fmt.Errorf("rename: %w", err)
		}
		os.Remove(tmpName)
	}
	return nil
}

// copyZipEntry copies one entry verbatim from src to dst, preserving the
// file header (method, mode, mtime, extra) so the round-trip is lossless.
func copyZipEntry(zw *zip.Writer, f *zip.File) error {
	hdr := f.FileHeader // shallow copy of the struct
	w, err := zw.CreateHeader(&hdr)
	if err != nil {
		return err
	}
	if f.FileInfo().IsDir() {
		return nil
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	_, err = io.Copy(w, rc)
	return err
}

// AddToArchive appends new files to an existing archive.
//
// Currently only .zip is supported. The archive is rewritten with all old
// entries preserved plus the new ones appended. Source paths are stored
// relative to their parent directory (matching the Compress behavior).
func AddToArchive(dest string, sources []string, opts Options) error {
	f := opts.Format
	if f == FormatAuto {
		f = Detect(dest)
		if f == FormatAuto {
			return fmt.Errorf("cannot detect format from %s", dest)
		}
	}
	switch f {
	case FormatZip:
		return addToZip(dest, sources, opts)
	default:
		return fmt.Errorf("add is only supported for .zip archives (got %s)", f)
	}
}

// addToZip rewrites the zip at dest with all existing entries plus the
// new sources appended. Existing entries are copied verbatim (preserving
// method, mode, mtime, extra).
func addToZip(dest string, sources []string, opts Options) error {
	// Read existing entries (if any).
	var existing *zip.ReadCloser
	if fi, err := os.Stat(dest); err == nil && !fi.IsDir() && fi.Size() > 0 {
		r, err := zip.OpenReader(dest)
		if err != nil {
			return fmt.Errorf("open existing zip: %w", err)
		}
		existing = r
		defer existing.Close()
	}

	// Collect new files to add.
	files, err := collect(sources, opts.RootDir)
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(dest), ".openbox-add-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := func() {
		tmp.Close()
		os.Remove(tmpName)
	}

	zw := zip.NewWriter(tmp)

	// 1) Copy existing entries verbatim.
	if existing != nil {
		for _, f := range existing.File {
			if err := copyZipEntry(zw, f); err != nil {
				zw.Close()
				cleanup()
				return fmt.Errorf("copy entry %q: %w", f.Name, err)
			}
		}
	}

	// 2) Append new entries.
	for _, fe := range files {
		if strings.HasSuffix(fe.arcPath, "/") {
			if _, err := zw.CreateHeader(&zip.FileHeader{Name: fe.arcPath, Method: zip.Store}); err != nil {
				zw.Close()
				cleanup()
				return err
			}
			continue
		}
		header, err := zip.FileInfoHeader(fe.info)
		if err != nil {
			zw.Close()
			cleanup()
			return err
		}
		header.Name = fe.arcPath
		header.Method = zip.Deflate
		if opts.Level == LevelStore {
			header.Method = zip.Store
		}
		w, err := zw.CreateHeader(header)
		if err != nil {
			zw.Close()
			cleanup()
			return err
		}
		if err := copyFileToWriter(w, fe.absPath); err != nil {
			zw.Close()
			cleanup()
			return err
		}
	}

	if err := zw.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close writer: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp: %w", err)
	}

	if fi, err := os.Stat(dest); err == nil {
		_ = os.Chmod(tmpName, fi.Mode())
	}
	if err := os.Rename(tmpName, dest); err != nil {
		if err := replaceViaCopy(tmpName, dest); err != nil {
			os.Remove(tmpName)
			return fmt.Errorf("rename: %w", err)
		}
		os.Remove(tmpName)
	}
	return nil
}

// copyFileToWriter opens the file at absPath and streams it into w.
func copyFileToWriter(w io.Writer, absPath string) error {
	r, err := os.Open(absPath)
	if err != nil {
		return err
	}
	defer r.Close()
	_, err = io.Copy(w, r)
	return err
}

// replaceViaCopy is a Windows-friendly fallback when os.Rename fails because
// the destination is locked. It copies bytes from src to dst, then deletes
// src. If dst doesn't exist, os.Rename should have worked in the first
// place, so this is only used when Rename returns an error.
func replaceViaCopy(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	// Preserve the original mode of dst if it had one.
	if fi, err := os.Stat(src); err == nil {
		_ = os.Chmod(dst, fi.Mode())
	}
	return nil
}

// Make sure io/fs stays referenced (used indirectly by collect in
// archiver.go, but keep the import alive here too for clarity).
var _ = fs.ModeDir
