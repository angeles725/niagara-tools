package readtools_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/golden"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// filesScript runs one n4_read_file, n4_px_summary or n4_list_px_views case in tools_read.py
// with a client whose get_file answers the case's file (or raises its HTTP status) and whose
// get_ord answers its BQL body, and prints the reply text and the output file's bytes and mode.
const filesScript = `import json, os, sys
from mcp_n4 import box, tools_read
case = json.load(sys.stdin)
class Client:
    def get_file(self, url_path, timeout=None, max_bytes=None):
        f = case["file"]
        if f["code"]:
            raise box.HttpStatusError(f["code"], "HTTP %d from station" % f["code"], "file", "get")
        body = f["body"].encode("latin-1")
        return {"body": body[:max_bytes], "truncated": len(body) > max_bytes,
                "content_type": f["ctype"], "content_length": len(body)}
    def get_ord(self, ord_text, timeout=None, max_bytes=None):
        return case["bql"], False
class Ctx:
    observations, write, progress_path, state_dir = [], None, None, case["state"]
    session = type("S", (), {"client": Client()})
    def clock(self):
        return 1000.0
    def scrub(self, text):
        return text
try:
    out = json.dumps(getattr(tools_read, case["tool"])(Ctx(), json.loads(case["args"])))
except (tools_read.ToolError, box.BoxError, ValueError) as exc:
    out = "error: %s" % exc
except Exception as exc:
    out = "error: internal error (%s)" % type(exc).__name__
print(json.dumps([out, bool(case["out"]) and os.path.lexists(case["out"]) and
                  [open(case["out"], "rb").read().decode("latin-1"), oct(os.stat(case["out"]).st_mode)]]))`

// TestFilesDifferentialPython runs the three file tools in tools_read.py on the target CPython
// and in Go on the same answers and state dir, and requires the same reply text and the same
// output file bytes and mode: text and binary reads, the byte cap (inside a UTF-8 sequence),
// a missing Content-Type, typed 403/404 results, every refusal class of station_path and a
// lone surrogate url_path cannot quote, output_file saved, refused and skipped, the Px summary
// and its refusals, and the Px view rows with NULL and slot-path owners.
func TestFilesDifferentialPython(t *testing.T) {
	g := golden.Open(t, "file-tools")
	view, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "mcp-n4-kit", "tests", "fixtures", "synthetic_view.px"))
	if err != nil {
		t.Fatal(err)
	}
	type file struct {
		Body  string  `json:"body"`
		Ctype *string `json:"ctype"`
		Code  int     `json:"code"`
	}
	str := func(s string) *string { return &s }
	px := file{Body: string(view), Ctype: str("text/xml")}
	big := file{Body: strings.Repeat("\xc3\xa9", 600), Ctype: str("text/plain; charset=UTF-8")}
	png := file{Body: "\x89PNG\r\n\x1a\n\x00\xff", Ctype: str("image/png")}
	rows := "Slot Path,Display Name,Px File,Media\nNULL,General,file:^px/General.px,workbench:WbPxMedia\n" +
		"slot:/Plant/AHU1/graphic,AHU,file:^px/AHU.px,m\nslot:/top/,T,,null\n/a/b,B,NULL,\nslot:,C,x,y\nslot:/,D,x,y\n"
	cases := []struct {
		tool, args string
		file       file
		bql, out   string
	}{
		{"n4_read_file", `{"file_ord": "file:^px/General.px"}`, px, "", ""},
		{"n4_read_file", `{"file_ord": "station:|file:^images/logo.png", "timeout_s": 5}`, png, "", ""},
		{"n4_read_file", `{"file_ord": "notes", "max_bytes": 1025}`, big, "", ""},
		{"n4_read_file", `{"file_ord": "notes"}`, file{Body: "a\xffb"}, "", ""},
		{"n4_read_file", `{"file_ord": "notes.txt"}`, file{Body: "a\xe2\x82b\xed\xa0\x80c\xf0\x9f\x98"}, "", ""},
		{"n4_read_file", `{"file_ord": "a.PROPERTIES"}`, file{Body: "k=v"}, "", ""},
		{"n4_read_file", `{"file_ord": "nav/NavFile.nav"}`, file{Code: 404}, "", ""},
		{"n4_read_file", `{"file_ord": "px/Secret.px"}`, file{Code: 403}, "", ""},
		{"n4_read_file", `{"file_ord": "px/floor plan#1?%.px"}`, px, "", ""},
		{"n4_read_file", `{"file_ord": "px/a\ud800.px"}`, px, "", ""},
		{"n4_read_file", `{"file_ord": "images/logo.png", "output_file": "logo.png"}`, png, "", "files/logo.png"},
		{"n4_read_file", `{"file_ord": "px/General.px", "output_file": "g.px"}`, px, "", "files/g.px"},
		{"n4_read_file", `{"file_ord": "notes", "max_bytes": 1024, "output_file": "big.txt"}`, big, "", "files/big.txt"},
		{"n4_read_file", `{"file_ord": "px/General.px", "output_file": "../escape.px"}`, px, "", ""},
		{"n4_read_file", `{"file_ord": "px/General.px", "output_file": "a..b.px"}`, px, "", ""},
		{"n4_read_file", `{"file_ord": "px/General.px", "output_file": "noext"}`, px, "", ""},
		{"n4_read_file", `{"file_ord": "../config.bog", "output_file": "x.y"}`, px, "", ""},
		{"n4_px_summary", `{"file_ord": "file:^px/General.px"}`, px, "", ""},
		{"n4_px_summary", `{"file_ord": "px/General.px", "max_items": 1}`, px, "", ""},
		{"n4_px_summary", `{"file_ord": "px/General.px", "max_bytes": 1024}`, px, "", ""},
		{"n4_px_summary", `{"file_ord": "px/Bad.px"}`, file{Body: "<html/>", Ctype: str("text/xml")}, "", ""},
		{"n4_px_summary", `{"file_ord": "px/Bad.px"}`, file{Body: "<px><a></b></px>"}, "", ""},
		{"n4_px_summary", `{"file_ord": "px/Bad.px"}`, file{Body: "<!DOCTYPE px><px/>"}, "", ""},
		{"n4_px_summary", `{"file_ord": "px/Bad.px"}`, file{Body: "\xef\xbb\xbf<px>\xff</px>"}, "", ""},
		{"n4_px_summary", `{"file_ord": "px/Nope.px"}`, file{Code: 404}, "", ""},
		{"n4_px_summary", `{"file_ord": "px/a.px:s"}`, px, "", ""},
		{"n4_list_px_views", `{}`, px, rows, ""},
		{"n4_list_px_views", `{"root": "station:|slot:/Plant", "max_rows": 1, "timeout_s": 9}`, px, rows, ""},
		{"n4_list_px_views", `{}`, px, "Slot Path,Display Name\nNULL,G\n", ""},
		{"n4_list_px_views", `{}`, px, "A,B,C,D\nx,y\n", ""},
		{"n4_list_px_views", `{"root": "/Plant|bql:select x from y"}`, px, rows, ""},
	}
	for _, bad := range []string{"", "..", "px/../config.bog", "/etc/passwd", "file:/etc/passwd", "px|slot:/", "px/",
		"config.bog", "config.bog.", "config.bog ", "config.bog::$DATA", "px/.. /config.bog", "C:/px/a.px", "~/a.px",
		"px/a\x00.px", "px/a\u2028.px", "station:|slot:/x", "file:^", strings.Repeat("x", 1025)} {
		o := pyjson.NewObject()
		o.Set("file_ord", bad)
		arg, _ := pyjson.Marshal(o, pyjson.Options{})
		cases = append(cases, struct {
			tool, args string
			file       file
			bql, out   string
		}{"n4_read_file", string(arg), px, "", ""})
	}
	root := t.TempDir()
	g.Alias(root, "$STATE_ROOT") // the cases and replies name the state dirs under it
	for i, c := range cases {
		state := filepath.Join(root, fmt.Sprint(i), "state")
		out := ""
		if c.out != "" {
			out = filepath.Join(state, c.out)
		}
		latin := make([]rune, len(c.file.Body))
		for j := 0; j < len(c.file.Body); j++ {
			latin[j] = rune(c.file.Body[j])
		}
		pf := c.file
		pf.Body = string(latin)
		in, _ := json.Marshal(map[string]any{"tool": c.tool, "args": c.args, "file": pf, "bql": c.bql, "state": state, "out": out})
		raw := g.Run(filesScript, in)
		var want []any
		if err := json.Unmarshal(raw, &want); err != nil {
			t.Fatal(err)
		}
		os.RemoveAll(state)
		s := readtoolstest.NewServer(readtoolstest.NewStation(), session.Config{StateDir: state})
		s.Connect(t)
		f := readtoolstest.File{Body: c.file.Body, Code: c.file.Code, NoType: c.file.Ctype == nil}
		if c.file.Ctype != nil {
			f.ContentType = *c.file.Ctype
		}
		s.Station.Files = map[string]readtoolstest.File{}
		s.Station.Files[strings.TrimPrefix(strings.TrimPrefix(fileOrd(t, c.args), "station:|"), "file:^")] = f
		s.Station.OrdRoutes = []readtoolstest.OrdRoute{{Needle: "", Body: c.bql}}
		args, err := pyjson.Unmarshal([]byte(c.args))
		if err != nil {
			t.Fatal(err)
		}
		var kv []any
		for _, k := range args.(*pyjson.Object).Keys() {
			v, _ := args.(*pyjson.Object).Get(k)
			kv = append(kv, k, v)
		}
		text, isErr, _ := s.Call(t, c.tool, kv...)
		if isErr {
			text = "error: " + text
		}
		var written any = false
		if _, err := os.Lstat(out); out != "" && err == nil {
			data, _ := os.ReadFile(out)
			info, _ := os.Stat(out)
			lat := make([]rune, len(data))
			for j, b := range data {
				lat[j] = rune(b)
			}
			written = []any{string(lat), fmt.Sprintf("0o%o", uint32(info.Mode().Perm())|0o100000)}
		}
		if text != want[0].(string) {
			t.Errorf("case %d %s %s:\nGo     %.500s\nPython %.500s", i, c.tool, c.args, text, want[0])
		}
		if g, w := jsonText(written), jsonText(want[1]); g != w {
			t.Errorf("case %d file:\nGo     %.300s\nPython %.300s", i, g, w)
		}
	}
}

// fileOrd is the file_ord of a case's arguments ("" when it has none or it is not a str).
func fileOrd(t *testing.T, args string) string {
	v, err := pyjson.Unmarshal([]byte(args))
	if err != nil {
		t.Fatal(err)
	}
	s, _ := v.(*pyjson.Object).Get("file_ord")
	f, _ := s.(string)
	return f
}
