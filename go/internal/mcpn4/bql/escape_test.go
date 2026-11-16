package bql

import "testing"

func TestDecodeForBqlIsOnePassOfEscUtilUnescape(t *testing.T) {
	for in, want := range map[string]string{
		"plain": "plain", "a$20b": "a b", "$27": "'", "$7C$7c": "||", "$u0027x": "'x",
		"$2427":  "$27", // an escaped '$' then "27": one pass leaves the literal text $27
		"$u00e9": "é",
	} {
		if got, err := DecodeForBql(in); err != nil || got != want {
			t.Errorf("DecodeForBql(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"$", "$2", "$zz", "$u12", "$u12g4", "a$"} {
		if _, err := DecodeForBql(bad); err == nil {
			t.Errorf("DecodeForBql(%q) accepted a malformed escape", bad)
		}
	}
}
