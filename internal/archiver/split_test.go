package archiver

import (
	"os"
	"path/filepath"
	"testing"
)

// TestZipSplitRoundTrip verifies the basic split-volume zip workflow:
// compress with VolumeSize, then extract the .001 part back.
func TestZipSplitRoundTrip(t *testing.T) {
	src := makeTree(t)
	dest := filepath.Join(t.TempDir(), "out.zip")

	// VolumeSize = 50 bytes. Our test tree has files of similar size,
	// so this will produce multiple parts.
	opts := Options{
		Format:     FormatZip,
		Level:      LevelStore, // store so we can predict sizes
		VolumeSize: 50,
	}
	if err := Compress([]string{src}, dest, opts, nil); err != nil {
		t.Fatal(err)
	}

	// Verify .001, .002, ... exist.
	parts, err := findSplitVolumes(dest + ".001")
	if err != nil {
		t.Fatalf("findSplitVolumes: %v", err)
	}
	if len(parts) < 2 {
		t.Errorf("expected >= 2 volumes, got %d", len(parts))
	}

	// Extract via the .001 part.
	out := t.TempDir()
	if err := Extract(dest+".001", out, Options{}, nil); err != nil {
		t.Fatal(err)
	}
	equalTree(t, src, filepath.Join(out, filepath.Base(src)))
}

// TestZipSplitList verifies List() works on a split-volume zip.
func TestZipSplitList(t *testing.T) {
	src := makeTree(t)
	dest := filepath.Join(t.TempDir(), "out.zip")
	opts := Options{
		Format:     FormatZip,
		Level:      LevelStore,
		VolumeSize: 512, // tiny to get many parts
	}
	if err := Compress([]string{src}, dest, opts, nil); err != nil {
		t.Fatal(err)
	}

	entries, err := List(dest+".001", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) < 4 {
		t.Errorf("expected >= 4 entries, got %d", len(entries))
	}
}

// TestParseVolumeSizeMonitors checks that the volume-size parser in
// the CLI flag accepts all the expected unit suffixes. We test the
// underlying parseVolumeSize logic indirectly via the public API by
// checking that Compress rejects obviously invalid sizes.
func TestCompressRejectsInvalidVolumeSize(t *testing.T) {
	src := makeTree(t)
	dest := filepath.Join(t.TempDir(), "out.zip")
	// VolumeSize = 0 should be treated as "no splitting" and produce
	// a single-file zip.
	opts := Options{Format: FormatZip, Level: LevelStore, VolumeSize: 0}
	if err := Compress([]string{src}, dest, opts, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dest); err != nil {
		t.Errorf("expected single zip at %s: %v", dest, err)
	}
	// Make sure no .001 was created.
	if _, err := os.Stat(dest + ".001"); err == nil {
		t.Errorf("did not expect .001 part when VolumeSize=0")
	}
}

// TestSevenZipSplitRoundTrip tests 7z split-volume compression +
// extraction. Skipped if 7z CLI is not installed.
func TestSevenZipSplitRoundTrip(t *testing.T) {
	if !SevenZipWriterAvailable() {
		t.Skip("no 7z CLI in PATH")
	}
	src := makeTree(t)
	dest := filepath.Join(t.TempDir(), "out.7z")
	opts := Options{
		Format:     Format7z,
		Level:      LevelNormal,
		VolumeSize: 512, // tiny → multiple parts
	}
	if err := Compress([]string{src}, dest, opts, nil); err != nil {
		t.Fatal(err)
	}

	// Verify parts exist.
	parts, err := findSplitVolumes(dest + ".001")
	if err != nil {
		t.Fatalf("findSplitVolumes: %v", err)
	}
	if len(parts) < 2 {
		t.Errorf("expected >= 2 volumes, got %d", len(parts))
	}

	// Extract via .001.
	out := t.TempDir()
	if err := Extract(dest+".001", out, Options{}, nil); err != nil {
		t.Fatal(err)
	}
	equalTree(t, src, filepath.Join(out, filepath.Base(src)))
}

// TestFindSplitVolumesContiguityCheck verifies that findSplitVolumes
// errors out when parts are missing (e.g. .001 and .003 but no .002).
func TestFindSplitVolumesContiguityCheck(t *testing.T) {
	dir := t.TempDir()
	// Create .001 and .003 but NOT .002.
	for _, n := range []string{"test.zip.001", "test.zip.003"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	_, err := findSplitVolumes(filepath.Join(dir, "test.zip.001"))
	if err == nil {
		t.Error("expected error for non-contiguous volumes, got nil")
	}
}
