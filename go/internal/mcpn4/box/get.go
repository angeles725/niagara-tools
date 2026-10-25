package box

import (
	"fmt"
	"math/big"
	"strings"
	"unicode/utf8"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The stateless HTTP Basic reads: GET /ord/<ORD> (BQL and ORD tables), GET /file/<path> and
// GET /obix/about/. They use the same request path as Call (no redirects, 401/403 latched as
// AuthError) and need no BOX session.

// MaxOrdBytes and MaxFileBytes are get_ord's and get_file's default read caps; MaxAboutBytes is
// about()'s.
const (
	MaxOrdBytes   = 32 << 20
	MaxFileBytes  = 2 << 20
	MaxAboutBytes = 64 << 10
)

// GetOrd is BoxClient.get_ord: the text of GET /ord/<url-encoded ORD> and whether the body was
// longer than maxBytes (0: MaxOrdBytes). The text is decoded with errors="replace" and ends
// wherever the cap fell: the CSV reader drops a cut last record. An HTML answer (by its
// Content-Type or its first bytes) is refused: it is a login or error page, never a table.
// A zero timeout takes the client's.
func (c *Client) GetOrd(ord string, timeout float64, maxBytes int) (string, bool, error) {
	if maxBytes <= 0 {
		maxBytes = MaxOrdBytes
	}
	path, err := quote(ord)
	if err != nil {
		return "", false, err
	}
	resp, err := c.do(c.interrupt, "GET", "/ord/"+path, nil, map[string]string{"Accept": "text/csv"}, timeout, maxBytes+1)
	if err != nil {
		return "", false, c.mapError(err, "ord", "get", true)
	}
	ctype, _ := header(resp.header, "Content-Type")
	if strings.Contains(strings.ToLower(ctype), "html") || markupStart(resp.body) {
		if ctype == "" {
			ctype = "none"
		}
		return "", false, boxError("HTTP 200 from station with an HTML page (Content-Type "+ctype+
			"), not a CSV table: a login or error page holds no rows", "ord", "get")
	}
	truncated := len(resp.body) > maxBytes
	if truncated {
		resp.body = resp.body[:maxBytes]
	}
	text, _ := pyjson.DecodeUTF8(resp.body, true)
	return text, truncated, nil
}

// File is get_file's result. ContentType is nil when the header is absent; ContentLength is
// int(Content-Length) as a pyjson int, nil when absent or not an int.
type File struct {
	Body          []byte
	Truncated     bool
	ContentType   *string
	ContentLength any
}

// GetFile is BoxClient.get_file: GET of a /file/ path, at most maxBytes (0: MaxFileBytes). A
// 401 is an AuthError; a 403 or 404 an HttpStatusError (denied or absent, no failed login).
func (c *Client) GetFile(urlPath string, timeout float64, maxBytes int) (*File, error) {
	if !strings.HasPrefix(urlPath, "/file/") {
		return nil, &pyjson.ValueError{Msg: "get_file reads /file/ paths only, got " + pyjson.StrRepr(urlPath)}
	}
	if maxBytes <= 0 {
		maxBytes = MaxFileBytes
	}
	resp, err := c.do(c.interrupt, "GET", urlPath, nil, nil, timeout, maxBytes+1)
	if err != nil {
		if st, ok := err.(*httpStatus); ok && (st.code == 403 || st.code == 404) {
			return nil, &Error{Kind: KindHTTPStatus, Msg: fmt.Sprintf("HTTP %d from station", st.code),
				Channel: "file", Key: "get", Code: st.code}
		}
		return nil, c.mapError(err, "file", "get", false)
	}
	f := &File{Body: resp.body, Truncated: len(resp.body) > maxBytes}
	if f.Truncated {
		f.Body = f.Body[:maxBytes]
	}
	if ct, ok := header(resp.header, "Content-Type"); ok {
		f.ContentType = &ct
	}
	if cl, ok := header(resp.header, "Content-Length"); ok {
		if n, ok := pyInt(cl); ok {
			f.ContentLength = pyjson.IntLiteral(n.String())
		}
	}
	return f, nil
}

// About is BoxClient.about: the text of GET /obix/about/ (at most maxBytes, 0: MaxAboutBytes).
func (c *Client) About(timeout float64, maxBytes int) (string, error) {
	if maxBytes <= 0 {
		maxBytes = MaxAboutBytes
	}
	resp, err := c.do(c.interrupt, "GET", "/obix/about/", nil, nil, timeout, maxBytes)
	if err != nil {
		return "", c.mapError(err, "obix", "about", false)
	}
	text, _ := pyjson.DecodeUTF8(resp.body, true)
	return text, nil
}

// markupStart is BoxClient._MARKUP_START on the first 256 bytes: optional whitespace, an
// optional UTF-8 BOM, whitespace, then <!doctype, <html, <head, <body or <?xml (any case).
func markupStart(b []byte) bool {
	if len(b) > 256 {
		b = b[:256]
	}
	s := strings.TrimLeft(string(b), " \t\n\r\f\v")
	s = strings.TrimPrefix(s, "\xef\xbb\xbf")
	s = strings.TrimLeft(s, " \t\n\r\f\v")
	if !strings.HasPrefix(s, "<") {
		return false
	}
	low := strings.ToLower(s[1:])
	for _, tag := range []string{"!doctype", "html", "head", "body", "?xml"} {
		if strings.HasPrefix(low, tag) {
			return true
		}
	}
	return false
}

// quote is urllib.parse.quote(s, safe=""): every byte outside A-Z a-z 0-9 _.-~ as %XX of its
// UTF-8. A lone surrogate (WTF-8 from json.loads) cannot be encoded: Python's UnicodeEncodeError.
func quote(s string) (string, error) {
	if err := surrogates(s); err != nil {
		return "", err
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || strings.IndexByte("_.-~", c) >= 0 {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String(), nil
}

// surrogates fails like s.encode("utf-8") on the first run of lone surrogates in s.
func surrogates(s string) error {
	pos, start, n := 0, -1, 0
	var first string
	for i := 0; i < len(s); pos++ {
		isSur := i+2 < len(s) && s[i] == 0xed && s[i+1] >= 0xa0 && s[i+1] <= 0xbf
		if isSur {
			if start < 0 {
				start = pos
				r := rune(s[i]&0x0f)<<12 | rune(s[i+1]&0x3f)<<6 | rune(s[i+2]&0x3f)
				first = fmt.Sprintf("'\\u%04x'", r)
			}
			n++
			i += 3
			continue
		}
		if start >= 0 {
			break
		}
		_, size := utf8.DecodeRuneInString(s[i:])
		i += size
	}
	switch {
	case n == 1:
		return &pyjson.ValueError{Msg: fmt.Sprintf("'utf-8' codec can't encode character %s in position %d: "+
			"surrogates not allowed", first, start)}
	case n > 1:
		return &pyjson.ValueError{Msg: fmt.Sprintf("'utf-8' codec can't encode characters in position %d-%d: "+
			"surrogates not allowed", start, start+n-1)}
	}
	return nil
}

// pyInt is int(text) for a header value: Python whitespace around, an optional sign, ASCII
// digits with single underscores between them.
func pyInt(text string) (*big.Int, bool) {
	s := strings.TrimFunc(text, func(r rune) bool {
		return r == ' ' || r >= '\t' && r <= '\r' || r >= 0x1c && r <= 0x1f || r == 0x85 || r == 0xa0
	})
	neg := false
	if s != "" && (s[0] == '+' || s[0] == '-') {
		neg, s = s[0] == '-', s[1:]
	}
	if s == "" || s[0] == '_' || s[len(s)-1] == '_' || strings.Contains(s, "__") {
		return nil, false
	}
	n, ok := new(big.Int).SetString(strings.ReplaceAll(s, "_", ""), 10)
	if !ok || strings.ContainsAny(s, "+-") {
		return nil, false
	}
	if neg {
		n.Neg(n)
	}
	return n, true
}
