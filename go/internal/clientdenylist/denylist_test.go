package clientdenylist

import (
	"strings"
	"testing"
)

func TestParseAndMatch(t *testing.T) {
	rules, err := Parse(strings.NewReader("# comment\n\n  # indented comment\n\\bacme\\b\nzeta[0-9]+\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got := rules.Match("The ACME module"); len(got) != 1 || got[0] != 4 {
		t.Errorf("case-insensitive match = %v, want rule #4", got)
	}
	if got := rules.Match("acmeish"); len(got) != 0 {
		t.Errorf("a whole-word rule matched inside a word: %v", got)
	}
	if got := rules.Match("Zeta42"); len(got) != 1 || got[0] != 5 {
		t.Errorf("match = %v, want rule #5", got)
	}
	if got := rules.Match("nothing here"); len(got) != 0 {
		t.Errorf("match = %v", got)
	}
}

func TestParseRefusesABadPatternWithoutNamingIt(t *testing.T) {
	_, err := Parse(strings.NewReader("ok\n(unclosed-secret\n"))
	if err == nil || !strings.Contains(err.Error(), "line 2") || strings.Contains(err.Error(), "secret") {
		t.Errorf("err = %v, want a line-2 error without the pattern", err)
	}
}

func TestLoadMissingFileIsNotFound(t *testing.T) {
	rules, found, err := Load(t.TempDir() + "/none.txt")
	if err != nil || found || rules != nil {
		t.Errorf("Load = %v %v %v, want not found without error", rules, found, err)
	}
}
