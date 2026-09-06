# Spec: campaign close — v0.22.0 (R5)

**Capability**: `campaign-close`
**PR**: PR5 (`chore/c11-close`)
**QA RED**: `qa/c11-close-checklist` (skeleton; QA freezes the `TODO(freeze)` pins at close) — PR opens only AFTER PR1–PR4 have merged. `[ev: proposal PR5 row; explore §3.2]`
**Repo**: `niagara-tools` (kit)
**Cross-cutting**: see `../cross-cutting.md` (K11/K12/K13/K19/K20/K21/K22/K24(7)/D9b/comment-strip/fragment-merge)
**Precondition**: PR5 opens ONLY after PR1–PR4 have merged and their retros have been created and INDEX rows flipped.

---

## Delta — ADDED/MODIFIED requirements

| ID | Requirement | ev |
|----|-------------|-----|
| R-C11.1 | `VERSION` file MUST read **`0.22.0`** after PR5 merges. `0.22.0` is a MINOR bump: two new capabilities (`kit-method-boundary-parser`, `module-guard-pin-lint`), two additive verdict/behaviour widenings (`module-timer-lint` + `module-silent-protection-lint` now catch one-liner methods, `module-ext-writable-shape-lint` gains the accessor skip), one additive advisory class (`DRIFT` in `module-write-path-matrix`), one new test-harness library; no CLI removal. | `[ev: proposal R5; CONTRIBUTING §4]` |
| R-C11.2 | `CHANGELOG.md` MUST contain a `## 0.22.0` section with an entry per CONTRIBUTING §4-5. | `[ev: proposal R5]` |
| R-C11.3 | `retros/INDEX.md` `pending` count MUST be **0** at the time PR5 is opened. Every kit-changing push (PR1–PR4, one retro each) MUST have its retro file created and its `retros/INDEX.md` row flipped before PR5 opens. | `[ev: proposal R5; BUILD-LOOP.md §7]` |
| R-C11.4 | `tests/c11-close.bats` MUST exist and assert all of the following under `C11_CLOSE=1`: `VERSION = 0.22.0`; `CHANGELOG.md` has a `## 0.22.0` section; `retros/INDEX.md` pending = 0; `sweep-build-state.sh` exits 0; `sweep-fold-audit.sh --strict` exits 0; BASE pin = `dab0807`; `C11_CLOSE_COMMIT` is set; tool-pins include `toolbelt/lib/method-boundary.sh` AND `toolbelt/lint-guard-pins.sh` (two new files added in C11). | `[ev: qa/c11-close-checklist skeleton; proposal PR5 row]` |
| R-C11.5 | `shellcheck 0.10.0` MUST exit 0 on every new or modified `toolbelt/*.sh` and `toolbelt/lib/*.sh` in the PR1–PR5 range. This includes `toolbelt/lib/method-boundary.sh` and `toolbelt/lint-guard-pins.sh`. | `[ev: proposal R5]` |
| R-C11.6 | The close gate MUST scan the full commit body range PR1–PR5 and assert **0 attribution trailers** (Co-Authored-By, Generated-by, or equivalent) (K11). | `[ev: K11]` `[ev: proposal §Process invariants]` |
| R-C11.7 | `kit-links.bats` MUST exit 0 after PR5, verifying every script touched or added in C11 is reachable from both `BUILD-LOOP.md` §5 and `skill/SKILL.md`. This includes `toolbelt/lib/method-boundary.sh` (routing added in PR1) and `toolbelt/lint-guard-pins.sh` (routing added in PR4). | `[ev: K19]` `[ev: proposal SC-11]` |
| R-C11.8 | `sweep-build-state.sh` and `sweep-fold-audit.sh --strict` MUST exit 0. | `[ev: proposal SC-13]` |
| R-C11.9 | The C11 close retros fold the following campaign lessons into the kit as candidate K-entries (pending kit-ticket / METHODOLOGY edit): (a) a real-tree smoke MUST pin the current clean state, not a known bug (the LD5 lesson — R-T2.6/R-T2.10); (b) when the kit has N copies of a logic fragment, a regression in one copy silently diverges from the others (the three-parser lesson — R-T1.1). No retro fold is a blocker for PR5 if the retro files are created and pending = 0; the K-entry content is a best-effort at close. | `[ev: proposal §Close; BUILD-LOOP §7]` |

---

## Scenarios

### close-gate (c11-close.bats under C11_CLOSE=1)

**Given** `tests/c11-close.bats` run with `C11_CLOSE=1` and `C11_CLOSE_COMMIT` set to the actual merged close commit SHA.

**When** the bats suite runs.

**Then** all assertions pass:
- `VERSION = 0.22.0`
- BASE pin = `dab0807`
- `retros/INDEX.md` pending = **0**
- `sweep-build-state.sh` exits 0
- `sweep-fold-audit.sh --strict` exits 0
- Tool-pins include `toolbelt/lib/method-boundary.sh` and `toolbelt/lint-guard-pins.sh`

`[ev: qa/c11-close-checklist skeleton]` `[ev: R-C11.4]`

---

### no-trailer (0 attribution trailers in entire PR1–PR5 range)

**Given** all commit bodies in the PR1–PR5 range.

**When** scanned for attribution trailers (`Co-Authored-By`, `Generated-by`, or equivalent).

**Then** **0 trailers** found.

`[ev: K11]` `[ev: R-C11.6]`

---

### kit-links-green (routing completeness — new scripts present)

**Given** all scripts touched or added in C11: `toolbelt/lib/method-boundary.sh`, `toolbelt/lint-timers.sh`, `toolbelt/lint-silent-protection.sh`, `toolbelt/lint-ext-writable-shape.sh`, `toolbelt/lint-write-path.sh`, `toolbelt/lint-guard-pins.sh`.

**When** `kit-links.bats` runs after PR5.

**Then** exits 0 — every script is reachable from both `BUILD-LOOP.md` §5 and `skill/SKILL.md`, including routing rows for `lib/method-boundary.sh` (from PR1) and `lint-guard-pins.sh` (from PR4).

`[ev: K19]` `[ev: R-C11.7]`

---

### retros-pending-zero (no retro debt at close)

**Given** `retros/INDEX.md` at the time PR5 is opened.

**When** the pending count is read.

**Then** pending count = **0**. Each of the four kit-changing PRs (PR1–PR4) has a retro file created and its row flipped.

`[ev: R-C11.3]` `[ev: BUILD-LOOP.md §7]`

---

## Success criteria (this capability)

- [ ] `tests/c11-close.bats` green under `C11_CLOSE=1` with `C11_CLOSE_COMMIT` set.
- [ ] BASE pin `dab0807`; `retros/INDEX.md` pending = **0**.
- [ ] `VERSION` = **`0.22.0`**; `CHANGELOG.md` has a `## 0.22.0` section per CONTRIBUTING §4-5.
- [ ] Tool-pins include `toolbelt/lib/method-boundary.sh` AND `toolbelt/lint-guard-pins.sh`.
- [ ] `sweep-build-state.sh` and `sweep-fold-audit.sh --strict` green.
- [ ] `shellcheck 0.10.0` exits 0 on all modified/added toolbelt scripts.
- [ ] **0 attribution trailers** in the entire PR1–PR5 range (K11).
- [ ] `kit-links.bats` green with both new scripts routed.
- [ ] C11 close lessons recorded as retro folds (pending = 0 is the hard gate; lesson content is best-effort).
