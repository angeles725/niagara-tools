<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-23 · kit · r8-t9i-packaging

**Session**: niagara-research quality round 8, task T9i unit 1 (register WU-P0-7: C23, C24, C25, C26, C29), branch `docs/r8-t9i-packaging` from origin/main 6416fb0
**Delta count**: 2

## What happened
Folding the packaging items found two kinds of drift the register did not name. First, two guide line cites
carried into the register and blocks point past the end of their 4.15.6.20 files: `DeployingTemplates-054F175F.txt:33`
(the file has 21 lines) and `InstallingSoftwareByDraggingFromThe.txt:36-46` (32 lines); the claims hold at
`:7-11` and `:16-26`. Second, the wrong "a JACE loads `-se`" premise lived in five places, not the one line C24
names: the load table, the headless AWT trap, the `-wb` dependency trap, the JACE resource note and
`METHODOLOGY.md`'s profile bullet.

## Evidence
- `niagara-help/versions/4.15.6.20/guides-clean/Templates/DeployingTemplates-054F175F.txt` (21 lines), `…/Provisioning/InstallingSoftwareByDraggingFromThe.txt` (32 lines) `[ev: corpus B1325 §1325.1 rows 5, 8]`
- `Platform/aPlat4PlatformAdminConfigureRuntimeProfiles.txt:24,30` (compact3 JACE, `se` "Windows-based hosts only") `[ev: corpus B1341 §1341.2 C1]`
- RED 6/6 then GREEN 6/6: `tests/kit-packaging-pins.bats` `[ev: tests/kit-packaging-pins.bats]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Re-open every guide line cite a fold copies from a block and cite the line read now; a cite past the end of the file is a stale version | `types/distribution.md` § 12 Fleet install | `[ev: corpus B1325 §1325.1]` |
| Δ2 | A corrected premise is swept across the whole kit (types/ and top-level docs) and pinned by a bats ban on each wrong phrasing, not fixed only at the register's line | `types/structure.md` § The `-se` (Java SE) runtime profile | `[ev: corpus B1341 §1341.2 C1]` |

## Lessons
- A register row names one line; the wrong idea it fixes usually has siblings.
- Block line numbers belong to the version the block read; re-read before citing.

---
**Status**: FOLDED (T9i unit 1, same unit) — Δ1 `types/distribution.md` § 12, Δ2 `types/structure.md` § The `-se` runtime profile.
