# Spec: build-n4-module-campaign7

**Status**: spec · **Source**: v0.17.0 (c136e3b) · **Target**: v0.18.0
**Topic key**: `sdd/build-n4-module-campaign7/spec`
**Based on**: proposal.md (post-explore) + facts injected at spec time

---

## Cross-cutting discipline (all PRs)

| Rule | Requirement |
|------|-------------|
| CD1 | Every kit-changing push range contains: retro file + INDEX row flip + `BUILD-STATE.md` self-envelope — in the same push range. |
| CD2 | Doc-only PRs (PR1, PR2, PR3, PR7) carry no new bats tests. |
| CD3 | Every doc delta is preceded by a grep-before-fold check using rg; a double-fold is never written. |
| CD4 | QA RED branch tip is re-read at apply time before the branch is merged into the PR branch. |
| CD5 | Pending-row count = `grep -c '| pending |' retros/INDEX.md` (not a raw word grep). |
| CD6 | The launcher is the TRACKED copy at `build-n4-module-kit/skill/SKILL.md`; PRs that edit it are followed by an orchestrator re-install via `scripts/install-skill.sh`. |

---

## PR1 — Fold 9 campaign-6 retros

**Branch**: `feat/c7-fold-retros` | **Scope** (A): METHODOLOGY.md, BUILD-LOOP.md, CONTRIBUTING.md, retros/INDEX.md, 9 retro marker files | **Type**: doc-only

### Requirements

| ID | Requirement |
|----|-------------|
| R1.1 | `retros/INDEX.md` pending-row count (CD5 predicate) drops from 9 to 0 after the PR merges. |
| R1.2 | Every flipped INDEX row cites `[ev: retro <slug>]` pointing to evidence in a core kit file. |
| R1.3 | Each of the 9 retro files gains a `<!-- review-status: folded -->` marker replacing `pending`. |
| R1.4 | METHODOLOGY.md gains K11–K14 under §Kit-maintenance; no other section is modified. |
| R1.5 | BUILD-LOOP.md §7 gains the envelope-pairing rule: every close-exit (a/b/c) pairs its retro anchor with the `BUILD-STATE.md` self-envelope in the same push range. |
| R1.6 | CONTRIBUTING.md gains the SDD-ledger rule and the two-independent-reads rule. |
| R1.7 | `sweep-build-state.sh` exits 0 (green) after the PR merges. |
| R1.8 | `sweep-fold-audit.sh --strict` exits 0 after the PR merges. |

### Scenarios

**Given** `retros/INDEX.md` has 9 rows with `| pending |`;
**When** PR1 merges;
**Then** `grep -c '| pending |' retros/INDEX.md` == 0, every formerly-pending row contains `[ev: retro <slug>]`, and `sweep-fold-audit.sh --strict` exits 0.

**Given** any row that was already folded in METHODOLOGY.md or BUILD-LOOP.md before PR1;
**When** the fold is applied;
**Then** the grep-before-fold check finds the existing text and the INDEX row + marker are flipped without writing the delta again (double-fold prevented).

---

## PR2 — Tool integration

**Branch**: `feat/c7-tool-integration` | **Scope** (B): BUILD-LOOP.md, `build-n4-module-kit/skill/SKILL.md` (tracked copy) | **Type**: doc-only

### Requirements

| ID | Requirement |
|----|-------------|
| R2.1 | BUILD-LOOP.md §0.b names `toolbelt/preflight.sh` as the preflight step. |
| R2.2 | BUILD-LOOP.md §5 lists `lint-timers.sh` and `slot-coverage.sh` as pre-gate steps before `verify-module.sh`. |
| R2.3 | BUILD-LOOP.md §7 names `sweep-build-state.sh --age` at the orient and close steps. |
| R2.4 | `skill/SKILL.md` §References lists all 10 toolbelt scripts. |
| R2.5 | `skill/SKILL.md` §Execution step 5 routes to the pre-gate checks (lint-timers, slot-coverage) before the verify gate. |
| R2.6 | A grep over BUILD-LOOP.md and `skill/SKILL.md` finds each of the 10 script names at least once. |
| R2.7 | Orchestrator re-installs the tracked launcher via `scripts/install-skill.sh` after this PR merges. |

### Scenarios

**Given** BUILD-LOOP.md and `skill/SKILL.md` before PR2 (preflight, lint-timers, slot-coverage each cited 0 times);
**When** PR2 merges;
**Then** `grep -c 'preflight' BUILD-LOOP.md` >= 1, `grep -c 'lint-timers' BUILD-LOOP.md` >= 1, `grep -c 'slot-coverage' BUILD-LOOP.md` >= 1; same for `skill/SKILL.md`.

---

## PR3 — Dashboard fold (B796)

**Branch**: `feat/c7-dashboard-fold` | **Scope** (D): `types/dashboard.md` | **Type**: doc-only

### Requirements

| ID | Requirement |
|----|-------------|
| R3.1 | `types/dashboard.md` gains the DashboardPan-ux write-surface exemplar from B796 §796.4 (4 of 5 gates documented). |
| R3.2 | Gate 4 (REQUIRED-but-absent) is cited as a known gap with a reference to issue #49. |
| R3.3 | The folded content is faithful to B796 §796.4 — confirmed by a two-independent-reads fidelity check. |
| R3.4 | `tests/kit-links.bats` exits 0 after the PR merges (no broken links introduced). |

### Scenarios

**Given** `types/dashboard.md` before PR3 (no -ux exemplar);
**When** PR3 merges;
**Then** `grep -c 'DashboardPan-ux' types/dashboard.md` >= 1 and `tests/kit-links.bats` exits 0.

---

## PR4 — scaffold-module.sh + fixtures/MinimalPan

**Branch**: `feat/c7-scaffold` | **Scope** (C): `toolbelt/scaffold-module.sh`, `fixtures/MinimalPan/**`, `tests/scaffold-module.bats` | **Type**: code | **QA RED**: `qa/c7-scaffold`

### CLI contract

```
scaffold-module.sh <ModuleName> <out-dir> [--vendor <v>] [--target-version <x.y>] [--plugin-version <v>]
```

Exits: 0 = emitted OK · 2 = usage/validation error · 3 = env prerequisite missing (skeleton not found)

Skeleton resolved as: `"${BASH_SOURCE[0]%/*}/../fixtures/MinimalPan"` — never from `$HOME`.

### Requirements

| ID | Requirement |
|----|-------------|
| R4.1 | `scaffold-module.sh` with fewer than 2 positional arguments exits 2 and prints usage to stderr. |
| R4.2 | A `ModuleName` that starts with a digit or contains non-alphanumeric characters exits 2. |
| R4.3 | The emitted tree for `ModuleName=MinimalPan` is byte-equal to `fixtures/MinimalPan/` (excluding `build/` and `.gradle/`) under `diff -r`. |
| R4.4 | The skeleton is resolved via `${BASH_SOURCE[0]%/*}/../fixtures/MinimalPan`; the script exits identically under `HOME=/nonexistent` (no `$HOME` reference in the resolution path). |
| R4.5 | If the skeleton directory is missing (no `gradle/wrapper/` found), the script exits 3. |
| R4.6 | The emitted `B<ModuleName>.java` contains `stopped()` cancelling the `Clock.Ticket`; removing that cancel causes `lint-timers.sh` to report a FAIL (named mutation). |
| R4.7 | TC4 (round-trip build + verify gate ALL PASS) is SKIPped in CI when JDK 8 or `niagara_home` is absent; it is run locally by QA before bless and the output is pasted in the PR body. A SKIP is never recorded as a PASS. |
| R4.8 | The fixture commits as its own commit before the script commit so the reviewer diff for the script is ~300 lines. |
| R4.9 | `shellcheck` exits 0 on `toolbelt/scaffold-module.sh`. |

### Scenarios (match qa/c7-scaffold)

**TC1** Given no arguments; When run; Then exits 2.

**TC2** Given `ModuleName=1bad`; When run; Then exits 2 (digit-first validation).

**TC3** Given `ModuleName=MinimalPan` and a clean temp `out-dir`; When run; Then `diff -r <out-dir> build-n4-module-kit/fixtures/MinimalPan --exclude=build --exclude=.gradle` exits 0.

**TC-K8** Given `HOME=/nonexistent` in the environment; When run with `ModuleName=MinimalPan` and a valid `out-dir`; Then exits 0 and the emitted tree is identical to TC3.

**TC4** (local-only) Given JDK 8 and a valid `niagara_home`; When `build.sh` is run on the emitted tree then `verify-module.sh` is run on the resulting jar; Then all checks PASS. SKIP this test in CI; record the exact local command and output in the PR body.

---

## PR5 — schema-risk.sh

**Branch**: `feat/c7-schema-risk` | **Scope** (E): `toolbelt/schema-risk.sh`, `tests/schema-risk.bats`, fixtures (B799) | **Type**: code | **QA RED**: `qa/c7-schema-risk`

### CLI contract

```
schema-risk.sh <before-dir> <after-dir>
```

Output: one row per changed slot with columns `slot | change_kind | verdict`, then a summary verdict line `VERDICT: SAFE|LOSSY|OUTAGE`.

Exits: 0=SAFE · 1=LOSSY · 2=OUTAGE · 3=usage error · 4=env error (missing tools or unreadable input)

**Exit-code justification**: verdict exits 0–2 form a contiguous severity range. Callers checking `exit 2` for OUTAGE must not receive a usage error; usage (3) and env (4) sit above the verdict domain to prevent a bad invocation from silently masquerading as OUTAGE in a deployment script.

### change_kind domain

Derived from the B795 §795.4 CSV embedded verbatim in the script:

| change_kind | verdict |
|-------------|---------|
| `add` | SAFE |
| `reorder` | SAFE |
| `remove` | LOSSY |
| `rename` | LOSSY |
| `retype_simple` | OUTAGE |
| `*_unknown` (subtype unresolved) | OUTAGE |
| unrecognized kind | OUTAGE |

Module verdict = worst cell (OUTAGE > LOSSY > SAFE).

### Requirements

| ID | Requirement |
|----|-------------|
| R5.1 | The B795 §795.4 CSV is embedded verbatim in `schema-risk.sh` as the classification table; verdict logic is not hand-coded outside the table. |
| R5.2 | A `before-dir` or `after-dir` argument missing exits 3. |
| R5.3 | An unreadable `module-include.xml` in either snapshot exits 4. |
| R5.4 | Each of the six fixture classes (add, remove, retype_simple, reorder, rename, unknown kind) produces the correct per-slot verdict as defined by the CSV. |
| R5.5 | The module verdict equals the worst cell across all changed slots. |
| R5.6 | A slot whose subtype cannot be resolved against the CSV falls back to `*_unknown` → OUTAGE. |
| R5.7 | The mutation "replace worst-cell verdict with first-cell verdict" causes the mixed fixture (containing both LOSSY and OUTAGE slots) to produce a wrong verdict — proving the worst-cell predicate is tested. |
| R5.8 | `shellcheck` exits 0 on `toolbelt/schema-risk.sh`. |
| R5.9 | Fixtures are the before/after pairs produced as B799 by companero; the spec references them as the fixture set. |

### Scenarios (match qa/c7-schema-risk)

**Given** add-fixture (slot added in after-dir); **When** run; **Then** exits 0, verdict line = `VERDICT: SAFE`.

**Given** remove-fixture (slot removed); **When** run; **Then** exits 1, verdict line = `VERDICT: LOSSY`.

**Given** retype_simple-fixture (slot retyped to incompatible type); **When** run; **Then** exits 2, verdict line = `VERDICT: OUTAGE`.

**Given** reorder-fixture (slot order changed, no type change); **When** run; **Then** exits 0, verdict line = `VERDICT: SAFE`.

**Given** rename-fixture (slot renamed); **When** run; **Then** exits 1, verdict line = `VERDICT: LOSSY`.

**Given** unknown-kind-fixture (slot kind unrecognized in CSV); **When** run; **Then** exits 2, verdict line = `VERDICT: OUTAGE`.

**Mutation** Given mixed-fixture (LOSSY + OUTAGE slots); When verdict logic uses first-cell instead of worst-cell; Then the verdict changes from OUTAGE to LOSSY — proving the worst-cell assertion catches the defect.

---

## PR6 — verify-module.sh --plano

**Branch**: `feat/c7-plano` | **Scope** (G): `toolbelt/verify-module.sh`, `tests/verify-module.bats` | **Type**: code | **QA RED**: `qa/c7-plano` | **Not research-gated** (B797 done, niagara-research 4e7486c55)

### --plano contract

`verify-module.sh --plano <jar>` checks that all aspect-ratio declarations inside the jar's `rc/` CSS agree and agree with the image geometry.

Three quantities derived per image:
- **Rc** = intrinsic PNG aspect ratio (width/height from base64-decoded PNG header)
- **Ri** = `IMG_W / IMG_H` (numeric `width` / `height` attribute values on the same element)
- **Rv** = aspect-ratio implied by the SVG `viewBox` (first two non-zero dimensions)

PASS iff `Rc == Rv == Ri` **and** every NUMERIC `aspect-ratio` declaration in the CSS equals `Rc`. Declarations set to `auto` are exempt.

Real DashboardPan-ux FAILS today at `.frame` (line 84): a stale numeric `aspect-ratio` disagrees with the intrinsic geometry.

### Requirements

| ID | Requirement |
|----|-------------|
| R6.1 | `--plano` added to `verify-module.sh` usage and check loop; it is skipped when not requested (backward-compatible). |
| R6.2 | PASS is emitted only when Rc == Rv == Ri and every NUMERIC `aspect-ratio` declaration equals Rc. |
| R6.3 | A CSS `aspect-ratio: auto` declaration is exempt from the check (does not trigger FAIL). |
| R6.4 | A disagreement between viewBox dimensions and image intrinsic size (Rv ≠ Rc) triggers FAIL. |
| R6.5 | The mutation "replace the exact-agreement check with a count-only check" causes the disagreeing-pair fixture to pass — proving the exact-agreement assertion. |
| R6.6 | `shellcheck` exits 0 on the modified `toolbelt/verify-module.sh`. |

### Scenarios (match qa/c7-plano — from B797 contract)

**PL1** Given a jar where Rc == Rv == Ri and all numeric aspect-ratio declarations equal Rc; When `--plano` is run; Then PASS is emitted, exit 0.

**PL2** Given a jar with a stale numeric `aspect-ratio` that disagrees with Rc (the DashboardPan-ux `.frame:84` pattern); When `--plano` is run; Then FAIL is emitted, exit 1.

**PL3** Given a jar where `aspect-ratio` is set to `auto`; When `--plano` is run; Then auto is exempt, PASS is emitted, exit 0.

**PL4** Given a jar where the viewBox dimensions imply a ratio different from the PNG intrinsic size (Rv ≠ Rc); When `--plano` is run; Then FAIL is emitted, exit 1.

---

## PR7 — logic.md split

**Branch**: `feat/c7-logic-split` | **Scope** (F): `types/logic.md`, `types/logic-authoring.md` (new), `skill/SKILL.md`, `BUILD-LOOP.md` §2, `tests/kit-links.bats` | **Type**: doc restructure

### Requirements

| ID | Requirement |
|----|-------------|
| R7.1 | `types/logic.md` retains lines 1–79 (control authoring for the proven timer/interlock/HOA/fail-mode layer). |
| R7.2 | `types/logic-authoring.md` is created with lines 80–136 (framework-extension authoring: SPIs, point extensions, containers, queries, templates, jobs, action protection, minimal module). |
| R7.3 | The `skill/SKILL.md` decision table moves atomically in the same commit range as the file creation. |
| R7.4 | BUILD-LOOP.md §2 references are updated atomically in the same commit range. |
| R7.5 | `tests/kit-links.bats` is extended to assert both `types/logic.md` and `types/logic-authoring.md` resolve from `skill/SKILL.md` and BUILD-LOOP.md §2. |
| R7.6 | `tests/kit-links.bats` exits 0 after the PR merges. |
| R7.7 | The mutation "break one moved link (e.g. reference to logic-authoring.md → wrong path)" causes `kit-links.bats` to FAIL — proving link coverage. |
| R7.8 | Orchestrator re-installs the tracked launcher via `scripts/install-skill.sh` after this PR merges. |

### Scenarios

**Given** `types/logic.md` at 136 lines before PR7; **When** PR7 merges; **Then** `wc -l types/logic.md` <= 91 (split boundary = line 91, `## Author-side SPIs`; corrected from 80) and `types/logic-authoring.md` exists with the framework-extension content.

**Given** a broken reference to `types/logic-authoring.md` in `skill/SKILL.md` (mutation); **When** `tests/kit-links.bats` runs; **Then** at least one test FAILS.

---

## PR8 — report-module.sh (stretch)

**Branch**: `feat/c7-report` | **Scope** (H): `toolbelt/report-module.sh`, `tests/report-module.bats` | **Type**: code | **QA RED**: `qa/c7-report` | **Not research-gated** (B798 done, niagara-research d267d4e5a)

### CLI contract

```
report-module.sh <jar> [--src <module-dir>]
```

Runs: `lint-timers.sh` + `slot-coverage.sh` + `verify-module.sh --src` on the jar. Prints one aggregated report. A FAIL from any member check surfaces as FAIL in the aggregate.

### Requirements

| ID | Requirement |
|----|-------------|
| R8.1 | A FAIL from `lint-timers.sh` causes the aggregate report exit to be non-zero. |
| R8.2 | A FAIL from `slot-coverage.sh` causes the aggregate report exit to be non-zero. |
| R8.3 | A FAIL from `verify-module.sh` causes the aggregate report exit to be non-zero. |
| R8.4 | When all member checks PASS, the aggregate exits 0. |
| R8.5 | The mutation "drop lint-timers FAIL aggregation" causes the report to exit 0 on a jar with a lint-timers defect — proving the aggregation is tested. |
| R8.6 | `shellcheck` exits 0 on `toolbelt/report-module.sh`. |

### Scenarios

**Given** a jar that `lint-timers.sh` marks FAIL (e.g. missing `stopped()`-cancel); **When** `report-module.sh` runs; **Then** the aggregate report shows FAIL and exits non-zero.

**Mutation** Given the same jar; When the lint-timers aggregation step is dropped; Then the report exits 0 — proving the aggregation assertion catches the defect.

---

## Success criteria (verifiable)

| Criterion | Assertion |
|-----------|-----------|
| SC1 | `grep -c '| pending |' retros/INDEX.md` == 0 at campaign close. |
| SC2 | `grep -rn 'preflight\|lint-timers\|slot-coverage' build-n4-module-kit/BUILD-LOOP.md skill/SKILL.md` returns >= 1 hit per script per file. |
| SC3 | TC3 `diff -r` exits 0; TC-K8 exits 0 under `HOME=/nonexistent`. TC4 locally PASS, PR body contains the exact output. |
| SC4 | `schema-risk.sh` classifies all 6 fixture classes correctly; worst-cell mutation flips the mixed fixture. |
| SC5 | `tests/kit-links.bats` exits 0 after PR7; both `types/logic.md` and `types/logic-authoring.md` resolve. |
| SC6 | bats suite total >= 175 passing; each new case has a named mutation recorded; `shellcheck` exit 0 across all scripts; `sweep-fold-audit.sh --strict` exits 0. |
| SC7 | Every kit-changing push range in the chain contains retro + INDEX row + `BUILD-STATE.md` self-envelope. |
| SC8 | `VERSION` = 0.18.0; `CHANGELOG.md` entry present per CONTRIBUTING §4-5. |
| SC9 | No commit body in any PR contains an attribution trailer. |
| SC10 | PL1–PL4 scenarios all pass against `verify-module.sh --plano` after PR6. |

## Corrections bound to the executable REDs and the design (orchestrator, 2026-09-05)

Where this spec and the QA RED branches disagree, the RED wins (cross-cutting discipline CD4); the design records the binding:

- **D4a schema-risk verdict line**: the final line is `verdict=<SAFE|LOSSY|OUTAGE>` (qa/c7-schema-risk 6d27ff0), not `VERDICT: <V>`; B799's `expected.txt` (`verdict: <V>`) is an oracle for the token and the change_kind set only.
- **D6a `--plano` operand**: `verify-module.sh --plano <index.html>` takes the explicit HTML file (qa/c7-plano c49504f); labels per B797 §797.2 (Rc = IMG_W/IMG_H, Ri = intrinsic PNG size, Rv = viewBox); integer cross-multiplication.
- **D7a report-module signature/exits**: `report-module.sh <module-root> [--target-version X.Y]`, exit 0 clean / 1 any FAIL / 3 env; rows aggregated from each tool's own rows ORed with member exits so WARN rows (slot-coverage exits 0) are preserved (qa/c7-report-module 412ee8e).
- **PR4 fixture**: `fixtures/MinimalPan/` is the PRE-slotomatic source tree (no AUTO region — its header hash is name-derived; B793 C3), with no operator paths in `gradle.properties` (K8); the fixture lexicon is English (kit artifact convention). The emitted root is `<out-dir>/<ModuleName>` (design D2); QA aligns TC3/TC4 paths to that layout.
- **PR7 split boundary**: line 91 (`## Author-side SPIs` starts the framework-extension half); `## kitControl patterns` stays with control authoring.
