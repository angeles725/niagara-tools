package ordescape

import "testing"

func TestUnescapeDecodesOnePassLikeEscUtil(t *testing.T) {
	for in, want := range map[string]string{
		"plain": "plain", "a$20b": "a b", "$27": "'", "$7C$7c": "||", "$u0027x": "'x",
		"$2427":  "$27", // an escaped '$' then "27": one pass leaves the literal text $27
		"$u00e9": "é",
	} {
		if got, err := Unescape(in); err != nil || got != want {
			t.Errorf("Unescape(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"$", "$2", "$zz", "$u12", "$u12g4", "a$"} {
		if _, err := Unescape(bad); err == nil {
			t.Errorf("Unescape(%q) accepted a malformed escape", bad)
		}
	}
}

func TestForbiddenChecksEveryDecodedStage(t *testing.T) {
	for _, c := range []struct {
		in   string
		want rune
	}{{"x$27y", '\''}, {"x$2427", '\''}, {"a$20b", ' '}, {"a$7cb", '|'}, {"a$0ab", '\n'}, {"a$u0022", '"'}, {"a\tb", '\t'}} {
		r, ok, err := Forbidden(c.in, "'\" |")
		if err != nil || !ok || r != c.want {
			t.Errorf("Forbidden(%q) = %q %v %v, want %q", c.in, r, ok, err, c.want)
		}
	}
	if _, ok, err := Forbidden("Drivers$2dNet$41", "'\" |"); ok || err != nil {
		t.Errorf("a harmless escape was refused: %v %v", ok, err)
	}
	if _, ok, _ := Forbidden("a$20b", "|"); ok {
		t.Error("a space is allowed when forbid does not name it")
	}
}
