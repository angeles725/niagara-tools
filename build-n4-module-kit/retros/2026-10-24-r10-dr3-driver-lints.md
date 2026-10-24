<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 4 -->
# 2026-10-24 · kit · r10-dr3-driver-lints

**Session**: niagara-research quality round 10, driver-authoring item DR3, branch `r10/dr3-driver-lints`
**Delta count**: 4

## What happened
Four owed driver lints landed natively on the Go lint engine, test-first (golden cases from each deferred spec,
then a bats suite on synthetic fixtures under `tests/fixtures/driver-lints/`, then observed mutations): G65
`private-api-import`, G66 `ndriver-static-collection` and G67 `ndriver-tagless-correlation` (specs in
`ledger.d/2026-10-23-r8-u5-c59-c62.md`) and G36 `bacnet-status-flags` (spec in `ledger.d/2026-10-23-r8-p1-drivers.md`).
`go/internal/javasrc` gained `Classes`, `Fields` and `Descendants` (class headers, class-body fields, superclass
chains through module-local classes). G65/G66/G67 run in `report-module.sh` step 13; G36 reads a station bog, which a
module report does not name. Real trees: the course trafficLightDriver tree meets every spec acceptance number (G65 0
rows, 60 with `--no-default-allow`; G66 1 row, the static `subscribedPoints` set; G67 1 row, no `getTag()` and the
`return 1` hint commented out); the course certification driver gives one more true G67 row; client module trees give
2 true G65 rows (`com.tridium.util.ContextThread`) and no driver class; the decompiled corpus has no part `src/`
tree (exit 3, not a module tree); local station bogs with BACnet proxies give one G36 row per device, all true (no
point carries the facet).

Spec decisions taken here (the specs were silent or ambiguous):
- Rows print the kit's `lint-<name>` check column (the specs wrote the bare name) and ASCII ` -- ` for the em dash.
- G65: an exact allow-list entry also allows its members (`DrUtil.getParent` by static import or a call); a glob
  matches the whole name, `*` one segment and `**` any. `--no-default-allow` is the wrapper's spelling of the engine
  option `--default-allow off` (Mode Valued takes value options only).
- G66/G67: a module root whose `*.java` all lie outside a `-rt` part's `src/` is the ERROR row (exit 3) like a root
  with no `*.java`, so `report-module.sh` relays it as a SKIP row; `java.util.concurrent` collections count as
  `java.util` implementations; `Set.of`/`List.of` are not exempt (Java 9; a module builds as Java 8).
- G67: comm-config and message chains are followed through module-local classes; `return (1);` counts as 1.
- G36: roots are station directories and every `*.bog` under one is read; the device of a row is the parent of the
  point's `…PointDeviceExt`, because real stations use vendor device types of other modules (a first run grouped
  every proxy as "no BACnet device"); "the ext's own override" is read as a `useStatusFlags=true` slot on the ext,
  since in bacnet-rt `useStatusFlags` is a cached field, not a slot.

## Deltas
| Δ | Change | Where | Evidence |
|---|---|---|---|
| Δ1 | New `toolbelt/lint-private-api-import.sh` (G65, rule `go/internal/lint/rules/privateapiimport`): WARN each `com.tridium.` import, static import and qualified use in part sources; default allow-list `com.tridium.ndriver.**` + `DrUtil`; `--allow`, `--no-default-allow`; PAI1/PAI2 pins. | `types/module-wiring.md` § 4.5 | `[ev: retro n4-course-handson-ndriver Δ8]` `[ev: retro n4-course-build-deps-permissions-signing Δ3]` |
| Δ2 | New `toolbelt/lint-ndriver-static-collection.sh` (G66, rule `rules/ndriverstaticcollection`): WARN a static collection field of a driver network/device/point-device-ext class, module-local subclasses included; NSC1/NSC2 pins. | `types/driver-authoring.md` § 1.7 | `[ev: retro n4-course-handson-ndriver Δ7]` |
| Δ3 | New `toolbelt/lint-ndriver-tagless-correlation.sh` (G67, rule `rules/ndrivertaglesscorrelation`): one WARN per module with `NMessage` subclasses, no `getTag()` and no comm config returning 1; NTC1/NTC2 pins. | `types/driver-authoring.md` § 1.7 | `[ev: retro n4-course-drivers-bql-ords-files Δ3]` |
| Δ4 | New `toolbelt/lint-bacnet-status-flags.sh` (G36, rule `rules/bacnetstatusflags`): one WARN per device counting the Present_Value BACnet proxies whose point facets lack `statusFlags=true`; BSF1-BSF3 pins. | `types/driver-authoring.md` § 2 | `[ev: retro devguide415-r2-drivers-protocols Δ4]` |

**Status**: FOLDED 2026-10-24 — Δ1 in `types/module-wiring.md` § 4.5, Δ2-Δ4 in `types/driver-authoring.md` § 1.7 and § 2.
