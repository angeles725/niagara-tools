<!-- ledger: kit -->
<!-- date: 2026-11-14 -->
<!-- last-change-note: ng-deploy.sh is a plain launcher; the eight sourced bats cases are Go tests -->
log: 2026-11-14 · all-Go pass, retro go-ng-deploy-port Δ6 → the eight `tests/ng-deploy.bats` cases that sourced `scripts/ng-deploy.sh` with `BATS_TEST_MODE=1` are Go tests of `go/internal/ngdeploy` that run the whole deploy through `Main` (`verify_single_jar_test.go`: type count match, mismatch, the `<types>` wrapper, no `BUILD_ID`, a cache-buster of another build; `backup_archive_test.go`: the lightweight default, `FULL_BACKUP=1` and the keep-3 purge, read from the archive the host's tar wrote). Each was seen RED under a mutation of the behaviour it pins. The launcher lost its `BATS_TEST_MODE` branch and the `verify_jar`, `verify_cachebuster` and `backup` shims; `steps.go`, `StepEnv` (`NG_DEPLOY_STEP`) and their two messages are deleted.
open_issue: retro go-ng-deploy-port Δ1-Δ5, Δ7, Δ8 pending
