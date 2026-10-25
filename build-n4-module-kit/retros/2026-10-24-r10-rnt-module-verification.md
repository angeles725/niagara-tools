<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-24 · kit · r10-rnt-module-verification

**Session**: niagara-tools quality round 10, kit tool G33 `run-niagara-test run`, an opt-in
`--module-verification low` for the disposable copy, branch `r10/rnt-module-verification` from 0f4494b9
(`r10/rnt-run-live-fixes`)
**Delta count**: 2

## What happened
A self-signed test jar runs on 4.14 and is rejected on 4.15. The cause is a shipped default, read from the
4.14.0.162 decompile and the two installs' `defaults/system.properties`:
- 4.14 ships `niagara.moduleVerificationMode=low` (line 442, warn only); 4.15 ships `#niagara.moduleVerificationMode=medium`
  at the same line, commented out, so the code default `medium` applies (`ModuleVerificationMode`).
- With `medium`, a jar whose cert chain cannot be validated fails `failed signature validation` (`ModuleManager`);
  `TestRunner` then prints `invalid test target(s)` and still exits 0. `system.properties` is read from
  `<niagara_home>/defaults/` (`NiagaraFiles`).
- `low` still fails a module that requests one of the six signature-required permission groups (`ModuleManager`).

The runner gained `--module-verification low`, off by default and test-only. Each guard got a fake-runner RED
first; no Windows boot was run:
1. `low` is the only value; anything else (`medium`, `LOW`, an empty value) is exit 2 before any call.
2. The line is written only after robocopy and only to the copy's `defaults/system.properties`, symlinks
   resolved: a file that resolves inside `--install`, outside the run's `os.MkdirTemp` dir or off the drives is
   refused (exit 2) before the build, nothing written. The install guard sits before the scratch guard: the scratch
   guard alone already refuses an install path (guard 1 keeps the install out of the scratch), so the first
   mutation of the install guard stayed green until each refusal case asserted its own reason.
3. Exactly one line, `niagara.moduleVerificationMode=low`, in the file's line ending (CRLF if it has any), with a
   line break first when the file does not end in one; `PASS  run-niagara-test  module verification: low (copy only)`.
4. The install's `defaults/system.properties` is hashed before the copy and after cleanup: `install
   defaults/system.properties unchanged (sha256 <8hex>)`, a change is FAIL. The install snapshot does not list
   `defaults/`, so without this hash an in-place edit there would go unseen.
5. A run without the flag whose console shows `failed signature validation` adds a `hint:` row naming the flag.

## Evidence
- the REDs, then green: `TestRunModuleVerificationValues`, `TestRunModuleVerificationLowAppendsOneLine`, `TestRunModuleVerificationRefusesOutsideCopy`, `TestRunModuleVerificationNeedsInstallFile`, `TestRunModuleVerificationInstallFileHashed`, `TestRunSignatureRejectionHintsFlag` `[ev: go/internal/niagaratest/run_verification_test.go]`
- the guards and the append `[ev: go/internal/niagaratest/verification.go]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | `run-niagara-test run --module-verification low` lowers verification in the disposable copy only (one appended line, refused outside the copy, the install's file hashed before and after); TEST-ONLY, never on a commissioning or production home | `types/moduleTest.md` § Running from WSL | `[ev: go/internal/niagaratest/verification.go]` |
| Δ2 | 4.14 ships `niagara.moduleVerificationMode=low`, 4.15 leaves it commented out (code default `medium`), so a self-signed test jar that ran on 4.14 is an `invalid test target(s)` with exit 0 on 4.15; `low` still rejects the six signature-required permission groups | `types/driver-authoring.md` § 12.1 | `[ev: go/internal/niagaratest/verification.go]` |

## Lessons
- A guard that a stronger sibling guard subsumes is unpinned until each refusal asserts its own reason.
- A snapshot that skips a directory needs its own check for any file the run touches in that directory's copy.

## Fold
- The flag is code already shipped in the PR that wrote this retro (`go/internal/niagaratest/verification.go`, `run_verification_test.go`); not re-implemented.
- Δ1 was cited at merge in `types/moduleTest.md` § Running from WSL (the TEST-ONLY bullet); Δ2 in `types/driver-authoring.md` § 12.1 (the 4.15 rejection bullet). No doctrine text added.
- Lesson 1 (a subsumed guard is unpinned until each refusal asserts its own reason) is the existing `BUILD-LOOP.md` § 5 rule "A truth case is RED on the old code for the reason it names"; lesson 2 is in the Δ1 bullet (the install file's sha256). Nothing owed.

---
**Status**: FOLDED 2026-10-25 (branch `docs/r10-fold-tail-4`; log in `ledger.d/2026-10-25-r10-fold-tail4.md`) — Δ1 in `types/moduleTest.md` § Running from WSL; Δ2 in `types/driver-authoring.md` § 12.1.
