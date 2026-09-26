<!-- review-status: pending -->
# 2026-09-26 · kit · roll-forward-recovery

**Session**: PANCCADIA León field rule stated by the user on 2026-09-26, while preparing the CompPan 2.6.0 / DashboardPan 2.8.0 deploy: when a new version crashes the station, you cannot go back to the previous version; recovery needs a higher version.
**Delta count**: 4

## What happened
The user stated a field rule: once a new module version is loaded and it crashes
the station, there is no regression to the previous version, and the only way out
is to install a higher version. The official Tridium documentation says otherwise
in principle. The Software Manager supports "Downgrade to <version>", but only when
"the earlier version [is] available in the Supervisor PC's software database". In
this project that condition is almost never met, because every version of our jars
has the same file name (`CompPan-rt.jar`, `DashboardPan-ux.jar`). Copying the new
jar onto the deploy laptop overwrites the old one, so the older version is gone
from the software database at the moment it is needed. The 2026-09-25 unknown-unit
outage shows the real path: the retro "offered" a rollback to DashboardPan 2.6.2,
but the station was recovered by a new version (2.7.1). The kit still states the
opposite in two places. The 2026-09-03 retro says "rollback = redeploy del jar
previo + reinicio", and the 2026-09-25 outage retro presents a rollback to the
previous jars as an option.

## Evidence
- Official downgrade exists but is conditional: `niagara-help guides-clean/Platform/platDaemon-SoftwareManager.txt:78,117` ("Downgrade to <version>", "Downgrade appears if the installed item is an newer version than your locally available one"); `guides-clean/Platform/InstallingModulesInARemotePlatform-2C3324A0.txt:74-75` ("The earlier version must be available in the Supervisor PC's software database"); `:84-85` (up/downgrade stops the station and reboots the host). `[ev: niagara-help Software Manager guides]`
- Same jar file name across versions: `fast-build.sh` prints `<module>/build/libs/<module>.jar` for every version; the version lives only in `META-INF/module.xml vendorVersion`. `[ev: fast-build.sh]`
- Outage recovery was roll-forward: DashboardPan 2.7.0 crash -> fixed and recovered by 2.7.1 (`ae98f79`). `[ev: retro 2026-09-25-panccadia-unknown-unit-outage.md:5-10]`
- Kit text that contradicts the field rule: `retros/2026-09-03-slot-type-change-rompe-bog-station-no-arranca.md:45` ("rollback = redeploy del jar previo + reinicio"). `[ev: retro 2026-09-03-slot-type-change-rompe-bog-station-no-arranca.md]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **Doctrine: recovery from a bad deploy is ROLL-FORWARD.** A version that crashes or degrades the station is replaced by a HIGHER `vendorVersion` (a fix, or the previous code rebuilt under a new version), never by reinstalling the older version. Reason, stated in the doctrine: Software Manager can only downgrade to a version that is still in the local software database, and our same-named jars erase it; the up/downgrade also forces a station stop + host reboot, so there is no cheaper path to save. | `BUILD-LOOP.md` § `6. Deploy (station) — operator` + `METHODOLOGY.md` § `Schema / upgrade safety` | `[ev: niagara-help Software Manager guides]`; `[ev: retro 2026-09-25-panccadia-unknown-unit-outage.md:5-10]` |
| Δ2 | **Pre-build a "revert build" before every production deploy.** Before loading version N, build and sign the LAST-KNOWN-GOOD source under version N+0.0.1 (e.g. before loading CompPan 2.6.0, build the 2.5.0 source as 2.6.1). Keep it next to the N package, with its jar hashes in the feature doc. If N crashes the station, the operator loads the revert build immediately, instead of starting a fix-and-build cycle while the plant is down. The real fix then ships as the next version up. | `BUILD-LOOP.md` § `6. Deploy (station) — operator` + `§ 4.c Version-bump checklist (before any shipped-bytes commit)` | `[ev: retro 2026-09-25-panccadia-unknown-unit-outage.md]` |
| Δ3 | **Keep every shipped jar under a VERSIONED file name outside `modules/`** (e.g. `deploy/CompPan-rt-2.6.0.jar`), plus the hash. The station never needs the old file name back, but the team needs to know exactly which bytes each version was, so a revert build can be rebuilt reproducibly and a crash can be matched to the bytes that caused it. | `BUILD-LOOP.md` § `6. Deploy (station) — operator` | `[ev: fast-build.sh]` |
| Δ4 | **Correct the two retros that describe a downgrade as the rollback path.** Add a pointer in `2026-09-03-slot-type-change-rompe-bog-station-no-arranca.md:45` and in the 2026-09-25 unknown-unit outage retro's recovery line to this retro's Δ1/Δ2: "rollback" means roll-forward to a revert build. Leaving the old wording live invites the next operator to try a downgrade in the middle of an outage. | `retros/2026-09-03-slot-type-change-rompe-bog-station-no-arranca.md` + `retros/2026-09-25-panccadia-unknown-unit-outage.md` | `[ev: retro 2026-09-03-slot-type-change-rompe-bog-station-no-arranca.md:45]` |

## Lessons
- Tridium's downgrade depends on the older bytes still being in the local database; with same-named jars that almost never holds, so plan for roll-forward.
- The cheapest outage recovery is prepared before the deploy: a signed revert build under a higher version.
- Version numbers only move up; a revert is new bytes with a new number, not old bytes.
- An old retro that says "rollback = previous jar" is a live hazard until it points to the corrected doctrine.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-26-roll-forward-recovery.md | kit | 2026-09-26 | pending | 4 |`
