package writeops_test

// Issue #818: a failed Spyder job names where the compiler's own message is. Station facts:
// BSpyderDownloadJob.java:236-237 catches the compiler's exception and logs only "Download failed
// due to compilation error" (the message is dropped); BCompilationJob.java:88/138 pass
// exception.getMessage() (null for many exceptions) to JobLog.failed and :99/:107 set hasErrors
// only for a validation failure.

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

const dlReply = "slot:/Services/JobService/SpyderDownloadJob4"

// failedJob is a Spyder 2012 whose job ended failed with the log and hasErrors given ("" leaves
// the slot out, as a SpyderDownloadJob has none).
func failedJob(log, hasErrors string) *asks {
	c := dlStation("failed", log)
	if hasErrors != "" {
		job, _ := c.nodes.Get(dlJob)
		job.(*pyjson.Object).Set("hasErrors", obj("t", "baja:Boolean", "v", hasErrors))
	}
	return c
}

// readBack plans action on the Spyder and returns the observed side of its read-back.
func readBack(t *testing.T, c *asks, args *pyjson.Object) string {
	t.Helper()
	p, err := writeops.InvokeAction.PlanWith(c, args, clock(), nil)
	if err != nil {
		t.Fatal(err)
	}
	_, _, observed, verdict, err := writeops.InvokeAction.Readback(c, args, p, []any{obj("v", dlReply)}, nil)
	if err != nil || verdict != "mismatch" {
		t.Fatalf("verdict %q err %v", verdict, err)
	}
	return js(t, observed)
}

// The download job drops the compiler's message; the read-back says so and names compile, whose
// CompilationJob logs it.
func TestAFailedDownloadThatDroppedTheCompilerMessageNamesCompile(t *testing.T) {
	c := failedJob("Message|t|Compiling the logic|\nFailed|t|Download failed due to compilation error|\n", "")
	got := readBack(t, c, obj("ord", bacnetNet+"/$32012", "action", "spyderDownload", "arg", "full"))
	for _, want := range []string{`"cause": "`, "does not keep the compiler's message", "run action compile", `"Failed: Download failed due to compilation error"`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %s in %s", want, got)
		}
	}
}

// A compile that failed validation says so (hasErrors true) and points at the validation lines.
func TestAFailedCompileWithValidationErrorsSaysSo(t *testing.T) {
	c := failedJob("Message|t|Compilation started on device - $32012|\nFailed|t|Terminal 3 is used twice|\n", "true")
	got := readBack(t, c, obj("ord", bacnetNet+"/$32012", "action", "compile"))
	for _, want := range []string{`"has_errors": true`, "validation", `"Failed: Terminal 3 is used twice"`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %s in %s", want, got)
		}
	}
}

// A compile whose job log holds only the IllegalArgumentException JobLog.failed throws for a null
// message (live 2026-11-18) and no validation error: the compiler's message is only in the station log.
func TestAFailedCompileWithANullMessageNamesTheStationLog(t *testing.T) {
	c := failedJob("Message|t|Compilation started on device - $32012|\nFailed|t|java.lang.IllegalArgumentException|\n", "false")
	got := readBack(t, c, obj("ord", bacnetNet+"/$32012", "action", "compile"))
	for _, want := range []string{`"has_errors": false`, "null message", "station log", "honeywellSpyderTool"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %s in %s", want, got)
		}
	}
}

// A failed job whose log shows none of this keeps the existing read-back: no cause, and a job
// without a hasErrors slot (a download) carries no has_errors.
func TestAFailedDownloadWithAnOrdinaryLogCarriesNoCause(t *testing.T) {
	c := failedJob("Message|t|Download started|\nFailed|t|Download Failed on device - $32012|\n", "")
	got := readBack(t, c, obj("ord", bacnetNet+"/$32012", "action", "spyderDownload", "arg", "full"))
	if strings.Contains(got, `"cause"`) || strings.Contains(got, "has_errors") || !strings.Contains(got, "BacnetNIFolder") {
		t.Errorf("observed %s", got)
	}
}
