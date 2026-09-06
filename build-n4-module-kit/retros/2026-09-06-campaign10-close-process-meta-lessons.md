<!-- review-status: folded -->
# 2026-09-06 · kit · campaign10-close-process-meta-lessons

**Session**: Campaign 10 CLOSE — v0.21.0 lint precision (S21-S25) + client hygiene (S26)
**Delta count**: 7

## What happened

Campaign 10 produced five lint-precision PRs (S21-S25) and one client hygiene PR (S26). The close process surfaced seven process-level lessons about gate contracts, per-row exemption scope, matrix-root coverage, worktree discipline, annotation harvesting, and method-boundary parsers.

[ev: retro campaign10-close-process-meta-lessons]

## Evidence

- S25 premise error: lint-write-path already exits 1 for uncovered FAIL — `--strict` was not a flag to add but a new STALE class to introduce. `[ev: retro campaign10-write-path-stale]`
- S25 per-row exemption: `[concept]` on one hoaMode row does NOT exempt the two other hoaMode rows (STALE is per-row, never per-name). `[ev: retro campaign10-write-path-stale D5b]`
- Matrix-root scope: passing CompPan-rt as root yields the same 5 STALE rows as ColdRoomPan-rt (root-invariant); a per-module root gives a wrong count. `[ev: retro campaign10-write-path-stale D5a]`
- Design executor reads origin tip: the design cites kit main anchors; a peer's worktree at an older tip produces stale line numbers that break the fold. `[ev: retro campaign10-close-process-meta-lessons]`
- Single-line @Niagara regex: `@NiagaraProperty.*name=` matches 56 names at ff1b659; the `name = "X"` field-line pattern matches 177 — the single-line form under-counts and causes false STALE. `[ev: retro campaign10-write-path-stale D5c]`
- S21 companion-flag: a forward brace-walk from a candidate fires on `@NiagaraProperty(` (annotated method name mismatch); the shared section-D method-boundary parser (brace_depth>=2) and a class-FIELD depth-1 guard fix it. `[ev: retro campaign10-lint-timers-scope]`
- Pin-attribution rule: every OBSERVED mutation named in a lead gate names the fixture it flips and QA confirms the flip; RED-pre-fix/GREEN-post-fix is not enough for a guard pin; a fixture in a note must match the shape of its proof. `[ev: retro campaign10-close-process-meta-lessons]`

## Proposed kit deltas (propose-never-apply)

| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | Verify a gate's real exit contract before proposing a new flag; if uncovered FAIL already exits 1, a flag must introduce a new class (STALE), not weaken the existing exit | METHODOLOGY.md — NEW K24 after K22 | `[ev: retro campaign10-close-process-meta-lessons]` |
| 2 | Name-level vs per-row exemption: a `[concept]` mark exempts the marked row only, never all rows sharing the same slot name | METHODOLOGY.md — NEW K24 | `[ev: retro campaign10-close-process-meta-lessons]` |
| 3 | Matrix-root scope: the covered set must be harvested matrix-root-wide (all modules), making the count invariant across module roots | METHODOLOGY.md — NEW K24 | `[ev: retro campaign10-close-process-meta-lessons]` |
| 4 | A design executor reads the origin tip, not a peer worktree; cite anchors from the tree the design pins (kit main) | METHODOLOGY.md — NEW K24 | `[ev: retro campaign10-close-process-meta-lessons]` |
| 5 | Multi-line annotation harvest: match the `name = "X"` field line, not a single-line `@Niagara…name=` pattern — the single-line approach undercounts (56 vs 177) | METHODOLOGY.md — NEW K24 | `[ev: retro campaign10-close-process-meta-lessons]` |
| 6 | Any lint check pairing a flag with a Clock.schedule call must use the shared section-D method-boundary parser + class-FIELD depth-1 guard | METHODOLOGY.md — NEW K24 | `[ev: retro campaign10-lint-timers-scope]` |
| 7 | Pin-attribution rule: every observed mutation named in a lead gate must name the exact fixture it flips; QA confirms the flip; RED-pre-fix / GREEN-post-fix alone is not a complete guard pin | METHODOLOGY.md — NEW K24 | `[ev: retro campaign10-close-process-meta-lessons]` |

## Lessons

- Verify a gate's real exit contract before proposing a flag — S25's premise was wrong: lint-write-path was ALREADY a hard exit-1 gate for uncovered FAIL, so `--strict` needed to introduce a new STALE class, not weaken the existing gate. Read the exit behavior first. `[ev: retro campaign10-close-process-meta-lessons]`
- A name-level exemption is a cross-row implicit exemption; STALE is PER-ROW — each row carries its own `[concept]` mark. One marked hoaMode row never silently exempts two other hoaMode rows. `[ev: retro campaign10-close-process-meta-lessons]`
- A coverage lint's covered set is harvested matrix-root-wide (all modules), else the count depends on which module root was passed; a per-module root produces false STALE on documented cross-module slots. `[ev: retro campaign10-close-process-meta-lessons]`
- A design executor must read the artifact at the origin tip the design pins, not a peer's scratch worktree; stale line numbers break fold discipline. `[ev: retro campaign10-close-process-meta-lessons]`
- The single-line `@Niagara…name=` regex under-counts — annotations are multi-line; match the `name = "X"` field line (177 names vs 56) or the covered set is wrong and STALE over-produces. `[ev: retro campaign10-close-process-meta-lessons]`
- A heuristic pairing a companion flag with a Clock.schedule call needs the shared section-D method-boundary parser (brace_depth>=2 guard) and a class-FIELD depth-1 check; a forward brace-walk fires false on annotation lines. `[ev: retro campaign10-lint-timers-scope]`
- Every OBSERVED mutation named in a lead gate names the exact fixture it flips and QA confirms the flip; RED-pre-fix/GREEN-post-fix alone is not enough for a guard pin; a fixture in a note must match the shape of its proof. `[ev: retro campaign10-close-process-meta-lessons]`

---
**Status**: FOLDED — METHODOLOGY.md K24 after K22 (:88). `[ev: retro campaign10-close-process-meta-lessons]`
