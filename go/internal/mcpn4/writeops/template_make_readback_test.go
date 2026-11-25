package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

var mkReply = obj("t", "baja:Ord", "v", "slot:/Services/JobService/MakeStationTemplateJob")

func mkReadback(t *testing.T, c *asks, reply any) (any, string) {
	t.Helper()
	args := obj()
	p := mkPlan(t, c, args)
	_, _, observed, verdict, err := writeops.MakeStationTemplate.Readback(c, args, p, []any{reply}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return observed, verdict
}

func wantAll(t *testing.T, observed any, parts ...string) {
	t.Helper()
	got := js(t, observed)
	for _, part := range parts {
		if !strings.Contains(got, part) {
			t.Errorf("missing %s in %s", part, got)
		}
	}
}

// A job that ended success returns its state, log and the ^temp file it saved (the job's
// templateFileOrd), so n4_read_file can fetch it.
func TestMakeStationTemplateSuccessReturnsTheFile(t *testing.T) {
	log := "Message|2026-11-25T10:00:00|Reading the station|\nSuccess|2026-11-25T10:00:02|Job Success|\n"
	observed, verdict := mkReadback(t, mkStation("success", log, mkFile), mkReply)
	if verdict != "verified" {
		t.Errorf("verdict = %q", verdict)
	}
	wantAll(t, observed, `"job": "`+mkJob+`"`, `"job_state": "success"`, `"Reading the station"`, `"Success: Job Success"`,
		`"file_ord": "`+mkFile+`"`, `"template_name": "Station"`, "n4_read_file")
}

// A failed job is a mismatch: its state and log, and no file to fetch.
func TestMakeStationTemplateFailedJobIsAMismatchWithItsLog(t *testing.T) {
	log := "Message|t|Reading the station|\nFailure|t|java.io.IOException: disk full|\n"
	observed, verdict := mkReadback(t, mkStation("failed", log, ""), mkReply)
	if verdict != "mismatch" {
		t.Errorf("verdict = %q", verdict)
	}
	wantAll(t, observed, `"job_state": "failed"`, `"Failure: java.io.IOException: disk full"`, `"file_ord": null`)
	if got := js(t, observed); strings.Contains(got, "n4_read_file") {
		t.Errorf("a failed job points at a file: %s", got)
	}
}

// A job still running at the deadline is unverified, with where to read it later.
func TestMakeStationTemplateRunningJobIsUnverified(t *testing.T) {
	observed, verdict := mkReadback(t, mkStation("running", "", ""), mkReply)
	if verdict != "unverified" {
		t.Errorf("verdict = %q", verdict)
	}
	wantAll(t, observed, `"job_state": "running"`, "still running", mkJob)
}

// doMakeStationTemplate returns null for a user that is not a super user
// (BTemplateService.java:275-282): no job, so nothing is proved and the cause is named.
func TestMakeStationTemplateNullReplyNamesTheSuperUserRule(t *testing.T) {
	observed, verdict := mkReadback(t, mkStation("success", "", mkFile), nil)
	if verdict != "unverified" {
		t.Errorf("verdict = %q", verdict)
	}
	wantAll(t, observed, "super user", `"job": null`)
}

// A job that succeeded but kept no file is not proof of a template: mismatch.
func TestMakeStationTemplateSuccessWithoutAFileIsAMismatch(t *testing.T) {
	observed, verdict := mkReadback(t, mkStation("success", "", ""), mkReply)
	if verdict != "mismatch" {
		t.Errorf("verdict = %q", verdict)
	}
	wantAll(t, observed, "templateFileOrd")
}

// sized is a station whose file servlet answers, one per read, the next of files (an error when it is one).
type sized struct {
	*asks
	files []any
	urls  []string
	caps  []int
}

func (s *sized) GetFile(urlPath string, _ float64, maxBytes int) (*box.File, error) {
	s.urls, s.caps = append(s.urls, urlPath), append(s.caps, maxBytes)
	n := len(s.urls) - 1
	if n >= len(s.files) {
		n = len(s.files) - 1
	}
	if err, ok := s.files[n].(error); ok {
		return nil, err
	}
	return s.files[n].(*box.File), nil
}

// size is a file reply: length is the Content-Length ("" = no header), body the bytes read.
func size(length string, body int, truncated bool) *box.File {
	f := &box.File{Body: make([]byte, body), Truncated: truncated}
	if length != "" {
		f.ContentLength = pyjson.IntLiteral(length)
	}
	return f
}

func sizeReadback(t *testing.T, s *sized) (any, string) {
	t.Helper()
	args := obj("accept_runaway_risk", true)
	p, err := writeops.MakeStationTemplate.PlanWith(s, args, clock(), &session.Session{StationName: "Station"})
	if err != nil {
		t.Fatal(err)
	}
	var c plan.Client = s
	_, _, observed, verdict, err := writeops.MakeStationTemplate.Readback(c, args, p, []any{mkReply}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return observed, verdict
}

// A job still running: the ^temp output is read twice a few seconds apart and its growth reported.
func TestMakeStationTemplateRunningJobReportsTheOutputGrowth(t *testing.T) {
	s := &sized{asks: mkStation("running", "", ""), files: []any{size("1000", 1000, false), size("61000000", 1<<20, true)}}
	observed, verdict := sizeReadback(t, s)
	if verdict != "unverified" {
		t.Errorf("verdict = %q", verdict)
	}
	wantAll(t, observed, `"file_ord": "file:^temp/Station.ntpl"`, `"first_bytes": 1000`, `"second_bytes": 61000000`,
		`"growth_bytes": 60999000`, `"seconds_apart": 5`, `"growing": true`, "RUNAWAY", "stop the station process")
	if len(s.urls) != 2 || s.urls[0] != "/file/temp/Station.ntpl" {
		t.Errorf("reads = %v", s.urls)
	}
}

// Without a Content-Length a read that hits the cap only proves "at least" that many bytes.
func TestMakeStationTemplateCappedReadSaysAtLeast(t *testing.T) {
	s := &sized{asks: mkStation("running", "", ""), files: []any{size("", 1<<20, true)}}
	observed, _ := sizeReadback(t, s)
	wantAll(t, observed, `"first_bytes": 1048576`, `"first_at_least": true`, "at least 1048576 bytes")
	if got := js(t, observed); strings.Contains(got, `"growing"`) {
		t.Errorf("growth claimed from two capped reads: %s", got)
	}
}

// A file that does not grow is reported as such, never as progress; one not found says where it looked.
func TestMakeStationTemplateStableAndMissingOutput(t *testing.T) {
	stable, _ := sizeReadback(t, &sized{asks: mkStation("running", "", ""), files: []any{size("500", 500, false)}})
	wantAll(t, stable, `"growth_bytes": 0`, `"growing": false`)
	missing, verdict := sizeReadback(t, &sized{asks: mkStation("running", "", ""),
		files: []any{&box.Error{Kind: box.KindHTTPStatus, Code: 404, Msg: "not found"}}})
	if verdict != "unverified" {
		t.Errorf("verdict = %q", verdict)
	}
	wantAll(t, missing, `"file_ord": "file:^temp/Station.ntpl"`, "could not be read", "^temp")
}

// A job that ended is not sampled: its templateFileOrd is the answer.
func TestMakeStationTemplateFinishedJobIsNotSampled(t *testing.T) {
	s := &sized{asks: mkStation("success", "", mkFile), files: []any{size("1", 1, false)}}
	if _, verdict := sizeReadback(t, s); verdict != "verified" || len(s.urls) != 0 {
		t.Errorf("verdict %q, reads %v", verdict, s.urls)
	}
}
