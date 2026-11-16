### Added — mcp-n4: `n4_event_report`, alarm times and the source's history peak in one call (retro row P3)

- `n4_event_report(source | source_glob, history_id, since, until, margin_minutes=15, max_events=10)` returns, per
  alarm event of the source (newest first), the alarm database times (`alarm_time`, `normal_time`, `ack_time` as
  ISO-8601 with the station's offset, `acked_by`) and the `n4_read_history` summary (count, min, max with its time,
  first, last) of the source's history over `[alarm_time - margin, normal_time + margin]`, up to now while the alarm
  is active. The history is `history_id` or the history extension beside the alarm extension, found once per source;
  a source with no history gets a null `summary` and a `summary_note`, never a failed report. Read-only.
  "A cuanto llego" took six steps in the 2026-11-16 client A over-amps retro.
