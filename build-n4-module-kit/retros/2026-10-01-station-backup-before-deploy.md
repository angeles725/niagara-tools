<!-- review-status: folded -->
# 2026-10-01 · kit · station-backup-before-deploy

**Session**: PANCCADIA gap review — station-level state (persisted operator values, run hours) is not protected by the module-jar backup
**Delta count**: 3

## What happened
Two PANCCADIA incidents showed that module upgrades and station restarts put operator-persisted state at risk, while the
kit's backup story covers only module jars (ng-deploy) and a read-only audit snapshot (config.bog + console logs). On
2026-09-28 a restart reverted HOA modes to AUTO; the restart-sequencing feature found compressor run hours at about 10.6 h
live against about 65.5 h earlier because they reach disk only on a station save. Neither a full station backup before an
`-rt` install nor a "seed/save operator state before the first restart" step exists in the kit, and the corpus block on
provisioning and backup is not cited by any kit file.

## Evidence
- Restart reverted operator state: TRANSIENT HOA slots reset to AUTO after the 09:03 restart; no code requested a station save `[ev: retros/2026-09-28-panccadia-persistent-config-hoa.md]`
- Hours persisted only on station save: "`condenserNHours` are persisted slots written every execute, but they only reach disk on a station save. Live hours were ~10.6 h vs ~65.5 h on 2026-09-06" `[ev: Cliente/panccadia-leon odd/tasks/restart-seq-comp-lockout-hours.md:12]`
- `ng-deploy.sh` backs up only this module's own `-rt/-ux/-wb` jars by default (whole-dir backup opt-in via `--full-backup`) `[ev: BUILD-LOOP.md:143]`
- `station-snapshot.sh` copies only `config.bog` + `console*.txt`; history/alarm db are pointers, never copied `[ev: toolbelt/station-snapshot.sh:1-13]`
- Kit has no station-backup procedure: `grep -l 'BBackupService\|platform backup'` over kit top-level docs and `types/` hits only `types/logic-authoring.md` (not a deploy procedure) `[ev: kit grep 2026-10-01]`
- Corpus B39 "Provisioning + Backup + Supervisor Replication + HA Operacional + Flota Management" exists and is cited nowhere in `corpus-index.md`, `types/*.md` or `BUILD-LOOP.md` `[ev: corpus B39]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Before any `-rt` install on a live station: take a full station backup (platform/station backup per B39, not only module jars), record where it is stored, and write the restore path in the deploy notes; `--no-backup` on ng-deploy requires this to exist. | `BUILD-LOOP.md` § `6. Deploy (station) — operator` | `[ev: corpus B39]` |
| Δ2 | Add a multi-site / fleet section: provisioning and backup model across stations, what the Supervisor can restore, and what stays per-station. | `types/distribution.md` (new §) "Station backup, provisioning and fleet" | `[ev: corpus B39]` |
| Δ3 | Commissioning checklist row: "persisted operator state (HOA modes, setpoints, run hours) is seeded or saved BEFORE the first restart, and re-read after it". | `BUILD-LOOP.md` § `6.b Commissioning-verify requirement (modules needing station rewiring)` | `[ev: retros/2026-09-28-panccadia-persistent-config-hoa.md]` |

## Lessons
- A module-jar backup protects code, not the operator's state.
- Persisted does not mean saved: a slot reaches disk on a station save, not on write.
- Cite the corpus block that owns the topic, or the kit will keep re-deriving it.
