<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 6 -->
# 2026-11-14 · kit · go-build-module-port

**Session**: "all tool logic in Go" pass, wave 2: `toolbelt/build.sh` (508 lines of bash) becomes a launcher over `go/cmd/build-module`.
**Delta count**: 6

## What happened
`build.sh` was ported to Go with a differential of 233 cases against the original (stdout, stderr, exit code, every
stub call and the resulting tree): 224 byte-identical, 9 differing only in diagnostics bash itself printed. The
port had to move two helpers out of `preflight` (`fstype`, `findwalk`) and create two shared packages
(`vendorversion`, `slotomaticskip`). Reading every branch to reproduce it showed six gaps (two of them raised by the native review of the port) that the port kept
byte-identical on purpose, because a port that also changes behaviour cannot be proven by a differential.

## Evidence
- The settings fallback pattern `niagaraPluginVersion[^)]*getOrElse("X")` cannot match the documented form
  `gradleProperty("niagaraPluginVersion").getOrElse("X")`: a `)` sits between the two, so the plugin-version
  default is never read from `settings.gradle.kts` in that form. Kept as is in `go/internal/buildmodule/patterns.go:40`.
  `[ev: 0b48eef7]`
- A module root that cannot be entered ends with `build.sh: cannot resolve module root: ` and no path: the failed
  command substitution had already emptied the variable. Pinned as is by `inputs_test.go:150`. `[ev: 0b48eef7]`
- The Go chain still runs `cp -p`, `mkdir -p`, `mktemp`, `unzip` and `df` as external commands: three bats cases
  stub `unzip` on PATH and one asserts `cp`'s own "Permission denied" text, so replacing them would have changed
  tests that had to pass unchanged. `[ev: 4ab3316e]`
- Turning the script into a short launcher broke a live doctrine cite by line number
  (`BUILD-LOOP.md` pointed at `build.sh:282`); `kit-cite-resolve --strict` caught that one, while a prose mention
  of `toolbelt/build.sh:15,:82-88` in the same card is not checked and stays inaccurate. `[ev: 5be7f193]`

- The deployed-baseline content hash concatenates the bytes of every non-META-INF entry in name order and never
  hashes the names, so a renamed entry with the same bytes hashes equal; an entry `unzip -p` cannot print
  contributes whatever it did print and the hash still counts as verified. Both come from the bash original.
  `go/internal/buildmodule/baseline.go:60-78`. `[ev: 0b48eef7]`
- A signal during the run removes the temporary directory and exits; the pre-build copy of the modules/ jar lives
  in that directory, so an interrupt after gradle replaced the jar leaves no baseline to restore.
  `go/internal/buildmodule/proc.go:110-119`. `[ev: 0b48eef7]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-14 (2fcf384e, be5a4ec5). The plugin-version fallback reads `gradleProperty("niagaraPluginVersion").getOrElse("X")` (and the `providers.gradleProperty(...)` form): fix the pattern, with a RED test on the documented settings file the scaffold emits | `go/internal/buildmodule/patterns.go` · `fixtures/` settings | `[ev: 0b48eef7]` |
| Δ2 | APPLIED 2026-11-14 (4324618b). An unenterable or missing module root names the path the operator typed in the message, and the test pins the path | `go/internal/buildmodule` (inputs, messages) | `[ev: 0b48eef7]` |
| Δ3 | The build chain does file work natively (copy with mode and times, mkdir, temp dir, zip listing, free space) instead of exec'ing `cp`, `mkdir`, `mktemp`, `unzip` and `df`; the bats cases that stub those commands move to Go tests with an injected filesystem error | `go/internal/buildmodule` (mirror, drift, baseline) · `tests/build-sh.bats` | `[ev: 4ab3316e]` |
| Δ4 | Live doctrine cites a Go symbol or a test name, never a line number of a script; `kit-cite-resolve --strict` also checks `file:line` pairs written in prose and reports one past the end of the file | `toolbelt/kit-cite-resolve.sh` (Go `internal/kitcite`) · `CONTRIBUTING.md` evidence tokens | `[ev: 5be7f193]` |
| Δ5 | APPLIED 2026-11-14 (7b13b65c, 3c6a86bc). The baseline content hash covers entry names and sizes as well as bytes, and an entry that cannot be read makes the comparison "cannot verify" (exit 52), never a pass | `go/internal/buildmodule/baseline.go` · drift tests | `[ev: 0b48eef7]` |
| Δ6 | APPLIED 2026-11-17 (`go/internal/buildmodule/signal.go`: from the baseline snapshot until the chain passed; `TestASignalBeforeTheGateRestoresThePreBuildJars`). On SIGINT/SIGTERM/SIGHUP after the jar copy started, the chain restores the pre-build modules/ jar before removing its temporary directory, and says so | `go/internal/buildmodule/proc.go` · `mirror.go` | `[ev: 0b48eef7]` |

## Lessons
- A port is proven by a differential only if it changes nothing; defects found on the way are filed, not fixed in
  the same change.
- A test that stubs a system command on PATH pins the implementation to that command. Inject the failure instead.
- A cite by line number is a cite to a file version. Cite a symbol.
- Helpers private to one tool get re-implemented by the next port; move them to a shared package the first time a
  second caller appears.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-11-14-go-build-module-port.md | kit | 2026-11-14 | pending | 6 |`
