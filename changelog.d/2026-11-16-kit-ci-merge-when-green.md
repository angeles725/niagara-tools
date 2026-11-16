### Added — merge-when-green, ci-local --changed, the here-document bats lint, the host-budget free-space floor
- `scripts/merge-when-green.sh <pr> [--rebase|--squash|--merge]` (`go/cmd/merge-when-green`, `go/internal/mergewait`): waits for the check runs of the PR's exact head commit and merges it with `gh pr merge --match-head-commit`; refuses a draft (exit 4) and a branch that lacks the current base (exit 5), stops when the head moves (exit 6), re-runs once a check that failed on a transient download error, prints the log tail of a failed check (exit 1), 7 on timeout, 8 when gh refuses the merge. Runs under bash and zsh.
- `scripts/ci-local.sh --changed [--range <from>..<to>]`: the pinned shellcheck, lint-bats-errexit, the new here-document lint and `gofmt -l` over the changed files only (`ci-steps changed`); the pre-push gate runs it on every pushed range.
- `ci-steps lint-bats-heredoc <file|dir>...`: flags a here-document line in a .bats file that starts with the test keyword (bats 1.10 reads it as a duplicate case); a new `lint-sweeps` step of ci.yml runs it over `tests/`.
- host-budget `--disk-min-free <size>` (40G, 500M, …): when given, a mount FAILs below that much free space instead of above `--disk-pct`, and the row names the free bytes and the floor; `scripts/ci-local.sh` passes 40G.

### Changed — CI on drafts, the pinned local bats, the retro refusal text, preflight.bats speed
- ci.yml runs every job on a draft pull request too.
- `scripts/ci-local.sh` runs a bats step with the bats-core version ci.yml installs (sha256-checked, cached) when the bats on PATH is another version; its helpers moved to `scripts/lib/ci-local-tools.sh` and `scripts/lib/ci-local-changed.sh`.
- The pre-push retro refusal names the directories it reads (`build-n4-module-kit/retros/`, `ledger.d/`) and the two ways out.
- `tests/preflight.bats` runs a stub `lsof` in every case: serially 847 s → 15 s on the WSL host.
