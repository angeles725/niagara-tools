package readtools_test

// A read that panics inside n4_histories_audit's worker pool is that history's failed read, as an
// ordinary failed read is: the reply lists the other histories, notes the failure by its history id,
// and the server goes on (issue #968 item 1, review finding R3).

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// panicClient is a station client whose GetOrd panics for an ORD holding needle.
type panicClient struct {
	readtools.Client
	needle string
}

func (c panicClient) GetOrd(ord string, timeout float64, maxBytes int) (string, bool, error) {
	if strings.Contains(ord, c.needle) {
		panic("boom")
	}
	return c.Client.GetOrd(ord, timeout, maxBytes)
}

// panicRig serves histories H1..H3 (recording, capped by 1000 records, flat at 0.0, 950 / 10 / 990
// records), the read of the ORD holding needle panics.
func panicRig(t *testing.T, needle string) *readtoolstest.Server {
	t.Helper()
	st := readtoolstest.NewStation()
	st.NoAbout = true
	s := readtoolstest.NewServer(st, session.Config{})
	var points, exts []string
	for i := 1; i <= 3; i++ {
		p := fmt.Sprintf("slot:/Pool/P%d", i)
		points = append(points, p)
		exts = append(exts, strings.Join([]string{p + "/hist", "history:NumericIntervalHistoryExt", "true",
			fmt.Sprintf("/FakeStation/H%d", i), "1:1000", "roll", "false:300000",
			time.Now().UTC().Add(-time.Minute).Format("2006-01-02T15:04:05.000Z")}, ","))
	}
	st.OrdRoutes = []readtoolstest.OrdRoute{
		{Needle: "control:ControlPoint", Body: "Slot Path\n" + strings.Join(points, "\n") + "\n"},
		{Needle: "history:HistoryExt", Body: "Slot Path,Type,Enabled,Id,Encode To String,Encode To String," +
			"Encode To String,Encode To String\n" + strings.Join(exts, "\n") + "\n"},
		{Needle: "?start=", Body: "min(value),max(value)\n0.0,0.0\n"},
		{Needle: "?end=", Body: "count(toString)\n600\n"},
		{Needle: "H1|bql:select count(*)", Body: "count(toString)\n950\n"},
		{Needle: "H2|bql:select count(*)", Body: "count(toString)\n10\n"},
		{Needle: "H3|bql:select count(*)", Body: "count(toString)\n990\n"}}
	// the same tools, on clients that panic: Register replaces the handlers NewServer wired
	readtools.Register(s.Server, &readtools.Tools{Ctx: s.Ctx, ServerVersion: "test",
		NewClient: func(cfg box.Config, hook func(*box.Error)) (readtools.Client, error) {
			c, err := st.Factory(cfg, hook)
			return panicClient{c, needle}, err
		}})
	s.Connect(t)
	return s
}

func auditReply(t *testing.T, s *readtoolstest.Server, kv ...any) *pyjson.Object {
	t.Helper()
	text, isErr, out := s.Call(t, "n4_histories_audit", kv...)
	if isErr {
		t.Fatalf("n4_histories_audit failed as a whole: %s", text)
	}
	return out
}

func ids(list any) (out []string) {
	for _, item := range list.([]any) {
		id, _ := item.(*pyjson.Object).Get("history_id")
		out = append(out, id.(string))
	}
	return out
}

func TestHistoriesAuditCountThatPanicsIsThatHistorysFailedRead(t *testing.T) {
	s := panicRig(t, "H2|bql:select count(*)")
	out := auditReply(t, s)
	near, _ := out.Get("near_capacity")
	if got := ids(near); !reflect.DeepEqual(got, []string{"/FakeStation/H1", "/FakeStation/H3"}) {
		t.Errorf("near_capacity = %q, want H1 and H3", got)
	}
	notes, _ := out.Get("notes")
	if got, want := fmt.Sprint(notes), "[1 record count read(s) failed, those histories are not in near_capacity; "+
		"first: /FakeStation/H2: the read panicked: boom]"; got != want {
		t.Errorf("notes = %s\nwant    %s", got, want)
	}
	reads, _ := out.Get("count_reads")
	requests, _ := out.Get("requests")
	if reads != 3 || requests != 5 {
		t.Errorf("count_reads %v requests %v, want 3 and 5 (2 tables, 3 counts)", reads, requests)
	}
	if again := auditReply(t, s); pyjson.Repr(again) != pyjson.Repr(out) {
		t.Errorf("the server did not answer the same again:\n%s\n%s", pyjson.Repr(again), pyjson.Repr(out))
	}
}

func TestHistoriesAuditFlatWindowThatPanicsIsThatHistorysFailedRead(t *testing.T) {
	s := panicRig(t, "H2?start=")
	out := auditReply(t, s, "flat", true)
	flat, _ := out.Get("flat")
	if got := ids(flat); !reflect.DeepEqual(got, []string{"/FakeStation/H1", "/FakeStation/H3"}) {
		t.Errorf("flat = %q, want H1 and H3", got)
	}
	notes, _ := out.Get("notes")
	if got, want := fmt.Sprint(notes), "[1 flat read(s) failed, those histories are not in flat; "+
		"first: /FakeStation/H2: the read panicked: boom]"; got != want {
		t.Errorf("notes = %s\nwant    %s", got, want)
	}
	// H1 and H3: one window read and one count before it each; H2's panicked read was not counted
	if reads, _ := out.Get("flat_reads"); reads != 4 {
		t.Errorf("flat_reads = %v, want 4", reads)
	}
}
