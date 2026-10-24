// Package plan is the write pipeline of mcp-n4-kit/mcp_n4/tools_write.py (76-142, 1260-1500):
// value coercion and BOX bodies, the per-process write state, and the engine every write tool
// runs through. Identity, tier gate, budget and scope come first; the plan (reads only) is then
// hashed with the station session it was made on; a dry run issues a single-use token bound to
// that hash, an execution spends it, journals its intent before any op is sent, reads the result
// back and journals the outcome. Every call leaves an audit line.
package plan

import (
	"crypto/rand"
	"encoding/hex"
	"strconv"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/journal"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/token"
)

// Client is what a write needs of the session's station client (box.Client).
type Client interface {
	LoadTree(ord string, depth int, opt box.LoadOptions) (*pyjson.Object, error)
	SSC(key string, arg any) (any, error)
	Sync(op any) (any, error)
	ClearHandles()
}

// StateFile is a journal or audit file: appended to, and checked private before a send.
type StateFile interface {
	Append(entry *pyjson.Object) error
	CheckPrivate() error
}

// State is WriteState: the write scope, tokens, journal, audit log and budget of one server
// process.
type State struct {
	Scope          *ord.Scope
	Tokens         *token.Tokens
	Journal, Audit StateFile
	MaxWrites      int
	// WritesExecuted counts the executed writes of this process (audit F5): a reconnect does not
	// reset it, so --max-writes bounds the whole run.
	WritesExecuted int
	// NewBatchID is uuid.uuid4().hex.
	NewBatchID func() string
}

// NewState is WriteState(scopes, state_dir, token_ttl, max_writes): the start checks in their
// order (positive limits, a state dir the server can trust, plain slot scopes).
func NewState(scopes []string, stateDir string, tokenTTL, maxWrites int) (*State, error) {
	for _, f := range []struct {
		flag  string
		value int
	}{{"--token-ttl", tokenTTL}, {"--max-writes", maxWrites}} {
		if f.value < 1 {
			return nil, &pyjson.ValueError{Msg: f.flag + " must be a positive integer, got " + strconv.Itoa(f.value)}
		}
	}
	if err := safety.CheckStateDir(stateDir); err != nil {
		return nil, err
	}
	scope, err := ord.NewScope(scopes)
	if err != nil {
		return nil, err
	}
	return &State{Scope: scope, Tokens: token.New(tokenTTL, nil), Journal: journal.New(stateDir),
		Audit: journal.NewAuditLog(stateDir), MaxWrites: maxWrites, NewBatchID: uuid4Hex}, nil
}

// checkStateFiles is check_state_files: a journal or audit file that became loose after start.
func (s *State) checkStateFiles() error {
	if err := s.Journal.CheckPrivate(); err != nil {
		return err
	}
	return s.Audit.CheckPrivate()
}

func uuid4Hex() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err) // crypto/rand never fails on a supported platform
	}
	b[6], b[8] = b[6]&0x0f|0x40, b[8]&0x3f|0x80
	return hex.EncodeToString(b)
}
