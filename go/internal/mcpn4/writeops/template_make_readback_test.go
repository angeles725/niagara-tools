package writeops_test

import (
	"strings"
	"testing"

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
