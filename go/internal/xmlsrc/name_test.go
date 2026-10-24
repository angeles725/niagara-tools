package xmlsrc

import "testing"

// Cases from niagara-go internal/bog/model_test.go TestUnescapeForms, plus the edges of each form.
func TestUnescapeName(t *testing.T) {
	for in, want := range map[string]string{
		"a$20b$3ac": "a b:c", "x$u00e9y": "xéy", "cost$": "cost$", "$u00E9$4": "é$4", "$$41": "$A",
		"plain": "plain", "": "", "$2": "$2", "$zz": "$zz", "$u00g9": "$u00g9", "$u12": "$u12",
		"in$2010": "in 10",
		// a UTF-16 surrogate pair is one character, as Java decodes it; a lone surrogate cannot be UTF-8
		"$ud83d$ude00!": "\U0001F600!", "a$ud83db": "a\uFFFDb", "$ude00$ud83d": "\uFFFD\uFFFD",
	} {
		if got := UnescapeName(in); got != want {
			t.Errorf("UnescapeName(%q) = %q, want %q", in, got, want)
		}
	}
}
