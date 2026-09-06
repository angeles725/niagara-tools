<!-- review-status: folded -->
# Retro: campaign8-build-pipeline

**Session**: 2026-09-05 · Campaign 8 PR14 — Build pipeline + module versioning doctrine  
**Scope**: BUILD-LOOP.md §4 (Gradle task matrix + version-bump checklist) + exit-31 regression pin  
**Kit version**: v0.18.0 → v0.19.0 (pending close)

---

## What happened

Campaign 8 PR14 adds the Gradle task matrix (§4.a) and version-bump checklist (§4.b) to BUILD-LOOP.md,
and pins the exit-31 station-lock behaviour in `tests/build-sh.bats` (BS-lock + BS-lock-hint).
The exit-31 fix was already live at `build.sh:82-88` (48fb210); this PR adds the regression guard so
the behaviour cannot be silently deleted.

The doc-only work was research-gated on block B807 (niagara-module plugin task matrix and station-lock
contract). The pin proves the existing behaviour is tested before folding the doctrine.

## Evidence: grep counts + anchors

**grep-before-fold** (`grep -cE 'gradle|niagara-module|Out-of-date|vendorVersion|mirror'`):
- `BUILD-LOOP.md`: 7 hits (existing; all pre-PR14 references in §0.b and §6)
- `METHODOLOGY.md`: 5 hits (existing; none relevant to §4)

**New-content check** (`grep -cE 'Gradle task matrix|mirror-niagara-home|vendorVersion.*bajaVersion' BUILD-LOOP.md`):
- 1 hit before edit (existing `mirror-niagara-home.sh` mention at §0.b line 22, forward-slash form)
- New §4.a and §4.b anchors confirmed distinct; no pre-existing task-matrix or version-bump checklist section

**Anchors after edit**:
- `BUILD-LOOP.md §4.a` — Gradle task matrix table (8 tasks: clean/slotomatic/compileJava/jar/build/moduleTestJar/niagaraTest/bajadoc) + safe-combinations + station-lock recipe `[ev: corpus B807]`
- `BUILD-LOOP.md §4.b` — version-bump checklist (vendorVersion, bajaVersion, restart mandate, schema-risk.sh cross-ref to §6) `[ev: corpus B807]` `[ev: corpus B795]`
- `tests/build-sh.bats` BS-lock (exit 31, locked text) + BS-lock-hint (mirror-niagara-home.sh text) — both green-on-arrival from commit 15e4808

**Guards**:
- `bats tests/build-sh.bats`: 12/12 ok (exit 0)
- `sweep-fold-audit.sh --strict retros/INDEX.md .`: 56 folded, 56 cited, 0 uncited (exit 0)

## Proposed kit deltas

1. **BUILD-LOOP.md §4.a** (new): Gradle task matrix — operators and developers can now look up what each `niagara-module` task does, when to use it, and the safe `clean slotomatic jar` combination without consulting the plugin source. `[ev: corpus B807]`
2. **BUILD-LOOP.md §4.b** (new): Version-bump checklist — `vendorVersion` / `bajaVersion` distinction and station reload consequence documented inline in the build section where the developer sees it. `[ev: corpus B807]` `[ev: corpus B795]`
3. **tests/build-sh.bats BS-lock + BS-lock-hint** (regression pin, already live): exit-31 station-lock detection and mirror-niagara-home.sh hint are pinned; deletion of the `build.sh:82-88` branch would flip both pins to FAIL.

## Lessons

1. `gradle :jar` alone is NOT a valid build — `slotomatic` must run first or the slot proxy is stale; `build.sh` enforces `clean slotomatic jar` per-profile; never invoke gradle tasks ad-hoc without the kit wrapper. `[ev: corpus B807]`
2. `vendorVersion` must be bumped on EVERY schema change (slot add/remove/retype/rename); on reload the station re-decodes `config.bog` against the installed module's type/slot registry — a mismatch is an OUTAGE, not a warning. `[ev: corpus B807]` `[ev: corpus B795]`
3. The station-lock exit 31 (`build.sh:82-88`) and the mirror recipe were already correct before this PR; adding a regression pin (BS-lock + BS-lock-hint) is the right move for a doc-only PR that cannot change the code — the pin proves the behaviour is tested even though the PR itself is doc-only.
