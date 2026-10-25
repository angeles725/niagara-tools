### Added — a changelog gate in pre-push and CI

- `.githooks/pre-push` and the new `changelog` CI job run `scripts/changelog-gate.sh`: a range that changes `go/`,
  `build-n4-module-kit/`, `tools/` or `scripts/` (except `go/bin/`, `ledger.d/`, `retros/` and `BUILD-STATE(.src).md`) fails
  unless it adds a `changelog.d/<YYYY-MM-DD>-<slug>.md` fragment, adds a line inside `## [Unreleased]`, or carries a
  `Changelog: none (<reason>)` commit trailer. A malformed fragment or trailer fails any range. See CONTRIBUTING.md § 6.2.
