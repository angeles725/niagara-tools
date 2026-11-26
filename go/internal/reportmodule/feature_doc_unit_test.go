package reportmodule

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseArgsFeatureDoc(t *testing.T) {
	t.Parallel()
	if got, ok := parseArgs([]string{"m", "--feature-doc", "odd/tasks/x.md"}); !ok || got.featureDoc != "odd/tasks/x.md" || got.root != "m" {
		t.Errorf("--feature-doc value: %+v, %v", got, ok)
	}
	if got, ok := parseArgs([]string{"m", "--feature-doc", ""}); !ok || got.featureDoc != "" {
		t.Errorf("an empty --feature-doc is the same as an absent one: %+v, %v", got, ok)
	}
	if _, ok := parseArgs([]string{"m", "--feature-doc"}); ok {
		t.Error("--feature-doc without its value must be a usage error")
	}
}

func TestFeatureDocGaps(t *testing.T) {
	t.Parallel()
	const tierGap, decGap, emptyGap, skipGap = "no tier line", "no `## Behavior decisions` heading", "has no row", "no `## Checks skipped` heading"
	tests := []struct {
		name, doc string
		want      []string // each a substring of the gap, in order
	}{
		{"all present", fdTier + fdDecided + fdSkipped, nil},
		{"tier at the start of the first line", "P1 additive indicator\n" + fdDecided + fdSkipped, nil},
		{"tier with a colon and bold", "Tier: **P3** (boot path)\n" + fdDecided + fdSkipped, nil},
		{"CRLF line ends", strings.ReplaceAll(fdTier+fdDecided+fdSkipped, "\n", "\r\n"), nil},
		{"case of the headings", fdTier + strings.ToUpper(fdDecided[:21]) + fdDecided[21:] + "## checks skipped\n", nil},
		{"the template's tier placeholder is no tier", "Locator: x · tier `<P0|P1|P2|P3>` and the evidence\n" + fdDecided + fdSkipped, []string{tierGap}},
		{"a tier after the first heading is no tier line", "# Feature\n## Specs\ntier P2\n" + fdDecided + fdSkipped, []string{tierGap}},
		{"P4 is no tier", "tier P4\n" + fdDecided + fdSkipped, []string{tierGap}},
		{"empty row", fdTier + fdUndecide + fdSkipped, []string{emptyGap}},
		{"the header row alone is no decision", fdTier + "## Behavior decisions\n| # | Question |\n|---|---|\n" + fdSkipped, []string{emptyGap}},
		{"a row needs a cell besides its id", fdTier + "## Behavior decisions\n| # | Question |\n|---|---|\n| B1 |  |\n" + fdSkipped, []string{emptyGap}},
		{"the template's prose is no decision", fdTier + "## Behavior decisions\nWrite \"none\" only when the change has none.\n" + fdSkipped, []string{emptyGap}},
		{"a ### heading does not end the section", fdTier + "## Behavior decisions\n### B1\n| # | Q |\n|--|--|\n| B1 | save? |\n" + fdSkipped, nil},
		{"decisions heading with no body", fdTier + "## Behavior decisions\n" + fdSkipped, []string{emptyGap}},
		{"empty document", "", []string{tierGap, decGap, skipGap}},
	}
	for _, tt := range tests {
		got := featureDocGaps(tt.doc)
		if len(got) != len(tt.want) {
			t.Errorf("%s: gaps %q, want %d containing %q", tt.name, got, len(tt.want), tt.want)
			continue
		}
		for i, w := range tt.want {
			if !strings.Contains(got[i], w) {
				t.Errorf("%s: gap %d = %q, want it to contain %q", tt.name, i, got[i], w)
			}
		}
	}
}

// TestFeatureDocTemplateShape ties the checker to the shipped template: the template as copied (placeholders kept)
// lacks a tier and a decision, and once its tier placeholder and its B1 row are filled it has no gap, so a document
// started from the template passes by filling what the template asks.
func TestFeatureDocTemplateShape(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile(filepath.Join("..", "..", "..", "build-n4-module-kit", "templates", "feature-doc", "feature-doc.md"))
	must(t, err)
	raw := string(b)
	got := featureDocGaps(raw)
	if len(got) != 2 || !strings.Contains(got[0], "no tier line") || !strings.Contains(got[1], "has no row") {
		t.Errorf("the template as copied has gaps %q, want the tier and the empty decisions", got)
	}
	filled := strings.Replace(raw, "tier `<P0|P1|P2|P3>`", "tier `P2`", 1)
	filled = strings.Replace(filled, "| B1 | | | | | |", "| B1 | Save on stop? | yes / no | yes | user | 2026-11-25 |", 1)
	if gaps := featureDocGaps(filled); !reflect.DeepEqual(gaps, []string(nil)) {
		t.Errorf("the filled template has gaps %q", gaps)
	}
}
