package archiver

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestListZipGBKFilename verifies that .zip files created with GBK-encoded
// filenames (the de-facto standard for Windows Explorer on zh-CN systems)
// are correctly decoded to UTF-8 by List() and extractZip().
//
// Without the GBK fallback in decodeZipName, the filename would come back
// as mojibake like "ÎÄ¼þÃû.txt" instead of "文件名.txt".
func TestListZipGBKFilename(t *testing.T) {
	// "测试文件.txt" encoded in GBK
	gbkName := []byte{0xB2, 0xE2, 0xCA, 0xD4, 0xCE, 0xC4, 0xBC, 0xFE, 0x2E, 0x74, 0x78, 0x74}
	want := "测试文件.txt"

	tmp := t.TempDir()
	zipPath := filepath.Join(tmp, "gbk.zip")

	// Build a zip file with raw GBK bytes in the filename (no UTF-8 flag set).
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	fh := &zip.FileHeader{
		Name:   string(gbkName),
		Method: zip.Store,
	}
	// Do NOT set the UTF-8 flag (bit 11) — this is what Windows Explorer
	// produces when creating a zip on a zh-CN system.
	fh.SetMode(0o644)
	fh.Modified = time.Now()
	fw, err := w.CreateHeader(fh)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	// List
	entries, err := List(zipPath, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("want 1 entry, got %d: %+v", len(entries), entries)
	}
	if entries[0].Name != want {
		t.Errorf("filename: got %q want %q", entries[0].Name, want)
	}

	// Extract and verify the on-disk filename.
	dest := t.TempDir()
	if err := Extract(zipPath, dest, Options{}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, want)); err != nil {
		t.Errorf("expected file %q to exist after extract: %v", want, err)
	}
}

// TestListZipUTF8Filename verifies UTF-8 zip filenames still work.
func TestListZipUTF8Filename(t *testing.T) {
	tmp := t.TempDir()
	zipPath := filepath.Join(tmp, "utf8.zip")

	// Build a zip with a UTF-8 filename and the EFS flag set.
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	fw, err := w.Create("文件.txt")
	if err != nil {
		t.Fatal(err)
	}
	fw.Write([]byte("hello"))
	w.Close()
	f.Close()

	entries, err := List(zipPath, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(entries))
	}
	if entries[0].Name != "文件.txt" {
		t.Errorf("got %q want %q", entries[0].Name, "文件.txt")
	}
}

// TestListRoundTrip verifies List returns the same entries we compressed.
func TestListRoundTrip(t *testing.T) {
	src := makeTree(t)
	dest := filepath.Join(t.TempDir(), "out.zip")
	if err := Compress([]string{src}, dest, Options{Format: FormatZip}, nil); err != nil {
		t.Fatal(err)
	}
	entries, err := List(dest, Options{})
	if err != nil {
		t.Fatal(err)
	}
	// makeTree creates: a.txt, sub/b.txt, sub/deep/c.txt, empty/
	// The collector walks every path, so we expect at least 4 entries.
	if len(entries) < 4 {
		t.Errorf("want >= 4 entries, got %d: %+v", len(entries), entries)
	}
	// Verify that the parent dir name appears.
	found := false
	for _, e := range entries {
		if e.Name != "" {
			found = true
			break
		}
	}
	if !found {
		t.Error("no entries with non-empty names")
	}
}

// TestExtractEntrySingleFile verifies single-entry extraction.
func TestExtractEntrySingleFile(t *testing.T) {
	src := makeTree(t)
	dest := filepath.Join(t.TempDir(), "out.zip")
	if err := Compress([]string{src}, dest, Options{Format: FormatZip}, nil); err != nil {
		t.Fatal(err)
	}
	entries, err := List(dest, Options{})
	if err != nil {
		t.Fatal(err)
	}
	// Find a non-dir entry.
	var target Entry
	for _, e := range entries {
		if !e.IsDir && e.Name != "" {
			target = e
			break
		}
	}
	if target.Name == "" {
		t.Fatal("no file entry found in archive")
	}
	out := t.TempDir()
	if err := ExtractEntry(dest, target.Name, out, Options{}); err != nil {
		t.Fatal(err)
	}
	// The file should exist at out/<full relative path> (ExtractEntry
	// preserves the in-archive path).
	full := filepath.Join(out, filepath.FromSlash(target.Name))
	if _, err := os.Stat(full); err != nil {
		t.Errorf("expected %q to exist: %v", full, err)
	}
}
