### Changed — `ci-local.sh --changed` runs every tree gate CI runs, and a test fails when a CI gate has no local twin (issues #983, #973)

- `scripts/ci-local.sh --changed` (and the pre-push hook) runs, for the files changed, the checks of the one table
  `go/internal/cisteps/gates.go`: `go test` of the Go packages whose tests read the repository (a `go/**/*.go` runs the
  file-size gate, a live doctrine card the 120-character cardwidth gate that broke PR #978 in CI only, a kit,
  mcp-n4-kit, script or fixture change the packages that read it), ci.yml's own lint steps replayed as CI runs them
  (lint-guard-pins, the `gen-*` checks, sweep-build-state, scaffold-diff, bats-shard and the fixture checks), and
  `scripts/pr-size.sh` against origin/main, a WARN only (the `size:exception` label decides in CI).
- The summary counts every failed check, gates included (`ci-local: --changed: N check(s) failed`); the extra "1 more
  check failed (the file-size gate)" line is gone.
- `TestEveryCIStepAndTreeGateHasALocalTwin` (go/internal/cisteps) fails when a ci.yml run step or a Go package whose
  tests read the repository tree has no gate row, a `FullOnly` reason or a `NotTreeGates` reason, naming the step or
  package and where to add it. Before, every new gate had to be remembered in `ci-local.sh`. Pinned by
  `tests/ci-local-gates.bats` (CG1-CG6).
