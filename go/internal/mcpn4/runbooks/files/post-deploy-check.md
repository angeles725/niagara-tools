---
title: Post-deploy check
description: After a module deploy and station restart, confirm the station runs the new jars and nothing regressed (read-only).
arg station!: Name of the configured station to check (as n4_describe_session lists it)
arg scope: Slot ORD of the part of the station the deploy touched = station:|slot:/
arg module: Module the deploy changed (name as in its jar) = the deployed module
---
# Post-deploy check: {{station}}

Read-only runbook. Report every value as the tools return it; a failed read is "not checked", never a pass.

1. `n4_describe_session`, then `n4_connect` to {{station}}. Confirm the reported `stationName`, `version` and `tier`.
2. `n4_station_runtime`: `restart_needed` must be `false`. Find {{module}} in `loaded_modules` and compare its `version` with the deployed jar. Any entry in `mismatches` (`version_differs`, `modified_after_boot`, `loaded_not_on_disk`, `not_loaded_modified_after_boot`) means the station is not running what was deployed: stop and report it.
3. `n4_station_health(base="{{scope}}")`: compare `points`, `networks` and `devices` (`total`, `ok`, `by_flag`) with the figures taken before the deploy, if the operator has them; any new `fault`, `down` or `stale` count is a finding.
4. `n4_read_alarms(source="{{scope}}")`: list the alarms raised since the restart (`since` = the boot time from step 2).
5. `n4_commissioning_report(scope="{{scope}}", sections=["points", "links"])`: read `summary` first; `n4_find_dangling_outputs` over {{scope}} for outputs nothing reads.
6. Report: station, version, tier, `restart_needed` and the {{module}} version, the health deltas, new alarms, the commissioning summary lines and every "not checked" with its reason.

Do not write anything in this runbook. A fix found here is a new job with its own dry run and confirmation token.
