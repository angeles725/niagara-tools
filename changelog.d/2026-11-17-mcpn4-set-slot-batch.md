### Added — n4_set_slot batch form (OD040)

- `n4_set_slot` takes `writes`: 1 to 50 entries `{ord, slot, value, value_type, status?, simple_type?}` under one
  dry run and one confirmation token (overamps retro row M6, backlog OD040). Each entry is planned and read back
  exactly as the single form; the batch refuses a mix with the single-form arguments, the same (component, slot)
  twice, and any entry outside the write scope. The batch spends one write of `--max-writes`, as `n4_set_tags` does. The read-back
  reports `{ord, slot, requested, observed, verdict}` per entry, `verified` only when all are; `n4_rollback`
  restores every entry; `incoming_links` carries each entry's links with its `target_ord`.
- The single form's plans and replies are unchanged. Its four arguments are no longer `required` in the schema, so
  a missing one is refused as a tool error with the same text (`missing required argument: ord`) instead of a
  -32602 JSON-RPC error.
