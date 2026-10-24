package files

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pytarget"
)

// refusalBodies are the paths of the refusal table: every rule of station_path, the Windows
// aliases (':', trailing dot or space, '..'), drive letters, control characters, Unicode
// spaces and case mappings, and lone surrogates, each also behind both file ORD prefixes.
var refusalBodies = []string{
	"", "px/General.px", "..", ".", "a/./b", "a//b", "a/", "/a", "~a", "~/a", "a\\b", "C:a", "c:/a", "1:a", "é:a", "a:b",
	"config.bog", "CONFIG.BOG", "a.Bog.Gz", "a.bog.gz.", "a.bog ", "a.bog. .", "a.bog::$DATA", "a.bog\u00a0", "a.bog\u3000",
	"px/.. /a", "px/... /a", "px/ /a", "px/a.px:s", "a.", "a ", "a\t", "a\x00", "a\x1f", "a\x7f", "a\u0085", "a\u2028",
	"a|b", "|", "a\\", "file:x", "station:x", "station:|slot:/x", "file:", "station:", "file:^", "station:|file:^",
	"px/floor plan#1?.px", "px/%41~_-.x", "px/日本.px", "px/\U0001F600.px", "px/İ.bog", "px/a.propertİes", "px/a.PROPERTIES",
	"px/a\xed\xa0\x80.px", "px/\xed\xb0\x80\xed\xa0\x80/b\xed\xa0\x80", "x/" + strings.Repeat("é", 1022), "x/" + strings.Repeat("é", 1023),
	strings.Repeat("a", 1025), "a/b.bog/c", "nav/NavFile.nav", "px/a.PX", "images/a.png", "v1.2/a", "a.b/c", ".hidden",
	"a.bajascript", "a.json.bak", "a.js", "a.md", "a.log", "a.htm", "a.svg",
}

// contentTypes are Content-Type values for is_text (None first).
var contentTypes = []any{nil, "", "text/plain; charset=UTF-8", " TEXT/X ", "\u0085text/x", "\x1ftext/x", "image/png",
	"application/octet-stream", "application/atom+XML", "application/xml;x", "application/json ", "a/jsonx", "text",
	"Text/\u0130", "x/+xml", "application/ld+json"}

const helperScript = `import json, sys
from mcp_n4 import files
cases = json.load(sys.stdin)
out = []
for given in cases["paths"]:
    try:
        path = files.station_path(given)
        out.append("ok " + path)
        try:
            out.append(files.url_path(path))
        except ValueError as exc:
            out.append("error: %s" % exc)
    except ValueError as exc:
        out.append("error: %s" % exc)
    for kind in cases["types"]:
        out.append(str(files.is_text(given, kind)))
print(json.dumps(out))`

// TestHelpersDifferentialPython runs station_path, url_path (on each accepted path) and is_text
// (on every path with every content type) in files.py on the target CPython and in Go, and
// requires the same results and the same refusal words.
func TestHelpersDifferentialPython(t *testing.T) {
	py := pytarget.Python(t)
	kit, _ := filepath.Abs(filepath.Join("..", "..", "..", "..", "mcp-n4-kit"))
	var paths []any
	for _, b := range refusalBodies {
		paths = append(paths, b, "file:^"+b, "station:|file:^"+b)
	}
	in := pyjson.NewObject()
	in.Set("paths", paths)
	in.Set("types", contentTypes)
	data, err := pyjson.Marshal(in, pyjson.Options{})
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(py, "-I", "-c", "import sys; sys.path.insert(0, sys.argv[1]); exec(sys.argv[2])", kit, helperScript)
	cmd.Stdin = strings.NewReader(string(data))
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("python: %v\n%s", err, stderr.String())
	}
	var got []any
	for _, p := range paths {
		given := p.(string)
		if path, err := StationPath(given); err != nil {
			got = append(got, "error: "+err.Error())
		} else if u, err := URLPath(path); err != nil {
			got = append(got, "ok "+path, "error: "+err.Error())
		} else {
			got = append(got, "ok "+path, u)
		}
		for _, k := range contentTypes {
			var kind *string
			if s, ok := k.(string); ok {
				kind = &s
			}
			got = append(got, map[bool]string{true: "True", false: "False"}[IsText(given, kind)])
		}
	}
	g, _ := pyjson.Marshal(got, pyjson.Options{})
	if want := strings.TrimSpace(string(out)); string(g) != want {
		gs, ws := strings.Split(string(g), `", "`), strings.Split(want, `", "`)
		for i := range min(len(gs), len(ws)) {
			if gs[i] != ws[i] {
				t.Errorf("first difference at item %d:\nGo     %s\nPython %s", i, gs[i], ws[i])
				break
			}
		}
		t.Fatalf("results differ (%d vs %d items)", len(gs), len(ws))
	}
}
