<!-- review-status: folded -->
# Retro: Campaign 8 — lint-structure.sh module source-tree structure lint (Campaign 8 PR18, B817)

**Date:** 2026-09-06
**Change:** build-n4-module-campaign8 PR18 — `feat/c8-structure`
**Scope:** `build-n4-module-kit/toolbelt/lint-structure.sh` (new); `build-n4-module-kit/types/structure.md` (new); K19 routing in `BUILD-LOOP.md` §5 + `skill/SKILL.md`; scaffold-module.sh verified (no changes needed); LS-PASS bats pin already in RED
**Lead:** Campaign 8 PR18 executor (apply worker — feat/c8-structure worktree)
**RED tip:** `902f2d7` (LS4, LS9, LS10, LS11, LS7, LS-prune, LS-usage, LS-PASS — 8 pins)

---

## What happened

`lint-structure.sh <module-root>` iterates every profile under `<module-root>` by finding each
`module-include.xml` (dot-directories pruned, D9b). Source-tree-only; L8 (signed-jar) stays in
`verify-module.sh` (D15). Ten checks implemented: L1–L7 + L9–L11 (L3 is the only WARN).

Row format: `FAIL|WARN  lint-structure  <path>  L<n>: <reason>`. Exit 0 clean / 1 FAIL / 3 usage.

**Key fix during GREEN**: L2 (`@NiagaraType` count) was matching Javadoc comment text as well
as actual annotations. The pattern `grep -c '@NiagaraType'` matched a `* @NiagaraType / ...`
comment in `BMinimalPan.java`, returning count 2 instead of 1. Fixed by anchoring the pattern
to `^[[:space:]]*@NiagaraType` (annotation-position only, not comment text).

**Key fix for palette empty-detection**: The initial `grep -q '<p '` check considered any `<p`
tag as a non-empty palette, but real DashboardPan-wb palette has `<p m="b=baja" t="b:Folder">`
(an outer container) with no named component children. Changed to `grep -q '<p n='` to detect
actual named component entries. The DashboardPan-wb fixture (which has no `<p` at all) passes
either check; the real tree now correctly registers as empty.

K19 routing: `lint-structure.sh` added to `BUILD-LOOP.md` §5 pre-gate (before `lint-delays.sh`)
and `skill/SKILL.md` step 5 + toolbelt References list. `types/structure.md` added to SKILL.md
Decision Gates table. `[ev: retro campaign8-structure]` in script header and routing lines.
kit-links.bats L4/L5: green.

`scaffold-module.sh` output already passes L1–L11 (MinimalPan fixture correct by inspection).
LS-PASS pin was in the RED; it turns green once `lint-structure.sh` exists.

Final bats: **292/292** green (8 new LS pins). shellcheck exit 0. All sweep guards exit 0.
kit-links.bats 7/7 (L4/L5 green — `lint-structure.sh` and `types/structure.md` routed).

---

## Evidence

### TDD evidence

| Phase | Evidence |
|-------|----------|
| RED | Commit `902f2d7` on `feat/c8-structure`: LS4–LS11 + LS-prune + LS-usage + LS-PASS all failing because `lint-structure.sh` was absent (exit 127). |
| GREEN | `lint-structure.sh` implemented; all 8 LS tests turn green. Fix 1: L2 Javadoc comment matching (anchored to `^[[:space:]]*@NiagaraType`). Fix 2: palette empty-detection (`<p n=` not `<p `). Full suite 292/292. |
| REFACTOR | None: two targeted fixes during GREEN; no structural change needed post-green. |

### Named mutations (all on mktemp copies; originals untouched)

| Mutation | Change applied | Test flipped | Observed output |
|----------|---------------|--------------|-----------------|
| (a) remove 3-part floor | Use `deps` fixture (`:baja:4.14` 2-part floor — fixture is the mutation target per R18.5) | LS7 expected FAIL L7 | `FAIL  lint-structure  Foo-rt/Foo-rt.gradle.kts  L7: dependency version is not a 3-part floor (use X.Y.Z e.g. 4.14.0, not X.Y e.g. 4.14)` + L4/L5 FAILs on fixture (no lexicon/palette); exit 1 |
| (b) drop L4 rule | `sed 's|_row FAIL.*L4:.*|: # L4 disabled|g'` on copy | LS4 flips | No L4 output on chihuahua-rt; only `FAIL  lint-structure  chihuahua-rt  L5: module.palette missing for -rt profile`; exit 1 (L4 not present — LS4 bats would fail `*"L4"*` check) |
| (c) drop L9 rule | `sed 's|_row FAIL.*L9:.*|: # L9 disabled|g'` on copy | LS9 flips | Empty output on dashboardpan-wb fixture; exit 0 (was 1) |
| (d) drop dot-dir prune | `sed "s|-name '\\.\*' -prune|-name 'DISABLED_PRUNE' -prune|g"` on copy | LS-prune flips | `FAIL  lint-structure  ...coldroompan-rt/.deploy-baseline/gradle.properties  L10: absolute host path in tracked gradle.properties`; exit 1 — L10 now appears alongside L11 |

### Real smokes (read-only trees; verbatim rows)

**Smoke 1: chihuahua module root** (`/home/cristian/modulos_niagara_n4/Cliente/Honeywell/MX60/chihuahua/chihuahua`)

```
FAIL  lint-structure  chihuahua-rt/module.lexicon  L4: lexicon empty (8 type(s) declared, zero key=value entries)
FAIL  lint-structure  chihuahua-ux/module.lexicon  L4: lexicon empty (1 type(s) declared, zero key=value entries)
FAIL  lint-structure  chihuahua-wb  L4: module.lexicon missing (1 type(s) declared in module-include.xml)
exit: 1
```

L4 fires on chihuahua-rt (8 declared types, empty lexicon) + chihuahua-ux (1 type, empty lexicon) + chihuahua-wb (1 type, no lexicon at all). No L5/L9/L10/L11 false positives.

**Smoke 2: DashboardPan module root** (`/home/cristian/modulos_niagara_n4/Cliente/Leon-Guanjuato/Dashboard/DashboardPan`)

```
exit: 0
```

False negative noted (recorded as Δ1): DashboardPan-wb has no `module-include.xml` in the real tree (not processed as a profile by the tool's find-by-module-include.xml strategy). The `gradle.properties` with absolute paths is at the parent `Dashboard/` level, not inside the `DashboardPan/` module subdir. The sanitized fixture (`dashboardpan-wb`) correctly represents the design target; the real tree has a non-standard layout. Campaign-9 seed records this as structure-lint false-positive/negative tuning.

**Smoke 3: ColdRoomPan module root** (`/home/cristian/modulos_niagara_n4/Cliente/Leon-Guanjuato/Paccadia/ColdRoomPan`)

```
exit: 0
```

Real srcTest uses pure JUnit (no BTest/BTestNg import) — not mixed → L11 correct absence. The sanitized `coldroompan-rt` fixture uses a synthetic `BColdTest.java` with BOTH imports to prove the rule.

**Smoke 4: CompPan module root** (`/home/cristian/modulos_niagara_n4/Cliente/Leon-Guanjuato/Compresores/CompPan`)

```
exit: 0
```

Same as ColdRoomPan — pure JUnit srcTest, no BTest mix → L11 does not fire.

**Smoke 5: scaffold-module.sh MinimalPan output**

```
scaffold-module: emitted MinimalPan -> /tmp/tmp.2te0adztmj/MinimalPan
exit: 0
```

Scaffold output passes L1–L11 at exit 0. LS-PASS bats pin green.

---

## Proposed kit deltas

| ID | Target | Delta |
|----|--------|-------|
| Δ1 ✓ | `lint-structure.sh` profile discovery + L10 upward walk | **CLOSED in same PR (lead review).** Two root causes: (1) the original discover-by-`module-include.xml` strategy made a profile without that file (real DashboardPan-wb) invisible to L9/L6; (2) L10 only scanned under the module root, missing `gradle.properties` kept at the parent project dir (`Dashboard/gradle.properties`). Fix: profile discovery changed to direct-child dirs matching `*-(rt|ux|wb|se|doc)` or containing `build.gradle.kts`/`build.gradle`/`module-include.xml`; a profile without `module-include.xml` fires L6 and still runs L9; L10 walks parent dirs up to the `.git` sentinel. New pins: `LS9-real` + `LS10-real`. Real smokes all correct: chihuahua L4+L10; DashboardPan L6+L9+L10; ColdRoomPan L10; CompPan L10; scaffold exit 0. |
| Δ2 | `lint-structure.sh` L2 | Javadoc comment anchor: `@NiagaraType` in Javadoc `* @NiagaraType` lines is NOT an annotation. Fixed by anchoring to `^[[:space:]]*@NiagaraType`. Add a fixture comment explaining this in `tests/fixtures/lint-structure/` if a new fixture covers it. |
| Δ3 | `lint-structure.sh` L5/L9 | Empty palette must be `<p n=` (named component), not `<p ` (any tag). A palette with only a `<p m="b=baja" t="b:Folder">` outer container is empty. Fixed in this PR. |

---

## Lessons

1. `grep -c '@NiagaraType'` matches Javadoc text; anchor to `^[[:space:]]*@NiagaraType` to detect only annotation-position occurrences.
2. A Niagara palette is FUNCTIONALLY empty when it has no `<p n=` named component entries, even if it has a Folder container element (`<p m="b=baja" t="b:Folder">`).
3. A profile without `module-include.xml` must still be discovered (by name pattern `*-(rt|ux|wb|se|doc)` or by containing `build.gradle.kts`/`build.gradle`) and checked for L6 (missing) and L9 (empty skeleton). Discover-by-include-file alone is insufficient.
4. L10 must walk up parent dirs to the repo root (`/.git` sentinel) because client projects commonly keep `gradle.properties` (with absolute host paths) at the project level above the module subdir. Walking within the module root alone misses these.
5. Sanitized fixtures prove the design rules; a scaffold that fails its own lint is a RED signal that a rule has a false positive (`LS-PASS` pin).
