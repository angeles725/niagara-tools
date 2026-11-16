### Changed — changelog-gate refuses a line added inside [Unreleased] outside a release

- `scripts/changelog-gate.sh` (and the pre-push gate that runs it in process) now FAILS a range that adds a
  non-blank line inside `CHANGELOG.md`'s `## [Unreleased]` unless the range is a release: it deletes a
  `changelog.d/` fragment (the `gen-changelog.sh --write` fold) or changes `VERSION` (the bump). The refusal holds
  next to a valid fragment and in a range that changes no behaviour, and names `changelog.d/<YYYY-MM-DD>-<slug>.md`
  as the fix. Before, such a line was an accepted carrier, and four open PRs that each used it conflicted on
  `CHANGELOG.md`. The missing-entry message no longer offers an `[Unreleased]` line.
