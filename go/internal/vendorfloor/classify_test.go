package vendorfloor

import (
	"strings"
	"testing"
)

const guarded = " (floor 83) — feature-guarded? record it in THIRD-PARTY.md"

// rows classifies rec for vendor/lib.js with the shipped table and returns the printed rows.
func rows(rec record, strict bool) []string {
	var out []string
	for _, f := range classify(rec, "vendor/lib.js", Default, strict) {
		out = append(out, f.Row())
	}
	return out
}

func wantRows(t *testing.T, got []string, want ...string) {
	t.Helper()
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("rows:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestSyntaxFail(t *testing.T) {
	t.Parallel()
	rec := record{Syntax: &syntaxError{Line: 2, Message: "Unexpected token (2:4)"}}
	wantRows(t, rows(rec, false),
		"FAIL  vendor-floor  vendor/lib.js:2  syntax: does not parse at ecmaVersion 2020 (Chromium 83 floor): Unexpected token")
	// acorn gave no location: the row still names a line.
	rec = record{Syntax: &syntaxError{Message: "boom"}}
	wantRows(t, rows(rec, false),
		"FAIL  vendor-floor  vendor/lib.js:1  syntax: does not parse at ecmaVersion 2020 (Chromium 83 floor): boom")
}

func TestModuleOnlyWarn(t *testing.T) {
	t.Parallel()
	rec := record{Syntax: &syntaxError{Line: 1, Message: "'import' and 'export' may appear only with 'sourceType: module' (1:0)", Module: true}}
	wantRows(t, rows(rec, false),
		`WARN  vendor-floor  vendor/lib.js:1  syntax: parses only as an ES module ('import' and 'export' may appear only with 'sourceType: module' (1:0)) — load with type="module" or vendor the UMD/global build`)
}

func TestAPIFamilies(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		use  nameUse
		want string // the call and its Chrome version; "" for no row
	}{
		{"member call", nameUse{Name: "replaceAll", Prev: ".", Prev2: "s", Next: "(", Line: 7}, ".replaceAll( needs Chrome 85"},
		{"qualified call", nameUse{Name: "hasOwn", Prev: ".", Prev2: "Object", Next: "(", Line: 7}, "Object.hasOwn( needs Chrome 93"},
		{"global call", nameUse{Name: "structuredClone", Prev: "=", Prev2: "x", Next: "(", Line: 7}, "structuredClone( needs Chrome 98"},
		{"global constructor", nameUse{Name: "WeakRef", Prev: "new", Prev2: "=", Next: "(", Line: 7}, "new WeakRef needs Chrome 84"},
		{"member read, not a call", nameUse{Name: "at", Prev: ".", Prev2: "o", Next: ";", Line: 7}, ""},
		{"optional call", nameUse{Name: "at", Prev: "?.", Prev2: "o", Next: "(", Line: 7}, ""},
		{"bare call of a member name", nameUse{Name: "at", Prev: "=", Prev2: "x", Next: "(", Line: 7}, ""},
		{"qualified name on another owner", nameUse{Name: "hasOwn", Prev: ".", Prev2: "Foo", Next: "(", Line: 7}, ""},
		{"global call name as a method", nameUse{Name: "structuredClone", Prev: ".", Prev2: "self", Next: "(", Line: 7}, ""},
		{"constructor name without new", nameUse{Name: "WeakRef", Prev: "=", Prev2: "w", Next: ";", Line: 7}, ""},
		{"first token of the file", nameUse{Name: "hasOwn", Next: "(", Line: 7}, ""},
	}
	for _, c := range cases {
		got := rows(record{Names: []nameUse{c.use}}, false)
		if c.want == "" {
			wantRows(t, got)
			continue
		}
		wantRows(t, got, "WARN  vendor-floor  vendor/lib.js:7  api: "+c.want+guarded)
	}
}

func TestEveryTableEntryWarns(t *testing.T) {
	t.Parallel()
	var uses []nameUse
	for _, name := range sortedKeys(Default.Member) {
		uses = append(uses, nameUse{Name: name, Prev: ".", Prev2: "x", Next: "(", Line: 1})
	}
	for _, q := range sortedKeys(Default.Qualified) {
		uses = append(uses, nameUse{Name: method(q), Prev: ".", Prev2: q[:len(q)-len(method(q))-1], Next: "(", Line: 1})
	}
	for _, name := range sortedKeys(Default.GlobalCall) {
		uses = append(uses, nameUse{Name: name, Next: "(", Line: 1})
	}
	for _, name := range sortedKeys(Default.GlobalNew) {
		uses = append(uses, nameUse{Name: name, Prev: "new", Next: "(", Line: 1})
	}
	got := rows(record{Names: uses}, false)
	if len(got) != len(uses) || len(uses) != 14 {
		t.Fatalf("%d rows for %d table entries, want 14 of each:\n%s", len(got), len(uses), strings.Join(got, "\n"))
	}
}

func TestStrictPromotesWarnToFail(t *testing.T) {
	t.Parallel()
	rec := record{
		Syntax: &syntaxError{Line: 1, Message: "m (1:0)", Module: true},
		Names:  []nameUse{{Name: "at", Prev: ".", Prev2: "a", Next: "(", Line: 3}},
	}
	wantRows(t, rows(rec, true),
		`FAIL  vendor-floor  vendor/lib.js:1  syntax: parses only as an ES module (m (1:0)) — load with type="module" or vendor the UMD/global build`,
		"FAIL  vendor-floor  vendor/lib.js:3  api: .at( needs Chrome 92"+guarded)
	for _, f := range classify(rec, "vendor/lib.js", Default, false) {
		if strings.HasPrefix(f.Row(), "FAIL") {
			t.Errorf("without --strict: %s", f.Row())
		}
	}
}

func TestSyntaxRowComesFirst(t *testing.T) {
	t.Parallel()
	rec := record{
		Syntax: &syntaxError{Line: 9, Message: "Unexpected token (9:1)"},
		Names:  []nameUse{{Name: "at", Prev: ".", Prev2: "a", Next: "(", Line: 2}},
	}
	got := rows(rec, false)
	if len(got) != 2 || !strings.HasPrefix(got[0], "FAIL  vendor-floor  vendor/lib.js:9  syntax:") ||
		!strings.HasPrefix(got[1], "WARN  vendor-floor  vendor/lib.js:2  api:") {
		t.Errorf("rows:\n%s", strings.Join(got, "\n"))
	}
}
