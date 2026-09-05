# Proposal: build-n4-module-kit-v0.2

**Status**: proposal
**Source version**: niagara-tools `v0.3.0` · kit `v0.1`
**Target version**: niagara-tools `v0.4.0` (MINOR — new toolbelt scripts, new tests, new docs) · kit `v0.2`
**Phase**: propose (post-explore)
**Topic key**: `sdd/build-n4-module-kit-v0.2/proposal`
**Inputs**: engram `#7937` (explore) · engram `#7935` (lessons inventory, QA-annotated) · `openspec/changes/build-n4-module-kit-v0.2/explore.md`
**Selected option**: Option 2 from exploration — fold retros + toolbelt scripts + tests + launcher fix.

---

## 1. Intent

Three field builds (ColdRoomPan rt hardening, DashboardPan 3→5 rooms, DashboardPan touch-HMI UX) produced 42 proven lessons that live only in retro files and the operator bitácora. The kit that the `build-n4-module` skill loads on every new module build does not contain them, so the next build re-learns them the expensive way — on a live plant.

The cost is already documented, not hypothetical:

| Pain | Evidence |
|---|---|
| A jar built against the wrong `niagara_home` is rejected by the station | 4.15→4.14 rebuild (B1) |
| Workbench re-sign fails with `invalid entry compressed size`; the 2026-08-31 "transient build state" hypothesis was wrong and cost a full debug cycle | B7 / H1, field-confirmed `[CERT-live 2026-09-01]` |
| A stale `.frame` aspect-ratio silently offsets every plano polygon; the same stale value is **still alive** in DashboardPan `index.html` today | C4, QA-reproduced |
| A deployed jar can be stale/unsigned/wrong-bytecode and nothing catches it — `ng-deploy.sh` verifies only the `<type` count | QA audit of `scripts/ng-deploy.sh` |
| `build.sh` adds `-wb` tasks for an empty stub subproject and checks bytecode on only the first `.class` per jar | QA-confirmed against DashboardPan |
| The launcher tells the agent to build with `toolbelt/build.sh`; the kit says the primary is `ng-deploy.sh --strict-slotomatic` | F1 |

There is also a doctrine gap: the kit has no automated verify gate, so "verified" currently means "an agent read a checklist". This change turns the automatable half of that checklist into a script with tests.

**Why now**: the three retros are the canonical propose-never-apply artifacts and are *still untracked in git*. They are one `rm` away from being lost, and every day they stay unfolded the kit ships known-stale guidance.

---

## 2. Scope

### 2.1 In scope

1. **Fold 41 doc lessons** into the kit (groups A, B, C, D, G, J) with the H-corrections applied.
2. **Toolbelt**: new `verify-module.sh` (THE verify gate), new `stored-repack.sh`, rewritten `build.sh`, promoted `mirror-niagara-home.sh` with a safety guard.
3. **Tests**: 4 new bats files + 1 stored-repack case, each guarding a named real regression.
4. **Structural fixes**: 2 dangling-link renames, SOURCES.md ColdRoomPan path, BUILD-LOOP.md preflight sub-step, "See also" pointers into `niagara-research` docs.
5. **Release**: kit README → v0.2, retro `review-status` markers, `CONTRIBUTING.md` bats step, `docs/GOTCHAS.md` entries, `CHANGELOG.md` + `VERSION` 0.3.0→0.4.0, launcher `SKILL.md` F1 fix + version 0.2.
6. **Repo defect**: restore the `scripts/ng-deploy.sh` exec bit (mode-only, zero content diff).

### 2.2 Out of scope (non-goals)

| Non-goal | Reason |
|---|---|
| Restructuring the kit into new type guides (JACE.md, signing.md, supervisor.md) | Option 3 rejected — breaks links, over-engineers a v0.1 seed |
| Promoting `types/wb-widgets.md` past SEED | No wb build happened; link fix + "See also" only |
| Any `ng-deploy.sh` behavior change | Exec bit only; API surface stays frozen |
| Integrating `verify-module.sh` into `ng-deploy.sh` | Deferred to a later change; the `-ux` slotomatic gap is **documented, not fixed** (H3) |
| Any change to the `niagara-research` repo | Kit points at those docs, never duplicates them (group I) |
| Any edit to the `Cliente/Leon-Guanjuato/bitacora/` folder | Bitácora is the operator's raw record; it is read, never written |
| Fixing the live stale `.frame 1247/771` in DashboardPan `index.html` | Module-side defect; the kit rule (C4) ships now, the module fix is separate work |
| Committed binary fixtures | Every jar fixture is generated in-test with `printf` + `zip` |

---

## 3. Capabilities

> `openspec/specs/` does not exist in this repo. The prior change (`niagara-tools-slotomatic-integration`) kept its spec change-local; this change follows the same convention — `sdd-spec` writes one change-local `spec.md`.

### New Capabilities

- `module-verify-gate`: `toolbelt/verify-module.sh` — the automatable verify contract (checks, flags, exit codes) applied to any built jar set regardless of which builder produced it.
- `kit-build-toolbelt`: contracts for `build.sh` (usage, profile selection, exit codes, gate call), `mirror-niagara-home.sh` (safety guard), `stored-repack.sh` (B7 recipe).
- `build-n4-module-kit-doctrine`: kit content rules — build primary/fallback ordering, retro `review-status` marker convention, relative-link integrity, pointer-not-duplicate rule for `niagara-research` docs.

### Modified Capabilities

- None. No existing spec-level requirement changes; `ng-deploy.sh`'s contract is untouched.

---

## 4. Approach — three chained PR slices

Fixed merge order. PR1 targets `main`; PR2 branches after PR1 merges; PR3 after PR2.

### PR1 — `feat/kit-v0.2-docs-foldin` (owner: Investigador2)

Doc fold-in only. **First commit = the 3 untracked kit retros, committed verbatim** (no markers yet — markers land in PR3, so the fold-in diff and the audit-trail diff stay separable).

Then: all A–D + G + J lessons folded with H corrections applied, both dangling-link renames, SOURCES.md path, BUILD-LOOP.md preflight, "See also" pointers.

**Corrections that MUST land (do not fold the superseded claim):**

| ID | Superseded claim | Correct content to fold |
|---|---|---|
| H1 | B7 "transient build state, fixed by clean+rebuild" | Deflater mismatch; STORED repackage is the fix; local `jarsigner` is a false negative |
| H2 | "the build emits STORED jars" | STORED is a **manual post-build step**, only for the Workbench re-sign path; `build/libs` jars stay deflated |
| H3 | "ng-deploy runs slotomatic for all profiles" | `ng-deploy.sh` runs slotomatic **rt-only**; use `build.sh` when editing `-ux` annotations |
| H4 | `BFrozenEnum` for a cross-module linked value | Plain `double` (G1) — a shared enum forces a module-B-rt→module-A-rt dependency |
| B2 | Retro's "plugin version common to 7.6.1/7.6.3/7.6.5" | **Each install ships exactly ONE** `niagara-module` plugin version (4.13.2→7.3.40, 4.14→7.6.17, 4.15.3→7.6.22); `-PniagaraPluginVersion` override is **MANDATORY** |
| C4 | "keep the four values in sync" | Strengthened: **exactly ONE** aspect-ratio declaration for the frame, equal to `IMG_W/IMG_H`. Fix = **delete** the stale value, never shadow it with a higher-specificity `#frame` rule. Verify: `grep -c 'aspect-ratio'` for the plano frame == 1 |

### PR2 — `feat/kit-v0.2-toolbelt` (owner: QA)

| File | Action |
|---|---|
| `toolbelt/verify-module.sh` | NEW — the gate: all-classes bytecode major 52, `NIAGARA4.SF` presence, `module.xml` `<type>` ↔ `.class` correspondence + count vs `src/module-include.xml`, baja `vendorVersion` vs `--target-version`, raw-double facet grep, `--stored` opt-in (zero `Defl:` entries) |
| `toolbelt/build.sh` | REWRITE — usage text + exit 2 on bad args; refuse a non-`niagara_home` target; profile selection = has a gradle file **AND** has sources; `--profiles` override; `--target-version` forwarded; calls `verify-module.sh` after gradle; English messages |
| `toolbelt/mirror-niagara-home.sh` | PROMOTE from bitácora + safety guard: refuse the source install or any path inside it; refuse to wipe a dir lacking a `.niagara-mirror` marker; fix the tilde example |
| `toolbelt/stored-repack.sh` | NEW — small script implementing the B7 recipe |
| `scripts/ng-deploy.sh` | exec-bit restore only (`git update-index --chmod=+x`) |

### PR3 — `feat/kit-v0.2-release` (owner: Investigador1)

Kit `README.md` status → v0.2; `<!-- review-status: folded v0.2 · 2026-09-01 -->` on each of the 3 retros; `CONTRIBUTING.md` bats-core install step; `docs/GOTCHAS.md` entries (STORED repackage, mirror pattern, verify-module gate); `CHANGELOG.md` + `VERSION` 0.3.0→0.4.0; and the launcher `~/.claude/skills/build-n4-module/SKILL.md` — F1 fix + version → 0.2. **The launcher is outside git**: its before/after must be recorded in the PR description and in engram, because `git revert` cannot roll it back.

### 4.1 Doctrine (final wording to encode)

- `verify-module.sh` is **THE verify gate**, regardless of who built the jars.
- `build.sh` is the **recommended WSL build**: it calls the gate automatically and runs slotomatic for **every profile that has sources**.
- `ng-deploy.sh` is the **station DEPLOY wrapper** (backup → build → copy → type-count verify, slotomatic guard rt-only). After an `ng-deploy` build, the operator runs `verify-module.sh` on `build/libs`.

---

## 5. Delta inventory — counts by target file

### 5.1 Documentation (PR1)

| Target file | Items | IDs | Est. authored lines |
|---|---|---|---|
| `types/dashboard.md` | 22 | C1–C13, G2, G3, G4, G5, G10, J1–J4 | ~150 |
| `types/logic.md` | 9 + 2 link renames | A1–A4, G1, G6, G7, G8, G9 | ~70 |
| `build-verify.md` | 8 + checklist + STORED recipe + `-ux` gap note | B1–B8 (H1/H2/H3/B2 corrections) | ~90 |
| `METHODOLOGY.md` | 2 | D1, D2 | ~20 |
| `BUILD-LOOP.md` | 1 | preflight sub-step in step 0 | ~12 |
| `SOURCES.md` | 1 + See-also | ColdRoomPan path (WSL primary, `/mnt/c` fallback) | ~10 |
| `types/wb-widgets.md` | 0 lessons | 2 link renames + See-also only (stays SEED) | ~4 |
| `retros/` × 3 | — | verbatim import, first commit | ~200 (imported, not authored) |
| **Total** | **41 lessons** | A:4 · B:8 · C:13 · D:2 · G:10 · J:4 | **~356 authored** |

### 5.2 Toolbelt + tests (PR2)

| Target file | Action | Est. authored lines |
|---|---|---|
| `toolbelt/verify-module.sh` | NEW | ~90 |
| `toolbelt/build.sh` | rewrite | ~80 (net ~+50) |
| `toolbelt/mirror-niagara-home.sh` | promote + guard | ~50 |
| `toolbelt/stored-repack.sh` | NEW | ~30 |
| `tests/verify-module.bats` | NEW | ~130 |
| `tests/build-sh.bats` | NEW | ~90 |
| `tests/mirror-niagara-home.bats` | NEW | ~50 |
| `tests/kit-links.bats` | NEW | ~35 |
| `scripts/ng-deploy.sh` | mode only | 0 |
| **Total** | | **~555** |

### 5.3 Release (PR3)

`README.md` + 3 retro markers + `CONTRIBUTING.md` + `docs/GOTCHAS.md` + `CHANGELOG.md` + `VERSION` + launcher `SKILL.md` ≈ **~90 authored lines**.

---

## 6. Tests — each guards a named regression

Operator rule: no filler tests. Every case below names the regression it catches.

| Test file | Case | Regression it catches |
|---|---|---|
| `tests/verify-module.bats` | forged bytecode major `65` on a **non-first** `.class` | A Java-11 class shipping inside an otherwise-52 jar — the exact blind spot of the current first-class-only check in `build.sh` |
| | missing `NIAGARA4.SF` | An unsigned jar deployed to a station that enforces signing (B6) |
| | `module.xml` `<type>` with no matching `.class` | The `Type "…" not found` station boot-loop class of defect |
| | `<type>` count vs `src/module-include.xml` mismatch | A type added to the annotation but never regenerated by slotomatic |
| | baja `vendorVersion` ≠ `--target-version` | B1 — a 4.15-built jar rejected by a 4.14 station |
| | `--stored` on a deflated jar | B7 — a deflated jar handed to Workbench re-sign, producing `invalid entry compressed size` |
| | raw-double facet grep non-empty | `BFacets.make(BFacets.MIN, -40)` with a raw double instead of a typed facet |
| | stored-repack output has zero `Defl:` and keeps manifest/signature ordering | A repack that silently re-deflates or reorders `META-INF`, re-breaking the B7 fix |
| `tests/build-sh.bats` | stub `-wb` dir with a gradle file but **no sources** | The confirmed `DashboardPan-wb` regression — wb tasks added to every DashboardPan build |
| | no args | Bare `${1:?}` aborting with no usage message (must print usage, exit 2) |
| | target dir that is not a `niagara_home` | Building against a dir with no `etc/m2` → the "plugin not found" failure (B3/B4) |
| | `--profiles` override | Auto-detection silently overriding an explicit operator choice |
| | `verify-module.sh` invoked after gradle | The gate being silently skipped, which is today's default state |
| `tests/mirror-niagara-home.bats` | target == source install, or a path inside it | Destroying a real `niagara_home` |
| | target dir lacking a `.niagara-mirror` marker | Wiping an arbitrary directory the operator typed by mistake |
| `tests/kit-links.bats` | every relative ref in kit `*.md` resolves | The existing dangling `checklist-common.md` / `type-dashboard.md` refs, and any future rename |
| `tests/ng-deploy.bats` | **untouched** | Regression guard: must stay 26/26 green |

Fixtures: all jars generated in-test with `printf` + `zip`. Zero committed binaries.

---

## 7. Risks

| # | Risk | Likelihood | Mitigation |
|---|---|---|---|
| R1 | PR1 exceeds the 800-line session budget once the ~200-line verbatim retro import is counted | Med | Retro import is its own first commit (imported, not authored); `types/dashboard.md` fold-in splits into 2 commits if the diff grows |
| R2 | `bats` is not installed; the PR2 gate cannot run, but the `CONTRIBUTING.md` install step lands in PR3 | High | QA installs `bats-core` locally before PR2; ordering is a known documentation lag, not a blocker. Flag to the operator if a documented-before-used ordering is preferred |
| R3 | The launcher `SKILL.md` is outside git — not in any PR diff, not revertible with `git revert` | High | PR3 description carries the full before/after; engram records it; rollback is a manual re-edit |
| R4 | `mirror-niagara-home.sh` is destructive; a guard bug wipes a real install | Low | Two independent guards (source-install refusal + `.niagara-mirror` marker) each with its own bats case; guards are tested before promotion |
| R5 | Station-only lessons (B6, A1, G2, G3) cannot be QA-reproduced in WSL | Certain | Fold with the `[CERT-live]` marker and a bitácora § reference; never assert them in a test |
| R6 | A superseded claim (H1–H4, B2, C4) gets folded alongside its correction | Med | §4 correction table is the fold-in checklist; `sdd-verify` greps the kit for the superseded strings (`transient build state`, `BFrozenEnum`, `7.6.1`) |
| R7 | 41 items into 7 files produces a diff a reviewer cannot hold in mind | Med | `cognitive-doc-design`: each fold-in is a labelled subsection with evidence, grouped by lesson family, one commit per target file |
| R8 | Chained PRs drift — PR2's diff shows PR1's commits | Med | PR2 branches only after PR1 merges; rebase until the child diff is clean |
| R9 | `verify-module.sh` rejects a jar that a station would actually accept (false positive on a legitimate build) | Low | `--stored` and `--target-version` are opt-in; the default check set is only what QA reproduced against real DashboardPan jars |

---

## 8. Rollback plan

| Slice | Rollback |
|---|---|
| PR3 | `git revert` the release commit (VERSION/CHANGELOG/markers/docs). The launcher `SKILL.md` is **manual**: restore from the before/after block in the PR description |
| PR2 | `git revert` — the new toolbelt scripts and tests are additive; `build.sh` reverts to its previous body; `ng-deploy.sh` exec bit reverts with `git update-index --chmod=-x` |
| PR1 | `git revert` — doc-only. The retro import commit can be kept independently (it only adds already-existing untracked files) |

No station, no deployed jar, and no operator data is touched by any slice. Rollback risk is limited to the developer workstation.

---

## 9. Success criteria

- [ ] All 41 doc lessons appear in their target file with an evidence reference; the 6 corrections in §4 are applied and no superseded string survives.
- [ ] `tests/kit-links.bats` passes — zero dangling relative refs in kit `*.md`.
- [ ] `tests/ng-deploy.bats` stays 26/26 green, unmodified.
- [ ] New bats suites green; every case maps to a row in §6.
- [ ] `shellcheck` exit 0 on `toolbelt/*.sh`, `scripts/ng-deploy.sh`, and all `tests/*.bats`.
- [ ] `verify-module.sh` run against the real DashboardPan `build/libs` reproduces the QA-confirmed outcome (deflated jars pass without `--stored`, fail with it).
- [ ] `build.sh` on DashboardPan selects `-rt` and `-ux` and **skips** the `-wb` stub.
- [ ] `git ls-files -s scripts/ng-deploy.sh` shows mode `100755`.
- [ ] The 3 kit retros are tracked in git and carry `<!-- review-status: folded v0.2 · 2026-09-01 -->`; **no retro file is deleted** (propose-never-apply).
- [ ] `VERSION` = `0.4.0`, `CHANGELOG.md` entry written in repo format, kit `README.md` states v0.2.
- [ ] Launcher `SKILL.md` states `ng-deploy.sh --strict-slotomatic` primary / `build.sh` WSL fallback, and version `0.2`.
- [ ] Each slice merges in order PR1 → PR2 → PR3 with a clean child diff.

---

## 10. Review workload note

Session `review_budget_lines` = **800**; `delivery_strategy` = **auto-chain**.

| Slice | Est. authored lines | Budget risk |
|---|---|---|
| PR1 docs | ~356 authored (+~200 imported retro lines in a separate commit) | Medium |
| PR2 toolbelt + tests | ~555 | Medium |
| PR3 release | ~90 | Low |

`Decision needed before apply: No` (auto-chain already resolves the slicing).
`Chained PRs recommended: Yes` — 3 slices, fixed merge order.
`800-line budget risk: Medium` — each slice fits; PR2 is the tightest and must not absorb any PR1 doc work.

---

## 11. Dependencies

- `bats-core` installed before PR2's gate can run (`brew install bats-core`).
- `shellcheck` — already installed.
- PR1 must merge before PR2 branches; PR2 before PR3.
- Read-only: `niagara-research` docs (group I pointers), `Cliente/Leon-Guanjuato/bitacora/` (source of groups G and J).

---

## 12. Frozen decisions (do not re-open)

1. Option 2 selected; release as niagara-tools `v0.4.0` (MINOR), kit `v0.2`.
2. Three chained PRs, fixed order and ownership as in §4.
3. Doctrine wording as in §4.1; `verify-module.sh` integration into `ng-deploy.sh` is deferred; the `-ux` slotomatic gap is documented, not fixed.
4. Open questions Q1–Q6 from exploration are resolved: bitácora diff done (groups G/J); STORED = recipe + `stored-repack.sh` + `--stored` opt-in check; `verify-module.sh` standalone; SOURCES.md path updated; `-ux` gap documented only; retros committed verbatim in PR1 with markers added in PR3.
5. Every test names a real regression (§6). No filler tests.
6. Non-goals in §2.2 stand.

---

## 13. Next phases

- `sdd-spec` — formalize `verify-module.sh` check set, flags and exit codes; `build.sh` usage/profile-selection/exit contract; `mirror-niagara-home.sh` guard predicates; the kit doctrine rules. Can run in parallel with `sdd-design`.
- `sdd-design` — fold-in placement per target file (section structure, ordering, evidence-reference format), bats fixture-generation helpers, and the three-branch chain mechanics.
- Then `sdd-tasks` → `sdd-apply` (TDD strict for scripts) → `sdd-verify` → `sdd-archive`.
