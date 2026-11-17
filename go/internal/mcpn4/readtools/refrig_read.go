package readtools

import (
	"strconv"
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/refrig"
)

// refrigSeries is one history read for n4_refrigeration_faults: its samples in time order, how
// many records came back, whether more existed (the oldest are the ones cut), the offset of its
// timestamps, or why it could not be read.
type refrigSeries struct {
	id        string
	samples   refrig.Series
	records   int
	truncated bool
	loc       *time.Location
	failed    string
}

// readSeries reads history id over params with the locale-free summary query (ISO timestamps,
// encodeToString values). A boolean value is 1 or 0; a record whose value is neither a number
// nor a boolean is skipped (a null status writes no value).
func (t *Tools) readSeries(id, params string, maxRows, timeout int) (*refrigSeries, error) {
	s := &refrigSeries{id: id, loc: time.UTC}
	full, err := bql.ResolveHistoryId(id, t.Ctx.Session.StationName)
	if err != nil {
		return nil, toolError("%s", err.Error())
	}
	s.id = full
	ord, err := bql.ComposeHistoryOrd(full, params, bql.HistorySummaryQuery)
	if err != nil {
		return nil, toolError("%s", err.Error())
	}
	columns, rows, cut, _, err := t.fetchHistory(ord, maxRows, timeout)
	if err == nil {
		rows, err = bql.ByPosition(columns, rows, historyFields)
	}
	if err != nil {
		section, err := t.sectionError(err)
		if err != nil {
			return nil, err
		}
		s.failed = pyStr(lookup(section, "error"))
		return s, nil
	}
	s.records, s.truncated = len(rows), cut
	for i, r := range rows {
		ts, err := time.Parse(time.RFC3339Nano, cellText(r, "timestamp"))
		v, ok := sampleValue(cellText(r, "value"))
		if err != nil || !ok {
			continue
		}
		if i == 0 {
			s.loc = ts.Location()
		}
		s.samples = append(s.samples, refrig.Sample{T: ts, V: v})
	}
	s.samples = refrig.Sorted(s.samples)
	return s, nil
}

func sampleValue(text string) (float64, bool) {
	switch strings.ToLower(strings.TrimSpace(text)) {
	case "true":
		return 1, true
	case "false":
		return 0, true
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	return f, err == nil
}

func pyStr(v any) string { s, _ := v.(string); return s }
