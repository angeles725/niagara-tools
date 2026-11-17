### Added — n4_create_link batch form (field retro delta U11)

- `n4_create_link` takes `links`: 1 to 50 entries `{source_ord, source_slot, target_ord, target_slot}` under one
  dry run and one confirmation token (105 links had taken ~41 min one by one). Each entry is planned, inverted and
  read back exactly as the single form; the batch refuses a mix with the single-form arguments, the same link twice
  (keyed on the station's handles, so another ORD spelling of the same component is caught), an entry identical to
  an existing link, and any entry whose target is outside the write scope. The batch spends one write of
  `--max-writes` per link and is refused before its token when the remaining budget cannot hold it. The read-back
  reports `{source_ord, source_slot, target_ord, target_slot, requested, observed, verdict}` per entry; the batch
  takes the entries' verdict when they all agree, else `partial`. `n4_rollback` removes every link of the batch.
- The single form's plans and replies are unchanged. Its four arguments are no longer `required` in the schema, so
  a missing one is refused as a tool error with the same text (`missing required argument: source_ord`) instead of
  a -32602 JSON-RPC error.
