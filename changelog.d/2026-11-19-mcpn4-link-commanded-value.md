### Added — mcp-n4: `n4_create_link` dry run states what each link will command

- The dry run lists `link_sources`, one entry per link (single and batch form):
  `{source_ord, source_slot, target_ord, target_slot, value, status}`, the value and status the
  source slot holds now, which the link drives into the target as soon as it is made.
- An `{ok}` source linked into a writable's control input (`in1`..`in16`, `fallback`) adds a
  `note`: "this link will command <target> to <value> {ok} as soon as it is made". Linking 28
  schedule copies at false / 0.0 `{ok}` would have switched units OFF and set setpoints to 0
  (delta P2, live near-miss 2026-11-19); only a manual read of `out` caught it.
- `link_sources` is volatile and outside `plan_hash` (a new dry-run-only plan preview), so a
  source whose value moves between the dry run and the execute keeps the token valid. The
  source is now loaded at depth 2 (the same single read) to see its slot's value and status.
