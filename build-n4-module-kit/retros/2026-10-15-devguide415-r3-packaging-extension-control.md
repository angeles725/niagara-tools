<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 4 -->
# 2026-10-15 · kit · devguide415-r3-packaging-extension-control

**Session**: research-sdd focus `dev-capabilities-kit-gaps` round 3 closed the packaging, jsonToolkit and kitControl child gaps from source; this retro carries corpus B1345 (`noRunningApp`), B1350 (jsonToolkit contract) and B1351 (LeadLag/Interstart). It REFINES `2026-10-15-devguide415-r2-packaging-platform-signing.md` (**pps**) Δ1, `2026-10-15-devguide415-r2-shipped-libraries.md` (**shl**) Δ9 and `2026-10-15-devguide415-r2-control-alarms-schedules.md` (**cas**) Δ5.
**Delta count**: 4

## What happened
(1) `noRunningApp`/`noStation`=true is enforced: both the headless installer and Workbench stop every running app (stations and Sedona) before commit, and Workbench aborts the install if the user declines; a dist that omits the attribute installs over running stations (code default false — the devguide's "defaults to true" is wrong). (2) The jsonToolkit property contract is one method, `T getJsonValue()`; the type-override hook is `overrideValueEncoding(BValue)` because `doOverride` is `final` in the 4.15 jar although the guide tells developers to override it. (3) `BInterstartDelayControl` has no own min on/off — it inherits `minActiveTime`/`minInactiveTime` (default 0) from `BBooleanWritable`; `BLeadLagCycles` and `BLeadLagRuntime` are the same algorithm whose ten feedback inputs must be LINKED — with none linked, output A always wins.

## Evidence
- dist `[ev: code organized/platform/platform-rt/vineflower/com/tridium/install/installable/DistributionManifest.java:87]` `[ev: code organized/platform/platform-rt/vineflower/com/tridium/platform/daemon/PlatformInstallManager.java:353-356]` `[ev: code organized/platDaemon/platDaemon-wb/vineflower/com/tridium/platDaemon/ui/softwaremanager/CommitCommand.java:74-86]` vs `[ev: niagara-help 4.15.6.20 devguide-clean/distribution.txt:117-119]` `[ev: corpus B1345]`
- jsonToolkit `[ev: code organized/jsonToolkit/jsonToolkit-rt/vineflower/com/tridiumx/jsonToolkit/outbound/schema/BIJsonProperty.java:15-30]` `[ev: javap /mnt/c/Niagara/Niagara-4.15.6.20/modules/jsonToolkit-rt.jar!…/program/BTypeOverride (doOverride final)]` vs `[ev: niagara-help 4.15.6.20 guides-clean/JsonToolkit/TypeOverrideExample-94E2114D.txt:52]` `[ev: corpus B1350]`
- kitControl `[ev: code niagara-help/versions/4.15.6.20/source/kitControl-rt/com/tridium/kitControl/BInterstartDelayControl.java:63-160]` `[ev: code niagara-help/versions/4.15.6.20/source/kitControl-rt/com/tridium/kitControl/hvac/BLeadLagCycles.java:1116-1124,1187-1200]` `[ev: corpus B1351]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Refine pps Δ1: `noRunningApp`/`noStation`=true is enforced (installer stops ALL apps incl. Sedona; Workbench aborts on decline, or forces a reboot when station restart is not allowed); omitting the attribute = false; use `noRunningStation` to stop stations only. Lint: yes — our dists that replace station-loaded jars must set one of the attributes explicitly. | `types/distribution.md:37` (attribute table) | `[ev: corpus B1345]` |
| Δ2 | Extending jsonToolkit (concretises shl Δ9): property = extend `BJsonSchemaProperty<T>` and implement only `T getJsonValue()`, never directly under the schema root; type override = extend `BTypeOverride` and override `overrideValueEncoding(BValue)` — NOT `doOverride` (final, the guide is wrong); return the same instance for "no change"; depend on `jsonToolkit-rt` (vendor tridiumx). Lint: yes — `extends BJsonSchemaProperty` requires the dependency. | `types/` new § "Extending jsonToolkit" | `[ev: corpus B1350]` |
| Δ3 | Refine cas Δ5: LeadLag needs one feedback link per used output (cycle count or elapsed-active time from a totalizer on the real point) — unlinked means output A forever; Cycles vs Runtime differ only in that input type. Lint: yes — bog check that every `cycleCount*`/`runtime*` below `numberOutputs` has an inbound link. | `types/logic.md` "REUSE before you hand-roll" | `[ev: corpus B1351 §1351.2]` |
| Δ4 | InterstartDelayControl min on/off = the inherited `minActiveTime`/`minInactiveTime` (default 0, off); set them explicitly for compressors. Lint: partial — bog check for non-zero values on staged compressor outputs. | `types/logic.md` timers | `[ev: corpus B1351 §1351.1]` |

## Lessons
- When a guide and the shipped jar disagree on an API (`doOverride`), `javap` the jar: final methods settle it.
- A reuse block with LINKED feedback inputs fails silently when unlinked; the kit must say which links are mandatory.
- A doc default and a parser default can differ; the parser decides what an attribute-less manifest does.

---
**Status**: PENDING — INDEX row appended: `| 2026-10-15-devguide415-r3-packaging-extension-control.md | kit | 2026-10-15 | pending | 4 |`
