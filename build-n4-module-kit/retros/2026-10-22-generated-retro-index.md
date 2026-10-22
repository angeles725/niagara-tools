<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-22 · kit · generated-retro-index

**Session**: niagara-research quality round 8, task T8i slice A, branch `r8/t8i-a-retro-index` from origin/main 784ec45
**Delta count**: 2

## What happened
Four sessions worked on the kit in parallel, and every kit PR conflicted on `retros/INDEX.md`: each one appended a
row at the end or flipped a row's status. "Keep both sides" resolutions went wrong at least three times and flipped
rows. The index only restated facts each retro already held (file, status) or could hold (module, delta count), so
this unit makes it a generated file. Each retro now carries its row as header comments and
`toolbelt/gen-retro-index.sh` rebuilds the index; `--check` runs in CI and inside `sweep-build-state.sh`, so the
pre-push gate's content half fails a stale index. The gate no longer needs an `INDEX.md` diff for a new retro.

## Evidence
- 243 retros migrated by a one-off script that copied module, deltas and the trailing row comment from the
  committed INDEX row into `<!-- module: -->`, `<!-- deltas: -->`, `<!-- index-comment: -->` headers; 24 legacy
  retros without a line-1 marker also got `<!-- review-status: -->` from their row `[ev: tests/gen-retro-index.bats GRI-real]`
- The regenerated rows equal the committed rows as a set (sorted diff empty); only the order changed: the old file
  was insertion-ordered within a date, which is not derivable and is exactly what made every PR append at the same
  place `[ev: tests/gen-retro-index.bats GRI-rows]`
- The old file had a blank line inside the table (after the 2026-09-05 campaign8 wb-audit row), so every later row
  rendered outside the table; generation removes the defect class
- Gate: H11 (BUILD-STATE + retro, no INDEX diff, fresh index) was RED before the hook change and is GREEN after;
  H12 (stale index) fails through sweep `[ev: tests/build-retro-sync.bats H11 H12]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | **`retros/INDEX.md` is generated; never hand-edit it.** A retro's row lives in its header comments; edit the header and run `toolbelt/gen-retro-index.sh`; on a conflict, regenerate. Applied in this PR. | `BUILD-LOOP.md` §7; `CONTRIBUTING.md` §6.1 | `[ev: tests/gen-retro-index.bats]` |
| 2 | **`new-retro.sh` writes the header block and regenerates the index instead of appending a row.** Today it still appends a row and its stub has no `module`/`deltas` headers, so the generator rejects the new retro until the author adds them. Not applied: `toolbelt/new-retro.sh` and `tests/retro-loop.bats` RL2/RL5/RL6/RL9 were outside this unit's edit surface. | `toolbelt/new-retro.sh`; `tests/retro-loop.bats` | `[ev: retro generated-retro-index]` |

## Lessons
- A hand-edited registry that only restates facts its entries already hold is a merge-conflict hotspot; move the
  facts into the entries and generate the registry.
- An insertion-ordered list conflicts on every append; a sort key derived from the entry (date, then name) spreads
  new rows across the file.
- A freshness check placed inside the gate's existing content validator covers the hook without new git logic.

---
**Status**: PENDING — Δ1 applied in this PR, Δ2 owed (new-retro.sh). Row generated from this header.
