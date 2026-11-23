package plan

import (
	"errors"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
)

// A confirmation token is bound to the station session it was issued in (Hashed names the session
// id), so after a lost session and a new n4_connect every pending token is dead (retro
// mcpn4-history-batch-tools Δ3: a dropped tunnel mid-batch left 10 half-configured extensions and
// the operator got a bare mismatch). The server remembers the session each token was issued in and,
// when a refused token belongs to an earlier one, says so and how to get a new token. The re-plan is
// the operator's call: the refusal sends nothing and issues no token, so the dry run / confirmation
// contract is unchanged.

// issuedCap bounds issuedIn when tokens are issued and never spent.
const issuedCap = 4096

// tokenNonce is the nonce part of a token string, "" for anything else.
func tokenNonce(tok any) string {
	s, _ := tok.(string)
	if parts := strings.Split(s, "."); len(parts) == 3 {
		return parts[1]
	}
	return ""
}

// noteIssued records the session a token was issued in and drops what is no longer pending.
func (w *State) noteIssued(tok, sessionID string) {
	if w.issuedIn == nil || len(w.issuedIn) > issuedCap {
		w.issuedIn = map[string]string{}
	}
	for nonce := range w.issuedIn {
		if !w.Tokens.Pending(nonce) {
			delete(w.issuedIn, nonce)
		}
	}
	if nonce := tokenNonce(tok); nonce != "" {
		w.issuedIn[nonce] = sessionID
	}
}

// forgetIssued drops a spent token.
func (w *State) forgetIssued(tok any) { delete(w.issuedIn, tokenNonce(tok)) }

// msgReplan is the note a mismatch of an earlier session's token carries.
const msgReplan = "; the token was issued in an earlier station session (the station dropped the session or n4_connect ran since), " +
	"so it can no longer execute and nothing was sent: to re-plan, call %s again with the same arguments and dry_run=true " +
	"(it reads the station again and issues a new token), check the plan, then execute with that token"

// offerReplan is err, with the re-plan note when it is the mismatch of a token issued in a session
// other than sessionID.
func (w *State) offerReplan(err error, tool string, tok any, sessionID string) error {
	var se *safety.Error
	if !errors.As(err, &se) || !strings.HasPrefix(se.Msg, safety.ReasonTokenMismatch) {
		return err
	}
	if was, ok := w.issuedIn[tokenNonce(tok)]; !ok || was == sessionID {
		return err
	}
	return &safety.Error{Msg: se.Msg + strings.Replace(msgReplan, "%s", tool, 1)}
}
