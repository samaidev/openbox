package archiver

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// splitSuffixRe matches the trailing ".001", ".002", ... ".999" volume suffix.
var splitSuffixRe = regexp.MustCompile(`\.\d{3}$`)

// isSplitVolumePart reports whether name looks like the first part of a
// split-volume archive, e.g. "archive.zip.001" or "archive.7z.001".
//
// We only treat .001 as the entry point — extracting .002 directly wouldn't
// make sense because it's not the start of the archive.
func isSplitVolumePart(name string) bool {
	low := strings.ToLower(name)
	if !splitSuffixRe.MatchString(low) {
		return false
	}
	// Must end in ".001" specifically.
	return strings.HasSuffix(low, ".001")
}

// stripSplitSuffix removes the trailing ".001" from a split-volume filename,
// returning the "real" archive name (e.g. "archive.zip.001" → "archive.zip").
func stripSplitSuffix(name string) string {
	if m := splitSuffixRe.FindStringIndex(name); m != nil {
		return name[:m[0]]
	}
	return name
}

// findSplitVolumes returns the ordered list of volume parts for a given
// .001 filename. e.g. given "/path/archive.zip.001", returns
// ["/path/archive.zip.001", "/path/archive.zip.002", "/path/archive.zip.003"].
//
// Stops at the first missing part number. Always returns at least the
// input file if it exists.
func findSplitVolumes(firstPart string) ([]string, error) {
	dir := filepath.Dir(firstPart)
	base := filepath.Base(firstPart)
	if !splitSuffixRe.MatchString(base) {
		return nil, fmt.Errorf("not a split-volume filename: %s", base)
	}
	// Strip the .001 suffix to get the "stem" (e.g. "archive.zip").
	stem := base[:len(base)-4]
	// Find all sibling files matching stem + .NNN.
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var parts []string
	for _, e := range entries {
		n := e.Name()
		if !strings.HasPrefix(n, stem+".") {
			continue
		}
		suffix := n[len(stem)+1:]
		if !regexp.MustCompile(`^\d{3}$`).MatchString(suffix) {
			continue
		}
		parts = append(parts, filepath.Join(dir, n))
	}
	// Sort by numeric suffix so .001 comes before .002 etc.
	sort.Slice(parts, func(i, j int) bool {
		ai, _ := strconv.Atoi(parts[i][len(parts[i])-3:])
		aj, _ := strconv.Atoi(parts[j][len(parts[j])-3:])
		return ai < aj
	})
	if len(parts) == 0 {
		return nil, fmt.Errorf("no volume parts found for %s", firstPart)
	}
	// Sanity check: parts must start at .001 and be contiguous.
	for i, p := range parts {
		want := fmt.Sprintf("%s.%03d", stem, i+1)
		if filepath.Base(p) != want {
			return nil, fmt.Errorf("volume parts are not contiguous: expected %s, got %s", want, filepath.Base(p))
		}
	}
	return parts, nil
}

// writeZipSplit creates a split-volume zip archive by first writing a normal
// single-file zip to a temp location, then slicing it into N parts of
// opts.VolumeSize bytes each.
//
// The output files are named like:
//
//	archive.zip.001
//	archive.zip.002
//	...
//
// This matches the .zip.001 / .zip.002 convention used by the `zip --split-size`
// command on Unix and by 7-Zip when splitting a .zip.
func writeZipSplit(files []fileEntry, dest string, opts Options, p *Progress) error {
	// dest is "archive.zip" (no .001 suffix). We write the whole zip to a
	// temp file first, then split.
	tmpZip := dest + ".tmp.single"
	if err := writeZip(files, tmpZip, opts, p); err != nil {
		return fmt.Errorf("write temp zip: %w", err)
	}
	defer os.Remove(tmpZip)

	in, err := os.Open(tmpZip)
	if err != nil {
		return err
	}
	defer in.Close()

	fi, err := in.Stat()
	if err != nil {
		return err
	}
	totalSize := fi.Size()
	volSize := opts.VolumeSize
	if volSize < 1 {
		return fmt.Errorf("volume size must be at least 1 byte")
	}
	// Calculate how many parts we'll write.
	numParts := totalSize / volSize
	if totalSize%volSize != 0 {
		numParts++
	}

	// Remove any pre-existing parts so we don't leave stale .002 etc.
	for i := 1; i <= 999; i++ {
		partPath := fmt.Sprintf("%s.%03d", dest, i)
		if _, err := os.Stat(partPath); err == nil {
			_ = os.Remove(partPath)
		}
	}

	buf := make([]byte, 32*1024)
	remaining := totalSize
	partNum := 1
	for remaining > 0 {
		partPath := fmt.Sprintf("%s.%03d", dest, partNum)
		out, err := os.Create(partPath)
		if err != nil {
			return fmt.Errorf("create part %d: %w", partNum, err)
		}
		thisPart := volSize
		if thisPart > remaining {
			thisPart = remaining
		}
		written := int64(0)
		for written < thisPart {
			toRead := int64(len(buf))
			if toRead > thisPart-written {
				toRead = thisPart - written
			}
			n, err := in.Read(buf[:toRead])
			if n > 0 {
				if _, werr := out.Write(buf[:n]); werr != nil {
					out.Close()
					return fmt.Errorf("write part %d: %w", partNum, werr)
				}
				written += int64(n)
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				out.Close()
				return fmt.Errorf("read temp zip: %w", err)
			}
		}
		out.Close()
		remaining -= written
		partNum++
	}
	return nil
}

// extractZipSplit extracts a split-volume zip archive.
//
// src is the .001 part. We concatenate all parts (.001, .002, ...) into a
// temporary single zip file, then call the normal extractZip on it.
//
// We use a temp file rather than a streaming reader because Go's archive/zip
// requires random access to the central directory (which lives at the end of
// the zip, i.e. in the last volume).
func extractZipSplit(src, dest string, p *Progress) error {
	tmpFile, err := mergeSplitVolumes(src, p)
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile)
	return extractZip(tmpFile, dest, p)
}

// listZipSplit lists the contents of a split-volume zip archive by merging
// the parts into a temp file and calling listZip on it.
func listZipSplit(src string) ([]Entry, error) {
	tmpFile, err := mergeSplitVolumes(src, nil)
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFile)
	return listZip(tmpFile)
}

// mergeSplitVolumes concatenates all .001 / .002 / ... parts into a single
// temp file and returns its path. The caller is responsible for deleting it.
func mergeSplitVolumes(src string, p *Progress) (string, error) {
	parts, err := findSplitVolumes(src)
	if err != nil {
		return "", fmt.Errorf("find split volumes: %w", err)
	}
	tmpFile := filepath.Join(filepath.Dir(src), ".openbox-merge-"+strconv.Itoa(os.Getpid())+".zip")
	out, err := os.Create(tmpFile)
	if err != nil {
		return "", fmt.Errorf("create temp merge file: %w", err)
	}
	defer out.Close()

	totalParts := len(parts)
	for i, part := range parts {
		reportProgress(p, i, totalParts, filepath.Base(part))
		in, err := os.Open(part)
		if err != nil {
			return tmpFile, fmt.Errorf("open %s: %w", part, err)
		}
		_, err = io.Copy(out, in)
		in.Close()
		if err != nil {
			return tmpFile, fmt.Errorf("copy %s: %w", part, err)
		}
	}
	reportProgress(p, totalParts, totalParts, "merged")
	if err := out.Close(); err != nil {
		return tmpFile, err
	}
	return tmpFile, nil
}
