### Changed — `ci-local.sh --changed` runs every tree gate CI runs, and a test fails when a CI gate has no local twin (issues #983, #973)

- `scripts/ci-local.sh --changed` (and the pre-push hook) runs, for the files changed, the checks of the one table
  `go/internal/cisteps/gates.go`:
  - a `go/**/*.go` runs the file-size gate (`go test` of `go/internal/filesize`);
  - a live doctrine card runs the 120-character cardwidth gate that broke PR #978 in CI only;
  - a kit, mcp-n4-kit, script or fixture change runs `go test` of the Go packages that read it;
  - a change a ci.yml lint step watches replays that step as CI runs it (lint-guard-pins, the `gen-*` checks,
    sweep-build-state, scaffold-diff, bats-shard and the fixture checks);
  - every change runs `scripts/pr-size.sh` against origin/main, a WARN only: the `size:exception` label decides in CI,
    so any non-zero exit (over the limit, or no origin/main to measure against) prints WARN and never blocks the push.
- The summary counts every failed check, gates included (`ci-local: --changed: N check(s) failed`); the extra "1 more
  check failed (the file-size gate)" line is gone.
- `TestEveryCIStepAndTreeGateHasALocalTwin` (go/internal/cisteps) fails when a ci.yml run step or a Go package whose
  tests read the repository tree has no gate row, a `FullOnly` reason or a `NotTreeGates` reason, naming the step or
  package and where to add it. Before, every new gate had to be remembered in `ci-local.sh`. Pinned by
  `tests/ci-local-gates.bats` (CG1-CG7).
