package readtools

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/journal"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// ListBatches is n4_list_batches: the journaled write batches, newest first, at most limit of
// them, with the total, and the unreadable lines when there are any (issue #448). It reads the journal only, never the station, and needs no session.
func (t *Tools) ListBatches(args *pyjson.Object) (*pyjson.Object, error) {
	return journal.New(t.stateDir()).ListBatches(intArg(args, "limit", 50))
}
