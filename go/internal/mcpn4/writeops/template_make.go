package writeops

import (
	"errors"
	"strconv"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// MakeStationTemplate is n4_make_station_template: the TemplateService's makeStationTemplate action.
// Station code, template-rt (decompiled, niagara-research organized/template):
//   - the action takes a BBoolean useMinorVersionOnDeployment, default true, flags 4 (hidden) and
//     returns the job's BOrd: BTemplateService.java:70-76 and 275.
//   - only a super user runs it: for anyone else doMakeStationTemplate returns null, no job:
//     BTemplateService.java:275-282.
//   - it submits a BMakeStationTemplateJob whose run reads the local station into a NiagaraTemplate
//     (NiagaraTemplate.createFromLocalStation): job/BMakeStationTemplateJob.java:24-28.
//   - saveTemplateToTemporaryFile writes the template into the station's ^temp directory under the
//     template's own file name made unique, and sets the job's templateFileOrd and templateName
//     properties: job/BMakeTemplateJob.java:60-66. The job is transient, under Services/JobService.
//
// The file stays in ^temp until the station clears it: there is no inverse and no delete tool. The
// read-back (template_make_readback.go) waits for the job and returns its state, log and file ORD. Like
// n4_save_station it is whole-station (any write scope will do): it changes no component, it writes a ^temp file.
var MakeStationTemplate = plan.Impl{ScopeOrds: func(*pyjson.Object) []any { return nil },
	PlanWith: makeTemplatePlan, Readback: makeTemplateReadback}

const (
	makeAction      = "makeStationTemplate"
	makeServiceType = "template:TemplateService"
	makeDefService  = "station:|slot:/Services/TemplateService"
	makeDefWait     = 120
	makeMaxWait     = 600
	msgMakeWrites   = "writes a file on the station: a new ^temp/<template file name>.ntpl (made unique), " +
		"saved by the BMakeStationTemplateJob; the file ORD is read back from the job's templateFileOrd " +
		"and n4_read_file fetches it"
	msgMakeNoInverse = "no inverse: the file stays in the station's ^temp directory until the station clears " +
		"it, and no tool deletes it"
	msgMakeSuperUser = "only a super user may run the action: for any other user the station answers null " +
		"and starts no job"
)

// makeData is what the read-back needs: the wait and the write state's clock.
type makeData struct {
	wait int
	w    *plan.State
}

func makeServiceOrd(args *pyjson.Object) string {
	if s := str(args, "service_ord"); s != "" {
		return s
	}
	return makeDefService
}

func makeTemplatePlan(c plan.Client, args *pyjson.Object, w *plan.State, _ *session.Session) (*plan.Planned, error) {
	minor := "true"
	if raw, given := args.Get("use_minor_version"); given && raw != nil {
		b, ok := raw.(bool)
		if !ok {
			return nil, plan.ToolError("use_minor_version must be true or false, got " + pyjson.Repr(raw))
		}
		minor = strconv.FormatBool(b)
	}
	wait, err := intArg(args, "wait_seconds", makeDefWait, 1, makeMaxWait)
	if err != nil {
		return nil, err
	}
	service := makeServiceOrd(args)
	h, nodes, err := plan.Handle(c, service, 1)
	var be *box.Error
	if errors.As(err, &be) && be.Station {
		return nil, plan.ToolError("the station has no TemplateService at " + service + " (" + be.Msg +
			"): the template module is not installed or the service is not added under Services")
	}
	if err != nil {
		return nil, err
	}
	if t := rootType(nodes); t != makeServiceType {
		return nil, plan.ToolError(service + " is " + pyjson.Repr(t) + ", not a " + makeServiceType)
	}
	b := obj("nm", "p", "t", "baja:Boolean", "v", minor)
	targets := pyjson.NewObject()
	targets.Set(service, h)
	notes := []any{msgMakeWrites, msgMakeNoInverse, msgMakeSuperUser,
		"useMinorVersionOnDeployment = " + minor + "; the job reads the whole local station, then the read-back " +
			"waits up to " + strconv.Itoa(wait) + " s plus " + strconv.Itoa(int(discoverSlack)) + " s for it"}
	return &plan.Planned{Ops: []any{invokeOp(obj("h", h, "a", makeAction, "b", b))}, Inverse: []any{},
		Targets: targets, Notes: notes, Data: makeData{wait: wait, w: w}}, nil
}
