### Added — mcp-n4: `n4_replicate_points` (bulk replication under one token)

- `n4_replicate_points(source_parent_ord, points, targets)` copies M points of one parent onto N
  target parents: N x M adds in one batch under one confirmation token (at most 200), each the
  body `n4_copy_component` would add (address and `dataType` kept), with per-target `copies[]`
  name and `proxyExt/dataAddress` overrides. Each add spends one `--max-writes` write; the plan
  notes the count and a replication larger than `write_budget.remaining` is refused before any
  token. A client B field job had spent 630 calls and 39 minutes on 112 points.

### Changed — mcp-n4: copy and replicate dry runs note a non-null fallback

- `n4_copy_component` and `n4_replicate_points` note every writable point whose `fallback` is not
  null: each copy commands its device with that value on first communication. A note, never a
  refusal.
