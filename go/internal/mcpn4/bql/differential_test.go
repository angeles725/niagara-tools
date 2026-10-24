package bql_test

import (
	"encoding/json"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pytarget"
)

// diffScript runs each case on bql.py and prints one line per case: the JSON of the result, or
// "<Python type>: <message>" (csv.Error by type only: its text is never shown).
const diffScript = `import csv, json, sys
from mcp_n4 import bql
out = []
for fn, args in json.load(sys.stdin):
    try:
        if fn == "parse_csv":
            res = list(bql.parse_csv(args[0], args[1], name_positions=args[2]))
        elif fn == "summarize_inventory":
            res = bql.summarize_inventory(*args)
        else:
            res = getattr(bql, fn)(*args)
        out.append(json.dumps(res))
    except csv.Error as exc:
        out.append(type(exc).__name__)
    except ValueError as exc:
        out.append("ValueError: %s" % exc)
print(json.dumps(out))`

// goCall is the Go side of one case, worded as diffScript words it.
func goCall(fn string, args []any) string {
	var res any
	var err error
	str := func(i int) string { s, _ := args[i].(string); return s }
	rows := func(v any) []*pyjson.Object {
		var out []*pyjson.Object
		for _, r := range v.([]any) {
			out = append(out, r.(*pyjson.Object))
		}
		return out
	}
	switch fn {
	case "validate_query":
		res, err = bql.ValidateQuery(args[0])
	case "base_slot":
		res, err = bql.BaseSlot(args[0])
	case "compose_ord":
		res, err = bql.ComposeOrd(args[0], args[1])
	case "selected_slots":
		if s := bql.SelectedSlots(str(0)); s != nil {
			res = s
		}
	case "unescape_slot":
		res = bql.UnescapeSlot(str(0))
	case "complete_records":
		res = bql.CompleteRecords(str(0))
	case "parse_csv":
		var names []int
		if list, ok := args[2].([]any); ok {
			names = []int{}
			for _, n := range list {
				i, _ := n.(pyjson.Number).BigInt()
				names = append(names, int(i.Int64()))
			}
		}
		n, _ := args[1].(pyjson.Number).BigInt()
		cols, rs, cut, e := bql.ParseCSV(str(0), int(n.Int64()), names)
		list := []any{}
		for _, r := range rs {
			list = append(list, r)
		}
		res, err = []any{cols, list, cut}, e
	case "summarize_inventory":
		res, err = bql.SummarizeInventory(args[0], rows(args[1]), rows(args[2]), rows(args[3]))
	}
	var ve *pyjson.ValueError
	var ex *pyjson.Exception
	switch {
	case errorsAs(err, &ve):
		return "ValueError: " + ve.Msg
	case errorsAs(err, &ex):
		return ex.Type
	case err != nil:
		return "unexpected " + err.Error()
	}
	b, e := pyjson.Marshal(res, pyjson.Options{})
	if e != nil {
		return "marshal " + e.Error()
	}
	return string(b)
}

// TestHelpersDifferentialPython runs the bql helpers on the same inputs in bql.py on the target
// CPython and in Go and requires the same JSON or error text: csv.reader's quirks (a quote inside
// an unquoted cell, text after a closing quote, \r-only lines, a cell spanning lines, data ending
// inside quotes, the field size limit), str.strip()'s and \s's Unicode spaces, re.I's long s,
// escapes that decode to a control character or a lone surrogate, and the inventory walk.
func TestHelpersDifferentialPython(t *testing.T) {
	py := pytarget.Python(t)
	kit, _ := filepath.Abs(filepath.Join("..", "..", "..", "..", "mcp-n4-kit"))
	long := strings.Repeat("x", bql.FieldLimit)
	csvs := []string{"A,B\nx,y\n", "A,B\r\nx,\"y\"z\r\n", "A,B\rx,y\r", "A\n\n\nx\n", "A,B\nx,\"a\nb\"\n", "A,B\nx,\"open",
		"A,B\nx\"q,y\n", "A,B\nx,\"\"\n", "\"\"\n", "A,A,A#2\n1,2,3\n", " A ,\x1fB\u0085\nx,y\n", "A\n" + long + "\n", "A\n" + long + "x\n",
		"A,B\nx,\"" + long + "\"\"\"\n", "A\r\n\r\nx\r\n\ry\n", "A\n\"x\"\"\n", "Name,B\nM$20e$u00e9,$ud800\n", "Name\n$0a$7f\n",
		"\ufeff\ufeffA\nx\n", "A\nx\ny\nz\n", "A,B\n,\n", "A\n\x00\n", "A,B\n\"x\" ,y\n", "A\n\"\"\"\"\n", "A\n\"a\"b\"c\"\n"}
	var cases []any
	for _, c := range csvs {
		cases = append(cases, []any{"parse_csv", []any{c, 2, nil}}, []any{"parse_csv", []any{c, 5000, []any{1}}},
			[]any{"complete_records", []any{c}})
	}
	for _, q := range []any{"select a from b", " \x1cselect a from b\x1f", "\u00a0select\u3000a\u2028from\u205fb\u0085", "ſelect a from b",
		"SELECT A FROM B WHERE x = 'y'", "select from b", "select a from", "select a frombx", "select\ta\nfrom b", "select a from b c",
		"select " + strings.Repeat("a", 1990) + " from b", "select " + strings.Repeat("é", 1990) + " from b", "select a from b|c",
		"select a from b\ufeff", 5, nil, "select a,b , c from d", "select a, ,b from c", "select *,a from b", "select\u00a0x\u00a0from\u00a0y"} {
		cases = append(cases, []any{"validate_query", []any{q}})
		if s, ok := q.(string); ok {
			cases = append(cases, []any{"selected_slots", []any{s}})
		}
	}
	for _, b := range []any{"/", "station:|slot:/", "slot:/A/", "/A//B", "A", "station:|slot:/A|b", "/A\x7f", "/A\ufeff", "", 3, "station:|slot:", "///"} {
		cases = append(cases, []any{"base_slot", []any{b}}, []any{"compose_ord", []any{b, "select a from b"}})
	}
	for _, s := range []string{"$20$2D$u00E9$u0041", "$ud83d$ude00", "$u12", "$$20", "$2", "$uZZZZ$41"} {
		cases = append(cases, []any{"unescape_slot", []any{s}})
	}
	row := func(path, name, kind string) any {
		o := pyjson.NewObject()
		o.Set("Slot Path", path)
		o.Set("Name", name)
		o.Set("Type", kind)
		return o
	}
	nets := []any{row("slot:/Drivers/N$201", "N 1", "a:Net"), row("slot:/Other/X", "X", "a:Net"), row("slot:/Drivers", "D", "x")}
	devs := []any{row("slot:/Drivers/N$201/D", "D", "a:Dev"), row("slot:/Drivers/N$201/D/Sub", "Sub", "a:Dev"),
		row("slot:/Drivers/M/localDevice", "localDevice", "a:Loc"), row("slot:/Drivers/N$201/E", "E", "a:Dev"), row("", "", "")}
	pts := []any{row("slot:/Drivers/N$201/D/Sub/p", "p", "c:P"), row("slot:/Drivers/N$201/D/p", "p", "c:P"),
		row("slot:/Drivers/Q/p", "p", "c:P"), row("slot:/Elsewhere/p", "p", "c:P"), row("slot:/Drivers/N$201/Ex/p", "p", "c:P")}
	for _, base := range []any{"/Drivers", "station:|slot:/", "/Drivers/N$201", "bad"} {
		cases = append(cases, []any{"summarize_inventory", []any{base, nets, devs, pts}})
	}
	in, err := pyjson.Marshal(cases, pyjson.Options{})
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(py, "-I", "-c", "import sys; sys.path.insert(0, sys.argv[1]); exec(sys.argv[2])", kit, diffScript)
	cmd.Stdin = strings.NewReader(string(in))
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("python: %v\n%s", err, stderr.String())
	}
	var want []string
	if err := json.Unmarshal(out, &want); err != nil {
		t.Fatal(err)
	}
	decoded, err := pyjson.Unmarshal(in)
	if err != nil {
		t.Fatal(err)
	}
	for i, c := range decoded.([]any) {
		pair := c.([]any)
		fn := pair[0].(string)
		if got := goCall(fn, pair[1].([]any)); got != want[i] {
			t.Errorf("case %d %s %.120s:\nGo     %.300s\nPython %.300s", i, fn, pyjson.Repr(pair[1]), got, want[i])
		}
	}
}

func errorsAs(err error, target any) bool { return err != nil && errors.As(err, target) }
