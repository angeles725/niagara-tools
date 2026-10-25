### Added — gen-changelog folds changelog.d/ at release time

- `scripts/gen-changelog.sh --write` appends every `changelog.d/<YYYY-MM-DD>-<slug>.md` fragment, in date+slug
  order, to the end of `CHANGELOG.md`'s `## [Unreleased]` section and deletes the folded fragments; it writes
  nothing when a fragment is malformed. The release runs it first (CONTRIBUTING.md § 5 step 2).
- CI runs `scripts/gen-changelog.sh --check`, so a fragment the fold would refuse fails before it merges.
