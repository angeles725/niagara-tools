# Spec: campaign close — v0.21.0 (R7)

**Capability**: `campaign-close`
**PR**: PR7 (`chore/c10-close`)
**QA RED**: `qa/c10-close-checklist` tip `41bca42` (skeleton; QA freezes the `TODO(freeze)` pins at close) (base `df8c7ec`)
**Repo**: `niagara-tools` (kit)
**Cross-cutting**: see `../cross-cutting.md` (K11/K12/K13/K14/K19/K20/D9b/comment-strip/observed-flip/real-tree-smoke)
**Precondition**: PR7 opens only AFTER PR1–PR5 have merged (and PR6 is independent/client).

---

## Delta — ADDED/MODIFIED requirements

| ID | Requirement | ev |
|----|-------------|-----|
| R-C10.1 | `VERSION` file MUST read **`0.21.0`** after PR7 merges. `0.21.0` is a MINOR bump (one narrowed behaviour in S22 documented as a contract change; no CLI removal). | `[ev: proposal R7]` `[ev: CONTRIBUTING §4]` |
| R-C10.2 | `CHANGELOG.md` MUST contain a `## 0.21.0` section with an entry per CONTRIBUTING §4-5. | `[ev: proposal R7]` |
| R-C10.3 | `retros/INDEX.md` `pending` count MUST be **0** at close. Every kit-changing push (PR1–PR5, one retro each) MUST have its retro file created and its `retros/INDEX.md` row flipped before PR7 opens. | `[ev: proposal R7]` `[ev: BUILD-LOOP.md §7]` |
| R-C10.4 | `tests/c10-close.bats` MUST assert: `VERSION = 0.21.0`; `CHANGELOG.md` has a `## 0.21.0` section; `retros/INDEX.md` pending = 0; `sweep-build-state.sh` exits 0; `sweep-fold-audit.sh --strict` exits 0; BASE pin = `1fb63d6`; `C10_CLOSE_COMMIT` is set; tool-pins = the C9 set (no new tool file added in C10). | `[ev: qa/c10-close-checklist 41bca42]` |
| R-C10.5 | `shellcheck 0.10.0` MUST exit 0 on every new or modified `toolbelt/*.sh` in the PR1–PR7 range. | `[ev: proposal R7]` |
| R-C10.6 | The close gate MUST scan the full commit body range PR1–PR7 and assert **0 attribution trailers** (K11). | `[ev: K11]` `[ev: qa/c10-close-checklist 41bca42]` |
| R-C10.7 | `kit-links.bats` MUST exit 0 after PR7, verifying every script touched in C10 is reachable from both `BUILD-LOOP.md` §5 and `skill/SKILL.md`. | `[ev: K19]` |
| R-C10.8 | `sweep-build-state.sh` and `sweep-fold-audit.sh --strict` MUST exit 0. | `[ev: qa/c10-close-checklist 41bca42]` |

---

## Scenarios

### close-gate (c10-close.bats under C10_CLOSE=1)

**Given** `tests/c10-close.bats` run with `C10_CLOSE=1` and `C10_CLOSE_COMMIT` set to the actual merged close commit.

**When** the bats suite runs.

**Then** all assertions pass:
- `VERSION = 0.21.0`
- BASE pin = `1fb63d6`
- `retros/INDEX.md` pending = **0**
- `sweep-build-state.sh` exits 0
- `sweep-fold-audit.sh --strict` exits 0
- Tool-pins = the C9 set (no new tool file)

`[ev: qa/c10-close-checklist 41bca42]`

---

### no-trailer (0 attribution trailers in entire range)

**Given** all commit bodies in the PR1–PR7 range.

**When** scanned for attribution trailers (`Co-Authored-By`, `Generated-by`, or equivalent).

**Then** **0 trailers** found.

`[ev: K11]` `[ev: R-C10.6]`

---

### kit-links-green (routing completeness)

**Given** all scripts touched or added in C10 (lint-timers.sh, lint-ext-writable-shape.sh, lint-silent-protection.sh, lint-write-path.sh, run-pure-test.sh).

**When** `kit-links.bats` runs after PR7.

**Then** exits 0 — every script is reachable from both `BUILD-LOOP.md` §5 and `skill/SKILL.md`, including the new `--strict` flag row for `lint-write-path.sh`.

`[ev: K19]` `[ev: R-C10.7]`

---

## Success criteria (this capability)

- [ ] `tests/c10-close.bats` green under `C10_CLOSE=1` with `C10_CLOSE_COMMIT` set.
- [ ] BASE pin `1fb63d6`; `retros/INDEX.md` pending = **0**.
- [ ] `VERSION` = **`0.21.0`**; `CHANGELOG.md` has a `## 0.21.0` section per CONTRIBUTING §4-5.
- [ ] `sweep-build-state.sh` and `sweep-fold-audit.sh --strict` green.
- [ ] `shellcheck 0.10.0` exits 0; **0 attribution trailers** in the entire PR1–PR7 range.
- [ ] `kit-links.bats` green.
- [ ] Tool-pins = C9 set (no new tool file added in C10).
