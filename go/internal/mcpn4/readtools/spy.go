package readtools

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/spy"
)

// n4_spy_page: one page of the station's spy tree (GET /ord?spy:/<path>) as data. It only
// reads: the path rule (spy.CleanPath) refuses before any request what could leave the spy
// root or name an action.

// spyReader is a client that can read spy pages (*box.Client can).
type spyReader interface {
	GetSpy(urlPath string, timeout float64, maxBytes int) (*box.File, error)
}

// The typed statuses of a spy read and what the operator can do about a refused one.
const (
	spyOK               = "ok"
	spyNotFound         = "not_found"
	spyNotHTML          = "not_html"
	spyParseTimeout     = "parse_timeout"
	spyParseTimeoutHint = "the page was fetched but not parsed within timeout_s=%d; raise timeout_s or lower max_bytes"
	spyNoReader         = "this client cannot read spy pages"
	spyForbidden        = "the station user lacks the permission spy pages need (a super user reads them), or this station does not serve them at " + spy.Root
	spyNotHTMLHint      = "the station answered something that is not an HTML page; spy pages are HTML"
	cutBytes            = "max_bytes"
)

// spyFetch is one spy page read: the reply so far (path, status, http_status, elapsed_ms and,
// for a refused read, a hint) and, when status is ok, the parsed page. A 403 or 404, and a page
// not parsed within timeout seconds, is a typed status, never a latch; an unsafe path fails before any request.
func (t *Tools) spyFetch(path string, maxRows, maxBytes, timeout int) (*pyjson.Object, *spy.Page, error) {
	clean, err := spy.CleanPath(path)
	if err != nil {
		return nil, nil, toolError("%s", err.Error())
	}
	reader, ok := t.Ctx.Session.Client.(spyReader)
	if !ok {
		return nil, nil, toolError("%s", spyNoReader)
	}
	out := obj("path", clean)
	start := t.Ctx.Clock()
	got, err := reader.GetSpy(spy.URLPath(clean), float64(timeout), maxBytes)
	out.Set("elapsed_ms", int(math.RoundToEven((t.Ctx.Clock()-start)*1000)))
	var be *box.Error
	if errors.As(err, &be) && be.Kind == box.KindHTTPStatus {
		out.Set("status", fileStatus[be.Code])
		out.Set("http_status", be.Code)
		if be.Code == 403 {
			out.Set("hint", spyForbidden)
		}
		return out, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	out.Set("http_status", 200)
	out.Set("size_bytes", len(got.Body))
	if html := got.ContentType != nil && strings.Contains(strings.ToLower(*got.ContentType), "html"); !html && !box.IsMarkup(got.Body) {
		out.Set("status", spyNotHTML)
		out.Set("hint", spyNotHTMLHint)
		return out, nil, nil
	}
	text, _ := pyjson.DecodeUTF8(got.Body, true)
	// the parse gets the call's timeout of its own, so a slow page is a typed status, not a hang
	page, err := spy.Parse(text, spy.Limits{MaxRows: maxRows, Deadline: time.Now().Add(time.Duration(timeout) * time.Second)})
	if err != nil {
		out.Set("status", spyParseTimeout)
		out.Set("hint", fmt.Sprintf(spyParseTimeoutHint, timeout))
		return out, nil, nil
	}
	if got.Truncated {
		page.Cut = append([]string{cutBytes}, page.Cut...)
	}
	if page.NotFound {
		out.Set("status", spyNotFound)
		return out, nil, nil
	}
	out.Set("status", spyOK)
	return out, page, nil
}

// SpyPage is n4_spy_page: the page at path as its title, tables, child links, action names and
// text, with the caps that cut it.
func (t *Tools) SpyPage(args *pyjson.Object) (*pyjson.Object, error) {
	out, page, err := t.spyFetch(strArg(args, "path", ""), intArg(args, "max_rows", spy.DefaultMaxRows),
		intArg(args, "max_bytes", box.MaxSpyBytes), intArg(args, "timeout_s", 60))
	if err != nil {
		return nil, err
	}
	rows := 0
	if page != nil {
		rows = spyPageReply(out, page)
	}
	t.recordRead("n4_spy_page", "path", args, rows)
	return out, nil
}

// spyPageReply adds a parsed page to out and returns its row count.
func spyPageReply(out *pyjson.Object, page *spy.Page) int {
	tables, rows := []any{}, 0
	for _, tb := range page.Tables {
		cells := make([]any, len(tb.Rows))
		for i, row := range tb.Rows {
			cells[i] = strs(row)
		}
		rows += len(tb.Rows)
		var header any
		if tb.Header != nil {
			header = strs(tb.Header)
		}
		tables = append(tables, obj("title", tb.Title, "header", header, "rows", cells))
	}
	links := make([]any, len(page.Links))
	for i, l := range page.Links {
		links[i] = obj("name", l.Name, "path", l.Path)
	}
	out.Set("title", page.Title)
	out.Set("tables", tables)
	out.Set("row_count", rows)
	out.Set("links", links)
	out.Set("actions", strs(page.Actions))
	out.Set("text", page.Text)
	out.Set("truncated", len(page.Cut) > 0)
	if len(page.Cut) > 0 {
		out.Set("truncated_by", strs(page.Cut))
	}
	if page.LinksSkipped > 0 {
		out.Set("links_skipped", page.LinksSkipped)
	}
	return rows
}
