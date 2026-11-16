### Changed — BREAKING: mcp-n4 refuses unknown arguments and enum violations in every tool (retro row M11)

- BREAKING: every mcp-n4 tool now refuses an argument its `inputSchema` does not declare with JSON-RPC -32602
  `unknown argument '<name>' for <tool>`, plus `; did you mean '<name>'?` when one declared argument is close
  (`capacity_records` -> `capacity`), and a value outside an `enum` with `argument <name> must be one of a, b, c;
  got '<value>'` (an enum list names the item, `sections[1]`). Before, an unknown argument was silently dropped
  (`capacity_records: 50000` on `n4_add_history` kept the 500-record default) and enums were checked only by some
  tools, as a tool error with their own wording. A caller that sent a dead argument now gets an error instead of a
  call that ignored it. One shared validator (`go/internal/mcpn4/jsonrpc`) does it before the session gate, so
  nothing reaches a station. Objects a tool keeps open stay open (`alarm_values` through `additionalProperties`;
  `expect_histories`, `expect_alarms`, `wire_sheet`, `weekly` and the `n4_set_schedule` / `n4_set_tags` list items
  declare no properties); a nested object is checked only where its schema declares `properties`. A refusal echoes at
  most 64 characters of the value or argument name, then `…`. A did-you-mean never offers a key the unknown one only
  starts unless it covers half of it (`all` is not `allow_ad_hoc`). `n4_read_alarms` keeps `source_state: ""` and
  `ack_state: ""` as no filter: the empty string is now declared in their enums.
- `n4_neql_query`: the no-from rule of its `bql` select now matches `from` between any BQL space (a decoded tab,
  newline or no-break space, `$09from$09`, `$0afrom$0a`, `$u00a0from$u00a0`) and at the end, as the table-base check
  does; before only ` from ` was refused (the station answered 404 anyway).
