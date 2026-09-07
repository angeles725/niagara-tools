<!-- review-status: pending -->
# 2026-09-07 · kit · gradle-properties-consistency

**Session**: PANCCADIA multi-module session — operator noticed CompPan's gradle.properties differs from the others.
**Delta count**: 3

## What happened
Comparing the three module group `gradle.properties` (Compresores/Paccadia/Dashboard) revealed two inconsistencies that a WSL `build.sh` build MASKS: (1) CompPan's Java-toolchain block (`org.gradle.java.installations.*`) is COMMENTED OUT — it relies on Gradle auto-detect to find a JDK 8 — while ColdRoomPan and DashboardPan force `auto-detect=false` + Zulu-8; (2) the three point at THREE DIFFERENT `niagara_home` versions despite all deploying to ONE PANCCADIA station (4.14 Honeywell). The inconsistency is invisible to `build.sh` (it overrides `niagara_home` via arg 3 and handles Java 8), so all four jars built major-52; it only bites a Windows `gradlew` build.

## Evidence
- Compresores/gradle.properties: `niagara_home=C:\Honeywell\OptimizerSupervisor-N4.14.0.162` (4.14); `org.gradle.java.installations.*` all COMMENTED [ev: Compresores/gradle.properties]
- Paccadia/gradle.properties: `niagara_home=C:\PowerB\PowerB-4.15.3.28` (4.15.3); `auto-detect=false`, `paths=C:\Program Files\Zulu\zulu-8` [ev: Paccadia/gradle.properties]
- Dashboard/gradle.properties: `niagara_home=C:\Niagara\iC-Niagara-4.13.2.18` (4.13.2); `auto-detect=false`, Zulu-8 [ev: Dashboard/gradle.properties]
- `lint-structure` ALREADY FAILs L10 (absolute host path in tracked gradle.properties) — pre-existing, ignored [ev: lint-structure.sh]
- session builds used `build.sh <group> <MOD> /home/cristian/dpan-niagara-home` → all major 52 (masked the divergence) [ev: build.sh runs]

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | Extend `lint-structure`: flag a module whose `org.gradle.java.installations.paths`/`auto-detect` block is ABSENT or COMMENTED (relies on auto-detect → machine-dependent JDK → wrong-bytecode-major risk on a `gradlew` build). Require it set (or `auto-detect=true` explicitly). | toolbelt/lint-structure.sh (new L-check) | [ev: Compresores/gradle.properties] |
| 2 | New group-consistency check: modules in the same group / destined for one station must share ONE `niagara_home` version; flag divergent `niagara_home` across sibling `gradle.properties`. | toolbelt/lint-structure.sh or report-module.sh | [ev: 4.14/4.15.3/4.13.2 divergence] |
| 3 | Reinforce L10 doctrine: never version absolute host paths in `gradle.properties`; ship a template + a git-ignored local override, so a fresh checkout builds without host-path edits AND without version drift. | types/structure.md §gradle-properties; scaffold-module.sh | [ev: lint-structure L10] |

## Lessons
- `build.sh` MASKS a wrong/inconsistent `gradle.properties`: it overrides `niagara_home` (arg 3) and forces Java 8, so a WSL build passing does NOT prove the committed `gradle.properties` is correct — a Windows `gradlew` build can pick the wrong JDK or the wrong Niagara API.
- A commented-out `org.gradle.java.installations` block = reliance on auto-detect = the "gradle :jar with the default JDK is NOT a build" hazard.
- Modules deploying to ONE station must build against THAT station's Niagara version; three different `niagara_home` versions across a deploy group is a latent defect.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-07-gradle-properties-consistency.md | kit | 2026-09-07 | pending | 3 |`
