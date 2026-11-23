<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 4 -->
# 2026-11-23 · kit · deploy-target-version

**Session**: a dashboard module (2.4.0) rebuilt in WSL against a Supervisor installation of Niagara 4.14.0.162, three Niagara installations on the host (4.13.2.18, 4.14.0.162, 4.15.6.20). The user, in Spanish: the module updated the day before is not in the `modules` directory of the 4.14 installation, and "depending on the Niagara version a module is compiled for, it has to be put in that Niagara version" — to be recorded as a retro/delta of the build kit.
**Delta count**: 4

## What happened
`build.sh` ran in mirror mode: gradle's `:jar` step replaced the module's jars in the WSL mirror of the 4.14.0.162 installation only, and the run ended without saying that the installation the mirror was made from never received them, or which installation that was. A manual copy into the Windows installation then failed with "Permission denied": the station (a Windows service, `station.exe`) was running and held the jars, and nothing in the copy step told the operator to stop it. Nothing in the deploy tools compared the Niagara version a jar was compiled against with the version of the installation it was copied into, so a copy into the 4.15.6.20 or 4.13.2.18 installation would have gone through silently.

## Evidence
- A mirror records where it came from: `go/internal/mirrorhome/messages.go:25` writes `source=<resolved path>` into `.niagara-mirror`; `build.sh` only asks whether the marker exists (`go/internal/buildmodule/inputs.go:47,65`, `modjar.IsMirror`) and its last mirror line named the mirror, never the source (`go/internal/buildmodule/replaced.go:65`). `[ev: go/internal/buildmodule/replaced.go]`
- `ng-deploy.sh` copies into `STATION_MODULES_DIR` (`go/internal/ngdeploy/config.go`, `copy.go`); its only version checks were the module's own vendorVersion against the deployed jar (`version.go`, exits 51/52/54). `NIAGARA_HOME` (a mirror) and the station directory are unrelated settings, so a station of another Niagara version was never noticed. `[ev: go/internal/ngdeploy/version.go]`
- What a built jar records of the Niagara version it was compiled against: the niagara-module plugin writes `<dependency name="baja" vendor="Tridium" vendorVersion="4.14"/>` (major.minor floor, one per Tridium module used) in `META-INF/module.xml`; no gradle plugin version, JDK or `niagara_home` path is in the jar (its manifest carries only `Implementation-Vendor`, `Implementation-Version` and per-entry digests). Read from a module jar built the day before and from two other module jars of the same installation. `[ev: read-only inspection of three installations on this host, 2026-11-23]`
- What an installation records of its own version: `modules/baja.jar` → `META-INF/module.xml` root `vendorVersion="4.14.0.162"` (likewise 4.13.2.18 and 4.15.6.20); `nre-core-*` in the same file repeats it; the directory name is only the installer's convention (`Niagara-4.15.6.20`, `iC-Niagara-4.13.2.18`, a vendor-prefixed `...-N4.14.0.162`). `[ev: read-only inspection of three installations on this host, 2026-11-23]`
- The same jar checked against the three installations: PASS for 4.14.0.162, FAIL for 4.15.6.20 and for 4.13.2.18, each row naming both versions. `[ev: go/internal/niagaraversion/command_test.go]`
- The lock: the plugin's post-jar copy and `ng-deploy`'s rename answer a jar held by a running station on a Windows mount with `EACCES`/`EBUSY`; `ng-deploy` already printed "a running station or Workbench holds its module jars open; stop it and run again" (`go/internal/ngdeploy/heldopen.go`), build.sh exit 32 said "close the process holding modules/<jar>", and neither said which installation or how a Windows station is stopped. `[ev: go/internal/ngdeploy/heldopen_test.go]`
- Doc already in the kit that bears on the rule: `build-verify.md` § Build target & plugin version says a station of another version REJECTS a jar built for a higher one, and that `<dependency vendorVersion="4.14">` is a minimum floor satisfied by 4.15 (`BVersion.MEETS_MINIMUM`). The rule below is stricter on purpose: a jar is compiled against one API, and the deploy copies into the installation it was built for.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-23. A deploy refuses a target installation whose Niagara version differs (major or minor) from the one the jars were compiled against: `ng-deploy.sh` exit 55 before the schema-risk precheck, the gate and the copy, both versions in the line; `toolbelt/check-target-version.sh <jar> <niagara_home>` is the same check for a manual copy. A target with no `baja.jar` is not checked (NOTICE); a version that cannot be read is exit 55, never a pass | `BUILD-LOOP.md` § 6 Deploy (station) — operator, § 5 (ng-deploy exits); `build-verify.md` § Station deploy | `[ev: retro deploy-target-version Δ1]` |
| Δ2 | APPLIED 2026-11-23. A passing mirror build ends with one line naming the installation the mirror was made from (the marker's `source=`), its Niagara version, that the jars built were NOT installed there, and the command that installs them (`check-target-version.sh` on the jar, then `cp` into `<installation>/modules/`) | `BUILD-LOOP.md` § 6 Deploy (station) — operator; `build-verify.md` § Building against a running station (Mirror) | `[ev: retro deploy-target-version Δ2]` |
| Δ3 | APPLIED 2026-11-23. A copy into a modules directory that fails as a held jar (build.sh exit 32, `ng-deploy` exit 40 or 10, a manual `cp` with Permission denied) says the station and Workbench of THAT installation must be stopped first; on Windows the station is a service or `station.exe` | `BUILD-LOOP.md` § 4 Post-jar copy lock (exit 32); `build-verify.md` § Deploy (station) | `[ev: retro deploy-target-version Δ3]` |
| Δ4 | APPLIED 2026-11-23. How to read a Niagara version on each side (installation: `modules/baja.jar` `vendorVersion`; jar: the highest Tridium dependency floor), that a mirror is not its source installation, and which installation a module goes into | `types/platform.md` § 4 | `[ev: retro deploy-target-version Δ4]` |

## Lessons
- A mirror build installs nothing in the installation it was made from; the build has to say so and say where the jars are, or the operator looks for them in the wrong directory.
- A module goes into the installation of the Niagara version it was compiled against; the version is read from `modules/baja.jar`, never from a directory name.
- A held jar on a Windows mount is a station or Workbench of that installation: the message names it and says to stop it before the copy.

---
**Status**: FOLDED — INDEX row generated from the header: `| 2026-11-23-deploy-target-version.md | kit | 2026-11-23 | folded | 4 |`
