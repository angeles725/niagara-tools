# Spec: run-pure-test cwd independence (S24)

**Capability**: `kit-test-harness` — structural RED source resolution
**PR**: PR4 (`feat/c10-cwd-independent-reds`)
**QA RED**: `qa/c10-structural-cwd` tip `a792d7a` (base `df8c7ec`) `[CERT]`
**Repo**: `niagara-tools` (kit)
**Cross-cutting**: see `../cross-cutting.md` (K11/K12/K13/K14/K19/K20/D9b/comment-strip/observed-flip/real-tree-smoke)
**Change class**: runner-side fix; FP-only; no client file touched; lands before any further structural RED is authored in C10.

---

## Delta — MODIFIED requirements

The following MODIFY `run-pure-test.sh` at line `:62` (the `java` invocation @ `cb79676`). The `javac` compilation step and exit codes are UNCHANGED.

| ID | Requirement | ev |
|----|-------------|-----|
| R-S24.1 | `run-pure-test.sh` MUST change directory to `"$rt"` (the module-rt dir, already validated at `:30`) **before** the `java` call at `:62`, so `src` in the `-sourcepath` argument resolves from the module root, never from the caller's working directory. | `[ev: qa/c10-structural-cwd a792d7a]` `[ev: run-pure-test.sh:62 @ cb79676]` |
| R-S24.2 | The implementation MUST use a subshell (`(cd "$rt" && java …)`) or an equivalent mechanism so the caller's working directory is UNCHANGED after the call returns. | `[ev: qa/c10-structural-cwd a792d7a companero package 4d5e6092c]` |
| R-S24.3 | Running `run-pure-test.sh <module-rt-dir> <pkg.TestClass>` from the kit root, from the module profile dir, and from `/tmp` MUST yield the **same** pass/fail verdict. | `[ev: qa/c10-structural-cwd a792d7a]` |
| R-S24.4 | No existing test's verdict MUST change as a result of this fix (no tests that were passing start failing; no tests that were failing start passing except S24-cwd). | `[ev: proposal SC-4]` |
| R-S24.5 | No client test file MUST be touched by this PR. | `[ev: proposal SC-4]` `[ev: K12]` |
| R-S24.6 | Exit codes (0 = tests passed · 1 = test FAILED · 2 = usage · 3 = environment) are **unchanged**. | `[ev: run-pure-test.sh:19-20 @ cb79676]` `[ev: K20]` |

---

## Scenarios

### S24-cwd (RED → GREEN after fix)

**Given** `run-pure-test.sh` invoked from a working directory other than the module-rt dir (e.g. the kit root or `/tmp`), with a valid `<module-rt-dir>` argument pointing to a module root where `srcTest/` contains a passing test.

**When** the script runs.

**Then** it exits 0 and all test assertions pass. (Before the fix: the `-sourcepath "$rt/src:$testroot"` argument with a relative `$rt` resolves incorrectly from the caller's cwd → `javac` fails to find sources → test fails.)

`[ev: qa/c10-structural-cwd a792d7a S24-cwd]` `[ev: companero package 4d5e6092c — WiringTests read Paths.get("src/…") relative to cwd]`

---

### S24-cwd-regression (must stay GREEN after fix)

**Given** `run-pure-test.sh` invoked from the module-rt dir itself.

**When** the script runs against the same test.

**Then** it exits 0 and the verdict is **unchanged** from the pre-fix run.

`[ev: qa/c10-structural-cwd a792d7a S24-cwd-regression]`

---

### S24-three-cwd (same verdict from three locations)

**Given** the same test invoked from the kit root, from the module profile dir, and from `/tmp`.

**When** each run completes.

**Then** all three runs return the **same** exit code and produce the same test result count.

`[ev: qa/c10-structural-cwd a792d7a]` `[ev: proposal SC-4]`

---

### S24-observed-flip

**Given** the fix is applied (S24-cwd passes).

**When** the `cd "$rt"` or subshell is reverted.

**Then** S24-cwd flips from exit 0 → a compilation or test failure (OBSERVED, verbatim output captured).

`[ev: proposal SC-7]` `[ev: K13]`

---

## Success criteria (this capability)

- [ ] `qa/c10-structural-cwd` `a792d7a` goes green: S24-cwd flips FAIL → pass; S24-cwd-regression stays pass.
- [ ] Same verdict from three cwds (kit root, profile dir, `/tmp`).
- [ ] Reverting the `cd "$rt"` fails S24-cwd (OBSERVED flip).
- [ ] No existing test's verdict changes; no client test file touched.
- [ ] `shellcheck 0.10.0` exits 0; 0 attribution trailers.
