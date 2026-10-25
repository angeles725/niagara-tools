<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-25 · kit · r10-lint-test-types-in-prod

**Session**: niagara-tools quality round 10, kit check "a production module-include.xml must not register test
types", branch `r10/lint-test-types-in-prod` from origin/main 005b2565
**Delta count**: 2

## What happened
With Niagara gradle plugin 7.3.40 (iC-Niagara 4.13), `:<part>:compileModuleTestJava` runs the annotation
processor against the production `module-include.xml` and appends every `@NiagaraType` test class to it, so an
ordinary build after a moduleTest compile ships test types in the production jar's `module.xml` (reported by the
round coordinator on a client module, `ModuleA`; not reproduced on a plugin in this session).

- **Home.** A sub-check `test-type-in-prod` inside `lint-agent-on-include-sync` (rule
  `go/internal/lint/rules/agentonincludesync`), the way `type-outside-region` went into
  `lint-slotomatic-markers`. That rule already maps each part to its `module-include.xml` and is a report-module
  per-module member, so the new rows reach `report-module.sh` without a new member and without re-recording every
  report golden. `lint-agent-on-shape` reads only `<on>` elements and `lint-structure` checks layout, not types.
- **Check.** A production `<type class>` is a leak when the class is declared in a `*.java` under the part's
  `srcTest/` (binary names from `javasrc.Classes` plus the package, nested types joined with `$`) or is a `<type>`
  of the part's `moduleTest-include.xml`. The source reason wins when both hold. The rule now reads `srcTest/`
  (it was pruned) and `moduleTest-include.xml`; the `@AgentOn` check still skips every `srcTest/` file.
- **verify-module.** `agentonincludesync.PartTestLeaks(part)` is the file-system entry; the `moduletest` row calls
  it first on every part with `--src` (not only `-rt`), prints one FAIL row per leak, and a malformed XML file is
  the env error (exit 3). A clean part prints no extra row, so no existing verify-module or report-module golden
  changed. Before this, a malformed `moduleTest-include.xml` read as "present" (PASS).
- **Not done here.** No live plugin run: the 7.3.40 behavior is the coordinator's observation. The check cannot
  tell a test class that was meant to ship; none should.

## Evidence
- goldens tt-srctest-class, tt-nested-listed, tt-moduletest-listed, tt-clean, tt-no-tests, tt-malformed-moduletest `[ev: go/internal/lint/rules/agentonincludesync/testtypes.go]`
- verify-module goldens edge-moduletest-test-type-in-prod and edge-moduletest-malformed-test-include `[ev: go/internal/verifymodule/srcmore.go]`
- bats AOIS3, AOIS3-env, MT5 and RM7d, each mutation-killed `[ev: tests/lint-agent-on-include-sync.bats]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | After any moduleTest compile, run `git status` and revert new `<type>` lines in the production `module-include.xml`; plugin 7.3.40 appends srcTest types there | `types/moduleTest.md` § `moduleTest-include.xml` — separate from `module-include.xml` | `[ev: go/internal/lint/rules/agentonincludesync/testtypes.go]` |
| Δ2 | The build-verify gate names `test-type-in-prod` (report-module) and the verify-module `moduletest` FAIL row as the guard for test types in a production jar | `BUILD-LOOP.md` § 5. Verify gate (before "done") | `[ev: go/internal/verifymodule/srcmore.go]` |

## Lessons
- Folding a new check into an existing report-module member, and printing a row only on a finding, keeps the
  recorded goldens of every other tool unchanged.
- A row that printed PASS on a file it never parsed ("moduleTest-include.xml present") was a silent-pass channel;
  parsing it for the new check closed it.
