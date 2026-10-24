package box

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

func TestAuthMessageNamesWhatToCheck(t *testing.T) {
	if got := AuthMessage(500); got != "HTTP 500 from station" {
		t.Fatalf("%q", got)
	}
	want := "HTTP 403 from station: the user is authenticated but lacks permission on this " +
		"resource; do not retry (lockout: 5 failures in 30 s)"
	if got := AuthMessage(403); got != want {
		t.Fatalf("%q", got)
	}
}

func TestPyValueSemantics(t *testing.T) {
	one, oneF := pyjson.IntLiteral("1"), pyjson.FloatNumber(1)
	for _, tt := range []struct {
		a, b any
		eq   bool
	}{
		{one, oneF, true}, {true, one, true}, {false, pyjson.FloatNumber(0), true}, {"1", one, false},
		{nil, nil, true}, {mustParse(t, `{"a": [1, 2.0]}`), mustParse(t, `{"a": [1.0, 2]}`), true},
		{mustParse(t, `{"a": 1, "b": 2}`), mustParse(t, `{"b": 2, "a": 1}`), true},
		{pyjson.FloatNumber(nanValue()), pyjson.FloatNumber(nanValue()), false},
	} {
		if pyEqual(tt.a, tt.b) != tt.eq {
			t.Errorf("%s == %s: want %v", pyjson.Repr(tt.a), pyjson.Repr(tt.b), tt.eq)
		}
	}
	for v, want := range map[string]bool{`0`: false, `0.0`: false, `-0.0`: false, `""`: false, `[]`: false,
		`{}`: false, `null`: false, `false`: false, `"0"`: true, `[0]`: true, `NaN`: true, `2`: true} {
		if truthy(mustParse(t, v)) != want {
			t.Errorf("bool(%s) != %v", v, want)
		}
	}
}

func TestJSONKindAndNesting(t *testing.T) {
	for text, want := range map[string]string{`null`: "null", `true`: "a boolean", `1`: "a number",
		`1.5`: "a number", `"x"`: "a string", `[]`: "a list", `{}`: "an object"} {
		if got := jsonKind(mustParse(t, text)); got != want {
			t.Errorf("jsonKind(%s) = %q, want %q", text, got, want)
		}
	}
	deep := strings.Repeat("[", 3) + `{"a": [1]}` + strings.Repeat("]", 3)
	if nestsDeeper(mustParse(t, deep), 5) || !nestsDeeper(mustParse(t, deep), 4) {
		t.Fatal("a 5-level value must fit 5 and exceed 4")
	}
}

func nanValue() float64 {
	zero := 0.0
	return zero / zero
}

func mustParse(t *testing.T, s string) any {
	t.Helper()
	v, err := pyjson.Unmarshal([]byte(s))
	if err != nil {
		t.Fatalf("%s: %v", s, err)
	}
	return v
}

func mustObject(t *testing.T, s string) *pyjson.Object {
	t.Helper()
	return mustParse(t, s).(*pyjson.Object)
}
