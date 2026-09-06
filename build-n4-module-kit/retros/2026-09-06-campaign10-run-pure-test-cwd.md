<!-- review-status: pending -->
# 2026-09-06 · kit · campaign10-run-pure-test-cwd

**Session**: C10 PR4 — S24 cwd-independent structural REDs
**Delta count**: 1

## What happened
`run-pure-test.sh` invoked the `java` call at `:62` in the caller's working directory.
Structural WiringTests use `Paths.get("src/…")` and `../../build.gradle.kts` relative to
the JVM cwd — paths that only resolve when the cwd is the module-rt dir or its parent.
Running the runner from the kit root or `/tmp` caused the test to fail with a file-not-found
error, even though the runner compiled the test correctly (javac `-sourcepath "$rt/src:…"` is
absolute once `$rt` is an absolute path, so compilation succeeded from any cwd). The fix is a
single subshell wrapping the `java` call: `( cd "$rt" && java -cp … )`.
`[ev: retro campaign10-run-pure-test-cwd]`

## Evidence
- `run-pure-test.sh:62` — `java` runs in caller cwd, not `$rt` `[ev: kit run-pure-test.sh:62 @ df8c7ec]`
- `FreezeAlarmWiringTest.java:48-56` — `Paths.get("src/…")` is cwd-relative `[ev: client ff1b659]`
- S24-cwd FAIL → GREEN: bats flip on subshell cd `[ev: qa/c10-structural-cwd a792d7a]`
- Three-cwd smoke OK (5 tests) × 3 from kit root, module-rt dir, /tmp `[ev: apply 2026-09-06]`
- `shellcheck 0.11.0` exit 0; 384/384 bats pass `[ev: apply 2026-09-06]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Document 2-arg usage + cwd independence in build-verify.md §Unit tests | `build-n4-module-kit/build-verify.md:108` | `[ev: apply campaign10-run-pure-test-cwd]` |

## Lessons
- The cwd-sensitive part of `run-pure-test.sh` was the `java` step, not the `javac` step; `-sourcepath` resolves absolute paths regardless of the caller's cwd, but `Paths.get("src/…")` in the JVM is always relative to JVM cwd.
- A subshell `( cd "$rt" && java … )` is the minimal runner-side fix: it changes the JVM cwd without altering the parent's cwd or the `trap 'rm -rf "$tmp"' EXIT` handler.
- Under `set -euo pipefail`, a subshell exit code propagates naturally as the script's exit code, preserving the "bite" (exit 1 on JUnit failure) without any extra plumbing.
- An absolutise line `rt=$(cd "$rt" && pwd)` is inert under the subshell structure and unpinnable by a RED that passes an absolute path; a change no RED can bite is would-flip prose.
- Structural WiringTests that read relative paths must be documented as requiring a runner that controls the JVM cwd — not a test-side fix per module file.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-06-campaign10-run-pure-test-cwd.md | kit | 2026-09-06 | pending | 1 |`
