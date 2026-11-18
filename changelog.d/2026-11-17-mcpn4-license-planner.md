### Added — mcp-n4: licence planner in `n4_license_info`, device limit check in `n4_create_component` (field retro delta U8)

- `n4_license_info` adds `usage`: per global capacity rule (`devices`, `points`) `used`, `limit`,
  `headroom` and `excluded_modules`. `used` counts as the station's capacity recount does
  (`driver:Device` / `driver:ProxyExt` components, less the modules the `tridium:globalCapacity`
  feature lists in `excludedDevices` / `excludedPoints`), with one BQL select per rule. An unlimited
  or absent limit is `limit: null`; a count that cannot be read is `used: null` with a `note`.
- `n4_license_info(plan: {add_devices?, add_points?})` adds `plan_result: {fits, over: [{rule, by}]}`;
  `fits` is `null` when a named rule's count or limit is unknown, never guessed. Without `plan` the
  reply keeps its fields.
- `n4_create_component` refuses, in the dry run, a `driver:Device` subtype whose creation would take
  the counted devices over `device.limit`. Non-device types are not checked; an unreadable licence or
  count is never a refusal (a fixed plan note says the limit was not checked).
