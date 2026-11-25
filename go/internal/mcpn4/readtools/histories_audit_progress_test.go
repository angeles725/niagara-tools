package readtools_test

// n4_histories_audit writes progress lines to the operator's progress file (the channel n4_inventory
// uses) while it runs: a line per stage and one per haProgressEvery finished histories, each with the
// requests made so far and the elapsed ms (issue #906, retro R8 delta N5). The reply is not changed.

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// windowStart is the flat window's start, which follows the clock of the call.
var windowStart = regexp.MustCompile(`'window_start': '[^']*', `)

// progressRig serves n recording histories (capped by 1000 records, 10 records each, flat at 0.0)
// and writes its progress lines to the returned file.
func progressRig(t *testing.T, n int, withFile bool) (*readtoolstest.Server, string) {
	t.Helper()
	st := readtoolstest.NewStation()
	st.NoAbout = true
	s := readtoolstest.NewServer(st, session.Config{})
	var points, exts []string
	for i := 1; i <= n; i++ {
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
		{Needle: "bql:select count(*)", Body: "count(toString)\n10\n"}}
	path := ""
	if withFile {
		path = filepath.Join(t.TempDir(), "progress.jsonl")
		s.Ctx.ProgressPath = path
	}
	s.Connect(t)
	return s, path
}

// progressLines are the lines of the progress file as step, rows (requests so far).
func progressLines(t *testing.T, path string) (steps []string, rows []any) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("no progress file: %v", err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		v, err := pyjson.Unmarshal([]byte(line))
		if err != nil {
			t.Fatalf("progress line %q: %v", line, err)
		}
		o := v.(*pyjson.Object)
		if tool, _ := o.Get("tool"); tool != "n4_histories_audit" {
			t.Errorf("progress line of tool %v", tool)
		}
		if _, ok := o.Get("elapsed_ms"); !ok {
			t.Errorf("progress line without elapsed_ms: %s", line)
		}
		step, _ := o.Get("step")
		r, _ := o.Get("rows")
		steps, rows = append(steps, step.(string)), append(rows, r)
	}
	return steps, rows
}

func TestHistoriesAuditWritesAProgressLinePerStage(t *testing.T) {
	s, path := progressRig(t, 3, true)
	out := auditReply(t, s)
	steps, rows := progressLines(t, path)
	if got, want := strings.Join(steps, "|"), "extensions|points|counts"; got != want {
		t.Errorf("steps = %q, want %q", got, want)
	}
	if requests, _ := out.Get("requests"); fmt.Sprint(rows[len(rows)-1]) != fmt.Sprint(requests) || requests != 5 {
		t.Errorf("last progress rows %v, reply requests %v, want 5", rows[len(rows)-1], requests)
	}
	if fmt.Sprint(rows) != "[1 2 5]" {
		t.Errorf("rows = %v, want the requests so far [1 2 5]", rows)
	}
}

func TestHistoriesAuditWritesAProgressLineEveryTenHistories(t *testing.T) {
	s, path := progressRig(t, 12, true)
	auditReply(t, s)
	steps, rows := progressLines(t, path)
	if got, want := strings.Join(steps, "|"), "extensions|points|counts 10/12|counts"; got != want {
		t.Errorf("steps = %q, want %q", got, want)
	}
	if fmt.Sprint(rows) != "[1 2 12 14]" {
		t.Errorf("rows = %v, want [1 2 12 14]", rows)
	}
}

func TestHistoriesAuditFlatWritesAFlatProgressLine(t *testing.T) {
	s, path := progressRig(t, 3, true)
	out := auditReply(t, s, "flat", true)
	steps, rows := progressLines(t, path)
	if got, want := strings.Join(steps, "|"), "extensions|points|counts|flat"; got != want {
		t.Errorf("steps = %q, want %q", got, want)
	}
	if requests, _ := out.Get("requests"); fmt.Sprint(rows[len(rows)-1]) != fmt.Sprint(requests) || requests != 11 {
		t.Errorf("last progress rows %v, reply requests %v, want 11 (2 tables, 3 counts, 6 flat)", rows[len(rows)-1], requests)
	}
}

// The progress file is a side channel: the reply is the same with and without it, and without
// --progress-file nothing is written.
func TestHistoriesAuditReplyIsTheSameWithAndWithoutTheProgressFile(t *testing.T) {
	with, path := progressRig(t, 12, true)
	without, _ := progressRig(t, 12, false)
	a, b := auditReply(t, with, "flat", true), auditReply(t, without, "flat", true)
	if _, ok := a.Get("progress"); ok {
		t.Errorf("the reply carries a progress field: %s", pyjson.Repr(a))
	}
	ra, rb := windowStart.ReplaceAllString(pyjson.Repr(a), ""), windowStart.ReplaceAllString(pyjson.Repr(b), "")
	if ra != rb {
		t.Errorf("reply differs:\n%s\n%s", ra, rb)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("no progress file written: %v", err)
	}
}
