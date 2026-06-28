package archiver

import (
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// decodeZipName decodes a filename from a zip central-directory record.
//
// Per the ZIP spec (APPNOTE 6.3.x), filenames can be in any encoding.
// The de-facto standard on Windows is:
//
//   - Bit 11 of the general-purpose flag (EFS) set → UTF-8.
//   - Otherwise → CP437 for DOS-era zips, but in practice almost always
//     GBK / GB18030 on Windows in zh-CN locales.
//
// Go's stdlib archive/zip honors the EFS bit and returns UTF-8 when set,
// but it returns the raw bytes verbatim otherwise. So if the raw string
// is not valid UTF-8, we try GBK as a fallback.
//
// This is the same logic WinRAR / 7-Zip / Bandizip use; it's the only
// way to handle .zip files created by Windows Explorer in zh-CN regions.
func decodeZipName(raw string) string {
	if raw == "" {
		return raw
	}
	if utf8.ValidString(raw) {
		return raw
	}
	// Try GBK (CP936). If that fails too, return the original — losing
	// the encoding is better than a hard error.
	dec := simplifiedchinese.GBK.NewDecoder()
	out, _, err := transform.String(dec, raw)
	if err != nil {
		return raw
	}
	return out
}
