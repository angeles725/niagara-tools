<!-- review-status: folded -->
# 2026-10-22 · kit · n4-course-signed-jars-gate-fix

**Session**: niagara-research quality round 8, task T9d-1 (issue #260), branch `fix/r8-t9d-verify-signed-jars` on main e1fee85
**Delta count**: 6

## What happened
Issue #260 carried the six deltas of retro `2026-10-16-n4-course-handson-build-signing-windows` (branch
`docs/n4-course-handson-retros`): the kit gate FAILed every N4 course jar that a station running
`moduleVerificationMode=high` had loaded live. `signed` only accepted `META-INF/NIAGARA4.SF` (the course jars
carry `CODESIGN.SF` + `CODESIGN.RSA`, named after the signing alias), `bytecode` counted Jackson's
`META-INF/versions/11` classes that Java 8 never loads, and a pipefail/SIGPIPE race reported 15 of 15 types
missing on the uberjar. `build.sh` did not recognise the Windows-route copy-lock line, and nothing checked a
course/vendor sample that still pointed at its author's install. This unit lands Δ1, Δ3, Δ5 and Δ6; Δ2 landed in
#267 (T8f) while this unit ran; Δ4 is deferred to #266. The Δ numbering below is #260's.

## Evidence
- RED on the base before each fix, GREEN after `[ev: tests/verify-module.bats VSIG1-VSIG3 VMR1-VMR2]`
  `[ev: tests/build-sh.bats BS-copy-lock-win]` `[ev: tests/preflight.bats PF-sample-orig PF-sample-retargeted PF-sample-version]`:
  VSIG1 (CODESIGN/SERVER1/EC1 aliases) FAILed `signed`; VSIG2 (NIAGARA4.SF without .RSA) PASSed; VSIG3 detail named
  NIAGARA4.SF; VMR1/VMR2 reported the `META-INF/versions/11` class as the first bad one; BS-copy-lock-win exited 30;
  PF-sample-* exited 2 (unknown flag). PF-sample-off and BS-copy-lock (LF/slash) are regression guards, green throughout.
- Pipe race before #267: a ~200 KiB entry list with the probed entries first gave `FAIL signed` + `FAIL types`
  (`META-INF/module.xml missing`) on the pipe form and 3/3 PASS with a here-string `[ev: T9d session run]`.
- The four course jars and `training-rt.jar` are on the Windows host and were not re-run here; the fixtures model
  their entry names (`CODESIGN.SF`/`.RSA`, `META-INF/versions/11/...major 55`) and the captured exam log line
  `[ev: retro n4-course-handson-build-signing-windows Evidence]`.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | `signed` PASSes any `META-INF/<ALIAS>.SF` with its `<ALIAS>.RSA/.DSA/.EC`, names the alias; an orphan `.SF` or no `.SF` FAILs | `toolbelt/verify-module.sh` `check_signed`; `build-verify.md` § Verify, § Signing per deploy target; `METHODOLOGY.md` close checklist; `BUILD-STATE.md` `signed:`; `types/third-party-libraries.md` | `[ev: retro n4-course-handson-build-signing-windows Δ1]` |
| Δ2 | presence tests under pipefail use here-strings | `toolbelt/verify-module.sh` (landed in #267, gate `tests/pipefail-grep.bats`); `build-verify.md` § Verify | `[ev: retro pipefail-grep-q-race]` |
| Δ3 | `bytecode` skips `META-INF/versions/**` and counts it in the row | `toolbelt/verify-module.sh` `check_bytecode_major`; `types/third-party-libraries.md` § Java-8 class-file ceiling | `[ev: retro n4-course-handson-build-signing-windows Δ3]` |
| Δ4 | lint the ROOT `build.gradle.kts` signing block; template + scaffold | deferred to #266 (`lint-structure.sh`, `templates/`, `scaffold-module.sh`) | `[ev: retro n4-course-handson-build-signing-windows Δ4]` |
| Δ5 | exit-32 copy-lock pattern accepts `modules[/\\]`; qualified tasks; holder by `tasklist` | `toolbelt/build.sh`; `BUILD-LOOP.md` § 4 Post-jar copy lock; `build-verify.md` § Raw gradle | `[ev: retro n4-course-handson-build-signing-windows Δ5]` |
| Δ6 | `preflight.sh --sample`: sample-home / sample-user / sample-sign FAIL, sample-build WARN | `toolbelt/preflight.sh` Check 6; `BUILD-LOOP.md` § 0.b; `build-verify.md` § Build target & plugin version | `[ev: retro n4-course-handson-build-signing-windows Δ6]` |

## Fold
- Δ1 → `toolbelt/verify-module.sh` `check_signed` + header; `build-verify.md` § Verify and § Signing per deploy target
  (the "OEM jars are expected to FAIL" note is gone); `METHODOLOGY.md` signed checklist line; `BUILD-STATE.md` `signed:` comment;
  `types/third-party-libraries.md` Path A step 4 and Path B step 3.
- Δ2 → done by #267 (T8f); cited in `build-verify.md` § Verify. No kit-ticket opened: the coordinator moved Δ2 to T8f.
- Δ3 → `toolbelt/verify-module.sh` `check_bytecode_major`; `build-verify.md` § Verify; `types/third-party-libraries.md`
  § Java-8 class-file ceiling (multi-release jars; Jackson 2.21.2 `uberjar()` ran live).
- Δ4 → `[deferred]` in `build-verify.md` § Signing per deploy target, tracked by #266.
- Δ5 → `toolbelt/build.sh` exit-32 classifier; `BUILD-LOOP.md` § 4; `build-verify.md` § Raw gradle.
- Δ6 → `toolbelt/preflight.sh` `--sample` (opt-in: client trees keep Windows `gradle.properties` paths);
  `BUILD-LOOP.md` § 0.b; `build-verify.md` § Build target & plugin version.
- #260 is resolved for Δ1, Δ2 (#267), Δ3, Δ5 and Δ6; Δ4 continues in #266.

## Lessons
- A gate keyed to one file name (`NIAGARA4.SF`) encodes one machine's signing profile; key it to the structure (`*.SF` + block).
- Run the gate on jars known to load live before trusting its FAIL; fixtures must model real entry names and sizes.
- A check that legitimately FAILs client trees (Windows paths in `gradle.properties`) belongs behind an opt-in flag, not in the default preflight.

---
**Status**: FOLDED — INDEX row: `| 2026-10-22-n4-course-signed-jars-gate-fix.md | kit | 2026-10-22 | folded | 6 |`
