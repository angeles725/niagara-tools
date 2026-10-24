package wiringmap

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestSlots pins the harvest the golden cases replay end to end: OPERATOR wins over SUMMARY, READONLY is a
// display slot, a slot with neither flag or no name is no row, and a multi-line annotation is joined.
func TestSlots(t *testing.T) {
	src := "@NiagaraProperty(name = \"sp\", flags = Flags.SUMMARY | Flags.OPERATOR)\n" +
		"@NiagaraProperty(\n  name = \"temp\",\n  flags = Flags.SUMMARY\n)\n" +
		"@NiagaraProperty(name = \"ro\", flags = Flags.READONLY)\n" +
		"@NiagaraProperty(name = \"hidden\", flags = Flags.HIDDEN)\n" +
		"@NiagaraProperty(type = \"double\", flags = Flags.OPERATOR)\n"
	op, sum := Slots(src)
	if !reflect.DeepEqual(op, []string{"sp"}) || !reflect.DeepEqual(sum, []string{"temp", "ro"}) {
		t.Errorf("Slots = %q, %q; want [sp], [temp ro]", op, sum)
	}
}

// TestDisplayNames: the bare slot key names the row (first line wins, blanks trimmed, "|" escaped), a
// comment or a line with no "=" is no key, and a slot with no key gets the placeholder.
func TestDisplayNames(t *testing.T) {
	lex := filepath.Join(t.TempDir(), "module.lexicon")
	if err := os.WriteFile(lex, []byte("# sp=comment\nnoequals\n sp = Set|point \nsp=second\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := rows([]string{"sp", "temp"}, readLexicon(lex))
	want := []row{{"sp", `Set\|point`}, {"temp", "_(no lexicon key)_"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("rows = %q, want %q", got, want)
	}
}

// TestRenderEmpty: a tree with no slot still gets both tables, each with its no-slots placeholder row.
func TestRenderEmpty(t *testing.T) {
	var b bytes.Buffer
	render(&b, "a/Facade-rt/src/", nil, nil)
	for _, want := range []string{"# Wiring Map — src\n", "from a/Facade-rt/src/ -->", "_(no OPERATOR slots found", "_(no SUMMARY display slots found"} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("render lacks %q", want)
		}
	}
}
