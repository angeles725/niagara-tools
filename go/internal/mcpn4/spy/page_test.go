package spy

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// shxFixture is a recorded spy page (testdata, station details replaced by neutral values).
func shxFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestParseReadsTheTablesOfTheMetricsPage(t *testing.T) {
	p := Parse(shxFixture(t, "metrics.html"), 0)
	if p.Title != "metrics" || p.NotFound || len(p.Tables) != 2 || len(p.Links) != 0 || len(p.Cut) != 0 {
		t.Fatalf("page = %+v", p)
	}
	recount, capacity := p.Tables[0], p.Tables[1]
	if recount.Title != "Recount" || recount.Header != nil ||
		!reflect.DeepEqual(recount.Rows[0], []string{"recountLastRun", "never"}) ||
		!reflect.DeepEqual(recount.Rows[2], []string{"recountLastFailReason", ""}) {
		t.Errorf("recount = %+v", recount)
	}
	if capacity.Title != "Global Capacity" || !reflect.DeepEqual(capacity.Header, []string{"Type", "Limit", "Used"}) ||
		len(capacity.Rows) != 6 || !reflect.DeepEqual(capacity.Rows[2], []string{"Points", "none", "36"}) {
		t.Errorf("capacity = %+v", capacity)
	}
}

func TestParseListsChildLinksOnceAsRelativePaths(t *testing.T) {
	p := Parse(shxFixture(t, "index.html"), 0)
	if p.Title != "Index" || len(p.Links) != 19 { // 20 entries, niagaraNetwork twice
		t.Fatalf("title %q, %d links: %+v", p.Title, len(p.Links), p.Links)
	}
	if p.Links[0] != (Link{"sysInfo", "sysInfo"}) || p.Links[16] != (Link{"Firewall Rules", "Firewall Rules"}) {
		t.Errorf("links = %+v", p.Links)
	}
	if p.Text != "" {
		t.Errorf("text = %q", p.Text)
	}
}

func TestParseNamesPostActionsAndNeverCarriesTheirHiddenFields(t *testing.T) {
	body := shxFixture(t, "util.html")
	p := Parse(body, 0)
	if !reflect.DeepEqual(p.Actions, []string{"util/gc", "util/forceThreadDump"}) {
		t.Errorf("actions = %q", p.Actions)
	}
	if !reflect.DeepEqual(p.Links, []Link{{"JMX Info", "util/JMX Info"}}) {
		t.Errorf("links = %+v", p.Links)
	}
	if !strings.Contains(body, "FIXTURE-CSRF-VALUE") || strings.Contains(p.Text+strings.Join(p.Actions, ""), "FIXTURE-CSRF") ||
		strings.Contains(p.Text, "gc") {
		t.Errorf("a form field or button leaked: text %q", p.Text)
	}
}

func TestParseMarksTheNotFoundPage(t *testing.T) {
	if p := Parse(shxFixture(t, "not-found.html"), 0); !p.NotFound || p.Title != "nope" {
		t.Errorf("page = %+v", p)
	}
}

func TestParseSkipsLinksThatLeaveTheSpyRoot(t *testing.T) {
	p := Parse(`<a href='https://other.invalid/x'>out</a><a href='/ord?spy:/a|b'>chain</a>`+
		`<a href='/ord?spy:/util/gc'>gc</a><a href='/ord?spy:/ok'>ok</a>`, 0)
	if p.LinksSkipped != 3 || !reflect.DeepEqual(p.Links, []Link{{"ok", "ok"}}) {
		t.Errorf("skipped %d, links %+v", p.LinksSkipped, p.Links)
	}
}

func TestParseCapsRowsCellsLinksAndText(t *testing.T) {
	var b strings.Builder
	b.WriteString("<table><tr><td>" + strings.Repeat("x", MaxCellRunes+5) + "</td></tr><tr><td>2</td></tr><tr><td>3</td></tr></table>")
	for i := 0; i < MaxLinks+2; i++ {
		b.WriteString("<a href='/ord?spy:/p" + strings.Repeat("a", i%7) + string(rune('a'+i%26)) + string(rune('a'+i/26%26)) + "'>l</a>")
	}
	b.WriteString("<pre>" + strings.Repeat("y", MaxTextRunes+9) + "</pre>")
	p := Parse(b.String(), 2)
	if len(p.Tables[0].Rows) != 2 || len([]rune(p.Tables[0].Rows[0][0])) != MaxCellRunes || len(p.Links) != MaxLinks ||
		len([]rune(p.Text)) != MaxTextRunes {
		t.Errorf("rows %d, links %d, text %d", len(p.Tables[0].Rows), len(p.Links), len([]rune(p.Text)))
	}
	if !reflect.DeepEqual(p.Cut, []string{CutCell, CutRows, CutLinks, CutText}) {
		t.Errorf("cut = %q", p.Cut)
	}
}

func TestParseSurvivesBrokenMarkup(t *testing.T) {
	for _, body := range []string{"", "plain", "<table><tr><td>a", "<a href='x", "<!-- open", "a < b", "<script>if (a<b) {}",
		"<td>cell outside a table</td>", "</table></tr>"} {
		Parse(body, 0)
	}
	if p := Parse("<table><tr><td>a<td>b<tr><td>c</table>tail &amp; text", 0); !reflect.DeepEqual(p.Tables[0].Rows,
		[][]string{{"a", "b"}, {"c"}}) || p.Text != "tail & text" {
		t.Errorf("page = %+v", p)
	}
}
