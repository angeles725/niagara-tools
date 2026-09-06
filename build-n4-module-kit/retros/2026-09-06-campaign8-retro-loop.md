<!-- review-status: pending -->
# 2026-09-06 · kit · campaign8-retro-loop

**Session**: Campaign 8 PR16 — feat/c8-retro-loop (new-retro.sh + kit-ticket.sh)
**Delta count**: 5

## What happened
The kit had no automated retro sink: at-STOP retros fire only when a session stops
or focus-closes, so in a continuous lead-delegated chain (one unit → next task) no
STOP is ever reached and retros get skipped — observed live at ~8:1 (blocks landed :
retros written). PR16 closes this by making the retro a per-run PRECONDITION with
`new-retro.sh` as the atomic stub writer and `sweep-build-state.sh --age` as the
visible debt counter. The RED test suite (`tests/retro-loop.bats`, RL1–RL7) was
on the branch (RL1–RL6 at commit 452a840; RL7 added after lead review 1; RL8/RL9 added after lead review 2); GREEN was reached in this session.

## Evidence
- RL1–RL6 all GREEN (6/6) after implementing new-retro.sh + kit-ticket.sh. `[ev: retro campaign8-retro-loop]`
- Mutation (a): omit INDEX duplicate guard → when retro FILE is absent but INDEX row is already present, second call produced 2 rows in INDEX.md (OBSERVED via RL9 — guard neutralized on mktemp copy). Note: RL2 is protected by the PRIMARY file-exists guard (exit 3 fires before any INDEX append on second run); the INDEX grep guard is defense-in-depth and is pinned separately by RL9 (retro file ABSENT + INDEX row PRESENT fixture). `[ev: retro campaign8-retro-loop §RL9]`
- Mutation (b): write stub before staging → stub lands without INDEX/BUILD-STATE changes (OBSERVED partial write). `[ev: retro campaign8-retro-loop §D13-proof]`
- Mutation (c): drop review-status marker → RL1 flips (first line is title not marker). `[ev: retro campaign8-retro-loop §RL1]`
- Mutation (d): kit-ticket calls gh unconditionally → exit 127 when gh absent (RL4 flips). `[ev: retro campaign8-retro-loop §RL4]`
- Real smoke on copy: `new-retro.sh kit smoke-run-2026-09-06` → retro PASS + index PASS + envelope PASS; second call → exit 3 (already exists), no duplicate; `sweep-build-state.sh` exit 0; `kit-ticket.sh` with gh masked → SKIP + fallback file written. `[ev: retro campaign8-retro-loop §smoke]`
- Lead review surfaced Δ5 as a live defect (not future): `new-retro.sh` sed was flipping ALL `retro_pending: false` → `true`, falsely marking every module section. RL7 added (RED: ColdRoomPan and CompPan both showed `true`; GREEN after awk section-scoped fix). Post-fix smoke: kit=true, ColdRoomPan=false, CompPan=false, DashboardPan=false, chihuahua=false. `[ev: retro campaign8-retro-loop §RL7]`
- investigador1 review found slug floor bug (RL8): case pattern enforced only 5 mandatory chars; sweep-fold-audit.sh drops tokens < 6 chars (unfoldable). Fixed: six bracket classes before `*`; RL8 added (5-char slug → exit 3, 6-char → accepted). `[ev: retro campaign8-retro-loop §RL8]`
- QA proved INDEX guard unpinned; RL9 added: retro FILE absent + INDEX row present → SKIP + stub written + envelope flipped + 1 row (no duplicate). `[ev: retro campaign8-retro-loop §RL9]`
- bats: 291 tests, 0 failures (284 pre-existing + 9 new RL tests). shellcheck clean both scripts. All guards exit 0.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | `new-retro.sh` script — atomic triple write: stub + INDEX row + BUILD-STATE flag | `toolbelt/new-retro.sh` § (new file) | `[ev: retro campaign8-retro-loop]` |
| Δ2 | `kit-ticket.sh` script — gh absent → SKIP + fallback file; with gh → issue create | `toolbelt/kit-ticket.sh` § (new file) | `[ev: retro campaign8-retro-loop]` |
| Δ3 | BUILD-LOOP §7 retro gate — "every run ends by writing its retro" with tool reference | `BUILD-LOOP.md` § `7. Retro + close` | `[ev: retro campaign8-retro-loop]` |
| Δ4 | skill/SKILL.md step 7 close-of-run — names new-retro.sh + kit-ticket.sh tools | `skill/SKILL.md` § `Execution Steps` | `[ev: retro campaign8-retro-loop]` |
| Δ5 | BUILD-STATE retro_pending flip scoped to the named section via awk (same delimiters as sweep-build-state.sh); RL7 pin added (kit, ColdRoomPan, CompPan fixture; symmetric case) | `toolbelt/new-retro.sh` § Stage 3; `tests/retro-loop.bats` RL7 | `[ev: retro campaign8-retro-loop]` |

## Lessons
- Atomic triple write via temp staging (stage all three, then mv/cp) prevents partial state; the mutation (b) probe confirms the risk without it.
- Restricted-PATH test harness (no `tr`, no `mktemp`) forces the script to use only the tools the harness exposes — test RL4 is the enforcement; `awk tolower()` replaces `tr`, direct printf replaces temp-file write.
- The INDEX idempotent guard (grep before append) and the file-exists guard (exit 3 early) are redundant by design: two independent fences catch the same partial-state risk via different paths.
- `set -u` + `trap 'rm -rf "$_TMP"' EXIT` are the minimal safety net for any script that stages files.
- The test seam (`env -C <tmpkit> KIT=<tmpkit>`) means the script must resolve its kit root from `$KIT` env var BEFORE any positional-arg or BASH_SOURCE fallback — otherwise the test injects a temp kit but the script writes to the real one.
- Section-scoped BUILD-STATE edit requires the same delimiters as sweep-build-state.sh (`<!-- build-state.v1 -->` / `<!-- /build-state.v1 -->`); a global sed that ignores section boundaries is a live operational defect, not a cosmetic deviation — it falsely signals pending retros for modules that owe none.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-06-campaign8-retro-loop.md | kit | 2026-09-06 | pending | 5 |`
