---
title: Commissioning close-out
description: Close a job on a station: find what is still wrong or unfinished below its folder, fix or justify each item, save with persistence evidence.
arg station!: Name of the configured station (as n4_describe_session lists it)
arg scope!: Slot ORD of the job's folder or driver network, e.g. station:|slot:/Drivers/ModbusNetwork
---
# Commissioning close-out: {{scope}} on {{station}}

1. `n4_describe_session`, then `n4_connect` to {{station}}; confirm `stationName`, `version` and `tier` (with `tier_writes: refused`, plan the fixes as dry runs only).
2. `n4_commissioning_report(scope="{{scope}}")`, adding `expect_histories` and `expect_alarms` when the job says which points must log or alarm (e.g. `{"kinds": ["numeric"], "name_pattern": "*Temp*"}`). Read `summary` first.
3. For each section with findings, fix or justify every item: `n4_read_slots` on the item's `path` shows the detail. Raise `limit` or narrow the scope while `more` is above zero.
4. Each fix is a write tool call with its default dry run (`n4_set_slot`, `n4_add_history`, `n4_add_alarm`, `n4_create_link`, ...): show the plan, wait for the operator's approval, then repeat with `dry_run=false` and the `confirmation_token`. Report each `verdict` as returned; anything but `verified` is not success.
5. A "not checked" line is work left for a person, not a pass: read its reason, then check it another way (Workbench, `n4_bql_query`, `n4_spy_page`).
6. `n4_find_dangling_outputs` over {{scope}}, then `n4_commissioning_report(scope="station:|slot:/", sections=["station"])` for the station section.
7. `n4_save_station` and report `persisted` with its `evidence`; `unknown` is not success.
8. Report: the summary lines before and after, each fix as requested / observed / verdict with its `batch_id`, the justified items, the "not checked" lines and the save evidence. Close with `n4_session_retro_draft`.
