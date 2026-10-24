<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 4 -->
# 2026-10-24 · kit · r10-golden-runner

**Session**: niagara-research quality round 10 item R10-D14, branch `r10/golden-runner` from origin/main 11af042f
**Delta count**: 4

## What happened
Three Go ports (preflight, commissioning-verify, new-retro) each carried a private golden loop and a private
recorder because `golden.RunMain` compared only exit, stdout and stderr. Reviews kept finding the same gaps in
them: a recorder that could not prove it ran the bash, silent root skips of mode-000 cases, host PATH/HOME leaking
into the run, written files only partly pinned, no cleanup trap. One `golden.Runner` and one bash
`golden/record/record.sh` now hold that hardening; the three packages replay through it and their loops and
recorders are deleted. `golden.RunMain` is a thin wrapper, so every lint golden also proves its run left the tree
untouched. Every recorded case of the three sets was re-recorded from its pinned bash and matched the converted
case byte for byte apart from the new `@kind` header (and canonical tree order / one empty-dir spelling).

## Evidence
- RED on a stub Runner: `TestRunnerTree/undeclared_write: problems "", want one about "changed a"`,
  `TestRunnerEnvIsHermetic: env [] leak ""`, `TestRunnerKind ... problems [], want bad=true`, `TestRootVerdict`
  `[ev: go/internal/lint/rules/golden/runner_test.go]`
- Re-recording: preflight 119/119, new-retro 33/33, commissioning-verify 77/77 recorded cases identical to the
  converted ones; Go replays 125, 44 and 88 cases (unchanged counts) `[ev: go/internal/lint/rules/golden/record/record.sh]`
- The checksum pin refused the first preflight recording until `lib/signing-block.sh` joined `EXTRA`: every
  file the script sources is pinned beside it `[ev: go/internal/preflight/testdata/record.env]`
- The one-off section converter first read the cases in text mode, which turned the CRLF of
  `edge-sample-prop-spacing` and `edge-flip-crlf-and-module-wins` into LF; re-recording new-retro showed the bash
  flip a different envelope, which exposed it before the cases were kept

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **A Go port's goldens replay through `golden.Runner` and record through `golden/record/record.sh`**: hermetic env, the whole post-run tree (`@after`/`@gone`), modes, symlinks, dirs, exact count, `@kind` pinned to `record.env`; no package keeps a private loop or recorder. | `BUILD-LOOP.md` § 5 What to port | `[ev: go/internal/lint/rules/golden/runner.go]` |
| Δ2 | **Prove a golden-format conversion by re-recording, not by re-reading**: convert the inputs, re-record from the pinned bash, `diff -r`; a converter bug that changes an input changes the bash output too. | `BUILD-LOOP.md` § 5 golden recordings | `[ev: go/internal/lint/rules/golden/record/README.md]` |
| Δ3 | **Pin every file the recorded script sources** (`EXTRA` in `record.env`): the checksum of the main script alone let a recording run without `lib/signing-block.sh`. | `BUILD-LOOP.md` § 5 golden recordings | `[ev: go/internal/preflight/testdata/record.env]` |
| Δ4 | **A golden case's environment is only what it declares** (PATH, HOME, LC_ALL defaulted): commissioning-verify cases ran with the host PATH and HOME; the wrappers' Go binaries are built once before the cases, not under the case PATH. | `BUILD-LOOP.md` § 5 golden recordings | `[ev: go/internal/commissioningverify/golden_test.go]` |

## Review notes (independent review of the preflight migration, applied to all three)
- Environment: every case now gets `HOME=<case dir>` and `LC_ALL=C` unless it declares them; the old private
  recordings had both unset. All recorded cases re-recorded byte-identical under the defaults; no migrated port
  reads HOME or LC_ALL (preflight's `bats-pf2-home` already declares `HOME=/nonexistent`; commissioning-verify adds
  `LC_ALL=C` to its lints itself), so no extra case was needed.
- `CHDIR=1`: the bash was always recorded with the case dir as cwd (all three old recorders `cd`ed there); preflight's
  old Go loop did not chdir. Every truth-* case (preflight 6, new-retro 11, commissioning-verify 11) passes with and
  without `CHDIR=1`, so none depends on the cwd by accident.
- The root opt-in is one variable now: `PREFLIGHT_GOLDEN_ALLOW_ROOT_SKIP` and `CV_GOLDEN_ALLOW_ROOT_SKIP` became
  `GOLDEN_ALLOW_ROOT_SKIP` (`golden.RootOptIn`); `golden.Unreadable` uses the same verdict instead of a silent skip.
- The deleted recipes stay readable in history: `git show a999bac8^:go/internal/preflight/testdata/record/cases.sh`
  and `git show ba3db050^:go/internal/newretro/testdata/record/cases.sh` (the dated r10-preflight-go and
  go-new-retro retros still name them); commissioning-verify's `TestRecord` is at
  `git show 4d9ca3cd^:go/internal/commissioningverify/record_test.go`.
- `recordTools` (preflight `golden_test.go`) is pinned to setup.sh's `TOOLS` by `TestRecordToolsMatchSetup`.

## Follow-ups
- `go/internal/wiringmap` (unmerged branch) still has a private loop: migrate it once it lands on main.
- `go/internal/slotcoverage` and `go/internal/verifymodule` already use `golden.RunMain` (no private loop) and now get
  the tree check; verifymodule's `TestUnreadableSource` comment says a chmod 000 cannot be spelled in a golden; with
  `(mode 000)` it can be a case.
- `toolbelt/new-retro.sh` header still names `go/internal/newretro/testdata/record/record.sh`.

## Lessons
- Python's text mode rewrites CRLF on read: a converter of byte-exact fixtures opens files with `newline=''`.
- A twin materialization is the simplest honest "before" snapshot: the run dir cannot be unlocked to read it.
