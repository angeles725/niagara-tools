package livesmoke

import (
	"errors"
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// Caller is the MCP client the scenario talks to: one tools/call that answers whether the result
// is an error and its payload (the structuredContent, or the error text); err ends the step.
type Caller interface {
	Call(tool string, args *pyjson.Object) (isErr bool, payload any, err error)
}

// stepFn runs one step against its report entry and returns the verdict and its detail.
type stepFn func(entry *pyjson.Object) (verdict, detail any, err error)

// Scenario runs the thermostat steps in order; after a required step does not verify the rest
// are skipped, and a folder the station may hold is cleaned up.
type Scenario struct {
	client          Caller
	station         string
	settle          float64
	log             func(string)
	Steps           []*pyjson.Object
	aborted         bool
	created         bool
	savedWithFolder bool
	removalBatch    any
}

// NewScenario is a scenario on station through client; settle bounds the wait for the logic
// outputs (seconds) and log receives one line per step.
func NewScenario(client Caller, station string, settle float64, log func(string)) *Scenario {
	return &Scenario{client: client, station: station, settle: settle, log: log}
}

// step records one step entry and runs fn unless an earlier required step did not verify.
func (s *Scenario) step(name, tool string, fn stepFn, required, probe bool) *pyjson.Object {
	entry := pyjson.NewObject()
	for _, kv := range [][2]any{{"name", name}, {"tool", tool}, {"required", required}, {"probe", probe},
		{"dry_run_done", false}, {"batch_id", nil}, {"verdict", "skipped"}, {"detail", nil}} {
		entry.Set(kv[0].(string), kv[1])
	}
	s.Steps = append(s.Steps, entry)
	if s.aborted {
		entry.Set("detail", "skipped: an earlier required step did not verify")
	} else {
		verdict, detail, err := fn(entry)
		if err != nil {
			verdict, detail = "failed", errorDetail(err)
		}
		entry.Set("verdict", verdict)
		entry.Set("detail", detail)
		if required && verdict != "verified" {
			s.aborted = true
		}
	}
	verdict, _ := entry.Get("verdict")
	s.log(fmt.Sprintf("%-9s %s", pyjson.Str(verdict), name))
	return entry
}

// errorDetail is a SmokeError's message, or "<Type>: <message>" of any other failure.
func errorDetail(err error) string {
	var se *SmokeError
	if errors.As(err, &se) {
		return se.Msg
	}
	var pe *pyError
	if errors.As(err, &pe) {
		return pe.Error()
	}
	return "Exception: " + err.Error()
}

// call is one tools/call with arguments as key, value pairs in order.
func (s *Scenario) call(tool string, args *pyjson.Object) (bool, any, error) {
	return s.client.Call(tool, args)
}

func kwargs(kv ...any) *pyjson.Object {
	o := pyjson.NewObject()
	for i := 0; i+1 < len(kv); i += 2 {
		o.Set(kv[i].(string), kv[i+1])
	}
	return o
}

// read is a read tool call whose error ends the step.
func (s *Scenario) read(tool string, args *pyjson.Object) (any, error) {
	isErr, out, err := s.call(tool, args)
	if err != nil {
		return nil, err
	}
	if isErr {
		return nil, smokef("%s: %s", tool, pyjson.Str(out))
	}
	return out, nil
}

// Write is dry run -> token -> execute; it returns the executed reply. beforeExecute runs after
// the dry run succeeded and just before the execute call: the point from which the station may
// hold the change even if no reply comes back.
func (s *Scenario) Write(entry *pyjson.Object, tool string, beforeExecute func(), args *pyjson.Object) (any, error) {
	if err := AssertInBounds(tool, args); err != nil {
		return nil, err
	}
	isErr, plan, err := s.call(tool, args)
	if err != nil {
		return nil, err
	}
	if isErr {
		return nil, smokef("dry run refused: %s", pyjson.Str(plan))
	}
	hash, err := get(plan, "plan_hash")
	if err != nil {
		return nil, err
	}
	entry.Set("dry_run_done", true)
	entry.Set("plan_hash", hash)
	if beforeExecute != nil {
		beforeExecute()
	}
	token, err := item(plan, "confirmation_token")
	if err != nil {
		return nil, err
	}
	exec := kwargs("dry_run", false, "confirmation_token", token)
	for _, k := range args.Keys() {
		v, _ := args.Get(k)
		exec.Set(k, v)
	}
	isErr, out, err := s.call(tool, exec)
	if err != nil {
		return nil, err
	}
	if isErr {
		return nil, smokef("execute refused: %s", pyjson.Str(out))
	}
	batch, err := get(out, "batch_id")
	if err != nil {
		return nil, err
	}
	entry.Set("batch_id", batch)
	return out, nil
}

// verdictOf is the executed reply's verdict ("unverified" when absent) and a detail holding the
// extra key, value pairs, then what the server observed.
func verdictOf(out any, extra ...any) (any, any, error) {
	verdict, err := getOr(out, "verdict", "unverified")
	if err != nil {
		return nil, nil, err
	}
	observed, _ := get(out, "observed")
	detail := kwargs(extra...)
	detail.Set("observed", observed)
	return verdict, detail, nil
}

// Run executes the plan, then cleans up after an abort if the folder may exist.
func (s *Scenario) Run() {
	for _, r := range s.plan() {
		s.step(r.name, r.tool, r.fn, !r.optional, r.probe)
	}
	if s.aborted && s.created {
		s.cleanup()
	}
}
