package plan

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/token"
)

// Planned is what a tool's plan returns (tools_write.Planned): the exact ops and inverse, both
// hashed into the token (the inverse without its `prior` values), human notes, the ORD ->
// handle targets the intent records, the PlanKeys and IntentKeys the token and the intent carry
// (Python keeps them in the plan's data), and the private data its read-back needs. HashInverse,
// when not nil, is the inverse the plan shows and the token hashes instead of Inverse: the same
// entries with the values of runtime slots left out (writeops.runtimeSlot), which change by
// timing alone; the journal keeps Inverse whole.
type Planned struct {
	Ops, Inverse, Notes []any
	HashInverse         []any
	Targets, Keys       *pyjson.Object
	Data                any
	// Writes is what executing the batch spends of --max-writes: 0 counts as one write, the
	// cost of an ordinary batch; a bulk tool sets one per station change (n4_replicate_points).
	Writes int
	// Nothing, when not nil, is a plan with nothing to do: the engine returns it as the reply,
	// with no token, no journal and no station call (n4_ack_alarm's selection matching nothing).
	Nothing *pyjson.Object
	// Preview, when not nil, is volatile state shown to the operator (a link source's current
	// value): its keys join the dry-run reply, never the hashed plan, so a token stays valid while
	// that state moves.
	Preview *pyjson.Object
}

// Impl is one write tool (_Impl): the ORDs its scope check covers, its plan (reads only; PlanWith
// instead when it also reads the write state and session, Python's needs_ctx), the inverse it
// journals (nil: the planned one; else built from the station's replies, where an error means
// the reply cannot be trusted), the recover hook of such a reply (nil: none), the follow-up
// runs of the same batch (nil: none), its read-back, and the check that nothing of a batch the
// station refused was applied (nil: a refusal stays in-doubt).
type Impl struct {
	ScopeOrds func(args *pyjson.Object) []any
	// ScopeCheck, when set, replaces the ScopeOrds check: a tool whose scope rule is more than
	// "every listed ORD under a prefix" (create may make the scope root, a link is scoped on its target).
	ScopeCheck func(s *ord.Scope, args *pyjson.Object) error
	Plan       func(c Client, args *pyjson.Object) (*Planned, error)
	PlanWith   func(c Client, args *pyjson.Object, w *State, sess *session.Session) (*Planned, error)
	Run        func(b *Batch, p *Planned, replies []any) (*Runs, error)
	Inverse    func(p *Planned, replies []any) ([]any, error)
	Recover    func(c Client, args *pyjson.Object, p *Planned, replies []any) (observed any, inverse []any, err error)
	Readback   func(c Client, args *pyjson.Object, p *Planned, replies, inverse []any) (requested, accepted, observed any, verdict string, err error)
	// Unapplied reads back the batch's targets after the station refused its first op: true only
	// when the observation shows none of its ops applied.
	Unapplied func(c Client, args *pyjson.Object, p *Planned) (bool, error)
}

// Batch is one executing batch as its follow-up runs see it: the station client, the write
// state (scope, journal, budget) and the batch id.
type Batch struct {
	Client Client
	State  *State
	ID     string
}

// Runs is what follow-up runs leave for the execution (_run_components, _run_relinks): the
// frozen children not re-created (echoed when any), the relink report the result records (nil:
// no relinks), the inverse of the links created (journaled before the tool's own) and whether
// a relink reply was ambiguous (the batch is in-doubt).
type Runs struct {
	Frozen  []any
	Report  *pyjson.Object
	Inverse []any
	Doubt   bool
}

// Engine runs the write tools of one server process.
type Engine struct {
	Ctx   *session.Context
	State *State
}

// Hashed is the plan a confirmation token authorizes and its plan_hash, the sha256 hex of its
// canonical JSON. It names the station session (audit F1): a token cannot execute after a
// reconnect or on another station.
func Hashed(name string, p *Planned, sess *session.Session) (*pyjson.Object, string, error) {
	inverse := p.Inverse
	if p.HashInverse != nil {
		inverse = p.HashInverse
	}
	plan := obj("tool", name, "ops", list(p.Ops), "inverse", HashedInverse(list(inverse)), "notes", list(p.Notes),
		"station", obj("name", sess.StationName, "base_url", sess.BaseURL, "session_id", sess.SessionID))
	withKeys(plan, p, PlanKeys)
	canon, err := pyjson.Marshal(plan, pyjson.Canonical)
	if err != nil {
		return nil, "", err
	}
	sum := sha256.Sum256(canon)
	return plan, hex.EncodeToString(sum[:]), nil
}

func (e *Engine) dryRun(name string, args, plan *pyjson.Object, planHash, tierBlock string, inverse []any,
	extra *pyjson.Object) (*pyjson.Object, string, error) {
	tok, expires, err := e.State.Tokens.Issue(name, args, planHash)
	if err != nil {
		return nil, "", err
	}
	e.State.noteIssued(tok, e.Ctx.Session.SessionID)
	out := obj("dry_run", true, "plan", plan, "plan_hash", planHash, "confirmation_token", tok, "expires_at", expires,
		"write_budget", e.State.budgetReport())
	preview, err := LinkInputValues(inverse)
	if err != nil {
		return nil, "", err
	}
	if len(preview) > 0 { // volatile: shown to the operator, outside what the token authorizes
		out.Set("link_input_values", preview)
	}
	if extra != nil { // volatile, like link_input_values: outside the hash
		for _, k := range extra.Keys() {
			if _, taken := out.Get(k); !taken {
				v, _ := extra.Get(k)
				out.Set(k, v)
			}
		}
	}
	if tierBlock != "" { // outside the plan hash: the gate is policy, not part of the plan
		sess := e.Ctx.Session
		var version any
		if sess.Version != "" {
			version = sess.Version
		}
		out.Set("tier_gate", obj("tier", sess.Tier, "version", version, "would_block", true, "reason", tierBlock))
	}
	return out, "", nil
}

// record is _record: one audit line; false when it could not be written.
func (e *Engine) record(name string, args *pyjson.Object, batchID any, outcome, reason string) bool {
	dry, ok := args.Get("dry_run")
	if !ok {
		dry = true
	}
	return e.State.Audit.Append(obj("ts", session.Now(), "tool", name, "batch_id", batchID, "dry_run", dry,
		"outcome", outcome, "reason", e.Ctx.Scrub(reason), "args_redacted", token.Redact(args))) == nil
}
