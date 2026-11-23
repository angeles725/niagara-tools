package retrodup

import (
	"strings"
	"testing"
)

func TestTokensKeepIdentifiers(t *testing.T) {
	tok := Tokens("n4_read_history filter", "uses `status.isOk` and --max-rows in toolbelt/foo.sh; HistoryStatusWhere decides")
	for _, want := range []string{"n4_read_history", "--max-rows", "toolbelt/foo.sh", "historystatuswhere"} {
		if tok[want] != idWeight {
			t.Errorf("token %q weight %d, want identifier weight; tokens %v", want, tok[want], tok)
		}
	}
	if tok["decides"] != 1 || tok["filter"] != 1 {
		t.Errorf("plain words keep weight 1: %v", tok)
	}
	if _, ok := tok["source"]; ok {
		t.Error("boilerplate words must be dropped")
	}
}

func TestScore(t *testing.T) {
	a := Tokens("n4_read_slots drops a key", "the n4_read_slots filter loses the key when the slot name is long")
	same := Tokens("n4_read_slots drops a key", "the n4_read_slots filter loses the key when slot names are long")
	other := Tokens("history windows", "n4_read_history needs a window argument for histories")
	if s := Score(a, same); s < DefaultThreshold {
		t.Errorf("same change scores %.2f, want at least %.2f", s, DefaultThreshold)
	}
	if s := Score(a, other); s >= DefaultThreshold {
		t.Errorf("a different change scores %.2f, want below %.2f", s, DefaultThreshold)
	}
	if Score(nil, a) != 0 || Score(a, nil) != 0 {
		t.Error("an empty side scores 0")
	}
}

func TestTermsPreferIdentifiers(t *testing.T) {
	tok := Tokens("n4_read_slots drops a key", "the filter loses keys; see --depth")
	got := Terms(tok, 2)
	if !strings.Contains(got, "n4_read_slots") || !strings.Contains(got, "--depth") || len(strings.Fields(got)) != 2 {
		t.Errorf("Terms = %q, want the two identifiers", got)
	}
}
