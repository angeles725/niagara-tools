# Spec: client gitignore for build/tmp and *.class (S26)

**Capability**: `client-gitignore` — `angeles725/niagara-panccadia-leon` `.gitignore`
**PR**: PR6 (`chore/c10-gitignore-build-caches`)
**QA RED**: No RED (chore PR). Carries a diff-shows-no-jar check.
**Repo**: `angeles725/niagara-panccadia-leon` (client)
**Cross-cutting**: see `../cross-cutting.md` (K11/K12/K13/K14/K19/K20/D9b/comment-strip/observed-flip/real-tree-smoke) — K19 does not apply (no toolbelt script). No jar, no slot, no deploy.

**Note on S26 + S25 coupling**: This PR also marks the 5 concept rows in `docs/write-path-matrix.md` (hoaMode ×3, inhibit, freezeEnabled) with the literal `[concept]` token, changing the `lint-write-path.sh` STALE count from **5 → 0** (the OBSERVED real-tree flip). This is the single S26 chore that closes both the gitignore churn and the write-path concept-row advisory at once.

---

## Delta — ADDED/MODIFIED requirements

| ID | Requirement | ev |
|----|-------------|-----|
| R-S26.1 | The client `.gitignore` MUST include patterns that prevent `build/tmp/` and all `*.class` files from appearing as untracked or modified in `git status --porcelain` after a clean build. | `[ev: proposal R6]` `[ev: explore §4 S26 45d550dac]` |
| R-S26.2 | Every file currently tracked under `build/` — specifically the 4 module libs jars and the 4 `module.xml` files — MUST remain tracked. `git ls-files` output for tracked `build/**/*.jar` and `build/**/module.xml` MUST be **byte-identical before and after** the `.gitignore` change. | `[ev: proposal SC-6]` `[ev: proposal RK7]` |
| R-S26.3 | If R-S26.2 cannot be satisfied with the planned ignore patterns, PR6 is **dropped**, not weakened — the tracked jars are the authoritative build artifacts and MAY NOT be silently untracked. | `[ev: proposal RK7]` |
| R-S26.4 | No `vendorVersion` bump is required for this chore (no new jar, no slot change). | `[ev: proposal R6]` |
| R-S26.5 | The 5 matrix rows in `docs/write-path-matrix.md` that have no matching source slot (hoaMode at `:31`, `:32`, `:52`; inhibit at `:33`; freezeEnabled at `:36`) MUST be marked with the literal token `[concept]` in the row body, so `lint-write-path.sh` reports **0 STALE rows** on DashboardPan after this PR. | `[ev: coordinator 2026-09-07]` `[ev: specs/write-path-stale/spec.md R-S25.11]` |
| R-S26.6 | No other matrix rows MUST be changed by this PR (no existing FAIL rows added or removed). | `[ev: proposal K12]` |

---

## Scenarios

### S26-clean (git status clean after build)

**Given** a client build has run producing `build/tmp/` and `*.class` files.

**When** `git status --porcelain` is checked.

**Then** NO `build/tmp` or `*.class` entry appears (those paths are gitignored).

`[ev: proposal SC-6]`

---

### S26-jar-tracked (tracked jars unchanged)

**Given** the `.gitignore` change has been applied.

**When** `git ls-files | grep 'build/'` is run before and after the change.

**Then** the 4 libs jars and 4 `module.xml` files are **listed identically** both times. No tracked `build/**/*.jar` entry disappears.

`[ev: proposal SC-6]` `[ev: R-S26.2]`

---

### S26-stale-flip (write-path STALE 5 → 0 after concept markers)

**Given** DashboardPan module root at client `ff1b659` BEFORE this PR (5 STALE rows: hoaMode ×3, inhibit, freezeEnabled).

**When** `lint-write-path.sh` runs on DashboardPan AFTER the S26 `[concept]` edits.

**Then** exits **0** with **0 STALE rows** (OBSERVED real-tree flip from 5 → 0).

`[ev: coordinator 2026-09-07]` `[ev: specs/write-path-stale/spec.md R-S25.11]`

---

### S26-no-vendorVersion (chore — no version bump)

**Given** the PR6 diff.

**When** `build.gradle.kts` files are inspected.

**Then** no `vendorVersion` line has changed.

`[ev: R-S26.4]`

---

## Success criteria (this capability)

- [ ] After a client build, `git status --porcelain` shows no `build/tmp` or `*.class`.
- [ ] `git ls-files` for every tracked `build/**/*.jar` and `build/**/module.xml` is **identical before and after** — no tracked jar or module.xml becomes untracked.
- [ ] The 5 concept matrix rows marked with `[concept]`; `lint-write-path.sh` on DashboardPan reports **0 STALE rows** (OBSERVED flip from 5).
- [ ] No `vendorVersion` changes.
- [ ] No new jar, no deploy, `schema-risk.sh` has no jurisdiction.
- [ ] 0 attribution trailers.
