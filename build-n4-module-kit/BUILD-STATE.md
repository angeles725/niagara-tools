# BUILD-STATE — the build-session continuity ledger

Where the last build session for each module left off. A new session READS this at orient
(`BUILD-LOOP.md` §0.a) and UPDATES it at close (`BUILD-LOOP.md` §7). This is the kit's
"where did we leave off" registry — the build-loop analog of research-sdd's FOCUSES +
RESEARCH-STATE registries, collapsed into ONE file because the build corpus is a handful of
modules, not sixty focuses.

## How to read this file

- **One `build-state.v1` envelope per module**, between `<!-- build-state.v1 -->` and
  `<!-- /build-state.v1 -->`. Fields are `key: value`; a parser strips blank lines, the
  `open_issues:` list items, and any trailing `# comment`.
- **GATED vs DECLARED — never label a gate that cannot run.**
  - **GATED** (`retro_required`, `retro_pending`): machine-enforced, but ONLY for KIT-local
    changes inside the `niagara-tools` repo — that is the only diff the enforcement check
    (build-n4-retro-gate / PR2) can see.
  - **DECLARED** (`module_repo`, `module_root`, `last_build`, `bytecode_major`, `signed`,
    `verify_gate`, `deployed`, `target_station`, `pure_tests`, `last_commit`): these describe
    module trees that live in SEPARATE repos (`Cliente/Leon-Guanjuato`, `Cliente/Honeywell/MX60`).
    A `niagara-tools` check cannot see those diffs, so they are RECORDED here, not gated.
- **Unknown is honest.** A DECLARED field with no authoritative value is `unknown`, never a guess.

## Index

| Module | Repo | Type | last_build | verify_gate | deployed | retro_pending | open_issues |
|---|---|---|---|---|---|---|---|
| ColdRoomPan | Cliente/Leon-Guanjuato | logic | 2026-09-03 | pass | yes | no | 1 |
| CompPan | Cliente/Leon-Guanjuato | logic | 2026-09-04 | pass | yes | no | 2 |
| DashboardPan | Cliente/Leon-Guanjuato | dashboard | 2026-09-04 | unknown | yes | no | 2 |
| chihuahua | Cliente/Honeywell/MX60 | logic | unknown | unknown | unknown | no | 1 |
| kit | niagara-tools | self | 2026-09-04 | n/a | n/a | no | 2 |

---

## ColdRoomPan — canonical worked example (every field annotated)

<!-- build-state.v1 -->
module: ColdRoomPan                 # the module name (matches the -rt/-ux/-wb artifact prefix)
module_repo: Cliente/Leon-Guanjuato # DECLARED — modules live in a SEPARATE repo, not niagara-tools
module_root: /home/cristian/modulos_niagara_n4/Cliente/Leon-Guanjuato/Paccadia/ColdRoomPan  # DECLARED
type: logic                         # logic | dashboard | wb (the SKILL.md decision table)
profiles: rt                        # which profiles have sources (rt,ux,wb)
target_version: 4.14                # DECLARED — LOWEST niagara_home built against (settings.gradle.kts)
plugin_version: 7.6.17              # DECLARED — com.tridium.niagara plugin; must exist in <niagara_home>/etc/m2
last_build: 2026-09-03              # DECLARED — date of last successful build (unknown if never/uncertain)
bytecode_major: 52                  # DECLARED — must be 52 (Java 8); any other value is a FAIL signal
signed: yes                         # DECLARED — META-INF/NIAGARA4.SF present
verify_gate: pass                   # DECLARED — toolbelt/verify-module.sh outcome (pass|fail|unknown)
deployed: yes                       # DECLARED — reached a station (yes|no|unknown)
target_station: Leon-JACE           # DECLARED — where it runs; the station executes off the Atlas SNAP
pure_tests: 22                      # DECLARED — pure-Java JUnit count (ColdRoomControlTest)
open_issues:
  - DefrostController.java (742 lines) has ZERO pure tests — QA HIGH gap; extract a pure DefrostControl class + tests (a module change, OUT of this campaign's scope). It shipped the started()/interval production bug.
retro_required: true                # GATED (kit-local) — did the last session change kit behavior / prove a lesson?
retro_pending: false                # GATED — the enforcement hook: true until the owed retro exists; false here, its retros were written
last_commit: f89e44e                # DECLARED — short sha in module_repo of the last build's commit
last_session: 2026-09-03 · self-firing-timer defrost fix confirmed live [CERT-live]; next: extract DefrostControl pure class + tests
<!-- /build-state.v1 -->

## CompPan

<!-- build-state.v1 -->
module: CompPan
module_repo: Cliente/Leon-Guanjuato
module_root: /home/cristian/modulos_niagara_n4/Cliente/Leon-Guanjuato/Compresores/CompPan
type: logic
profiles: rt
target_version: 4.14
plugin_version: 7.6.17
last_build: 2026-09-04
bytecode_major: 52
signed: yes
verify_gate: pass
deployed: yes
target_station: Leon-JACE
pure_tests: 31
open_issues:
  - suctionPressure2 sensor stuck/frozen at 130.5342 psi — control UNAFFECTED (selectSuction uses the healthy primary); monitor / replace the sensor.
  - amps2 / amps3 read low or zero while the compressors physically run — an amperage-sensor issue, not equipment; amperage is visual-only and control does not depend on it.
retro_required: true
retro_pending: false
last_commit: d6eccaf
last_session: 2026-09-04 · HOA manual override per compressor + dischargeHighLimit 0=disabled deployed live [CERT-live]; next: watch the stuck suction-2 sensor
<!-- /build-state.v1 -->

## DashboardPan

<!-- build-state.v1 -->
module: DashboardPan
module_repo: Cliente/Leon-Guanjuato
module_root: /home/cristian/modulos_niagara_n4/Cliente/Leon-Guanjuato/Dashboard/DashboardPan
type: dashboard
profiles: rt,ux,wb
target_version: 4.14
plugin_version: 7.6.17
last_build: 2026-09-04
bytecode_major: 52
signed: yes
verify_gate: unknown
deployed: yes
target_station: Leon-JACE
pure_tests: 14
open_issues:
  - servlet generic write endpoint lacks an OPERATOR-flag / whitelist check (U5) — the write-surface is wider than the read-surface; the RULE is promoted into types/dashboard.md in PR3, the code fix is OUT of scope.
  - DashboardPan-rt and DashboardPan-wb ship empty srcTest — the pure_tests count is the -ux DashboardDispatch suite only.
retro_required: true
retro_pending: false
last_commit: 6b9b085
last_session: 2026-09-04 · Condensadoras tab + HOA control (preview -ux); next: wire live pressure/amps data into the -ux
<!-- /build-state.v1 -->

## chihuahua (reference exemplar — not built from this kit)

<!-- build-state.v1 -->
module: chihuahua
module_repo: Cliente/Honeywell/MX60
module_root: /home/cristian/modulos_niagara_n4/Cliente/Honeywell/MX60/chihuahua
type: logic
profiles: unknown
target_version: unknown
plugin_version: unknown
last_build: unknown
bytecode_major: 52
signed: unknown
verify_gate: unknown
deployed: unknown
target_station: unknown
pure_tests: unknown
open_issues:
  - reference exemplar only — cross-project knowledge lives in the engram project honeywell-mx60-chihuahua (BAlarmService ackAlarm no-op #1788; hot-reload rule #1779). Not built from this kit.
retro_required: false
retro_pending: false
last_commit: unknown
last_session: unknown · imported as the build exemplar; see SOURCES.md
<!-- /build-state.v1 -->

## kit — the kit's own evolution (self-section)

Kit-infrastructure work (changing the kit itself, not building a module) has no module build to
record, so it updates THIS `kit` self-section. Same `build-state.v1` envelope and same `BUILD-LOOP.md`
§7 close gate — this is the kit's no-registro fix applied to its own evolution.

Split-retro fold rule (from Campaign 2): when a retro's remaining lesson has a LATER PR that will carry it, revert its INDEX row to `pending`; when it has NO later home, fold it now — reverting would orphan the lesson.

<!-- build-state.v1 -->
module: kit                          # the kit itself — the module_repo/build fields do not apply
version: 0.9.3                       # current kit VERSION
last_change: 2026-09-04              # Campaign 2 PR-C: fold build/deploy/schema; Campaign 2 complete
open_issues:
  - Campaign 2 COMPLETE: logic L3-L22 (PR-A/A2), UX U1-U10 (PR-B), and the build/deploy/schema DOC lessons D1/D2/D3/B9 + S2/S3/S4 (PR-C) are folded into the core.
  - OWED SCRIPT/GATE IMPLEMENTATIONS (rule folded, impl pending — future MINOR feature PRs, test-first; Campaign 3 candidates): B4 (verify-module.sh WARN on non-asset files under rc/), B6 (build.sh auto-detect the module's own gradle target), B7 (build.sh detect the gradle root / accept :project:task), B8 (ng-deploy.sh grep `<type ` with a space so EXPECTED_*_TYPES matches the gate), B10 (ng-deploy.sh default no-backup + keep-N purge); soft-start build.sh (print ONE actionable line on the `modules/<jar>` clean-lock failure, not the raw stacktrace). Their retros stay review-status pending until implemented.
  - UNFOLDED T/process group (out of Campaign 2 scope, Campaign 3 candidate): T5/T6 (self-retro preview gate + WARNING classification), T7 (curated corpus-index), T8 (module.lexicon empty renders raw camelCase).
  - CompPan-rt/module.lexicon is empty (T8) — slots render raw camelCase; the rule is in METHODOLOGY, the module fix is out of scope.
retro_required: true                 # GATED — PR3 (and PRs 12/13) changed kit files
retro_pending: false                 # GATED — the two owed retros (campaign + run-pure-test) are FILED this session; their fold-status lives in retros/INDEX.md as review-status: pending
last_commit: unknown                 # set at PR3 merge
last_session: 2026-09-04 · Campaign 2 PR-C folded build/deploy/schema (D1/D2/D3/B9 + S2/S3/S4); Campaign 2 COMPLETE. next: Campaign 3 — the 5 owed script-impls (B4/B6/B7/B8/B10), then the T/process group; both tracked in open_issues above
<!-- /build-state.v1 -->
