### Fixed — ci-local --changed runs kit-links (retro row T25 of the 2026-11-15 coordination stretch)

- `scripts/ci-local.sh --changed`, which the pre-push hook runs, also runs `tests/kit-links.bats` when the change
  touches a kit doc, a `ledger.d` or a `changelog.d` file, on a snapshot of the change with the generated
  BUILD-STATE.md rebuilt there; a fragment that names a missing doc is refused at push, not first in CI.
- The host-budget gate of `ci-local.sh` moved to `scripts/lib/ci-local-host.sh` without a behaviour change.
