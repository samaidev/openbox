package archiver

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeTree builds a tiny temp directory tree for round-trip tests.
func makeTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	must := func(p string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("hello "+filepath.Base(p)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	must(filepath.Join(root, "a.txt"))
	must(filepath.Join(root, "sub", "b.txt"))
	must(filepath.Join(root, "sub", "deep", "c.txt"))
	// Empty dir to verify it survives a round-trip.
	if err := os.MkdirAll(filepath.Join(root, "empty"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

// equalTree compares two directory trees by relative path + content.
func equalTree(t *testing.T, want, got string) {
	t.Helper()
	wantFiles := map[string]string{}
	err := filepath.WalkDir(want, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(want, p)
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			wantFiles[rel+"/"] = ""
			return nil
		}
		b, _ := os.ReadFile(p)
		wantFiles[rel] = string(b)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	gotFiles := map[string]string{}
	err = filepath.WalkDir(got, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(got, p)
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			gotFiles[rel+"/"] = ""
			return nil
		}
		b, _ := os.ReadFile(p)
		gotFiles[rel] = string(b)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(wantFiles) != len(gotFiles) {
		t.Fatalf("file count mismatch: want %d got %d\nwant=%v\ngot=%v",
			len(wantFiles), len(gotFiles), wantFiles, gotFiles)
	}
	for k, v := range wantFiles {
		if gotFiles[k] != v {
			t.Errorf("mismatch for %s: want %q got %q", k, v, gotFiles[k])
		}
	}
}

func TestZipRoundTrip(t *testing.T) {
	src := makeTree(t)
	dest := filepath.Join(t.TempDir(), "out.zip")
	out := t.TempDir()
	if err := Compress([]string{src}, dest, Options{Format: FormatZip, Level: LevelNormal}, nil); err != nil {
		t.Fatal(err)
	}
	if err := Extract(dest, out, Options{}, nil); err != nil {
		t.Fatal(err)
	}
	equalTree(t, src, filepath.Join(out, filepath.Base(src)))
}

func TestTarRoundTrip(t *testing.T) {
	src := makeTree(t)
	dest := filepath.Join(t.TempDir(), "out.tar")
	out := t.TempDir()
	if err := Compress([]string{src}, dest, Options{Format: FormatTar}, nil); err != nil {
		t.Fatal(err)
	}
	if err := Extract(dest, out, Options{}, nil); err != nil {
		t.Fatal(err)
	}
	equalTree(t, src, filepath.Join(out, filepath.Base(src)))
}

func TestTarGzRoundTrip(t *testing.T) {
	src := makeTree(t)
	dest := filepath.Join(t.TempDir(), "out.tar.gz")
	out := t.TempDir()
	if err := Compress([]string{src}, dest, Options{Format: FormatTarGz, Level: LevelMaximum}, nil); err != nil {
		t.Fatal(err)
	}
	if err := Extract(dest, out, Options{}, nil); err != nil {
		t.Fatal(err)
	}
	equalTree(t, src, filepath.Join(out, filepath.Base(src)))
}

func TestSevenZipWriteRoundTrip(t *testing.T) {
	if !SevenZipWriterAvailable() {
		t.Skip("no 7z CLI in PATH; install p7zip-full / 7zip / 7-Zip to run this test")
	}
	src := makeTree(t)
	dest := filepath.Join(t.TempDir(), "out.7z")
	out := t.TempDir()
	if err := Compress([]string{src}, dest, Options{Format: Format7z, Level: LevelNormal}, nil); err != nil {
		t.Fatal(err)
	}
	if err := Extract(dest, out, Options{}, nil); err != nil {
		t.Fatal(err)
	}
	equalTree(t, src, filepath.Join(out, filepath.Base(src)))
}

func TestSevenZipWriteWithPassword(t *testing.T) {
	if !SevenZipWriterAvailable() {
		t.Skip("no 7z CLI in PATH; install p7zip-full / 7zip / 7-Zip to run this test")
	}
	src := makeTree(t)
	dest := filepath.Join(t.TempDir(), "out.7z")
	out := t.TempDir()
	if err := Compress([]string{src}, dest, Options{Format: Format7z, Level: LevelMaximum, Password: "secret"}, nil); err != nil {
		t.Fatal(err)
	}
	if err := Extract(dest, out, Options{Password: "secret"}, nil); err != nil {
		t.Fatal(err)
	}
	equalTree(t, src, filepath.Join(out, filepath.Base(src)))
}

func TestSevenZipWriteMultipleSources(t *testing.T) {
	if !SevenZipWriterAvailable() {
		t.Skip("no 7z CLI in PATH")
	}
	src1 := makeTree(t)
	src2 := t.TempDir()
	if err := os.WriteFile(filepath.Join(src2, "extra.txt"), []byte("extra"), 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "multi.7z")
	out := t.TempDir()
	if err := Compress([]string{src1, src2}, dest, Options{Format: Format7z}, nil); err != nil {
		t.Fatal(err)
	}
	if err := Extract(dest, out, Options{}, nil); err != nil {
		t.Fatal(err)
	}
	// Both source base names should appear as top-level entries.
	if _, err := os.Stat(filepath.Join(out, filepath.Base(src1))); err != nil {
		t.Errorf("src1 missing in archive: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, filepath.Base(src2))); err != nil {
		t.Errorf("src2 missing in archive: %v", err)
	}
}

func TestDetect(t *testing.T) {
	cases := map[string]Format{
		"a.zip":     FormatZip,
		"a.tar":     FormatTar,
		"a.tar.gz":  FormatTarGz,
		"a.tgz":     FormatTarGz,
		"a.7z":      Format7z,
		"a.rar":     FormatRar,
		"a.iso":     FormatIso,
		"weird":     FormatAuto,
		"UPPER.ZIP": FormatZip,
	}
	for name, want := range cases {
		if got := Detect(name); got != want {
			t.Errorf("Detect(%q)=%v want %v", name, got, want)
		}
	}
}

func TestSafeJoinRejectsTraversal(t *testing.T) {
	// safeJoin must never produce a path that escapes the base dir.
	// Traversal sequences are normalised so the result stays inside base.
	// We use filepath.Clean on the base so the comparison works on both
	// POSIX (where base "/tmp/dest" stays as-is) and Windows (where
	// filepath.Clean("/tmp/dest") returns "\tmp\dest" with backslashes,
	// matching what safeJoin returns internally).
	base := filepath.Clean("/tmp/dest")
	cases := []string{
		"../../etc/passwd",
		"/etc/passwd",
		"sub/../../../etc/shadow",
		"..",
	}
	for _, rel := range cases {
		got, err := safeJoin(base, rel)
		if err != nil {
			continue // explicit rejection is also acceptable
		}
		// On Windows the result may use backslashes; compare using
		// filepath.Clean + HasPrefix on the cleaned form.
		gotClean := filepath.Clean(got)
		want := base + string(filepath.Separator)
		if !strings.HasPrefix(gotClean, want) && gotClean != base {
			t.Errorf("safeJoin escaped base: rel=%q got=%q (clean=%q, base=%q)",
				rel, got, gotClean, base)
		}
	}
}
