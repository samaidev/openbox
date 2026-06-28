package archiver

import (
	"compress/gzip"
	"io"
)

// gzipWriter is a thin alias so callers can pass a Level value without
// spreading compress/gzip imports across the package.
type gzipWriter = gzip.Writer

// newGzipWriter wraps the stdlib gzip writer with the requested level.
// Falls back to gzip.DefaultCompression for out-of-range levels.
func newGzipWriter(w io.Writer, lvl Level) *gzip.Writer {
	gw, err := gzip.NewWriterLevel(w, int(lvl))
	if err != nil {
		return gzip.NewWriter(w)
	}
	return gw
}

// newGzipReader wraps the stdlib gzip reader.
func newGzipReader(r io.Reader) (*gzip.Reader, error) {
	return gzip.NewReader(r)
}
