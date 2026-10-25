package writeops

import (
	"bufio"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/golden"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

const pySorted = `import json, sys
for line in sys.stdin:
    try:
        print(json.dumps({"ok": sorted(json.loads(line), key=tuple)}))
    except Exception as e:
        print(json.dumps({"error": {"type": type(e).__name__, "message": str(e)}}))
`

// sortCases are lists of (target, target_slot, source_path) keys as _scan_outgoing sorts them:
// every pair and triple drawn from keys whose slots mix strings, numbers, bools, null and lists.
func sortCases() []string {
	slots := []string{`null`, `"in1"`, `"in10"`, `"in2"`, `1`, `1.0`, `true`, `false`, `0`, `2.5`, `-1e400`,
		`1e400`, `100000000000000000000001`, `[1]`, `[1, "a"]`, `["a"]`, `{"a": 1}`, `"é"`, `"\ud800"`}
	var keys []string
	for _, t := range []string{`"A"`, `"B"`} {
		for _, s := range slots {
			keys = append(keys, `[`+t+`, `+s+`, "p"]`)
		}
	}
	keys = append(keys, `["A", "in1", "o"]`, `["A", null, "o"]`, `["A", 1e400, "o"]`, `["A", -1e400, "o"]`,
		`["A", NaN, "o"]`, `["A", NaN, "p"]`, `["A", 1, "o"]`, `["A", "in1", "a"]`)
	var out []string
	for i, a := range keys {
		for j, b := range keys {
			out = append(out, "["+a+", "+b+"]")
			if (i+j)%7 == 0 {
				for _, c := range keys[:8] {
					out = append(out, "["+a+", "+b+", "+c+"]")
				}
			}
		}
	}
	for i := range 400 { // longer runs: ascending, descending and equal stretches
		var l []string
		for j := range 3 + i%6 {
			l = append(l, keys[(i*7+j*(i%5+1)*3)%len(keys)])
		}
		out = append(out, "["+strings.Join(l, ", ")+"]")
		var m []string // the same over string slots only: sorts that succeed
		for j := range 3 + i%9 {
			m = append(m, `["`+"AB"[(i+j)%2:(i+j)%2+1]+`", "in`+string(rune('0'+(i*j+i)%4))+`", "`+"pqo"[(i+2*j)%3:(i+2*j)%3+1]+`"]`)
		}
		out = append(out, "["+strings.Join(m, ", ")+"]")
	}
	return out
}

func TestPySortDifferentialPython(t *testing.T) {
	cases := sortCases()
	raw := golden.Open(t, "pysort").Run(pySorted, []byte(strings.Join(cases, "\n")+"\n"))
	sc := bufio.NewScanner(strings.NewReader(string(raw)))
	n, errs := 0, 0
	for ; sc.Scan(); n++ {
		v, err := pyjson.Unmarshal([]byte(cases[n]))
		if err != nil {
			t.Fatal(err)
		}
		items := v.([]any)
		got := obj("ok", items)
		if err := pySort(items, func(e any) []any { return e.([]any) }); err != nil {
			got = obj("error", obj("type", plan.PyType(err), "message", plan.PyStr(err)))
			errs++
		}
		line, _ := pyjson.Marshal(got, pyjson.Options{})
		if string(line) != sc.Text() {
			t.Errorf("case %d %s:\n go     %s\n python %s", n, cases[n], line, sc.Text())
		}
	}
	if n != len(cases) || errs == 0 || errs == n {
		t.Fatalf("python answered %d of %d cases, %d errors", n, len(cases), errs)
	}
	t.Logf("%d cases, %d TypeErrors", n, errs)
}
