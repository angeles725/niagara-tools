package spy

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

// shxHolds reports whether any text of the page (cells, titles, headers, links, actions, text)
// holds needle.
func shxHolds(p *Page, needle string) bool {
	return strings.Contains(fmt.Sprintf("%+v", *p), needle)
}

func TestParseNeverReturnsFormFieldContentWhereverTheFieldSits(t *testing.T) {
	for _, body := range []string{
		"<table><tr><td><form><textarea name=t>FIELDX</textarea></form></td></tr></table>",
		"<form><table><tr><td><select><option>FIELDX<option selected>FIELDX</select></td></tr></table></form>",
		"<table><tr><th><button>FIELDX</button></th></tr><tr><td><input value='FIELDX'></td></tr></table>",
		"<a href='/ord?spy:/ok'><button>FIELDX</button>ok</a><h1><textarea>FIELDX</textarea></h1><textarea>FIELDX</textarea>",
	} {
		if p := shxParse(t, body, 0); shxHolds(p, "FIELDX") {
			t.Errorf("%s\n returned a field: %+v", body, p)
		}
	}
	p := shxParse(t, "<form><table><tr><td>Level</td><td><select><option>FINE</select> of log</td></tr></table></form>", 0)
	if !reflect.DeepEqual(p.Tables[0].Rows, [][]string{{"Level", "of log"}}) {
		t.Errorf("a label beside a field was lost: %+v", p.Tables)
	}
}

func TestParseReportsAFormActionOnlyAsACleanSpyPath(t *testing.T) {
	p := shxParse(t, "<form method=post action='/ord?spy:/util/x?csrfToken=SECRETVAL'></form>"+
		"<form method=POST action='/ord?spy:/util/y#SECRETVAL'></form>"+
		"<form method=post action='/ord?spy:/util/x'></form>"+ // the same action once
		"<form method=post action='/ord?spy:/a|bql:SECRETVAL'></form>"+
		"<form method=post action='/ord?spy:/../SECRETVAL'></form>"+
		"<form method=post action='https://other.invalid/SECRETVAL'></form>"+
		"<form method=post action='/ord?spy:/util/gc'></form>", 0)
	if !reflect.DeepEqual(p.Actions, []string{"util/x", "util/y", "util/gc"}) || shxHolds(p, "SECRETVAL") {
		t.Errorf("actions = %q", p.Actions)
	}
	var b strings.Builder
	for i := 0; i < MaxActions+3; i++ {
		fmt.Fprintf(&b, "<form method=post action='/ord?spy:/a/n%d'></form>", i)
	}
	if p := shxParse(t, b.String(), 0); len(p.Actions) != MaxActions || !reflect.DeepEqual(p.Cut, []string{CutActions}) {
		t.Errorf("%d actions, cut %q", len(p.Actions), p.Cut)
	}
}

// The page shape that took 13 s: every script start tag made the reader scan the rest of the
// page again. The bound is generous; a linear reader needs milliseconds.
func TestParseTimeIsLinearInTheNumberOfScriptTags(t *testing.T) {
	body := strings.Repeat("<script></script>", 32000)
	start := time.Now()
	shxParse(t, body, 0)
	shxParse(t, strings.Repeat("<STYLE>a<b</Style>", 32000), 0)
	if took := time.Since(start); took > 2*time.Second {
		t.Errorf("parsing %d bytes of script tags took %s", len(body), took)
	}
}

func TestParseGivesUpAtItsDeadline(t *testing.T) {
	body := strings.Repeat("<table><tr><td>a</td></tr></table>", 5000)
	if p, err := Parse(body, Limits{Deadline: time.Now().Add(-time.Second)}); !errors.Is(err, ErrDeadline) || p != nil {
		t.Errorf("past deadline: page %v, err %v", p != nil, err)
	}
	if _, err := Parse(body, Limits{Deadline: time.Now().Add(time.Minute)}); err != nil {
		t.Errorf("far deadline: %v", err)
	}
}

func TestParseBoundsTablesCellsAndNames(t *testing.T) {
	p := shxParse(t, strings.Repeat("<table></table>", 139000), 1)
	if len(p.Tables) != MaxTables || !reflect.DeepEqual(p.Cut, []string{CutTables}) {
		t.Errorf("empty tables: %d kept, cut %q", len(p.Tables), p.Cut)
	}
	p = shxParse(t, "<table><tr>"+strings.Repeat("<th>", MaxRowCells+7)+"</tr><tr>"+strings.Repeat("<td>", 400000)+"</tr></table>", 0)
	if tb := p.Tables[0]; len(tb.Header) != MaxRowCells || len(tb.Rows[0]) != MaxRowCells || !reflect.DeepEqual(p.Cut, []string{CutCells}) {
		t.Errorf("cells: header %d, row %d, cut %q", len(tb.Header), len(tb.Rows[0]), p.Cut)
	}
	long := strings.Repeat("n", 1500000)
	p = shxParse(t, "<table class='spy-nav'><tr><td><a href='/ord?spy:/x'>"+long+"</a></td></tr></table>"+
		"<a href='/ord?spy:/x'>"+long+"</a>", 0)
	if len([]rune(p.Links[0].Name)) != MaxNameRunes || len([]rune(p.Title)) != MaxNameRunes || !reflect.DeepEqual(p.Cut, []string{CutName}) {
		t.Errorf("names: link %d, title %d, cut %q", len(p.Links[0].Name), len(p.Title), p.Cut)
	}
}

func TestCleanPathRefusesEveryUnicodeControlAndInvalidUTF8(t *testing.T) {
	for _, in := range []string{"a\u0085b", "a\u009bb", "\u0080", "a\u009f/x", "\xc0\xae\xc0\xae/x", "a\xffb", "\xed\xa0\x80"} {
		if got, err := CleanPath(in); err == nil || got != "" {
			t.Errorf("CleanPath(%q) = %q, %v; want a refusal", in, got, err)
		}
	}
	if got, err := CleanPath("Señales/día"); err != nil || got != "Señales/día" {
		t.Errorf("a plain non-ASCII name: %q, %v", got, err)
	}
}
