### Changed

- `toolbelt/retro-issues.sh` (Go: `go/internal/retroissues`), retro R17 delta D4 (issue #994): two more
  correctness advisories. (1) Alias collisions fail loudly: two retros that end up under one name (an `--alias`
  that spells the stem of a kit retro, or two files of one stem under one alias) are refused with exit 3 and
  `retros #N and #M of the command line share the name <name>` before any `gh` call; they would have shared every
  marker, so the second read as already filed. The message names only the alias, never an outside stem. (2) The
  closing line of `--apply` counts only the issues `gh` took: a probable duplicate or already-fixed delta whose
  comment or `retro-delta` label edit failed is no longer counted (only a created issue was held to this).
- Already done, pinned by existing tests: NaN and out-of-range `--dup-threshold` are refused
  (`TestDupThresholdRejectsNaN`), the duplicate path adds the `retro-delta` label to the issue it comments on
  (`TestProbableDuplicateOpenIsCommentedNotCreated`, `issue edit 30 --add-label retro-delta`), and two aliases of
  one value are refused (`TestTwoStemsCannotShareAnAlias`). New tests: `TestAnAliasCannotCollideWithAKitStem`,
  `TestTwoRetrosOfOneStemCannotShareAnAlias`, `TestApplySummaryDoesNotCountADuplicateWhoseCommentFailed`,
  `TestApplySummaryDoesNotCountADuplicateWhoseLabelFailed`.
