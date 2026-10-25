package box

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/golden"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

const readerNodes = `{"": {"t": "a:B"}, "out": {"t": "baja:StatusNumeric"}, "out/status": {"v": "40"},
	"sw": {"t": "baja:StatusBoolean"}, "sw/value": {"v": "true"}, "str": {"t": "baja:StatusString"},
	"out/x/y": {}, "outer": {}}`

func TestChildrenAndStatusValue(t *testing.T) {
	nodes := mustParse(t, readerNodes).(*pyjson.Object)
	for path, want := range map[string]string{"": "out sw str outer", "out": "status", "out/x": "y", "outer": ""} {
		if got := strings.Join(Children(nodes, path).Keys(), " "); got != want {
			t.Errorf("children(%q) = %q, want %q", path, got, want)
		}
	}
	for path, want := range map[string]string{"out": "0.0 40", "sw": "true 0"} {
		v, st, err := StatusValue(nodes, path)
		if got := pyjson.Str(v) + " " + pyjson.Str(st); err != nil || got != want {
			t.Errorf("status_value(%s) = %q, %v; want %q", path, got, err, want)
		}
	}
	var ve *pyjson.ValueError
	if _, _, err := StatusValue(nodes, "str"); !errors.As(err, &ve) || ve.Msg != "str is not a StatusBoolean/StatusNumeric" {
		t.Errorf("status_value(str): %v", err)
	}
}

// TestParseStatusDifferentialPython compares ParseStatus with box.parse_status on the target
// CPython (recorded in testdata/parse-status.golden.jsonl): int(_, 16)'s whitespace, sign,
// prefix and underscore rules, Unicode digits, the 200-character repr cut of its error, and the
// falsy and non-string values.
func TestParseStatusDifferentialPython(t *testing.T) {
	g := golden.Open(t, "parse-status")
	values := []any{"0", "40", "40;units=x", "0;", ";x", "", nil, false, 0, 7, []any{}, []any{1}, "ff",
		"-40", "+0x40", "-+1", " 0x_4_0　", "0X40", "4_0", "1__0", "_1", "1_", "0x", "0x__1", "0_x1",
		"-", "  ", "٤٠", "\U0001d7d2\U0001d7ce", "\x00", "4 0", "\xed\xa0\x80", strings.Repeat("z", 300),
		"-1", "-41", "ffffffffffffffffffffffffffffff40"}
	script := `import json, sys
from mcp_n4 import box
out = []
for v in json.load(sys.stdin):
    try:
        f = box.parse_status(v)
        out.append("%s %s" % (f["ok"], f["null"]))
    except Exception as exc:
        out.append("%s: %s" % (type(exc).__name__, exc))
print(json.dumps(out))`
	in, _ := pyjson.Marshal(values, pyjson.Options{})
	var want []string
	if err := json.Unmarshal(g.Run(script, in), &want); err != nil {
		t.Fatal(err)
	}
	for i, v := range values {
		ok, null, err := ParseStatus(v)
		got := fmt.Sprintf("%s %s", pyBool(ok), pyBool(null))
		var ve *pyjson.ValueError
		var ex *pyjson.Exception
		switch {
		case errors.As(err, &ve):
			got = "ValueError: " + ve.Msg
		case errors.As(err, &ex):
			got = ex.Type + ": "
		}
		if w := want[i]; got != w && !(ex != nil && strings.HasPrefix(w, got)) {
			t.Errorf("parse_status(%s) = %q, Python %q", pyjson.Repr(v), got, w)
		}
	}
}

func pyBool(b bool) string {
	if b {
		return "True"
	}
	return "False"
}
