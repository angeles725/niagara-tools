package citeresolve

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/citeparse"
)

func fixtureRoots(t *testing.T) Roots {
	t.Helper()
	abs, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	return Roots{Kit: abs + "/kit", Repo: abs, Research: abs + "/research", Modules: abs + "/modules", Clients: abs + "/modules/Cliente",
		Memory: abs + "/memory", Version: "4.15.6.20"}
}

// statuses resolves every pointer of text as a card at kit/types/card.md.
func statuses(t *testing.T, text string) []Result {
	t.Helper()
	r := fixtureRoots(t)
	e := New(r)
	c := e.NewCard(filepath.Join(r.Kit, "types", "card.md"))
	var out []Result
	for _, p := range citeparse.Card(text) {
		out = append(out, e.Resolve(p, c))
	}
	return out
}

type want struct {
	status Status
	detail string // a substring of the detail
}

func check(t *testing.T, cases []struct {
	name, text string
	want       []want
}) {
	t.Helper()
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := statuses(t, tt.text)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d results %+v, want %d", len(got), got, len(tt.want))
			}
			for i, w := range tt.want {
				if got[i].Status != w.status || !strings.Contains(got[i].Detail, w.detail) {
					t.Errorf("result %d = %s %q, want %s containing %q", i, got[i].Status, got[i].Detail, w.status, w.detail)
				}
			}
		})
	}
}

func TestRetroAndCorpus(t *testing.T) {
	t.Parallel()
	check(t, []struct {
		name, text string
		want       []want
	}{
		{"valid retro delta", "[ev: retro go-ci-steps Δ2]", []want{{Resolved, "| Δ2 | **A generator fails closed"}}},
		{"retro without delta shows its heading", "[ev: retro go-ci-steps]", []want{{Resolved, "# 2026-10-22 · kit · go-ci-steps"}}},
		{"delta as a numbered row of a Δ table", "[ev: retro table-deltas Δ2]", []want{{Resolved, "| 2 | second"}}},
		{"delta as a numbered row under a deltas heading", "[ev: retro table-deltas Δ3]", []want{{Resolved, "| 3 | third"}}},
		{"a numbered row outside a Δ table is no delta", "[ev: retro table-deltas Δ7]", []want{{MissingDelta, ""}}},
		{"retro lacking the delta", "[ev: retro go-ci-steps Δ9]", []want{{MissingDelta, "defines no Δ9"}}},
		{"a delta cited inside the retro is not its own", "[ev: retro go-ci-steps Δ3]", []want{{MissingDelta, ""}}},
		{"missing retro", "[ev: retro nope Δ1]", []want{{MissingFile, "retros/nope.md"}}},
		{"retro lines past EOF", "[ev: retro go-ci-steps.md:5-99]", []want{{LineOutOfRange, "file has 9"}}},
		{"block section", "[ev: corpus B1255 §1255.4]", []want{{Resolved, "## §1255.4 — Deck 6"}}},
		{"two-clause tag is two pointers", "[ev: retro go-ci-steps Δ1; corpus B1255 §1255.1]", []want{{Resolved, "Δ1"}, {Resolved, "§1255.1"}}},
		{"section number without §", "[ev: corpus B1023 §1023.1]", []want{{Resolved, "## 1023.1"}}},
		{"title word is not a section", "[ev: corpus B1023 §ND4]", []want{{MissingSection, "no §ND4"}}},
		{"a number prefix is not the section", "[ev: corpus B1255 §125]", []want{{MissingSection, "no §125"}}},
		{"a deeper section resolves to its own heading", "[ev: corpus B1255 §1255.4.1]", []want{{Resolved, "deeper"}}},
		{"rows, claim, rule, gap", "[ev: corpus B1255 rows 1-2; corpus B1255 C1; corpus B1255 rule 2; corpus B1255 §B1255-G1]",
			[]want{{Resolved, "| 1 | first row"}, {Resolved, "**C1**"}, {Resolved, "The rule 2"}, {Resolved, "B1255-G1"}}},
		{"missing row", "[ev: corpus B1255 row 3]", []want{{MissingSection, "no table row 3"}}},
		{"missing block", "[ev: corpus B9999]", []want{{MissingFile, "bloque9999"}}},
		{"bare ID", "[ev: corpus OBS-2]", []want{{BareID, ""}}},
		{"unterminated tag", "[ev: retro x\n", []want{{Unparsed, "unterminated"}}},
		{"unchecked notes and deferred are kept", "[deferred: corpus B1255 — see Foo.java:3]", []want{{Resolved, "[unchecked: — see Foo.java:3; deferred]"}}},
	})
}
