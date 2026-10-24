package citeparse

import (
	"reflect"
	"testing"
)

// brief is the part of a Pointer a tag-grammar case pins.
type brief struct {
	Line    int
	Kind    Kind
	Target  string
	Locator string
}

func briefs(ps []Pointer) []brief {
	var out []brief
	for _, p := range ps {
		out = append(out, brief{p.Line, p.Kind, p.Target, p.Locator()})
	}
	return out
}

func TestCardTags(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, text string
		want       []brief
	}{
		{"retro with a delta", "x `[ev: retro go-ci-steps Δ2]` y", []brief{{1, Retro, "go-ci-steps", "Δ2"}}},
		{"retro without a delta, .md suffix dropped", "[ev: retro 2026-09-25-outage.md]", []brief{{1, Retro, "2026-09-25-outage", ""}}},
		{"retro file with lines", "[ev: retro 2026-09-25-outage.md:5-10]", []brief{{1, Retro, "2026-09-25-outage", ":5-10"}}},
		{"two clauses are two pointers", "[ev: retro a Δ1; corpus B12 §12.3]", []brief{{1, Retro, "a", "Δ1"}, {1, Corpus, "B12", "§12.3"}}},
		{"comma-joined retros", "[ev: retro a, retro b]", []brief{{1, Retro, "a", ""}, {1, Retro, "b", ""}}},
		{"delta list", "[ev: retro a Δ1, Δ3]", []brief{{1, Retro, "a", "Δ1"}, {1, Retro, "a", "Δ3"}}},
		{"corpus section list", "[ev: corpus B1255 §1255.4, §1255.5]", []brief{{1, Corpus, "B1255", "§1255.4 §1255.5"}}},
		{"corpus slash and comma lists", "[ev: corpus B776/B789, B867 §867.2]",
			[]brief{{1, Corpus, "B776", ""}, {1, Corpus, "B789", ""}, {1, Corpus, "B867", "§867.2"}}},
		{"corpus rows, claim and rule", "[ev: corpus B1335 rows 1-4; corpus B1349 C1; corpus B9 rule 2]",
			[]brief{{1, Corpus, "B1335", "rows 1-4"}, {1, Corpus, "B1349", "C1"}, {1, Corpus, "B9", "rule 2"}}},
		{"bare block", "[ev: B1023 §ND4]", []brief{{1, Corpus, "B1023", "§ND4"}}},
		{"bare IDs", "[ev: corpus OBS-2] [ev: WIR-G3] [ev: Δ11 BLD2]",
			[]brief{{1, Bare, "OBS-2", ""}, {1, Bare, "WIR-G3", ""}, {1, Bare, "Δ11 BLD2", ""}}},
		{"unterminated tag", "a [ev: retro x\n\nnext", []brief{{1, Unparsed, "[ev: retro x", ""}}},
		{"a tag wrapped over two lines", "a `[ev: retro\n  x Δ1]` b", []brief{{1, Retro, "x", "Δ1"}}},
		{"deferred tag", "[deferred: retro mb Δ1]", []brief{{1, Retro, "mb", "Δ1"}}},
		{"placeholder is unparsed", "[ev: retro <stem> Δk]", []brief{{1, Unparsed, "retro <stem> Δk", ""}}},
		{"unknown kind is unparsed", "[ev: course Review 2 build]", []brief{{1, Unparsed, "course Review 2 build", ""}}},
		{"fenced block is skipped", "```\n[ev: retro x]\n```\n[ev: retro y]", []brief{{4, Retro, "y", ""}}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := briefs(Card(tt.text)); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got  %+v\nwant %+v", got, tt.want)
			}
		})
	}
}

func TestDeferredFlag(t *testing.T) {
	t.Parallel()
	ps := Card("[deferred: retro mb Δ1] [ev: retro mb Δ1]")
	if len(ps) != 2 || !ps[0].Deferred || ps[1].Deferred {
		t.Fatalf("got %+v", ps)
	}
}
