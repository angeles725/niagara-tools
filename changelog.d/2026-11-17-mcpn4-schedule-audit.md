### Added — mcp-n4: `n4_schedule_audit`, every schedule's next event, holidays and links (station-audits S12)

- `n4_schedule_audit(base, horizon="7d", max_schedules=500)` reads, in one read-only call, every boolean, numeric,
  enum and string schedule and every calendar below `base` (`select slotPath, knobCount from
  schedule:ControlSchedule`): per schedule `out_value`, `next_time`, `next_value`, `special_events` `{count,
  upcoming}` (the events, or a calendar's dates, whose first day falls in the horizon from the station's date, with
  references followed to their calendar and `calendars: [{ord, resolves}]`) and `linked_targets` (the knob count).
  `unlinked` lists the non-calendar schedules nothing reads, `no_next_event` those whose next time is null; a value
  that cannot be read is null with a note, and a rule the audit does not evaluate is noted, never guessed.
