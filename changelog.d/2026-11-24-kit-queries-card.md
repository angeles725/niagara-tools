### Added — `types/queries.md`, a card for reading data from module code (DOC-C)

- `types/queries.md`: which read to use (BQL, NEQL, a history or alarm query, a lease or a `Subscriber`), the Java
  recipe (the caller's Context, try-with-resources, `top` or a time window, off the engine thread), a cost table of
  what is indexed and what scans (history narrows only by `timestamp`; alarms by `openAlarms`, `ackPendingAlarms`,
  `uuid`, `source` and `timestamp`; no `skip`; `order by` materializes; rollup is a streaming fold; archive limit
  50000), history and alarm recipes, paging from the last timestamp plus 1 ms, the `add("name?")` /
  `DuplicateSlotException` / `linkTo` idioms, and a snippet bank from the course `ords/*` classes.
- `types/logic-authoring.md` § BQL from code: the lease caveat is replaced by its mechanism (read a leased proxy point
  on a later tick, not in the leasing callback); the Java lease is 60 000 ms; a pointer to the new card.
- `README.md` and `skill/SKILL.md` list the card the way `types/hx-px.md` is listed.
