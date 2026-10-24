package readtools

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/journal"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// ListBatches is n4_list_batches: the journaled write batches, newest first, at most limit of
// them, with the total. It reads the journal only, never the station, and needs no session.
func (t *Tools) ListBatches(args *pyjson.Object) (*pyjson.Object, error) {
	views, err := journal.New(t.stateDir()).Views()
	if err != nil {
		return nil, err
	}
	rows, err := journal.Batches(views)
	if err != nil {
		return nil, err
	}
	return obj("total", len(rows), "batches", rows[:min(intArg(args, "limit", 50), len(rows))]), nil
}
