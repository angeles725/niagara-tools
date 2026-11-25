package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// spyderErrorSlots are the Spyder's own slots that keep what a failed compile or download left
// (the commissioning report's spyder section reads the same ones).
var spyderErrorSlots = []string{"deviceStatus", "compileStatusFlag", "sErrorStatus"}

// msgJobFailed is the hint of a failed job: the job's log is where Workbench shows the cause, and
// live 2026-11-18 a compile that broke answered only "due to compilation error" with the log
// unreadable; the known cause is a renamed BacnetNIFolder slot.
const msgJobFailed = "the job failed: its log, log_error and device_state above are all the station keeps; the compiler's own " +
	"message is in the station log. A compile error after a rename usually means a BacnetNIFolder slot was renamed: " +
	"rename the BACnet objectName only, never the slot, then compile and download again"

// Where the compiler's own message is, by what the failed job left. Station facts:
// BSpyderDownloadJob.java:236-237 logs only the fixed text and drops the exception;
// BCompilationJob.java:99/107 set hasErrors for a validation failure (its lines are in the job
// log), :88/:138 pass exception.getMessage() to JobLog.failed, which throws on null [INFER: the
// IllegalArgumentException of live 2026-11-18], and SpyderLogger.java:9 logs as honeywellSpyderTool.
const (
	droppedText     = "Download failed due to compilation error"
	nullMessageText = "IllegalArgumentException"
	msgDropped      = "the download job does not keep the compiler's message (it logs only that text): run action compile on " +
		"this Spyder, whose CompilationJob logs the validation or compiler message in its own log"
	msgValidation = "validation failed (hasErrors true): the validation lines are in the job log above"
	msgNullMsg    = "the compiler threw with a null message (JobLog.failed rejects it) and no validation error (hasErrors false): " +
		"the cause is only in the station log, logger honeywellSpyderTool"
)

// failureDetail adds what a failed or canceled job leaves to observed: why the log is empty
// (log_error), the Spyder's error slots (device_state, when they can be read), the job's
// hasErrors and where the compiler's message is (cause), and the hint.
func failureDetail(c plan.Client, p *plan.Planned, observed *pyjson.Object, action string, job *pyjson.Object, log []any, logErr string) {
	if logErr != "" {
		observed.Set("log_error", logErr)
	}
	for _, ordText := range p.Targets.Keys() {
		nodes, err := c.LoadTree(ordText, 1, box.LoadOptions{})
		if err != nil {
			break
		}
		state := pyjson.NewObject()
		kids := box.Children(nodes, "")
		for _, slot := range spyderErrorSlots {
			if v := box.ChildValue(kids, slot, nil); v != nil {
				state.Set(slot, pyjson.Str(v))
			}
		}
		if len(state.Keys()) > 0 {
			observed.Set("device_state", state)
		}
		break
	}
	failureCause(action, job, log, observed)
	observed.Set("hint", msgJobFailed)
}

// failureCause sets has_errors (the compile job's hasErrors slot, when it has one) and cause, the
// one sentence saying where the compiler's message is, when the job's log tells which case it is.
func failureCause(action string, job *pyjson.Object, log []any, observed *pyjson.Object) {
	var hasErrors any
	if v := box.ChildValue(box.Children(job, ""), "hasErrors", nil); v != nil {
		hasErrors = strings.EqualFold(strings.TrimSpace(pyjson.Str(v)), "true")
		observed.Set("has_errors", hasErrors)
	}
	switch {
	case action == spyderDownload && logHas(log, droppedText):
		observed.Set("cause", msgDropped)
	case hasErrors == true:
		observed.Set("cause", msgValidation)
	case hasErrors == false && logHas(log, nullMessageText):
		observed.Set("cause", msgNullMsg)
	}
}
