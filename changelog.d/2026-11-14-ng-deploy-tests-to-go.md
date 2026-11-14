### Removed — scripts/ng-deploy.sh can no longer be sourced for single steps
- `scripts/ng-deploy.sh` is a plain launcher: it builds `go/bin/ng-deploy` on demand and execs it. The `BATS_TEST_MODE=1` branch and the `verify_jar`, `verify_cachebuster` and `backup` shell functions it offered to a sourcing test are gone, and the binary no longer reads `NG_DEPLOY_STEP` (`go/internal/ngdeploy/steps.go` is deleted). A deploy run as a command is unchanged: same lines, exit codes and command lines.
- The eight `tests/ng-deploy.bats` cases that used those functions are Go tests of `go/internal/ngdeploy` (`verify_single_jar_test.go`, `backup_archive_test.go`); the three backup cases read the archive the host's `tar` wrote.

### Removed — the bash libs `toolbelt/lib/vendor-version.sh` and `toolbelt/lib/slotomatic-skip.sh`
- No script sourced them since the build and the deploy moved to Go. Their cases are tests of `go/internal/vendorversion` and `go/internal/slotomaticskip`; `tests/vendor-version.bats` and `tests/slotomatic-skip.bats` are deleted with them, and `toolbelt/lib` now holds only `werror.init.gradle`.
