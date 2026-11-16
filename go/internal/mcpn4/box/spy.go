package box

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/spy"
)

// MaxSpyBytes is GetSpy's default read cap.
const MaxSpyBytes = 2 << 20

// GetSpy reads one spy page: GET of a URL spy.URLPath built (anything else is refused before a
// request is sent), at most maxBytes (0: MaxSpyBytes). Like GetFile, a 401 is an AuthError and
// a 403 or 404 an HttpStatusError: the user may not read spy pages, no failed login.
func (c *Client) GetSpy(urlPath string, timeout float64, maxBytes int) (*File, error) {
	if !spy.IsURLPath(urlPath) {
		return nil, &pyjson.ValueError{Msg: "get_spy reads " + spy.Root + " pages only, got " + pyjson.StrRepr(urlPath)}
	}
	if maxBytes <= 0 {
		maxBytes = MaxSpyBytes
	}
	return c.getPage(urlPath, "spy", timeout, maxBytes)
}

// getPage is the GET of GetFile and GetSpy: the body up to maxBytes with its Content-Type and
// Content-Length, a 403 or 404 as an HttpStatusError on channel.
func (c *Client) getPage(urlPath, channel string, timeout float64, maxBytes int) (*File, error) {
	resp, err := c.do(c.interrupt, "GET", urlPath, nil, nil, timeout, maxBytes+1)
	if err != nil {
		if st, ok := err.(*httpStatus); ok && (st.code == 403 || st.code == 404) {
			return nil, &Error{Kind: KindHTTPStatus, Msg: fmt.Sprintf("HTTP %d from station", st.code),
				Channel: channel, Key: "get", Code: st.code}
		}
		return nil, c.mapError(err, channel, "get", false)
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

// IsMarkup reports whether body starts like an HTML or XML page (the check GetOrd refuses a
// login page with).
func IsMarkup(body []byte) bool { return markupStart(body) }
