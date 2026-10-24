package readtools

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math"
	"os"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/files"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/px"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// Station files and Px views (tools_read.py:689-822): n4_read_file, n4_list_px_views and
// n4_px_summary, through the station's file servlet (GET /file/<path>) and one BQL query.

// fileOutputFile is _FILE_OUTPUT: the output_file of n4_read_file, a plain name with an extension.
var fileOutputFile = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,99}\.[A-Za-z0-9]{1,10}$`)

// fileStatus is FILE_STATUS: the typed status of a refused file read.
var fileStatus = map[int]string{403: "forbidden", 404: "not_found"}

// fetchFile is _fetch_file: the reply of one GET /file/<path> and the body read, ok false for a
// 403/404 (a typed status, never a latch). A client-side refusal fails before any request.
func (t *Tools) fetchFile(tool string, args *pyjson.Object) (out *pyjson.Object, body []byte, ok bool, err error) {
	fileOrd := strArg(args, "file_ord", "")
	path, err := files.StationPath(fileOrd)
	if err != nil {
		return nil, nil, false, toolError("%s", err.Error())
	}
	maxBytes := intArg(args, "max_bytes", files.DefaultMaxBytes)
	out = obj("file_ord", fileOrd, "path", path)
	start := t.Ctx.Clock()
	elapsed := func() int { return int(math.RoundToEven((t.Ctx.Clock() - start) * 1000)) }
	urlPath, err := files.URLPath(path)
	if err != nil {
		return nil, nil, false, err
	}
	got, err := t.Ctx.Session.Client.(Client).GetFile(urlPath, float64(intArg(args, "timeout_s", 60)), maxBytes)
	var be *box.Error
	if errors.As(err, &be) && be.Kind == box.KindHTTPStatus {
		out.Set("status", fileStatus[be.Code])
		out.Set("http_status", be.Code)
		out.Set("elapsed_ms", elapsed())
		if be.Code == 403 {
			out.Set("hint", "the station user lacks read permission on this file, or the "+
				"station refuses its type (directories, .bog)")
		}
		t.recordRead(tool, "file_ord", args, 0)
		return out, nil, false, nil
	}
	if err != nil {
		return nil, nil, false, err
	}
	var contentType, digest any
	if got.ContentType != nil {
		contentType = *got.ContentType
	}
	if !got.Truncated {
		sum := sha256.Sum256(got.Body)
		digest = hex.EncodeToString(sum[:])
	}
	for _, kv := range [][2]any{{"status", "ok"}, {"http_status", 200}, {"content_type", contentType},
		{"content_length", got.ContentLength}, {"size_bytes", len(got.Body)}, {"truncated", got.Truncated},
		{"sha256", digest}, {"elapsed_ms", elapsed()}} {
		out.Set(kv[0].(string), kv[1])
	}
	t.recordRead(tool, "file_ord", args, 1)
	return out, got.Body, true, nil
}

// ReadFile is n4_read_file: one station file as text (a text extension or content type) or as
// its size and digest, saved whole under <state-dir>/files/ when output_file names a file.
func (t *Tools) ReadFile(args *pyjson.Object) (*pyjson.Object, error) {
	path := ""
	if v, ok := args.Get("output_file"); ok {
		var err error
		if path, err = t.outputPath(v.(string), "files", fileOutputFile, "with an extension "); err != nil {
			return nil, err
		}
	}
	out, body, ok, err := t.fetchFile("n4_read_file", args)
	if err != nil || !ok {
		return out, err
	}
	ct, _ := out.Get("content_type")
	var kind *string
	if s, ok := ct.(string); ok {
		kind = &s
	}
	if p, _ := out.Get("path"); files.IsText(p.(string), kind) {
		text, _ := pyjson.DecodeUTF8(body, true)
		out.Set("text", text)
	} else {
		out.Set("binary", true)
	}
	if path == "" {
		return out, nil
	}
	if truncated, _ := out.Get("truncated"); truncated == true {
		out.Set("output_skipped", "the file was truncated at max_bytes; raise max_bytes to save it whole")
		return out, nil
	}
	if err := writePrivate(path, func(f *os.File) error { _, err := f.Write(body); return err }); err != nil {
		return nil, writeFailed(path, err)
	}
	out.Set("output_file", path)
	return out, nil
}

// pxViewQuery and pxViewColumns are PX_VIEW_QUERY and PX_VIEW_COLUMNS; pxOwnerNote is
// PX_OWNER_NOTE.
const (
	pxViewQuery = "select slotPath, displayName, pxFile, media from baja:PxView"
	pxOwnerNote = "A PxView is a property value (BStruct) of its owning component, not a " +
		"component: the station answers its slotPath as NULL, so slot_path and " +
		"owner_path are null then. Find the owner by the view's display name in " +
		"n4_navigate / n4_read_slots, or read the Px file directly."
)

var pxViewColumns = []string{"slot_path", "display_name", "px_file", "media"}

// null is _null: an empty or NULL cell is None.
func null(cell any) any {
	if cell == "" || cell == "NULL" || cell == "null" {
		return nil
	}
	return cell
}

// ListPxViews is n4_list_px_views: the baja:PxView values below root with one BQL query, each
// with its owner's ORD when the station answers a slot path.
func (t *Tools) ListPxViews(args *pyjson.Object) (*pyjson.Object, error) {
	root := strArg(args, "root", RootOrd)
	columns, rows, truncated, elapsed, err := t.query(root, pxViewQuery, intArg(args, "max_rows", bql.DefaultMaxRows),
		intArg(args, "timeout_s", 60))
	if err != nil {
		return nil, err
	}
	keyed, err := bql.ByPosition(columns, rows, pxViewColumns)
	if err != nil {
		return nil, toolError("%s", err.Error())
	}
	views := []any{}
	for _, row := range keyed {
		view := pyjson.NewObject()
		for _, k := range pxViewColumns {
			v, _ := row.Get(k)
			view.Set(k, null(v))
		}
		var owner any
		if slot, _ := view.Get("slot_path"); slot != nil {
			s := strings.TrimPrefix(slot.(string), "slot:")
			head := strings.TrimRight(s, "/")
			if i := strings.LastIndexByte(head, '/'); i >= 0 {
				head = head[:i]
			} else {
				head = ""
			}
			if head == "" {
				head = "/"
			}
			owner = bql.StationSlot + head
		}
		view.Set("owner_path", owner)
		views = append(views, view)
	}
	t.recordRead("n4_list_px_views", "root", args, len(views))
	return obj("root", root, "query", pxViewQuery, "views", views, "row_count", len(views),
		"truncated", truncated, "elapsed_ms", elapsed, "owner_note", pxOwnerNote), nil
}

// PxSummary is n4_px_summary: one .px file fetched like n4_read_file and summarized by
// parse_px; a file cut at max_bytes is refused, never summarized in part.
func (t *Tools) PxSummary(args *pyjson.Object) (*pyjson.Object, error) {
	out, body, ok, err := t.fetchFile("n4_px_summary", args)
	if err != nil || !ok {
		return out, err
	}
	path, _ := out.Get("path")
	if truncated, _ := out.Get("truncated"); truncated == true {
		return nil, toolError("%s is larger than max_bytes=%d: raise max_bytes (up to %d) to summarize it",
			path, len(body), files.MaxBytesLimit)
	}
	text, _ := pyjson.DecodeUTF8(body, true)
	summary, err := px.Parse(text, intArg(args, "max_items", files.DefaultMaxItems))
	var ve *pyjson.ValueError
	if errors.As(err, &ve) {
		return nil, toolError("%s: %s", path, ve.Msg)
	}
	if err != nil {
		return nil, err
	}
	out.Set("summary", summary)
	return out, nil
}
