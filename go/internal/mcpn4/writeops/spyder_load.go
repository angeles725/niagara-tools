package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// spyderLoad is the Spyder action that loads the ControlProgram the station unloads on its own
// (BBacnetSpyder.loadControlProgram(), no argument, returns the LoadJob ORD; BDeviceModeActions
// .java:237-257 submits a BLoadJob, answers null when logicLoaded is already true, and refuses
// with HeapLimitReached when the station's heap is past its configured limit; the unload,
// :265-307, removes the ControlProgram slot when it is unsubscribed and not loaded). BLoadJob sets
// the logicLoaded slot (BBacnetSpyder.java:475) when it ends.
const (
	spyderLoad     = "loadControlProgram"
	loadMaxWait    = 300 // seconds
	msgLoad        = "no inverse: loadControlProgram loads the Spyder's ControlProgram into the station (a LoadJob); the station unloads it again on its own when nothing uses it. It can fail with HeapLimitReached when the station's heap is past its configured limit. An already loaded program returns no job and nothing is done. The read-back waits for the job when there is one and then reports logic_loaded and whether the ControlProgram child is there"
	msgLoadMissing = "the load job ended but the program is not loaded (logicLoaded is false): the station unloaded it again or the load did not finish; run loadControlProgram again and repeat the call at once"
	msgLoadNoJob   = "the station returned no job and the program is not loaded (logicLoaded is false): it is not loading; read the station log, then run loadControlProgram again"
)

// invokeReadbackWithLoad is n4_invoke_action's read-back: loadControlProgram here, the rest as invokeReadback.
func invokeReadbackWithLoad(c plan.Client, args *pyjson.Object, p *plan.Planned, replies, other []any) (any, any, any, string, error) {
	if str(args, "action") == spyderLoad {
		return spyderLoadReadback(c, str(args, "ord"), p, replies[0])
	}
	return invokeReadback(c, args, p, replies, other)
}

// spyderLoadPlan plans loadControlProgram (no arg) on a BacnetSpyder.
func spyderLoadPlan(ordText string, h any, nodes, args *pyjson.Object) (*plan.Planned, error) {
	if t := rootType(nodes); t != spyderType {
		return nil, plan.ToolError("action 'loadControlProgram' needs a " + spyderType + ", got " + pyjson.StrRepr(t) + " at " + ordText)
	}
	if _, has := args.Get("arg"); has {
		return nil, plan.ToolError("action 'loadControlProgram' takes no arg")
	}
	targets := pyjson.NewObject()
	targets.Set(ordText, h)
	return &plan.Planned{Ops: []any{invokeOp(obj("h", h, "a", spyderLoad))}, Inverse: []any{},
		Targets: targets, Notes: []any{msgLoad}, Data: downloadData{action: spyderLoad}}, nil
}

// spyderLoadReadback waits for the LoadJob (when the station returned one) and reports whether
// the program is loaded: the logicLoaded slot, and whether the ControlProgram child exists.
func spyderLoadReadback(c plan.Client, ordText string, p *plan.Planned, reply any) (any, any, any, string, error) {
	d, _ := p.Data.(downloadData)
	requested := obj("action", spyderLoad)
	observed := obj("job", nil)
	job := jobOrd(reply)
	state := any("")
	if job != "" {
		if d.w == nil {
			return requested, reply, observed, "unverified", nil
		}
		var nodes *pyjson.Object
		var err error
		if state, nodes, err = waitJob(c, job, discoverData{wait: loadMaxWait, w: d.w}); err != nil {
			return nil, nil, nil, "", err
		}
		log, logErr := jobLog(c, nodes)
		observed = obj("job", job, "job_state", state, "log", log)
		if state == "failed" || state == "canceled" {
			failureDetail(c, p, observed, spyderLoad, nodes, log, logErr)
		}
	}
	loaded, err := loadedState(c, ordText, observed)
	switch {
	case err != nil:
		observed.Set("state_error", err.Error())
		return requested, reply, observed, "unverified", nil
	case state == "failed" || state == "canceled":
		return requested, reply, observed, "mismatch", nil
	case job != "" && state != "success":
		return requested, reply, observed, "unverified", nil
	case loaded:
		return requested, reply, observed, "verified", nil
	case job != "":
		observed.Set("hint", msgLoadMissing)
		return requested, reply, observed, "mismatch", nil
	}
	observed.Set("hint", msgLoadNoJob)
	return requested, reply, observed, "unverified", nil
}

// loadedState reads logic_loaded (the slot, absent when the station sends none) and
// control_program (the child) into observed; loaded is the slot, else the child.
func loadedState(c plan.Client, ordText string, observed *pyjson.Object) (bool, error) {
	nodes, err := c.LoadTree(ordText, 1, box.LoadOptions{})
	if err != nil {
		return false, err
	}
	kids := box.Children(nodes, "")
	program := box.Node(kids, "ControlProgram") != nil
	loaded := program
	if v := box.ChildValue(kids, "logicLoaded", nil); v != nil {
		loaded = strings.EqualFold(strings.TrimSpace(pyjson.Str(v)), "true")
		observed.Set("logic_loaded", loaded)
	}
	observed.Set("control_program", program)
	return loaded, nil
}
