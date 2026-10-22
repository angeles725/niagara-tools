<!-- review-status: folded -->
# 2026-10-22 · kit · n4-course-sample-check-precision

**Session**: niagara-research quality round 8, task T9d follow-ups (review findings of #269 / issue #260 and #268 / issue #259), branch `fix/r8-t9d-followups` on main 6b17b5d
**Delta count**: 5

## What happened
The reviews of #269 and #268 left non-blocking findings: `preflight.sh --sample` PASSed `sample-sign` on bare
substrings (a root `build.gradle.kts` with `allowDefaultProfile.set(true)` or a commented-out signing block PASSed),
PASSed `sample-user` when the `niagara_user_home` key was missing, and skipped the `sample-home` version compare
silently when a path had no X.Y.Z; `verify-module.sh` dropped the multi-release count when every class sat under
`META-INF/versions/`; `WsGrid.rowSpacingFor` let an int overflow near `Integer.MAX_VALUE` fall back to 10 and returned
spacings no sheet row can hold. The coordinator added `stored-repack.sh`, which hardcoded `NIAGARA4.SF`/`.RSA` and so
lost the signature-first entry order of a `CODESIGN`/`SERVER1`/EC-key signed jar. The toolbelt comments also cited the
origin retro `n4-course-handson-build-signing-windows` (not on main); they now cite `n4-course-signed-jars-gate-fix`,
the retro that is, like the docs.

## Evidence
- RED on 6b17b5d, GREEN after the fix `[ev: tests/preflight.bats PF-sample-sign-true PF-sample-sign-commented PF-sample-sign-partial PF-sample-user-missing PF-sample-version-unknown]`:
  the three sign rows exited 0 (`[ "$status" -eq 1 ]` failed), the user/version rows printed PASS instead of WARN.
- `[ev: tests/verify-module.bats VMR3]` RED: the FAIL detail was `no .class entries` with no count.
- `[ev: templates/ws-placement/srcTest/com/angeles/wsplacement/WsGridTest.java]` RED `Tests run: 9, Failures: 1`
  (`SUMMARY count 592 accepted`), compiling as written, GREEN `OK (9 tests)` under JDK 8.
- `[ev: tests/stored-repack.bats S4]` RED: the report line named no `CODESIGN.SF`/`EC1.EC`. The entry-order half passed
  by chance on the base (`zip -r` walks `META-INF/` in readdir order), so the report line is the deterministic pin.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | `sample-sign` checks the root code outside comments and strings (`"…"` with escapes, raw `"""…"""`, so `"**/*.jar"` or `"https://…"` is text) for `allowDefaultProfile.set(false)` or `allowDefaultProfile = false`, `niagaraSigning {`, `aliases`, `signingProfileFile`, and names each one missing | `toolbelt/preflight.sh` Check 6; `build-verify.md` § Build target & plugin version | `[ev: tests/preflight.bats PF-sample-sign-*]` |
| Δ2 | No `sample-*` row PASSes by omission: missing `niagara_user_home` WARNs, an uncomparable version WARNs | `toolbelt/preflight.sh` Check 6; `build-verify.md` § Build target & plugin version | `[ev: tests/preflight.bats PF-sample-user-missing PF-sample-version-unknown]` |
| Δ3 | `bytecode` FAIL for a jar with only multi-release classes keeps their count | `toolbelt/verify-module.sh` `check_bytecode_major`; `build-verify.md` § Verify | `[ev: tests/verify-module.bats VMR3]` |
| Δ4 | `stored-repack.sh` puts every `META-INF/<ALIAS>.SF` + `.RSA/.DSA/.EC` right after the manifest | `toolbelt/stored-repack.sh`; `build-verify.md` § Workbench re-sign: STORED repackage | `[ev: tests/stored-repack.bats S4]` |
| Δ5 | `WsGrid.rowSpacingFor` rejects `n > MAX_SUMMARY_COUNT` (591) | `templates/ws-placement/`; `types/logic-authoring.md` §Wire-sheet placement | `[ev: templates/ws-placement/srcTest/com/angeles/wsplacement/WsGridTest.java]` |

- Branch review WARNINGs, fixed in the same unit `[ev: tests/preflight.bats PF-sample-sign-strings PF-sample-sign-assign]`: RED,
  the first stripper took `/*` in `"**/*.jar"` as an unclosed block comment (every requirement reported missing) and
  `allowDefaultProfile = false` FAILed; GREEN once strings are tracked and the assignment form is accepted.

## Lessons
- A gate that matches a configuration by substring passes its own negation (`set(true)`) and its commented-out copy;
  match the uncommented code and the exact value.
- Every branch of a check prints a row: a missing key or an uncomparable value is a WARN, never a silent PASS.
- A file name derived from a user choice (the signing alias) is never hardcoded in a tool that reorders entries.
- An int bound computed by addition needs an upper limit before the sum, or overflow turns a reject into a default.

---
**Status**: FOLDED (same unit) — Δ1/Δ2 `build-verify.md` § Build target & plugin version, Δ3 `build-verify.md` § Verify,
Δ4 `build-verify.md` § Workbench re-sign: STORED repackage, Δ5 `types/logic-authoring.md` §Wire-sheet placement.
