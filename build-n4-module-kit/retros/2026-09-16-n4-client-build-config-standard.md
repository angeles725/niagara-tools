<!-- review-status: folded -->
# 2026-09-16 · kit · n4-client-build-config-standard

**Session**: `/build-n4-module` UmbrellaDashboard bring-up surfaced the client's canonical build-config files (gradle.properties + settings.gradle.kts).
**Delta count**: 3

## What happened
A fresh client module scaffold (`Cliente/Juarez/Umbrella`) ships `gradle.properties` with the node/JDK-toolchain
block COMMENTED (auto-detect), while the established client modules PIN explicit values. Comparing all four
module groups shows a consistent client standard the kit does not yet document, and a build.sh interaction
(the WSL gate overrides the JDK path, so the gradle.properties JDK path only governs the Windows Workbench
build). Aligning Umbrella to the standard and rebuilding proved the change is WSL-safe.

## Evidence
- Cross-module `gradle.properties` / `settings.gradle.kts` compare (4 groups) `[ev: corpus B1016]`:
  - niagara_home: Umbrella/Compresores `Honeywell-N4.14.0.162`; Dashboard `iC-Niagara-4.13.2.18`; Paccadia `PowerB-4.15.3.28`.
  - JDK: Dashboard/Paccadia/Umbrella pin `org.gradle.java.installations.paths=C:\Program Files\Zulu\zulu-8` + `nodeHome=C:\Program Files\nodejs` + `auto-detect=false`; Compresores (logic-only, no ux) leaves them commented.
  - plugins: all use `gradlePluginVersion 7.6.17` / `settingsPluginVersion 7.6.3` (4.13→4.15); Paccadia parameterizes `getOrElse("7.6.17")`.
- `build.sh:74` `GARGS=(-Pniagara_home=… -Porg.gradle.java.installations.paths="$J8")` — the WSL gate OVERRIDES the JDK path `[ev: build.sh:74]`.
- After aligning Umbrella (Zulu-8 + nodeHome uncommented): `build.sh --profiles rt,ux --target-version 4.14` → `verify-module: 17 passed, 0 failed → ALL PASS` (WSL-safe) `[ev: corpus B1016 §1016.3]`.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | document the CLIENT build-config STANDARD for a new module: `gradle.properties` pins `niagara_home`+`niagara_user_home` (same SDK), and for a ux/JS module also `nodeHome=…nodejs` + `org.gradle.java.installations.paths=…Zulu\zulu-8` + `auto-detect/-download=false`; `settings.gradle.kts` `gradlePluginVersion`/`settingsPluginVersion` match the SDK family | `types/structure.md` § new-module scaffold + `METHODOLOGY.md` common checklist | `[ev: corpus B1016]` |
| Δ2 | note the build.sh↔gradle.properties JDK split: the WSL gate overrides `installations.paths` (build.sh:74), so the gradle.properties JDK path is for the WINDOWS Workbench build only — a commented (auto-detect) block still builds in WSL but is not reproducible on Windows | `build-verify.md` § JDK toolchain | `[ev: build.sh:74]` |
| Δ3 | add a new-module BRING-UP checklist: (1) `chmod +x gradlew`; (2) align gradle.properties (niagara_home/user_home/nodeHome/JDK path) + settings.gradle.kts (plugin versions) to a SIBLING of the same SDK; (3) then first build | `BUILD-LOOP.md` §0 / scaffold-module.sh output | `[ev: corpus B1016]` |

## Lessons
- A fresh client scaffold's `gradle.properties` JDK/node block is COMMENTED (auto-detect); the client standard PINS an explicit Zulu-8 JDK path + nodeHome — align it to a sibling of the same SDK before shipping, for Windows-build reproducibility.
- The WSL gate (`build.sh`) is JDK-path-agnostic (it injects `$JAVA8` via `-P`), so a green WSL gate does NOT prove the Windows Workbench build is configured — that is what the gradle.properties JDK path governs.
- The three version pins move on different axes: SDK (`niagara_home`), gradle plugins (`settings.gradle.kts`), module vendorVersion (`defaultModuleVersion` in the group build) — keep them explicit and consistent per module group.
- Compresores (logic-only, no `-ux`) can leave node/JDK commented; a dashboard/ux module should pin them like DashboardPan/Paccadia.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-16-n4-client-build-config-standard.md | kit | 2026-09-16 | pending | 3 |`
