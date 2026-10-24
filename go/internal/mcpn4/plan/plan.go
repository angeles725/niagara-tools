package plan

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/token"
)

// Planned is what a tool's plan returns (tools_write.Planned): the exact ops and inverse, both
// hashed into the token, human notes, the ORD -> handle targets the intent records, and the
// private data its read-back needs.
type Planned struct {
	Ops, Inverse, Notes []any
	Targets             *pyjson.Object
	Data                any
}

// Impl is one write tool (_Impl): the ORDs its scope check covers, its plan (reads only) and
// its read-back. The inverse it journals is the planned one.
type Impl struct {
	ScopeOrds func(args *pyjson.Object) []any
	Plan      func(c Client, args *pyjson.Object) (*Planned, error)
	Readback  func(c Client, args *pyjson.Object, p *Planned, replies []any) (requested, accepted, observed any, verdict string, err error)
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
	plan := obj("tool", name, "ops", list(p.Ops), "inverse", list(p.Inverse), "notes", list(p.Notes),
		"station", obj("name", sess.StationName, "base_url", sess.BaseURL, "session_id", sess.SessionID))
	canon, err := pyjson.Marshal(plan, pyjson.Canonical)
	if err != nil {
		return nil, "", err
	}
	sum := sha256.Sum256(canon)
	return plan, hex.EncodeToString(sum[:]), nil
}

func (e *Engine) dryRun(name string, args, plan *pyjson.Object, planHash, tierBlock string) (*pyjson.Object, string, error) {
	tok, expires, err := e.State.Tokens.Issue(name, args, planHash)
	if err != nil {
		return nil, "", err
	}
	out := obj("dry_run", true, "plan", plan, "plan_hash", planHash, "confirmation_token", tok, "expires_at", expires)
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
