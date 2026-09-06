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
| DashboardPan | Cliente/Leon-Guanjuato | dashboard | 2026-09-04 | pass | yes | no | 3 |
| chihuahua | Cliente/Honeywell/MX60 | logic | unknown | unknown | unknown | no | 1 |
| kit | niagara-tools | self | 2026-09-05 | n/a | n/a | yes | 0 |

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
profiles: rt,ux
target_version: 4.14
plugin_version: 7.6.17
last_build: 2026-09-04
bytecode_major: 52
signed: yes
verify_gate: pass                   # 2026-09-05 · verify-module.sh --src . --target-version 4.14 on repo HEAD 4f5f1c7; rt 7/7, ux 7/7
deployed: yes
target_station: Leon-JACE
pure_tests: 14
open_issues:
  - U5 residue: `handleSetpointWrite` IS gated fail-closed by `DashboardRbacHelper.checkCanWrite` (BDashboardServlet.java:198) [ev: corpus B763]; residue = lost pure-RBAC test seam (DWS2), no per-Ord lock/423, optional slot allowlist (client punch-list, not kit).
  - DashboardPan-rt and DashboardPan-wb ship empty srcTest — the pure_tests count is the -ux DashboardDispatch suite only.
  - DashboardPan-wb is a scaffold (gradle/lexicon/palette/permissions, zero .java, never built); build or delete before declaring it.
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
version: 0.18.0                      # current kit VERSION
last_change: 2026-09-05              # Campaign 7 CLOSE: fold 9 C7 retros, CHANGELOG v0.18.0, close retro, contract refresh
open_issues:
  - Campaign 2 COMPLETE: logic L3-L22 (PR-A/A2), UX U1-U10 (PR-B), and the build/deploy/schema DOC lessons D1/D2/D3/B9 + S2/S3/S4 (PR-C) are folded into the core.
  - OWED SCRIPT/GATE IMPLEMENTATIONS: ALL DONE — the backlog is cleared. (B4, B6, B7, B8, B10, soft-start, palette all implemented and folded.)
  - DONE: B4 (verify-module rc/ WARN + --strict, C3-PR1 v0.10.0 → detail-render-doors folded); B6 (build.sh auto-detect module gradle target) + B7 (build.sh gradle-root walk-up) + soft-start build.sh clean-lock message exit 31 (all C3-PR2 v0.11.0 → dashboardpan-ux-direct-build + soft-start folded); B8 (ng-deploy.sh `verify_jar` counts `<type ` with a space → EXPECTED_*_TYPES = real count, no +1) + B10 (ng-deploy.sh lightweight-backup default own jars only + keep-N autopurge + `--no-backup` opt-in WARN + `--full-backup`) (both C3-PR3 v0.12.0 → ng-deploy-type-count-and-cwd + ng-deploy-backup-liviano-y-autopurga folded); PALETTE (verify-module.sh empty-palette WARN/--strict FAIL, default-on, type-count-guarded) (C3-PR4 v0.13.0 → module-palette-and-build-target folded — its authoring rule B5 + build target B6 were already in core, this closes its gate half).
  - FINAL-TIDY-PASS: ALL 3 DONE (Campaign 3 close, v0.13.1). (1) tag-convention normalized — the two mixed-format citations (comppan-fase3 and process-timers, which put a retro-local number after the citation separator) now use the full retro slug (neither has a global tag; the ~16 rt-hardening/5rooms retro-local citations are legitimately pre-global-tag and left as-is). (2) doc-vs-script folded-completeness + adversarial fidelity-grading rules folded into a new METHODOLOGY "Kit maintenance — retro promotion discipline" section; campaign2-promotion-process-meta-lessons flipped to folded. (3) detail-render #5's `height:100% (not auto)` nuance spelled out in types/dashboard.md U1.
  - CAMPAIGN 4 (final — exhausts the 42-lesson corpus, the T/process group): C4-PR1 (v0.13.3) folded T5 (preview-approval gate → BUILD-LOOP §3) + T6 (WARNING caught-cosmetic-vs-aborts → METHODOLOGY new Debugging section) + T8 (lexicon toFriendly consequence → METHODOLOGY reinforce); self-retro-preview-gate → folded (T4 live-anchor smoke + the pure-test recipe were already in build-verify.md, so its four lessons are all in core). T7 progress (corpus B729-B760 wiring): C4-PR2 (v0.13.4) DONE — folded the WB ladder + FieldEditor + Wizard recipes B751 → types/wb-widgets.md, and the three-way serving-recipe decision + vendor-unrestricted-RBAC contrast B752/B753 → types/dashboard.md (the RBAC core, pure-router, traversal guard, and Home-Page footgun of B752 were ALREADY in dashboard.md, so only the new parts were folded — no double-fold). C4-PR3 (v0.14.0, MINOR, structural — TERMINAL) DONE: added the curated corpus-index.md nav (B729-B760, priority-first) + in-repo pointers (BUILD-LOOP §2, METHODOLOGY top, types/logic.md) + out-of-repo SKILL wiring companion; corpus-index-rt-authoring → FOLDED (all 8 of its proposed deltas verified in core: corpus-index + pointers C4-PR3; WB/UX growth C4-PR2; lexicon C4-PR1; composition B737 at logic.md §Composition · L21; schema/versioning B739/B754 at METHODOLOGY §Schema · S1/S2/S3; build-target B756 at build-verify.md). **CAMPAIGN 4 COMPLETE — the 42-lesson mined corpus is EXHAUSTED; every mined lesson (L/U/B/S/D/T) is folded and fidelity-graded.**
  - CAMPAIGN 5 (activate the retro-enforcement gate — built but inert): AG-PR1 (v0.15.0) DONE — hardened `.githooks/pre-push` so the promotion exit accepts an in-range INDEX.md diff OR an in-range BUILD-STATE.md diff (fixes the partial-promotion FALSE-NEGATIVE found in C4-PR2), with the blanket-escape guard kept (trailer + NEITHER anchor → FAIL) and `sweep` run on both anchored paths; added `scripts/install-hooks.sh` (idempotent opt-in activation, `--uninstall`, refuses to clobber a custom hooksPath without `--force`); documented activation in BUILD-LOOP §7 + CONTRIBUTING §6.1. Tests: build-retro-sync H8/H9/H10 + install-hooks I1-I4. This closed the earlier gate-hardening open_issue. AG-PR1 carries its OWN feature-retro (campaign5-gate-activation) via the new-retro exit (a) — a pure feature PR is NOT a 4th gate shape, it uses exit (a); so the PR that activates the gate passes its own gate, and the Campaign 5 close is consolidated into this PR (no separate close-retro PR). NEXT: the LIVE ACTIVATION smoke (run the installer in this repo + prove a trivial kit-file push without a retro/trailer is BLOCKED, then passes with the trailer); CI (AG-PR2) awaits the user's explicit call.
  - CONTENT FOLD-AUDIT open_issue (#3, CLOSED by Campaign 6 PR7): the `sweep-fold-audit.sh --strict` gate now verifies every folded retro has a `[ev: retro <token>]` citation in a core kit file. 38 folded, 38 cited, exit 0.
  - B788 (own-modules-vs-exemplars OMV4): ColdRoomPan-rt lexicon partial (32 keys; fanMode/valveMode/freeze* missing — likely camelCase mismatch); DashboardPan-rt type-set coverage 100% but per-slot ~25% (many slots without lexicon keys); DashboardPan-wb palette is an empty scaffold (`<p t="b:Folder">` only). Module fixes are out of niagara-tools scope. [ev: corpus B788]
retro_required: true                 # GATED — close retro filed and folded
retro_pending: true                  # campaign8-lint-delays + campaign8-triage-console + campaign8-lint-timers-ext + campaign8-facets-lint + campaign8-slot-per-slot + campaign8-rc-scan + campaign8-doctrine-fold + campaign8-report-integration + campaign8-station-snapshot + campaign8-bog-audit + campaign8-wb-audit + campaign8-lint-servlet + campaign8-post-deploy-checklist + campaign8-build-pipeline + campaign8-orchestration + campaign8-structure (all pending fold)
last_commit: 2f4c4bb — Campaign 8 PR18 apply: lint-structure.sh L1-L11 + types/structure.md + K19 routing (Campaign 8 PR18)
last_session: 2026-09-06 · Campaign 8 PR18: lint-structure.sh (L1-L11 source-tree lint; 8/8 LS pins green); types/structure.md (B817 layout + L1-L11 table); K19 routing BUILD-LOOP.md §5 + skill/SKILL.md. Mutations: (a) L7 2-part floor FAIL, (b) L4 drop flips LS4, (c) L9 drop flips LS9, (d) prune drop flips LS-prune. Real smokes: chihuahua L4×3 exit 1; DashboardPan exit 0 (no module-include.xml in -wb, project-level gradle.properties); ColdRoomPan/CompPan exit 0 (pure JUnit srcTest); scaffold exit 0. Guards: 292/292 bats, shellcheck clean, sweep-build-state/sweep-fold-audit exit 0, kit-links 7/7. Retro + INDEX + BUILD-STATE filed.
2026-09-05 · Campaign 8 PR10: bog-audit.sh (CHECK1-CHECK12; python3 embedded engine). PANCCADIA smoke: 17 CHECK11 FAIL, 1 CHECK2 WARN, 0 CHECK5/7 FAIL. MX60 smoke: exit 0, 109 chihuahua components. All 13 bats green, shellcheck clean, 6/6 kit-links, sweeps clean. K19 routing: BUILD-LOOP.md §6 + skill/SKILL.md. Retro + INDEX + BUILD-STATE filed. Deviations: CHECK11 FAIL not WARN (K13 RED wins), CHECK12 WARN (advisory). Fixes: compound-type sub-slot direct-parent tracking, wsAnnotation platform-slot exclusion, Java numeric literal suffix in MIN facet regex.
<!-- /build-state.v1 -->
