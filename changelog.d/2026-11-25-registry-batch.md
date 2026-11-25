### Changed

- `n4_histories_audit` (#906): writes progress lines to the operator's progress file (`--progress-file`, the
  channel `n4_inventory` uses; nothing is written without it): one line per stage (`extensions`, `points`,
  `counts`, and `flat` with `flat=true`) and one per 10 finished histories of the counts and flat stages
  (`counts 10/90`). Each line carries the requests made so far in `rows` and `elapsed_ms`. The reply is not
  changed (no new field, no new request). The bounded concurrency (4) was done in #967. The combined
  `select min(value), max(value), count(*)` read is not done: the window's min/max skip records with a null
  status and read the window only, while the capacity count reads every record of the history, and the flat
  check's count reads the range before the window, so one query cannot give any of them without changing a
  reply field or the near-capacity and flat decisions. Tests `TestHistoriesAuditWritesAProgressLinePerStage`,
  `...EveryTenHistories`, `TestHistoriesAuditFlatWritesAFlatProgressLine`,
  `TestHistoriesAuditReplyIsTheSameWithAndWithoutTheProgressFile`.
- `n4_rollback chain=true` and `n4_remove_component compact=true` follow-ups (#971): a create's rollback after a
  chain no longer refuses on the later batches the chain removed with their component (they stay "not undone" in
  the journal; the later-work check now treats a batch journaled between a create and its completed, non-failed
  rollback that wrote only under the created component as gone; work journaled after the rollback and a failed
  rollback leave them as later work). Tests `TestLaterBatches*` and
  `TestRollbackOfAnAncestorCreateAfterAChainDoesNotRefuseOnVanishedBatches`. Already done, now pinned: item 1
  (a chain plan carries the note "the rollback itself has no automatic inverse for slot restores or removals" and
  records no inverse, `TestChainRollbackPlanHasNoInverseAndSaysSo`), item 3 (the note for a multi-component
  create, `TestRollbackChainOnAMultiComponentCreateSaysItFollowsNothing`), item 4 (compact keeps
  `outgoing_links_broken` listed and the plan hash, `TestCompactWithAcknowledgeLinksKeepsTheOutgoingLinksListed`),
  item 5 (`TestCompactSummaryCountsOutgoingLinksAndInputsLinkedFromOutside`).
