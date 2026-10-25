<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-25 · kit · r10-rnt-reuse-nre-proof

**Session**: niagara-tools quality round 10, kit tool G33 `run-niagara-test run`, proof that the NRE wrote into the
scratch `niagara_user_home` on a `--reuse-home` boot, branch `r10/rnt-reuse-nre-proof` from 4ebb0b6f (origin/main)
**Delta count**: 1

## What happened
The reuse gate (retro r10-rnt-reuse-home) left one weakness: the use of the scratch home rested on `etc\` and
`security\` existing in it, and on a reused home boot 1 had already made them, so only the real-home snapshot
proved boot 2. Each rule got a fake-runner RED first on generic `ModuleA` fixtures; no station boot was run:
1. Just before `bin/test.exe` starts the runner records every regular file under `<scratch>/nuh/` (mtime and size;
   that state holds everything the runner and its gradle build put there) and the start time; the exit time just
   after it returns.
2. After the boot a file qualifies when it is new or changed against that record and its mtime lies in
   start..exit widened by a 2 s clock-skew tolerance (the window ends are WSL clock, the mtimes Windows-stamped).
   None qualifying is `FAIL  run-niagara-test  scratch niagara_user_home not written by this boot: ...`, with the
   window and tolerance printed; the PASS row names up to 5 files relative to `nuh/` with their mtimes.
3. No file is required by name. The only recorded listing of an NRE-made home is the non-booting `nre.exe -version`
   probe on an empty home (`etc/nre.properties`, `logging/`, `security/` keystores, retro r10-rnt-run-live-fixes);
   which files a boot rewrites on a kept home is not recorded, so the check is generic.
4. It runs on fresh runs too: there the NRE creates `etc/nre.properties` in a home the build only populated, so it
   is cheap (the walk is of a small dir) and closes the same gap for a build that leaves `etc\` or `security\`.
   The `etc\`/`security\` check and the real-home snapshot stay; all must pass.
5. The walk is bounded: over 20000 entries or a dir over 16 deep below `nuh/` is an error (no proof, FAIL); maps
   grow by insert, nothing is sized from the tree.

## Evidence
- the REDs on 4ebb0b6f: (b) nothing under nuh changes, (c) a file only the build wrote, (d) an mtime an hour before
  or after the window all exited 0 on the old code; (a) and the fresh run found no PASS row. Then green:
  `TestRunReuseHomeNREWroteScratchHome` (a)-(d), `TestRunFreshNREWroteScratchHome` `[ev: go/internal/niagaratest/run_nuhproof_test.go]`,
  `TestWalkNUHBounds` `[ev: go/internal/niagaratest/nuhproof_test.go]`
- the walk and the comparison `[ev: go/internal/niagaratest/nuhproof.go]`, the window around test.exe `[ev: go/internal/niagaratest/run.go]`
- mutation: always passing the row turned (b), (c) and both (d) subtests red; dropping the pre-boot comparison
  turned (b) and (c) red too (boot 1's files and the build's sit inside the 2 s tolerance on a fast fake run)

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | `run-niagara-test run` proves the NRE used the scratch `niagara_user_home` on this boot: some regular file under `nuh\` must be new or changed since just before test.exe started, its mtime in the boot window ±2 s skew, or the run is FAIL; the PASS row names the files. On fresh and reuse runs, beside the `etc\`/`security\` check and the real-home snapshot | `types/moduleTest.md` § Running from WSL | `[ev: go/internal/niagaratest/nuhproof.go]` |

## Lessons
- A presence check proves a first boot only; a later boot on the same dir needs a write it made, bounded by its own start and exit.
- An mtime window alone admits what the runner wrote just before the boot: compare against a record taken at the boot's start as well.

## Fold
- The proof is code in the branch that wrote this retro (`go/internal/niagaratest/nuhproof.go`).
- Δ1 is cited in `types/moduleTest.md` § Running from WSL (the scratch-home-written bullet). The live check ran on 2026-10-25 (see Live evidence).

## Live evidence (2026-10-25, a client root on a 4.15 disposable copy, names only)
Files under `nuh\` that the NRE wrote inside the boot window. The row names at most 5; the counts are the row's totals.
- Fresh boot, 9 files: `logging/logging.properties`, `registry/registry.chk`, `registry/registry.db`, `security/.kr`, `security/exemptions.tes`, and 4 more.
- `--keep-home` boot, 7 files: `logging/logging.properties`, `registry/registry.chk`, `registry/registry.db`, `security/.kr`, `security/signing/signers`, and 2 more.
- `--reuse-home` boot on that kept home, 4 files: `registry/registry.chk`, `registry/registry.db`, `security/signing/signers`, `test/station/shared/<module data file>`.
- On every boot the NRE rewrote `registry/registry.chk` and `registry/registry.db`. Re-check this list after an NRE or plugin upgrade: if a later version stops rewriting them, the row still holds as long as some other file qualifies.

---
**Status**: FOLDED 2026-10-25 (branch `r10/rnt-reuse-nre-proof`; log in `ledger.d/2026-10-25-r10-rnt-reuse-nre-proof.md`) — Δ1 in `types/moduleTest.md` § Running from WSL.
