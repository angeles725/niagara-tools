package retro

import (
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/journal"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
)

// keepFunc is the window test of one entry timestamp.
type keepFunc func(ts any) bool

// sinceFilter is _since_filter: every entry at or after since ("" is None: every entry); an
// entry with no usable timestamp is kept, never hidden. A since that does not parse is its
// ValueError.
func sinceFilter(since string) (keepFunc, error) {
	if since == "" {
		return func(any) bool { return true }, nil
	}
	floor, ok := parseTS(since)
	if !ok {
		return nil, &pyjson.ValueError{Msg: "not an ISO-8601 timestamp: " + pyjson.StrRepr(since)}
	}
	return func(ts any) bool {
		t, ok := parseTS(ts)
		return !ok || t >= floor
	}, nil
}

func filter(items []*pyjson.Object, keep keepFunc) []*pyjson.Object {
	var out []*pyjson.Object
	for _, o := range items {
		if keep(get(o, "ts")) {
			out = append(out, o)
		}
	}
	return out
}

// Draft is retro.draft: {markdown, candidates, station, date} for the entries of stateDir and
// the observations at or after since ("" for the whole state dir), dated today in UTC.
func Draft(stateDir, station, since string, observations []*pyjson.Object) (*pyjson.Object, error) {
	return draft(stateDir, station, time.Now().UTC().Format("2006-01-02"), since, observations, sinceFilter)
}

func draft(stateDir, station, date, since string, observations []*pyjson.Object,
	newFilter func(string) (keepFunc, error)) (*pyjson.Object, error) {
	stateDir = safety.ExpandUser(stateDir)
	audit, err := journal.NewAuditLog(stateDir).Entries()
	if err != nil {
		return nil, err
	}
	views, err := journal.New(stateDir).Views()
	if err != nil {
		return nil, err
	}
	keep, err := newFilter(since) // the one place the window is filtered
	if err != nil {
		return nil, err
	}
	byBatch := map[string]*pyjson.Object{}
	for _, v := range views {
		key, _ := pyjson.DictKey(get(v, "batch_id")) // the journal keyed every view already
		byBatch[key] = v
	}
	audit, windowed := filter(audit, keep), filter(views, keep)
	candidates, err := derive(audit, windowed, filter(observations, keep), byBatch)
	if err != nil {
		return nil, err
	}
	markdown, err := render(candidates, station, date, audit, windowed, since)
	if err != nil {
		return nil, err
	}
	items := make([]any, len(candidates))
	for i, c := range candidates {
		items[i] = c
	}
	out := pyjson.NewObject()
	out.Set("markdown", markdown)
	out.Set("candidates", items)
	out.Set("station", station)
	out.Set("date", date)
	return out, nil
}
