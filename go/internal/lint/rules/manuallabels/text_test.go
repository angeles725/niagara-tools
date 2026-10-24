package manuallabels

import (
	"reflect"
	"testing"
)

// Values from the python3 helper of toolbelt/lint-manual-labels.sh (CPython 3.15): its UNITS finditer
// groups over decode(body), and its LABEL finditer groups, stripped.
func TestUnits(t *testing.T) {
	for in, want := range map[string][]string{
		"a>Uno<b>>Dos<<c>Tres\nx</c>":                              {"Uno", "Dos", "Tres\nx"},
		"x = \"a\\\"b\" + 'c \"d\"' + `e\nf` + \"open\n'g'":        {`a\"b`, `c "d"`, "e\nf", "g", ` "a\"b" + 'c "d"' + ` + "`e"},
		"\n\n  \n\tk=Uno\n#c=1\n!c=2\n=c=3\n k2 = Dos \nnovalue\n": {"Uno", " Dos "},
		`k=\u00e1&amp;\\u0042`:                                     {`á&\B`},
	} {
		if got, _ := units(in); !reflect.DeepEqual(got, want) {
			t.Errorf("units(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLabels(t *testing.T) {
	for in, want := range map[string][]string{
		`the "a" b "Guardar" and "c`:                 {"b", "and"},
		`“x”y” «Modo» "123" " 1a " "http://x" "_ab"`: {"Modo", "1a", "_ab"},
	} {
		var got []string
		for _, l := range labels(in) {
			got = append(got, l.text)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("labels(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMarkdownCode(t *testing.T) {
	in := "a `x \"k\"` b ``y ` z`` c `open\n```json\n{\"v\": 1}\n  ``\nstill \"code\"\n````\n\"Guardar\"\n~~~\nsee \"Modo\" `x`\n~~~"
	want := "a   b   c `open\n\n\n\n\n\n\"Guardar\"\n~~~\nsee \"Modo\" `x`\n~~~"
	if got := markdownCode(in); got != want {
		t.Errorf("markdownCode = %q, want %q", got, want)
	}
}
