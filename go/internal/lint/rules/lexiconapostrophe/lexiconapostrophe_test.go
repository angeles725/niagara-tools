package lexiconapostrophe

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: one hand-written case per testdata/*.txt. LX1 a lexicon value with a MessageFormat argument ({0},
// {1,number}) and a single apostrophe; a doubled pair is clean, an apostrophe in a value without arguments is clean (read with
// getText(key) it is not formatted), comments, a continuation line and a key-only line; --strict; a root with no
// lexicon is clean; usage.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 4) }

// TestLoneApostrophe: the quote reading behind LX1. An unclosed apostrophe is lone; a closed quote is deliberate only
// when its blank-free text holds a brace ('{0}'), so a closed quote with no brace or with a blank in it is lone too
// (the closed-quote true branch); a doubled pair is never lone, and the walk goes on after a deliberate quote.
func TestLoneApostrophe(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  bool
	}{
		{"Room {0} can't reach setpoint", true},
		{"Room {0} can''t reach setpoint", false},
		{"Value '{0}' {1} no", false},
		// Mutation: LX-closed-quote -- treating every closed quote as deliberate passes 'OK', '{ 0}' and '{0} x'
		{"Press 'OK' for {0}", true},
		{"Value '{ 0}' {1}", true},
		{"Value '{0} x' {1}", true},
		{"'{0}' and '{1}' stay literal for {2}", false},
		{"'{0}' then it's {1}", true},
	} {
		if got := loneApostrophe(tc.value); got != tc.want {
			t.Errorf("loneApostrophe(%q) = %v, want %v", tc.value, got, tc.want)
		}
	}
}
