package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// spyderDownload is the Honeywell Spyder action that validates, compiles and writes the station's
// program into the controller (BBacnetSpyder.spyderDownload(BSpyderDownloadParameters) returns
// the job ORD; it checks the controller's model first, niagara-research B1450). Live 2026-11-18 on
// a PUB1012S over MS/TP: about two minutes, the job log ends "Success|...|Job Success|".
const (
	spyderDownload  = "spyderDownload"
	downloadParams  = "honeywellSpyderTool:SpyderDownloadParameters"
	downloadMaxWait = 600 // seconds; a full download on MS/TP took 130 s
	msgDownload     = "no inverse: spyderDownload writes the station's program into the controller and restarts the controller, " +
		"which stops controlling until the download ends (about 2 min on MS/TP); the station validates, compiles and " +
		"checks the controller's model before writing, and refuses a mismatch. The read-back waits up to 600 s for the job"
)

// spyderCompile is the Spyder action that compiles the station's program
// (BBacnetSpyder.compile(BBoolean), a CompilationJob). A download recompiles only a program the
// station marked modified; an edit that does not mark it (a BACnet objectName, live 2026-11-18)
// reaches the controller only after compile.
const (
	spyderCompile = "compile"
	msgCompile    = "no inverse: compile rebuilds the station's compiled program files for the next download; " +
		"nothing reaches the controller. The read-back waits for the CompilationJob and returns its log"
	msgNoCompile = "no compile step in the log: the controller got the program compiled earlier; an edit that did " +
		"not mark the program modified (e.g. a BACnet objectName) is not in it: run compile, then download again"
)

type downloadData struct {
	action string
	full   bool
	w      *plan.State
}

// invokePlanWith is invokePlan plus the write state spyderDownload's read-back waits with.
func invokePlanWith(c plan.Client, args *pyjson.Object, w *plan.State, _ *session.Session) (*plan.Planned, error) {
	p, err := invokePlan(c, args)
	if err == nil {
		if d, ok := p.Data.(downloadData); ok {
			d.w = w
			p.Data = d
		}
	}
	return p, err
}

// spyderDownloadPlan plans a full or quick download on a BacnetSpyder.
func spyderDownloadPlan(ordText string, h any, nodes, args *pyjson.Object) (*plan.Planned, error) {
	if t := rootType(nodes); t != spyderType {
		return nil, plan.ToolError("action 'spyderDownload' needs a " + spyderType + ", got " + pyjson.StrRepr(t) + " at " + ordText)
	}
	raw, _ := args.Get("arg")
	kind, _ := raw.(string)
	if kind != "full" && kind != "quick" {
		return nil, plan.ToolError("spyderDownload needs arg = full or quick, got " + pyjson.Repr(raw))
	}
	full := "false"
	if kind == "full" {
		full = "true"
	}
	b := obj("nm", "p", "t", downloadParams, "s", []any{obj("nm", "p", "n", "fullDownload", "t", "baja:Boolean", "v", full)})
	targets := pyjson.NewObject()
	targets.Set(ordText, h)
	return &plan.Planned{Ops: []any{invokeOp(obj("h", h, "a", spyderDownload, "b", b))}, Inverse: []any{},
		Targets: targets, Notes: []any{msgDownload}, Data: downloadData{action: spyderDownload, full: kind == "full"}}, nil
}

// spyderCompilePlan plans compile (no arg) on a BacnetSpyder.
func spyderCompilePlan(ordText string, h any, nodes, args *pyjson.Object) (*plan.Planned, error) {
	if t := rootType(nodes); t != spyderType {
		return nil, plan.ToolError("action 'compile' needs a " + spyderType + ", got " + pyjson.StrRepr(t) + " at " + ordText)
	}
	if _, has := args.Get("arg"); has {
		return nil, plan.ToolError("action 'compile' takes no arg")
	}
	targets := pyjson.NewObject()
	targets.Set(ordText, h)
	b := obj("nm", "p", "t", "baja:Boolean", "v", "true")
	return &plan.Planned{Ops: []any{invokeOp(obj("h", h, "a", spyderCompile, "b", b))}, Inverse: []any{},
		Targets: targets, Notes: []any{msgCompile}, Data: downloadData{action: spyderCompile}}, nil
}

// spyderDownloadReadback waits for the download or compile job and returns its state and log.
func spyderDownloadReadback(c plan.Client, p *plan.Planned, reply any) (any, any, any, string, error) {
	d, _ := p.Data.(downloadData)
	requested := obj("action", d.action)
	if d.action == spyderDownload {
		requested.Set("full_download", d.full)
	}
	job := jobOrd(reply)
	if job == "" || d.w == nil {
		return requested, reply, obj("job", nil), "unverified", nil
	}
	state, nodes, err := waitJob(c, job, discoverData{wait: downloadMaxWait, w: d.w})
	if err != nil {
		return nil, nil, nil, "", err
	}
	log := jobLog(c, nodes)
	observed := obj("job", job, "job_state", state, "log", log)
	if d.action == spyderDownload && state == "success" && !logHas(log, "Compiling the logic") {
		observed.Set("hint", msgNoCompile)
	}
	switch state {
	case "success":
		return requested, reply, observed, "verified", nil
	case "failed", "canceled":
		return requested, reply, observed, "mismatch", nil
	}
	return requested, reply, observed, "unverified", nil
}

// jobLog is the job's readLog as lines: "message", or "Kind: message" for any kind but Message
// (each raw line is Kind|timestamp|message|). A log that cannot be read is an empty list.
func jobLog(c plan.Client, nodes *pyjson.Object) []any {
	root := box.Node(nodes, "")
	if root == nil {
		return []any{}
	}
	r, err := c.SSC("invokeAction", obj("h", get(root, "h"), "a", "readLog"))
	if err != nil {
		return []any{}
	}
	text := r
	if o, ok := r.(*pyjson.Object); ok {
		text = get(o, "v")
	}
	out := []any{}
	for _, line := range strings.Split(pyjson.Str(text), "\n") {
		parts := strings.Split(line, "|")
		if len(parts) < 3 {
			continue
		}
		if parts[0] == "Message" {
			out = append(out, parts[2])
		} else {
			out = append(out, parts[0]+": "+parts[2])
		}
	}
	return out
}

func logHas(log []any, text string) bool {
	for _, l := range log {
		if strings.Contains(pyjson.Str(l), text) {
			return true
		}
	}
	return false
}
