package readtools

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/schedule"
)

// exportImport is what the kinds reply says about moving schedules between stations: the
// product has it, this server has no tool for it.
const exportImport = "Schedules can be exported from one station and imported read-only in " +
	"another (niagaraDriver:NiagaraScheduleExport / NiagaraScheduleImportExt under the device's " +
	"schedule extension; bacnet:BacnetScheduleExport / BacnetScheduleImportExt for BACnet). " +
	"mcp-n4 has no tool for them."

// ListScheduleKinds is n4_list_schedule_kinds: the schedule kinds and special-event kinds that
// exist, what each is for and the parameters each takes. It reads no station.
func (t *Tools) ListScheduleKinds(*pyjson.Object) (*pyjson.Object, error) {
	kinds := make([]any, 0, len(schedule.Kinds))
	for _, k := range schedule.Kinds {
		var valueType, defaultOutput any
		if k.ValueType != "" {
			valueType, defaultOutput = k.ValueType, k.DefaultOutput
		}
		kinds = append(kinds, obj("name", k.Name, "type", k.Type, "value_type", valueType,
			"value", k.Value, "default_output", defaultOutput, "weekly", k.Weekly,
			"special_events", k.SpecialEvents, "dates", k.Dates, "times", k.Times,
			"purpose", k.Purpose, "behaviour", k.Behaviour, "links", k.Links))
	}
	events := make([]any, 0, len(schedule.EventKinds))
	for _, e := range schedule.EventKinds {
		fields := make([]any, 0, len(e.Fields))
		for _, f := range e.Fields {
			fields = append(fields, obj("name", f.Name, "values", f.Values))
		}
		events = append(events, obj("name", e.Name, "type", e.Type, "purpose", e.Purpose,
			"in_calendar", e.Name != "reference", "fields", fields))
	}
	return obj("schedule_kind_count", len(kinds), "schedule_kinds", kinds,
		"event_kind_count", len(events), "event_kinds", events,
		"priority", "special events are tried top of the list first; the first one active at a "+
			"time wins, and all of them come before the weekly schedule",
		"cleanup", "with cleanupExpiredEvents (default true) the station deletes a special event "+
			"whose date or date range has passed; other kinds never expire",
		"export_import", exportImport), nil
}
