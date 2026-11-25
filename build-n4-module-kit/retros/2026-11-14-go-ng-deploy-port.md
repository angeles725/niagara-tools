<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 13 -->
# 2026-11-14 · kit · go-ng-deploy-port

**Session**: "all tool logic in Go" pass, wave 3: `scripts/ng-deploy.sh` becomes a launcher over `go/cmd/ng-deploy`; the operator env file is read by `go/internal/envfile` instead of being sourced.
**Delta count**: 13

## What happened
`ng-deploy.sh` (726 lines) was ported phase by phase with a differential of 237 cases (stdout, stderr, exit code,
stub calls with cwd, argv and environment, resulting files): 191 identical, 46 declared, 0 undeclared. The declared
rows are the env file no longer being executed as shell (plain assignments, `$NAME`, `${NAME}` and a leading `~`
are read as bash read them; anything bash would have run is exit 10 with file and line), four false successes
turned into refusals, and "could not check" cases that now refuse. The port also showed thirteen defects of the
bash tool that were kept byte-identical; the deltas below group them.

## Evidence
- The env file is read after the flags, so a key named like an option variable (`MODE`, `NO_GATE`, `NO_BACKUP`,
  `ACCEPT_LOSSY`, `KEEP_N`, `VERIFY_MODULE_BIN`) silently replaces the flag and can switch the gate off; kept in
  `go/internal/ngdeploy/envload.go`. `[ev: 99db3b32]`
- The copy step copies the rt jar, then the ux jar: a ux failure is exit 40 with half a deploy in place; kept in
  `go/internal/ngdeploy/copy.go`. A failed `git rev-parse HEAD` or an unwritable `.last-deploy-sha` leaves the
  deploy a success, and its content reaches `git cat-file` / `git diff` unvalidated (a leading `-` is read as an
  option). `[ev: 99db3b32]`
- A jar with no `<type ` line is exit 1 with no message even when 0 types are expected; a failed unzip exits with
  unzip's status and no ERROR line; a missing built jar skips the version precheck; purge failures and, outside
  strict mode, a failed `git diff` are silent; `--env-file <directory>` is ignored. `[ev: 99db3b32]`
- The `build:` and `gate:` lines join task and jar names with newlines (the script's IFS); strict mode aborts on
  annotation changes even when `--with-slotomatic` is given; backup names have one-second resolution; a CRLF env
  file keeps the carriage return in its values. `[ev: 99db3b32]`
- Eight bats cases source the script with `BATS_TEST_MODE=1` and call `verify_jar`, `verify_cachebuster` or
  `backup`; the launcher keeps those three as shell shims over the binary (`NG_DEPLOY_STEP`). `[ev: 267b3eca]`
- `toolbelt/lib/vendor-version.sh` and `lib/slotomatic-skip.sh` have no shell caller left; only their own bats,
  two Go `TestMatchesTheShellLib` tests and live text that names them as the implementation keep them.
  `[ev: 718f1fb3]`
- The env-file reading and the new exit-10 refusals change the flag, exit and env surface that `CONTRIBUTING.md`
  puts under SemVer; `VERSION` was not bumped in the port. `[ev: 51d755ae]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-15 (the flag wins without a refusal; the refused keys are `NO_GATE=1` and `NO_BACKUP=1`). A flag given on the command line wins over the env file: the env file may not assign an option variable the operator also set by flag, and a key that would switch the gate or the backup off is refused with file and line unless the matching flag is given | `go/internal/ngdeploy/envload.go` · `tests/ng-deploy.bats` | `[ev: 99db3b32]` |
| Δ2 | APPLIED 2026-11-15 (the previous pair is restored from the jars the copy moved aside, not from the backup, so `--no-backup` is covered too). The copy step is all-or-nothing: both jars are staged beside their targets and renamed, and a failure restores the previous pair; exit 40 never leaves one new jar and one old | `go/internal/ngdeploy/copy.go` | `[ev: 99db3b32]` |
| Δ3 | APPLIED 2026-11-14. `.last-deploy-sha` is validated as a full hex object name before it reaches git, and a deploy whose sha could not be read or written says so on stderr and exits non-zero after the copy | `go/internal/ngdeploy/annotations.go` · `copy.go` | `[ev: 99db3b32]` |
| Δ4 | APPLIED 2026-11-14. Every silent exit gets its ERROR line and its own documented code: a jar with no type line (and exit 0 when 0 types are expected), a failed unzip, a missing built jar before the version precheck, a failed purge, `--env-file` naming a directory, a failed `git diff` outside strict mode | `go/internal/ngdeploy/verify.go` · `version.go` · `purge.go` · `options.go` · `messages.go` | `[ev: 99db3b32]` |
| Δ5 | APPLIED 2026-11-14. Cosmetic and timing defects are fixed together: task and jar names joined by spaces on one line, strict mode satisfied by `--with-slotomatic`, backup names unique within one second, a trailing carriage return stripped from env values with a warning | `go/internal/ngdeploy/build.go` · `gate.go` · `annotations.go` · `backup.go` · `go/internal/envfile` | `[ev: 99db3b32]` |
| Δ6 | The eight sourced bats cases move to Go tests of `verify`, `backup` and the cache-buster, and the launcher drops `BATS_TEST_MODE` and the `NG_DEPLOY_STEP` shims. APPLIED | `scripts/ng-deploy.sh` · `go/internal/ngdeploy/steps.go` · `tests/ng-deploy.bats` | `[ev: 267b3eca]` |
| Δ7 | APPLIED 2026-11-14. `lib/vendor-version.sh` and `lib/slotomatic-skip.sh` are deleted with their bats once the cases are pinned in Go; the lines of `build.sh` help, `BUILD-LOOP.md`, `METHODOLOGY.md` and `build-verify.md` that name them point at `go/internal/vendorversion` and `slotomaticskip` | `build-n4-module-kit/toolbelt/lib/` · `BUILD-LOOP.md` · `METHODOLOGY.md` · `build-verify.md` | `[ev: 718f1fb3]` |
| Δ8 | The release that carries this port bumps `VERSION` as a behaviour change of the deploy surface, and `CONTRIBUTING.md` says which port outcomes (a refusal of a former success, a changed exit code) require the bump | `VERSION` · `CONTRIBUTING.md` (versioning) | `[ev: 51d755ae]` |
| Δ9 | APPLIED 2026-11-15 through the wording of the restore outcome (after Δ2 a failure on the second jar puts the first back: the line says the previous pair is back, or, when a jar moved aside could not be moved back, which jar is new, which is the previous one, and the `.ng-deploy-old` file that holds the previous bytes). A copy that fails on the second jar (exit 40) says which jar is already in the station and that a restart or a restore from the backup is needed; today the line names only the jar that failed | `go/internal/ngdeploy/copy.go` · `usage.txt` exit 40 | `[ev: verifier of PR #649, 2026-11-15: mode A, rt copied, ux cp failing: exit 40, stderr "ERROR copy failed for demo-ux.jar"]` |
| Δ10 | APPLIED 2026-11-15 (the failure before the copy is exit 10; 41 keeps meaning "copied"). A `.last-deploy-sha` that is a directory or cannot be written is found before the copy, not after it (exit 41 after the copy stays for a failure that only appears then) | `go/internal/ngdeploy/record.go` | `[ev: verifier of PR #649, 2026-11-15: the detection is silent, the deploy copies, then exit 41 "Is a directory"]` |
| Δ11 | APPLIED 2026-11-15 (the NOTICE has the shape of the malformed one and its own first clause, "names a commit git does not know": the id is a full one). A recorded id that is a full object name but that git does not know gets the same NOTICE as a malformed one, and strict mode refuses it; today it falls back to `HEAD~1` without a word and strict mode passes | `go/internal/ngdeploy/annotations.go` | `[ev: verifier of PR #649, 2026-11-15; same behaviour on the base]` |
| Δ12 | APPLIED 2026-11-17 (decided: refuse both, exit 10 with file and line; MODE=B unless `--mode`, VERIFY_MODULE_BIN unless `--no-gate`; the environment still sets VERIFY_MODULE_BIN: `go/internal/ngdeploy/envguard.go`). Decide whether an env file may still avoid the verify-module gate by other keys: `VERIFY_MODULE_BIN` (any executable stands in for the gate) and a `MODE` that has no gate (`MODE=B`) are read from the env file as before; only `NO_GATE=1` and `NO_BACKUP=1` are refused (Δ1) | `go/internal/ngdeploy/optiontable.go` · `envload.go` · `usage.txt` | `[ev: verifier of the Δ1 branch, 2026-11-15: MODE=B or VERIFY_MODULE_BIN in the env file, no flag, deploys without the gate]` |
| Δ13 | Two deploys at once to one modules directory are not serialised: both stage and move aside under the same names, and the line of the run that loses can describe a station the other run changed; a lock on the modules directory for the copy step would refuse the second run | `go/internal/ngdeploy/copy.go` | `[ev: verifier of the Δ2 branch, 2026-11-15: two concurrent runs, the losing one reports a state that is not the station's]` |

## Lessons
- A tool that sources an operator file runs it: read assignments, expand the plain forms, refuse the rest with file
  and line, and never print the line.
- Check real operator files (by counting constructs, never by printing them) before narrowing what a config reader
  accepts; a stricter reader that breaks a working configuration is a regression, not hardening.
- Under `set -u` an unset variable in a sourced file aborts; a port keeps that as a refusal that names the variable.
- `[[ "$n" -ne "09" ]]` is an arithmetic error that reads as false: numeric settings are validated once, at load.
- A script kept sourceable for its tests keeps shell logic alive; port the tests with the tool.

---
**Status**: FOLDED — INDEX row generated from the header: `| 2026-11-14-go-ng-deploy-port.md | kit | 2026-11-14 | folded | 13 |`
