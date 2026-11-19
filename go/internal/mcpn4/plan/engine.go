package plan

import (
	"errors"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/jsonrpc"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// Handler is _make_handler(name): the tool's pipeline plus its audit line.
func (e *Engine) Handler(name string, impl Impl) jsonrpc.Handler {
	return func(args *pyjson.Object) (*pyjson.Object, error) {
		batchID := ""
		out, err := jsonrpc.Recover(func() (*pyjson.Object, error) { // a panic is audited as an error
			out, id, err := e.process(name, impl, args)
			batchID = id
			return out, err
		})
		var te *jsonrpc.ToolError
		var se *safety.Error
		var doubt *InDoubt
		var refused *Refused
		switch {
		case errors.As(err, &doubt): // the station call failed after the intent was journaled
			e.record(name, args, doubt.BatchID, "error", doubt.Msg)
			return nil, ToolError(doubt.Msg)
		case errors.As(err, &refused): // the station refused it and nothing was applied
			e.record(name, args, refused.BatchID, "refused", refused.Msg)
			return nil, ToolError(refused.Msg)
		case errors.As(err, &te) || errors.As(err, &se):
			e.record(name, args, nil, "refused", PyStr(err))
			return nil, ToolError(PyStr(err))
		case err != nil:
			e.record(name, args, nil, "error", PyType(err)+": "+PyStr(err))
			return nil, err
		}
		logged := true
		if batchID == "" {
			logged = e.record(name, args, nil, "planned", "dry run")
		} else {
			verdict, _ := out.Get("verdict")
			logged = e.record(name, args, batchID, "executed", "verdict="+pyjson.Str(verdict))
		}
		if !logged {
			warnings, _ := out.Get("warnings")
			list, _ := warnings.([]any)
			out.Set("warnings", append(list, "the audit log could not be written"))
		}
		return out, nil
	}
}

// process is _process: the plan and either its dry run or its execution; batchID is "" for a
// dry run.
func (e *Engine) process(name string, impl Impl, args *pyjson.Object) (out *pyjson.Object, batchID string, err error) {
	w, sess := e.State, e.Ctx.Session
	dry := true
	if v, ok := args.Get("dry_run"); ok {
		dry = pyjson.Truthy(v)
	}
	if sess == nil {
		return nil, "", ToolError(safety.ReasonNotConnected + ": call n4_connect first")
	}
	if !sess.IdentityVerified {
		return nil, "", ToolError(safety.ReasonIdentity + ": reconnect with n4_connect and pass " +
			"expected_station=<the station's stationName> before any write")
	}
	tierBlock := e.Ctx.TierBlock(sess) // METHODOLOGY section 5: a dry run is still allowed
	if tierBlock != "" && !dry {
		return nil, "", ToolError(tierBlock)
	}
	if spendsBudget(name) && w.WritesExecuted >= w.MaxWrites {
		return nil, "", budgetRefusal(w.MaxWrites)
	}
	if err := checkScope(impl, w.Scope, args); err != nil {
		return nil, "", err
	}
	client, ok := sess.Client.(Client)
	if !ok {
		return nil, "", &PyError{Type: "AttributeError", Msg: "the station client cannot write"}
	}
	var planned *Planned
	if impl.PlanWith != nil {
		planned, err = impl.PlanWith(client, args, w, sess)
	} else {
		planned, err = impl.Plan(client, args)
	}
	if err == nil {
		err = checkChannelOps(planned.Ops) // before the token and the journal: nothing planned
	}
	if err != nil {
		return nil, "", err
	}
	if planned.Nothing != nil {
		return planned.Nothing, "", nil
	}
	plan, planHash, err := Hashed(name, planned, sess)
	if err != nil {
		return nil, "", err
	}
	if dry {
		return e.dryRun(name, args, plan, planHash, tierBlock, list(planned.Inverse), planned.Preview)
	}
	if err := w.checkStateFiles(); err != nil { // a loose file fails before the token and any send
		return nil, "", err
	}
	tok, _ := args.Get("confirmation_token")
	if err := w.Tokens.Consume(name, args, planHash, tok); err != nil {
		return nil, "", err
	}
	batchID = w.NewBatchID()
	targets := planned.Targets
	if targets == nil {
		targets = pyjson.NewObject()
	}
	intent := obj("batch_id", batchID, "ts", session.Now(), "tool", name, "ops", list(planned.Ops),
		"inverse_plan", list(planned.Inverse), "station_name", sess.StationName, "phase", "intent", "targets", targets)
	withKeys(intent, planned, IntentKeys)
	if w.Journal.Append(intent) != nil { // write-ahead: no intent on disk, no op on the wire
		return nil, "", ToolError("nothing was sent: the journal intent could not be written (batch " + batchID +
			"); the confirmation token is spent, run the dry run again")
	}
	if spendsBudget(name) {
		w.WritesExecuted += planned.cost() // counted once sent, whatever the station answers
	}
	// From here on the batch exists: every failure is its in-doubt error, a panic (Python's
	// unexpected exception) too.
	out, err = jsonrpc.Recover(func() (*pyjson.Object, error) { return e.execute(client, impl, args, planned, batchID) })
	var doubt *InDoubt
	var refused *Refused
	if err != nil && !errors.As(err, &doubt) && !errors.As(err, &refused) {
		err = inDoubt(batchID, err)
	}
	if out != nil {
		out.Set("write_budget", w.budgetReport())
	}
	return out, batchID, err
}

// checkScope is the scope step of a write: the tool's own rule when it has one, else every ORD it
// lists under a prefix, and at least one prefix configured when it lists none (whole-station or
// journal-driven writes).
func checkScope(impl Impl, scope *ord.Scope, args *pyjson.Object) error {
	if impl.ScopeCheck != nil {
		return impl.ScopeCheck(scope, args)
	}
	ords := impl.ScopeOrds(args)
	for _, o := range ords {
		if err := scope.Check(o); err != nil {
			return err
		}
	}
	if len(ords) == 0 {
		return scope.RequireAny()
	}
	return nil
}
