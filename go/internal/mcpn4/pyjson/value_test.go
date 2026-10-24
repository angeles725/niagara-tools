package pyjson

import "testing"

func TestTruthyIsPythonBool(t *testing.T) {
	t.Parallel()
	o := NewObject()
	o.Set("k", nil)
	for _, tc := range []struct {
		line string
		want bool
	}{
		{`null`, false}, {`false`, false}, {`true`, true}, {`0`, false}, {`-0`, false}, {`0.0`, false},
		{`-0.0`, false}, {`1e-400`, false}, {`7`, true}, {`0.5`, true}, {`100000000000000000000000`, true},
		{`NaN`, true}, {`""`, false}, {`"0"`, true}, {`[]`, false}, {`[0]`, true}, {`{}`, false}, {`{"a": null}`, true},
	} {
		v, err := Unmarshal([]byte(tc.line))
		if err != nil {
			t.Fatal(err)
		}
		if got := Truthy(v); got != tc.want {
			t.Errorf("Truthy(%s) = %v, want %v", tc.line, got, tc.want)
		}
	}
	if !Truthy(o) || Truthy(0) || !Truthy(2) {
		t.Error("built values")
	}
}
