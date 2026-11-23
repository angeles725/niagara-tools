### Fixed — mcp-n4 small fixes

- `n4_connect`: a station-name mismatch explains that `about/serverName` is the host name, not the station's, and suggests the name the station answered as `expected_station` (issue #860).
- `n4_discover_devices`: `not_answering` lists only the registered devices whose instance is inside `low_instance..high_instance`; it listed every device under the network (issue #929).
- mcp-n4 skill and METHODOLOGY: rename is `n4_rename_component`, not a Workbench-only step (#872); the stale Workbench Device Manager / Edit dialog warning after a BOX write (#888); the inspect-then-decide steps for a create that answered "Unable to process request" (#864); the recipes to brief a writer that cannot reach the station and to audit overrides around a dashboard change (#894).
