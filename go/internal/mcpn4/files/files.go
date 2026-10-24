// Package files holds the pure helpers of station file reads (mcp-n4-kit/mcp_n4/files.py):
// station_path's refusals, url_path's quoting and is_text. A station serves its files on
// GET /file/<path> (web-rt FileServlet): the file ORD file:^a/b is the URL path /file/a/b.
// Nothing here touches the network.
package files

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The read caps of n4_read_file and n4_px_summary (DEFAULT_MAX_BYTES, MAX_BYTES_LIMIT,
// MIN_BYTES), the path length bound and the default bound on every Px summary list.
const (
	DefaultMaxBytes = 2 * 1024 * 1024
	MaxBytesLimit   = 16 * 1024 * 1024
	MinBytes        = 1024
	MaxPathLength   = 1024
	DefaultMaxItems = 200
)

// fileOrdPrefixes is FILE_ORD_PREFIXES, tried in order.
var fileOrdPrefixes = []string{"station:|file:^", "file:^"}

// textExtensions is TEXT_EXTENSIONS.
var textExtensions = map[string]bool{"px": true, "xml": true, "nav": true, "chart": true, "json": true,
	"csv": true, "html": true, "htm": true, "txt": true, "css": true, "js": true, "svg": true,
	"properties": true, "md": true, "log": true, "bajascript": true}

var (
	control = regexp.MustCompile(`[\x00-\x1f\x7f]`)
	drive   = regexp.MustCompile(`^[A-Za-z]:`)
)

func refuse(msg string) error { return &pyjson.ValueError{Msg: msg} }

// StationPath is station_path: the station-relative path (px/General.px) of a file ORD or a
// relative path, or the ValueError of the first rule it breaks. The rules run in Python's order,
// so a path breaking several is refused with the same words.
func StationPath(fileOrd string) (string, error) {
	if fileOrd == "" {
		return "", refuse("file_ord must be a file ORD such as file:^px/General.px or a " +
			"relative path such as px/General.px")
	}
	path, cut := fileOrd, false
	for _, prefix := range fileOrdPrefixes {
		if path, cut = strings.CutPrefix(fileOrd, prefix); cut {
			break
		}
	}
	if !cut && (strings.HasPrefix(path, "file:") || strings.HasPrefix(path, "station:")) {
		return "", refuse("only station-home file ORDs (file:^...) are read, got " + pyjson.StrRepr(fileOrd))
	}
	r := pyjson.StrRepr(path)
	segments := strings.Split(path, "/")
	switch {
	case path == "":
		return "", refuse("file_ord names no file: " + pyjson.StrRepr(fileOrd))
	case pyjson.Len(path) > MaxPathLength:
		return "", refuse("file path is longer than 1024 characters")
	case control.MatchString(path):
		return "", refuse("file path must not contain control characters")
	case strings.Contains(path, "|"):
		return "", refuse("file path must not contain '|' (no ORD hop after the file)")
	case strings.Contains(path, "\\") || drive.MatchString(path) || strings.HasPrefix(path, "/") || strings.HasPrefix(path, "~"):
		return "", refuse("file path must be relative to the station home, got " + r)
	case strings.HasSuffix(path, "/"):
		return "", refuse("file path names a directory (" + r + "): the station serves files only")
	case anySegment(segments, func(s string) bool { return s == "" || s == "." || s == ".." }):
		return "", refuse("file path must not contain empty, '.' or '..' segments: " + r)
	// Windows aliases: a station on Windows strips trailing dots and spaces from a name and
	// reads "a:b" as an NTFS stream.
	case strings.Contains(path, ":"):
		return "", refuse("file path must not contain ':' (drive or NTFS stream): " + r)
	case anySegment(segments, func(s string) bool { return s[len(s)-1] == '.' || s[len(s)-1] == ' ' }):
		return "", refuse("file path segments must not end in '.' or ' ' (a Windows station strips " +
			"them, aliasing another name): " + r)
	case strings.HasSuffix(Lower(path), ".bog") || strings.HasSuffix(Lower(path), ".bog.gz"):
		return "", refuse("the station never serves .bog / .bog.gz files (FileServlet); use the " +
			"offline config.bog route instead")
	}
	return path, nil
}

func anySegment(segments []string, f func(string) bool) bool {
	for _, s := range segments {
		if f(s) {
			return true
		}
	}
	return false
}

// URLPath is url_path: /file/<path> with each segment quoted by urllib.parse.quote(seg,
// safe=""): its UTF-8 bytes as %XX except letters, digits and "_.-~". A lone surrogate is the
// UnicodeEncodeError of its segment.
func URLPath(path string) (string, error) {
	var b strings.Builder
	b.WriteString("/file/")
	for i, seg := range strings.Split(path, "/") {
		if err := pyjson.EncodeUTF8(seg); err != nil {
			return "", err
		}
		if i > 0 {
			b.WriteByte('/')
		}
		for j := 0; j < len(seg); j++ {
			c := seg[j]
			if 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z' || '0' <= c && c <= '9' || strings.IndexByte("_.-~", c) >= 0 {
				b.WriteByte(c)
			} else {
				fmt.Fprintf(&b, "%%%02X", c)
			}
		}
	}
	return b.String(), nil
}

// IsText is is_text: the file is returned as text when its extension is a known text one or its
// content type (nil when the header is absent) is text/*, */xml, *+xml or */json.
func IsText(path string, contentType *string) bool {
	if last := path[strings.LastIndexByte(path, '/')+1:]; strings.Contains(last, ".") {
		if textExtensions[Lower(path[strings.LastIndexByte(path, '.')+1:])] {
			return true
		}
	}
	kind := ""
	if contentType != nil {
		kind, _, _ = strings.Cut(*contentType, ";")
		kind = Lower(strings.TrimFunc(kind, pyjson.IsSpace))
	}
	return strings.HasPrefix(kind, "text/") || strings.HasSuffix(kind, "+xml") ||
		strings.HasSuffix(kind, "/xml") || strings.HasSuffix(kind, "/json")
}

// Lower is str.lower() as far as a comparison with ASCII text can tell: U+0130 lowers to "i"
// and U+0307 (Unicode SpecialCasing), every other code point to its simple lowercase; bytes
// that are not UTF-8 (a WTF-8 surrogate) are kept.
func Lower(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		r, size := pyjson.DecodeRune(s[i:])
		switch {
		case r == 0x130:
			b.WriteString("i̇")
		case r >= 0xd800 && r <= 0xdfff:
			b.WriteString(s[i : i+size])
		default:
			b.WriteRune(unicode.ToLower(r))
		}
		i += size
	}
	return b.String()
}
