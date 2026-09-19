<!-- review-status: folded -->
# 2026-09-16 · UmbrellaDashboard · umbrelladashboard-module-creation

**Session**: `/build-n4-module` — new dashboard module for the Juárez/Umbrella "Productos de Agua" 3D HMI, replicating DashboardPan.
**Delta count**: 4

## What happened
Built a NEW dashboard-type module `UmbrellaDashboard` (vendor Angeles, symbol UMD) from the empty scaffold at
`Cliente/Juarez/Umbrella/`, replicating the DashboardPan exemplar (B791/B796): a `BUmbrellaService`
(BAbstractService) + `BPackageUnit` facade (rt), and a `BWebServlet` + pure `UmbrellaDispatch` router +
`UmbrellaReader` + RBAC (ux), then packaged the 2.73MB three.js "Productos de Agua" SPA into `src/rc/` and
wired it live to `/umbrelladashboard/api/equipment`. Both profiles build green (major 52, signed,
`baja 4.14 <= target 4.14`), verify-module 17/0 ALL PASS, 14 pure-JUnit router tests pass. Three tool WARN/FAIL
turned out to be non-defects (documented below), which is the source of the proposed kit deltas.

## Evidence
- `build.sh --profiles rt,ux --target-version 4.14` → `BUILD SUCCESSFUL`; `verify-module: 17 passed, 0 failed, 4 skipped, 3 warned -> ALL PASS` `[ev: corpus B1015 §1015.4]`
- `run-pure-test.sh … UmbrellaDispatchTest` → `OK (14 tests)` `[ev: corpus B1015 §1015.2]`
- `report-module.sh` false FAIL `plano Rc: IMG_W/IMG_H not found` on a 3D SPA `[ev: corpus B1015 §1015.5]`
- `verify-module.sh` false WARN `phantom-dep UmbrellaDashboard-rt` — declared as `api(project(":UmbrellaDashboard-rt"))` `[ev: corpus B1015 §1015.4]`
- `build.sh` exit 10 on a non-executable `gradlew` (fixed with `chmod +x`) `[ev: corpus B1015]`
- cross-module `gradle.properties` compare: L10 absolute-host-path is client-universal (Paccadia/Dashboard/Compresores/Umbrella) `[ev: corpus B1015 §1015.5]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | plano check must SKIP (not FAIL) when the SPA has no plano overlay (no IMG_W/IMG_H) — a 3D/self-contained SPA is not a plano dashboard; make plano opt-in via `--plano` only, not auto-run in report-module | `toolbelt/report-module.sh` + `toolbelt/verify-module.sh` (plano sub-check) | `[ev: corpus B1015 §1015.5]` |
| Δ2 | phantom-dep must recognize `api(project(":<MOD>-rt"))` project deps as declared, not only `api(":x")`/`nre(":x")` string deps — else every rt→ux client module gets a false phantom-dep WARN | `toolbelt/verify-module.sh` (phantom-dep) | `[ev: corpus B1015 §1015.4]` |
| Δ3 | add a "self-contained 3D SPA → /api/equipment live-wiring" recipe: override `reading()`/`currentState()`, poll the absolute `/prefix/api/equipment` with `X-Requested-With`, ordinal `UP-0N→UnitN` map, demo fallback | `types/dashboard.md` § "Extending an existing dashboard" | `[ev: corpus B1015 §1015.3]` |
| Δ4 | **when the Niagara SDK version changes, the gradle plugin versions must be re-checked**: `settings.gradle.kts` `gradlePluginVersion`/`settingsPluginVersion` are coupled to what `niagara_home/etc/m2` ships for that SDK family. build.sh should VERIFY the settings value against the SDK-present plugin and WARN on mismatch; and the kit should recommend the PARAMETERIZED form `providers.gradleProperty("niagaraPluginVersion").getOrElse("7.6.17")` (as Paccadia@4.15 uses) over the hardcoded literal (Umbrella/Dashboard/Compresores), so switching SDK is a gradle.properties override, not a settings edit | `toolbelt/build.sh` (plugin-version resolution, lines ~66-70) + `types/structure.md` § settings | `[ev: corpus B1015]` |

## Lessons
- A new dashboard module is a near-mechanical rename+reslot of DashboardPan — the exemplar's exact Java idiom (slotomatic AUTO region, audit trio, servlet/dispatch/reader split) transfers 1:1; slotomatic regenerates the AUTO region so the annotations are what must be right.
- `build.sh` needs an executable `gradlew`; a fresh client scaffold may ship it `-rw-r--r--` → exit 10 (environment), fixed by `chmod +x` — worth a one-line hint in the build.sh error.
- The verify GATE (verify-module.sh, 0 FAIL) is authoritative; `report-module.sh` aggregates extra checks that can false-positive by dashboard type (plano) — read a report FAIL against the module's actual UI kind before treating it as a defect.
- Keep the servlet write-surface CLOSED (`WRITABLE_SLOTS = emptySet()`) until real config slots exist — the full RBAC/XHR/traversal/audit path stays wired and every write is safely rejected 400.
- `gradle.properties` L10 (absolute niagara_home) is the client-repo norm across all four module groups; not a per-module defect.
- **Niagara version and gradle plugin version are two separate pins that must move together**: `gradle.properties` (niagara_home/niagara_user_home) sets the SDK; `settings.gradle.kts` (gradlePluginVersion 7.6.17 / settingsPluginVersion 7.6.3) sets the build plugins — 4.13 and 4.14 share 7.6.17, but a different SDK family (4.15/PowerB) can diverge, so re-check both when retargeting. Paccadia's `getOrElse("7.6.17")` gradle-property form is the portable pattern.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-16-umbrelladashboard-module-creation.md | UmbrellaDashboard | 2026-09-16 | pending | 4 |`
