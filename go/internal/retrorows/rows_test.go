package retrorows

import (
	"reflect"
	"strings"
	"testing"
)

const kitRetro = `<!-- review-status: pending -->
# 2026-01-02 · kit · sample

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-01-02. ok filters use status.isOk | ` + "`go/internal/mcpn4/bql/history_audit.go`" + ` | ` + "`[ev: retro sample Δ1]`" + ` |
| Δ2 | OPEN. Every new BQL predicate carries a live-check note #812 | ` + "`go/internal/mcpn4/bql/`" + ` | ` + "`[ev: retro sample Δ2]`" + ` |
| Δ3 | pipe \| in text | toolbelt/x.sh | e |

## Not a kit delta
| Δ | Delta | Target | Token |
|---|---|---|---|
| Δ9 | ignored | x | y |

## Lessons
| 1 | a table outside a delta section | x | y |
`

func TestParseKitFormat(t *testing.T) {
	rows := Parse(kitRetro)
	if len(rows) != 3 {
		t.Fatalf("rows = %d, want 3: %+v", len(rows), rows)
	}
	if rows[0].ID != "Δ1" || rows[0].Done != "APPLIED" || rows[0].Line != 7 {
		t.Errorf("row 0 = %+v", rows[0])
	}
	r := rows[1]
	if r.ID != "Δ2" || r.Issue != 812 || r.Done != "" || strings.Contains(r.Text, "#812") {
		t.Errorf("row 1 = %+v (the trailing #N is the issue, not text)", r)
	}
	if r.Target != "`go/internal/mcpn4/bql/`" || r.Evidence != "`[ev: retro sample Δ2]`" {
		t.Errorf("row 1 columns: target %q evidence %q", r.Target, r.Evidence)
	}
	if !strings.Contains(rows[2].Text, "pipe | in text") {
		t.Errorf("escaped pipe lost: %q", rows[2].Text)
	}
}

const researchRetro = `## Proposed kit deltas (/mcp-n4)

| # | Proposed change | Target (file · section) | Evidence | Type | Priority |
|---|---|---|---|---|---|
| M1 | **Read the limit from the feature.** ` + "`n4_license_info.usage`" + ` takes it | tool | the page | fix | P1 |
| 2 | numeric id | x | y | z | P2 |
| B3 | letter id | x | y | z | P2 |
`

func TestParseResearchFormatNormalisesIDs(t *testing.T) {
	var ids []string
	for _, r := range Parse(researchRetro) {
		ids = append(ids, r.ID)
	}
	if want := []string{"ΔM1", "Δ2", "ΔB3"}; !reflect.DeepEqual(ids, want) {
		t.Errorf("ids = %v, want %v", ids, want)
	}
	if r := Parse(researchRetro)[0]; r.Evidence != "the page" || r.Heading != "Proposed kit deltas (/mcp-n4)" {
		t.Errorf("row = %+v", r)
	}
}

func TestAreaOrder(t *testing.T) {
	cases := []struct {
		name string
		row  Row
		file string
		def  string
		want string
	}{
		{"tool name in the text", Row{Text: "n4_read_slots drops a key"}, "x.md", "", "mcp-n4"},
		{"explicit name in the heading", Row{Heading: "Proposed kit deltas (/mcp-n4)", Text: "something"}, "x.md", "", "mcp-n4"},
		{"kit words in the target", Row{Target: "toolbelt/x.sh", Text: "t"}, "x.md", "", "build-n4-module"},
		{"go/internal is the kit", Row{Target: "go/internal/foo", Text: "t"}, "x.md", "", "build-n4-module"},
		{"generic heading word kit is not a signal", Row{Heading: "Proposed kit deltas", Text: "t"}, "x.md", "", ""},
		{"research-sdd rows are not ours", Row{Target: "toolbelt/scan.sh", Text: "research-sdd loop"}, "x.md", "", ""},
		{"strong beats weak", Row{Target: "toolbelt/x.sh", Text: "n4_foo"}, "x.md", "", "mcp-n4"},
		{"file name", Row{Text: "t"}, "2026-01-01-mcpn4-foo.md", "", "mcp-n4"},
		{"kit target beats a name only in the evidence", Row{Target: "`toolbelt` fold audit", Text: "make the fold audit warn", Evidence: "~40 mcp-n4 deltas unfolded"}, "x.md", "", "build-n4-module"},
		{"evidence still decides when nothing else does", Row{Text: "t", Evidence: "seen in n4_read_slots"}, "x.md", "", "mcp-n4"},
		{"default for a kit retro", Row{Text: "t"}, "2026-01-01-foo.md", "build-n4-module", "build-n4-module"},
	}
	for _, c := range cases {
		if got := Area(c.row, c.file, c.def); got != c.want {
			t.Errorf("%s: Area = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestSummary(t *testing.T) {
	cases := map[string]string{
		"**Read the limit from the feature.** more words follow": "Read the limit from the feature",
		"OPEN. A BQL where clause. Second sentence":              "A BQL where clause",
		"plain text with no stop":                                "plain text with no stop",
	}
	for in, want := range cases {
		if got := Summary(in, 100); got != want {
			t.Errorf("Summary(%q) = %q, want %q", in, got, want)
		}
	}
	long := Summary(strings.Repeat("palabra ", 40), 30)
	if n := len([]rune(long)); n > 30 || !strings.HasSuffix(long, "…") {
		t.Errorf("long summary = %q (%d runes)", long, n)
	}
}

func TestStem(t *testing.T) {
	if got := Stem("/a/b/2026-11-19-mcpn4-status-isok.md"); got != "mcpn4-status-isok" {
		t.Errorf("Stem = %q", got)
	}
}

func TestAppendIssue(t *testing.T) {
	out, err := AppendIssue(kitRetro, 7, 99)
	if err != nil {
		t.Fatal(err)
	}
	rows := Parse(out)
	if rows[0].Issue != 99 || rows[0].Text != Parse(kitRetro)[0].Text {
		t.Errorf("row 0 after = %+v", rows[0])
	}
	if strings.Count(out, "\n") != strings.Count(kitRetro, "\n") {
		t.Error("the line count changed")
	}
	if again, err := AppendIssue(out, 7, 99); err != nil || again != out {
		t.Errorf("same number twice must be a no-op: %v", err)
	}
	if _, err := AppendIssue(out, 7, 100); err == nil {
		t.Error("a row that carries another issue must be refused")
	}
	if _, err := AppendIssue(kitRetro, 3, 1); err == nil {
		t.Error("a line that is not a delta row must be refused")
	}
}
