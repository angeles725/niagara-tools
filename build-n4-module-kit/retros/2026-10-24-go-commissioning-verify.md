<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 4 -->
# 2026-10-24 · kit · go-commissioning-verify

**Session**: niagara-research quality round 10, branch `r10/commissioning-verify-go` from origin/main f86eeddf
**Delta count**: 4

## What happened
`toolbelt/commissioning-verify.sh` (bash orchestrator + one awk values-owed reader) is now the thin `go/go-tool.sh`
wrapper over `go/cmd/commissioning-verify` (`go/internal/commissioningverify`), a standalone `cli.Main` command like
bog-audit. The four per -rt/src source lints still run through their toolbelt wrappers: the wrapper exports its own
unresolved directory as `COMMISSIONING_VERIFY_TOOLBELT`, so a stubbed toolbelt copy (`tests/helpers/stub-toolbelt.bash`,
CV-ltf4/CV-ltf5) keeps running its stubs while `here=` still resolves the real path for `go/` (GWS3). bog-audit runs in
process through `go/internal/bogaudit` (Out and Err into one buffer, as `2>&1` did). Golden cases reuse
`rules/golden.Read`; the test tree adds `(symlink)`, `(dir)`, `(mode 000)` sections and `toolbelt/<member>` stubs.

## Evidence
- RED (package missing) → GREEN per piece: owedRows, sourceDirs/run, Main `[ev: go/internal/commissioningverify/command_test.go TestMainEnvErrors]`
- 85 goldens recorded from the bash tool (every bats run plus usage, env, discovery, stub-exit, station and values-owed
  edges): Go byte-identical on stdout, stderr and exit, and again through the switched wrapper `[ev: go/internal/commissioningverify/golden_test.go TestGolden]`
- client station A config.bog, three own modules, with and without --strict (6 runs), bash vs Go after T1-T4: 2 runs
  byte-identical, 4 differ only by T2 (4 CHECK19 link-direction WARN rows the bash fold dropped, and the WARN count)
- Truth fixes T1-T4 (Δ1-Δ4 below), each one commit with its RED, re-recorded or new goldens and a bats pin `[ev: tests/commissioning-verify.bats CV-bog-skip]` `[ev: tests/commissioning-verify.bats CV-bog-warn]` `[ev: tests/commissioning-verify.bats CV-src-failclosed]` `[ev: tests/commissioning-verify.bats CV-owed-unreadable]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | DONE (T1, this branch): the bog-audit exit-3 SKIP row named python3 (bog-audit is Go); it now names what exit 3 means (unreadable or malformed bog, no component of the module) | `go/internal/commissioningverify/station.go` | `[ev: go/internal/commissioningverify/testdata/truth-edge-bog-nocomponent.txt]` |
| Δ2 | DONE (T2, this branch): the CHECK13/15-20 fold relayed FAIL and PASS only, so the WARN-only CHECK15 sensor-crossed and CHECK19 link-direction never reached the punch-list (client station A: 4 hidden link-direction WARNs); every folded check relays FAIL, WARN and PASS | `go/internal/commissioningverify/station.go` | `[ev: go/internal/commissioningverify/testdata/truth-edge-bog-check19.txt]` |
| Δ3 | DONE (T3, this branch): config-sanity, status-parity and recovery-path PASSed on a non-zero exit with no row (126, 127, a crash); the link-target-flags fail-closed rule now holds for all four | `go/internal/commissioningverify/source.go` sourceChecks | `[ev: go/internal/commissioningverify/testdata/truth-edge-stub-cs-exit127.txt]` |
| Δ4 | DONE (T4, this branch): an unreadable values-owed table was a PASS row beside an awk error; it is exit 3 before any row | `go/internal/commissioningverify/command.go` Main | `[ev: go/internal/commissioningverify/testdata/truth-owed-unreadable.txt]` |

## Lessons
- An orchestrator whose tests stub toolbelt members needs the invoked (unresolved) toolbelt directory passed to the Go
  binary; resolving it like `here=` would silently run the real lints instead of the stubs.
- A fold that lists statuses per check drops whatever status a check later starts emitting; relay a fixed status set.
- A structural pin over a ported script goes vacuous at the switch: RM45 grepped commissioning-verify.sh for the bash
  `set -u` array idiom, so it now covers report-module.sh only (mutation RM-setu).
- Review follow-ups applied in this unit: goldens re-record reproducibly from the pre-port bash (`TestRecord`,
  `testdata/README.md`), declared-fix cases are named `truth-*` and counted exactly, a mode-644 stub is a real EACCES
  case beside the ENOEXEC (no `#!`) case, a mode-000 case fails as root unless opted out, and the lints get an
  injected PATH/HOME-only environment in tests.
- Follow-up (pre-existing, not changed here): the golden cases run the real toolbelt wrappers, which rebuild
  `go/bin/<tool>` on demand inside the live tree (`go/go-tool.sh` `go_tool_bin`); a test run can therefore write
  `go/bin`. A hermetic option is a per-test GO_DIR/bin or a prebuilt binary handed to the wrappers.
- Δ1-Δ4 are applied in this unit as code fixes; the retro stays pending until a fold review decides whether any of them
  becomes doctrine.

---
**Status**: PENDING
