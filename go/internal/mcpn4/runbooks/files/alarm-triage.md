---
title: Alarm triage
description: Sort the open alarms of a station by source and class, size each event from its history, and propose (never perform) acknowledgements.
arg station!: Name of the configured station (as n4_describe_session lists it)
arg source: Slot ORD to narrow the triage to (a network, device or folder) = station:|slot:/
arg since: Start of the window, an ISO timestamp = the last 24 hours
---
# Alarm triage: {{station}}

1. `n4_describe_session`, then `n4_connect` to {{station}}; confirm `stationName`, `version` and `tier`.
2. `n4_list_alarm_classes`: the classes, their open and unacked counts and whether each has a recipient. A class without a recipient is a finding by itself: nobody is notified of its alarms.
3. `n4_read_alarms(source="{{source}}", since=<{{since}}>, state_counts=true)`: the open alarms, newest first, with `total`. Group them by `source` and `alarmClass`; a source with many records is one problem, not many.
4. For the top sources, `n4_event_report(source=<that source>, since=<{{since}}>)`: how far the value went during each alarm and when it returned to normal.
5. Classify each group: real process event, device or network down (check with `n4_station_health`), wrong limit or delay on the alarm extension (`n4_read_slots` on the extension), or a test/stale alarm.
6. Propose the acknowledgements as a list of uuids with the reason for each. Acknowledging is a write: only after the operator approves, `n4_ack_alarm(uuids=[...], notes="...")` with its dry run, then the confirmation token. For a whole group, `n4_ack_alarm(select={source: "<that source>", alarm_class: "<class>", before: "<time>"}, notes="...")`: show the operator the dry run's `plan.selection.records` (and `truncated`), then execute with its token, which acknowledges exactly those records. Never acknowledge an alarm whose cause is still active without the operator saying so.
7. Report: per group the source, class, count, first and last time, the event peak, the classification and the proposed action; the classes without recipients; the batch ids of any acknowledgement.
