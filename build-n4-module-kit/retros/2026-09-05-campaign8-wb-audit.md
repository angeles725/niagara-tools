<!-- review-status: folded -->
# Retro: Campaign 8 — wb checks (WB-LEX1/SCAFFOLD1/THREAD1/AGENT1/DEP1, Campaign 8 PR11)

**Date:** 2026-09-05
**Change:** build-n4-module-campaign8 PR11 — `feat/c8-wb-audit`
**Scope:** `toolbelt/lint-wb-threading.sh` (new); `toolbelt/slot-coverage.sh` WB-LEX1 extension; `toolbelt/verify-module.sh` WB-SCAFFOLD1 + WB-DEP1 extensions; `types/wb-widgets.md` DWB1 doctrine; K19 routing in `BUILD-LOOP.md` + `skill/SKILL.md`
**Lead:** Campaign 8 PR11 executor
**RED tip:** `734a0b6` (WBT1/WBT1c/WBA1/WBA1c/WBT-strict/WBT-prune/WBT-usage/WB-LEX1/WB-SCAFFOLD1/WB-DEP1)

---

## What happened

Three tools extended and one created to operationalize the five wb-audit checks
(B809 §809.7, WB-LEX1/SCAFFOLD1/THREAD1/AGENT1/DEP1):

- **`lint-wb-threading.sh <wb-src-dir> [--strict]`** (new): two heuristic WARNs over
  a `-wb` Java source tree. `ui-thread-traversal` fires when a `doInvoke` body directly
  calls `getNavChildren|getNavNodes|BqlQuery` without `invokeLater|BJobService|JobThread`
  in the same body (B809: flag for human review only, never a FAIL by design).
  `agent-breadth` fires when `@AgentOn(types="baja:Component")` appears without a comment
  containing `justif|why|broad` within 3 lines above. `--strict` promotes WARNs to exit 1.
  Dot-dirs pruned (D9b). LC_ALL=C, set -u, shellcheck clean. Exits 0/1/3.

- **`slot-coverage.sh` WB-LEX1** (parse subcommand extension): a missing lexicon file
  that pairs with a module-include.xml declaring ≥1 type now exits 1 (FAIL) instead of
  exit 3 (env). Was exit 3 before; the change mirrors D6a's empty-lexicon FAIL rationale
  (every operator-facing type name renders raw camelCase — this is a ship-blocker, not an
  env error).

- **`verify-module.sh` WB-SCAFFOLD1** (new `check_wb_scaffold`): a `-wb`-named jar with
  0 `.class` entries AND 0 palette `<p n=` entries is an empty scaffold — WARN by default,
  FAIL under `--strict`. Closes the `:245-257` dead angle (the `check_palette` check
  only fires when declared types ≥ 1; a typeless all-empty `-wb` jar slipped through).

- **`verify-module.sh` WB-DEP1** (new `check_phantom_dep`): under `--src`, compares
  each `<dependency name="X">` in `META-INF/module.xml` against the `api(":X")`/`nre(":X")`
  declarations in the profile `*.gradle.kts`. Undeclared deps → WARN `phantom-dep`.
  WARN only (never changes exit code independently).

- **`types/wb-widgets.md`**: 10-rule good-`-wb`-artifact doctrine with every line
  citing `[ev: corpus B809] [ev: corpus B817]`; chihuahua-wb `model/` tree (commit
  `175eee8`) cited as DWB1 exemplar.

---

## Evidence

### TDD evidence

| Phase | Evidence |
|-------|----------|
| RED | Commit `734a0b6` on `feat/c8-wb-audit` (cherry from `qa/c8-wb-audit 1e82419`): all 10 pins failing (exit 127 for absent script; incorrect exit codes for WB-LEX1/SCAFFOLD1/DEP1). |
| GREEN | All 10 pins turn green; `lint-wb-threading.sh` written; slot-coverage WB-LEX1 exit-1 path added; verify-module check_wb_scaffold + check_phantom_dep added. Full suite: 254/254. |
| REFACTOR | shellcheck exit 0 on all three scripts; SC2034 unused-local fix in check_phantom_dep local decl. |

### Named mutations (all on mktemp copies; original restored after each)

| Label | Mutation | Test that flipped | Observed output |
|-------|----------|------------------|-----------------|
| (a) drop empty-scaffold detection | Removed `check_wb_scaffold` from verify-module.sh + call loop | WB-SCAFFOLD1 | `wb-scaffold` row absent; output contains only `FAIL bytecode … no .class entries` |
| (b) drop traversal rule | Removed `_AWK_THREAD` section and its awk invocation from lint-wb-threading.sh | WBT1 | No output, exit 0 — `ui-thread-traversal` row absent |
| (c) drop invokeLater exemption | Set `has_guard = 0` in `_AWK_THREAD` awk | WBT1c | `ui-thread-traversal  WARN  GuardedTraversal.java:4  doInvoke body calls nav traversal…` |
| (d) drop dot-dir prune | Replaced `find … -type d -name '.*' -prune -o … -print` with `find … -name '*.java'` | WBT-prune | `ui-thread-traversal  WARN  .deploy-baseline/Stale.java:4  …` |
| (e) drop depth-3 callee expansion | Reverted `expand_body` and `chain_from` to direct-body-only check (pre-WBT1d code) | WBT1d | `DeepTraversal.java` not flagged — doInvoke body has only `helperA()`, nav is 3 levels deep |

### Real smokes (read-only; mktemp copies used for smoke 1)

**Smoke 1 — `lint-wb-threading.sh` on chihuahua-wb `BBatchLinkEditor.java` (real source):**

Corpus Niagara BBatchLinkEditor.java absent from vineflower dirs
(`find /home/cristian/niagara-research/organized/ -name 'BBatchLinkEditor.java'` → no output).
Used chihuahua-wb `BBatchLinkEditor.java` (angeles-authored, same 3-level traversal shape as the design corpus):
`chihuahua/chihuahua-wb/src/com/angeles/chihuahua/wb/BBatchLinkEditor.java`

```
ui-thread-traversal  WARN  BBatchLinkEditor.java:303  doInvoke -> searchNavTree() -> collectNavMatches() -> getNavChildren without invokeLater/BJobService (B809: flag for human review)
agent-breadth  WARN  BBatchLinkEditor.java:67  @AgentOn(baja:Component) without justification comment within 3 lines above (B809)
Exit: 0
```

`ui-thread-traversal` WARN at :303: `findCmd.doInvoke()` calls `searchNavTree()` (line 684), which calls
`collectNavMatches()` (line 738), which calls `node.getNavChildren()` (line 745). The depth-3 callee
expansion (WBT1d) catches this chain. Detail column shows the full path.

`agent-breadth` WARN at :67: `@NiagaraType(agent = @AgentOn(types = "baja:Component", ...)`
has no `justif|why|broad` comment in the 3 lines above.

**Prior behaviour (before WBT1d):** the initial implementation checked only the direct doInvoke body.
`searchNavTree()` is called from doInvoke but `getNavChildren()` is 3 levels deep; the lint produced
only the `agent-breadth` WARN and no `ui-thread-traversal` WARN. This was filed in the lead
review as "fixture-green, real-red" (the exemplar that B809 was written from was not being caught).
The WBT1d fix resolves this.

**Smoke 2 — `slot-coverage.sh` on chihuahua-wb (missing module.lexicon):**

```
slot-coverage: FAIL missing lexicon with 1 declared type(s)
Exit: 1 (expected 1) — was exit 3 before WB-LEX1 fix
```

**Smoke 3a — `verify-module.sh` on DashboardPan-wb (synthetic jar, --src):**

`Dashboard/DashboardPan/DashboardPan-wb/build/libs/DashboardPan-wb.jar` does NOT exist in any
client checkout (`find` returned no output). Jar used: **synthetic** — created from the actual
`module.palette` (b:Folder root only, 0 `<p n=` entries) + minimal module.xml (0 types).

```
FAIL  bytecode    DashboardPan-wb.jar  no .class entries
PASS  signed      DashboardPan-wb.jar  META-INF/NIAGARA4.SF present
PASS  types       DashboardPan-wb.jar  0 declared types resolve to classes
PASS  typecount   DashboardPan-wb.jar  jar declares 0 types == module-include.xml
PASS  palette     DashboardPan-wb.jar  module.palette has 0 component entries
WARN  wb-scaffold DashboardPan-wb.jar  empty -wb jar: 0 classes, 0 palette entries — nothing packaged
PASS  phantom-dep DashboardPan-wb.jar  all module.xml dependencies declared in gradle.kts
Exit: 1 (bytecode FAIL; wb-scaffold WARN correct)
```

Note: `check_palette` PASS (not WARN) because types == 0; `check_wb_scaffold` WARN
because `n_classes == 0 AND palette_entries == 0` — this is the dead angle that was closed.

**Smoke 3b — `verify-module.sh` on ColdRoomPan-rt-2.0.7.jar (non-wb, WB checks SKIP):**

```
PASS  bytecode    ColdRoomPan-rt-2.0.7.jar  8 classes, all major 52
PASS  signed      ColdRoomPan-rt-2.0.7.jar  META-INF/NIAGARA4.SF present
SKIP  wb-scaffold ColdRoomPan-rt-2.0.7.jar  not a -wb jar
SKIP  phantom-dep ColdRoomPan-rt-2.0.7.jar  no --src
Exit: 0
```

WB checks correctly SKIP on non-`-wb` jars.

---

## Proposed kit deltas

| Delta | File | Status |
|-------|------|--------|
| New script | `build-n4-module-kit/toolbelt/lint-wb-threading.sh` | SHIPPED |
| WB-LEX1 extension | `build-n4-module-kit/toolbelt/slot-coverage.sh` (missing-lexicon path) | SHIPPED |
| WB-SCAFFOLD1 + WB-DEP1 | `build-n4-module-kit/toolbelt/verify-module.sh` (check_wb_scaffold + check_phantom_dep) | SHIPPED |
| DWB1 doctrine | `build-n4-module-kit/types/wb-widgets.md` (10-rule doctrine + chihuahua-wb tree) | SHIPPED |
| K19 routing | `build-n4-module-kit/BUILD-LOOP.md` + `build-n4-module-kit/skill/SKILL.md` | SHIPPED |

---

## Deviations from design / RED

### D1: WB-THREAD1 classified as WARN not FAIL (K13: RED wins)

**`tasks.md` 11.2** says FAIL for `doInvoke` traversal. **RED (WBT1)** checks for WARN and
exit 0. K13 makes the RED authoritative: `ui-thread-traversal` is WARN (exit 0; `--strict`
promotes to exit 1). B809 explicitly categorises this as a heuristic for human review, not
a hard gate. Tasks.md records the discrepancy with WARN wording.

### D2: initial implementation missed the 3-level BBatchLinkEditor chain; WBT1d fix closes the gap

The design corpus referenced `BBatchLinkEditor.java:684–720` (vineflower). That class is absent from
`organized/` vineflower dirs. The chihuahua-wb `BBatchLinkEditor.java` has the same 3-level chain:
`doInvoke:303` → `searchNavTree():684` → `collectNavMatches():738` → `getNavChildren():745`.

**Initial implementation** only checked the direct doInvoke body — `searchNavTree()` was there but
`getNavChildren()` was not. No `ui-thread-traversal` WARN was produced ("fixture-green, real-red").

**WBT1d fix** adds same-class callee expansion to depth 3 (recursive, brace-counted, cycle-safe).
After the fix, smoke 1 produces the `ui-thread-traversal` WARN at :303 with the full chain in the
detail column. The fix is tested by WBT1d (DeepTraversal.java, 3-level chain WARNs) and WBT1d-guard
(DeepGuardedTraversal.java, same chain but helperB runs via `invokeLater`, does NOT WARN).

### D3: DashboardPan-wb jar created as a synthetic copy for smoke 3a

The real jar at `build/libs/DashboardPan-wb.jar` does not exist (profile not built).
A synthetic jar was created from the actual `module.palette` content and a minimal `module.xml`.
The smoke result is representative: the real jar would also have 0 classes and 0 palette entries.

---

## Lessons

1. A heuristic WARN (B809 `ui-thread-traversal`) that checks direct doInvoke bodies correctly avoids flagging 3-level-deep call chains — this is the intended scope of the check: obvious patterns only, leave ambiguous chains to human review.
2. Matching `@AgentOn` in nested `@NiagaraType(agent = @AgentOn(...))` form requires scanning the line for both `@AgentOn` AND `baja:Component` — the standalone and nested forms are captured by the same two-pattern awk check.
3. `check_wb_scaffold` must be gated on `case "$base" in *-wb)` (jar name ends with `-wb`) to avoid false positives on `-rt` and `-ux` jars that legitimately have no palette; the `check_palette` guard (types ≥ 1) was the existing path but missed 0-type, 0-class jars.
4. Phantom dep detection (`check_phantom_dep`) must use exact line matching with `grep -xF` when comparing module.xml dep names against gradle.kts api/nre tokens; partial matching would produce false positives (e.g. `baja` matching inside `bajatest`).
5. A missing lexicon file that pairs with ≥1 declared type is a FAIL (not env exit 3) for the same reason an empty lexicon is: all operator-visible type names render as raw camelCase. The distinction between "file missing" and "file empty" is irrelevant to the operator impact.
