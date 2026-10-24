package ord

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pytarget"
)

const helperScript = `import json, sys
from mcp_n4 import box, safety
cases = json.load(sys.stdin)
out = []
for prefixes in cases["scopes"]:
    try:
        scope = safety.WriteScope(prefixes)
    except safety.SafetyError as exc:
        out.append("refused: %s" % exc)
        continue
    for target in cases["ords"]:
        try:
            scope.check(target)
            out.append("ok")
        except safety.SafetyError as exc:
            out.append("refused: %s" % exc)
    try:
        scope.require_any()
        out.append("ok")
    except safety.SafetyError as exc:
        out.append("refused: %s" % exc)
for parent in cases["parents"]:
    for name in cases["names"]:
        for fn in (box.child_ord, box.join_ord):
            try:
                out.append(fn(parent, name))
            except ValueError as exc:
                out.append("error: %s" % exc)
print(json.dumps(out))`

// ords is the ORD table: boundaries, trailing and doubled slashes, '..' in and around
// segments, chained hops, other schemes, case, Unicode and surrogates, and non-strings.
var ords = []any{
	"station:|slot:/", "station:|slot:", "station:|slot:/A", "station:|slot:/A/", "station:|slot:/A//", "station:|slot:/AB",
	"station:|slot:/A/B", "station:|slot:/B/A", "station:|slot:/a", "station:|slot:/A/..", "station:|slot:/A/../B",
	"station:|slot:/../A", "station:|slot:/A/..x", "station:|slot:/A/x..", "station:|slot:/A/...", "station:|slot:/A|h:1f",
	"station:|slot:/A|", "station:|h:1f", "station:|slot:/A/B|bql:select *", "local:|foo", "/A/B", "", "Station:|slot:/A",
	"station:|slot:A", " station:|slot:/A", "station:|slot:/A ", "station:|slot:/Été", "station:|slot:/A/\xed\xa0\x80",
	"station:|slot:/Été/x", "station:|slot:/A\\B", "station:|slot:/A/./B", nil, 5, true, []any{"station:|slot:/A"},
	pyjson.FloatNumber(1.5),
}

var scopes = [][]string{
	{}, {"station:|slot:/"}, {"station:|slot:/A"}, {"station:|slot:/A/"}, {"station:|slot:/A//"}, {"station:|slot:/A", "station:|slot:/B"},
	{"station:|slot:/Été"}, {"station:|slot:/A/.."}, {"station:|slot:/A|h:1"}, {"station:|h:1"}, {"slot:/A"}, {""},
	{"station:"}, {"station:|slot:/A", "station:|slot:/A/../B"}, {"station:|slot://"}, {"station:|slot:/A\xed\xa0\x80"},
}

var parents = []string{"station:", "station:|", "station:||", "station:|slot:/", "station:|slot:", "station:|slot:/A", "station:|slot:/A/",
	"station:|slot:/A//", "station:|slot://A", "local:|foo", "", "station:|slot:/A|h:1", "station:|slot:/é"}

var names = []string{"X", "", "a/b", "a|b", "..", ".", "...", "A/B", "A//B", "/A", "A/", "é", "\xed\xa0\x80", "a b", "a\\b"}

func TestScopeAndJoinDifferentialPython(t *testing.T) {
	py := pytarget.Python(t)
	kit, _ := filepath.Abs(filepath.Join("..", "..", "..", "..", "mcp-n4-kit"))
	in := pyjson.NewObject()
	var scopeList []any
	for _, s := range scopes {
		scopeList = append(scopeList, s)
	}
	in.Set("scopes", scopeList)
	in.Set("ords", ords)
	in.Set("parents", parents)
	in.Set("names", names)
	data, err := pyjson.Marshal(in, pyjson.Options{})
	if err != nil {
		t.Fatal(err)
	}
	// Go sees what Python sees: the table after one JSON round trip (an int is a Number).
	loaded, _ := pyjson.Loads(data)
	table, _ := loaded.(*pyjson.Object).Get("ords")
	cmd := exec.Command(py, "-I", "-c", "import sys; sys.path.insert(0, sys.argv[1]); exec(sys.argv[2])", kit, helperScript)
	cmd.Stdin = strings.NewReader(string(data))
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("python: %v\n%s", err, stderr.String())
	}
	var got []any
	verdict := func(err error) string {
		if err != nil {
			return "refused: " + err.Error()
		}
		return "ok"
	}
	for _, prefixes := range scopes {
		s, err := NewScope(prefixes)
		if err != nil {
			got = append(got, verdict(err))
			continue
		}
		for _, o := range table.([]any) {
			got = append(got, verdict(s.Check(o)))
		}
		got = append(got, verdict(s.RequireAny()))
	}
	for _, p := range parents {
		for _, n := range names {
			for _, fn := range []func(string, string) (string, error){ChildOrd, JoinOrd} {
				if o, err := fn(p, n); err != nil {
					got = append(got, "error: "+err.Error())
				} else {
					got = append(got, o)
				}
			}
		}
	}
	gotText, _ := pyjson.Marshal(got, pyjson.Options{})
	if want := strings.TrimSpace(string(out)); string(gotText) != want {
		t.Fatalf("Go and Python differ\ngo:     %.4000s\npython: %.4000s", gotText, want)
	}
	if len(got) < 400 {
		t.Fatalf("only %d results", len(got))
	}
}
