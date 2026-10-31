<!-- review-status: pending -->
<!-- module: ClientEDashboard -->
<!-- deltas: 4 -->
# 2026-10-12 · ClientEDashboard · client-e-dashboard-phase1-close

**Session**: ClientEDashboard phase 1 close: T5b (VAV facade aligned to the real PIC-SD v3.21.0 BACnet point list), T6 (build, lints, verify, preview), T7 (wiring map + this ledger row).
**Delta count**: 4

## What happened
The operator supplied the real VAV point export (the client VAV point export (CSV), 21 BACnet points on BacnetNetwork/COM4) and chose to realign the
`BVavPanel` facade to it (decision D10): three slots dropped, `occupied` changed from BStatusBoolean to BStatusEnum, three enum slots added.
A delegated writer did T5b test-first and ran T6. Four kit-level frictions surfaced: the drift gate hard-failed on a 2-part installed version,
`new-retro.sh` reported a PASS envelope edit for a module that had no envelope, the native review could not take a 480-line slice whose
generated fixtures are 128 KB, and BACnet multi-state values arrive at a BStatusEnum facade slot with `$XX`-escaped tags.

## Evidence
- build.sh drift gate exit 52 on installed baseline vendorVersion "1.0" (not X.Y.Z); worked around with 1.1.0 + `--no-drift-check` `[ev: 25ceab8]`
- `new-retro.sh module client-e-dashboard-phase1-close` printed `envelope PASS BUILD-STATE.md retro_pending: true` while BUILD-STATE.md had no ClientEDashboard envelope (git diff empty) `[ev: new-retro.sh 2026-10-12 run]`
- gentle-ai review START `lens_context_budget_exceeded` for a8c29fa (16 files, two 64 KB fixture JSONs); split into 4 slices, fixtures proven by byte-identical regeneration with `gen_fixtures.py` `[ev: bf9162e]`
- PIC-SD MSV range tags such as `Default$20Mode$20$28Occupied$29` reach the SPA through `getEnum().getTag()`; the SPA had to unescape `$XX` `[ev: d60359b]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Drift gate: a non-X.Y.Z installed baseline prints a WARN with the fix (bump to X.Y.Z) instead of exit 52 | `toolbelt/build.sh` § drift check | `[ev: 25ceab8]` |
| Δ2 | new-retro.sh: when the module has no build-state.v1 envelope, print `envelope WARN … no envelope` (not PASS) and exit non-zero or tell the caller to add one | `toolbelt/new-retro.sh` § envelope step | `[ev: new-retro.sh 2026-10-12 run]` |
| Δ3 | Commit generated fixtures in their own work-unit commit and record a regeneration-determinism check (rerun generator, `git status` clean) as their proof; native review budgets cannot take them | `BUILD-LOOP.md` § commits / verify | `[ev: bf9162e]` |
| Δ4 | Dashboard reader/SPA rule: an enum tag read from a linked BStatusEnum is SlotPath-escaped; unescape it (or send the display string) before label lookup | `types/dashboard.md` § reader serialization | `[ev: d60359b]` |

## Lessons
- A facade slot type change (Boolean -> Enum) is a relink on every station already wired; record it in the module's open issues before deploy.
- Build the facade from the real controller point export, not the prototype's point list; the prototype had slots the PIC-SD does not have.
- Keep generated data out of reviewable commits; prove it by deterministic regeneration instead.

---
**Status**: PENDING — INDEX row appended: `| 2026-10-12-client-e-dashboard-phase1-close.md | module | 2026-10-12 | pending | 4 |`
