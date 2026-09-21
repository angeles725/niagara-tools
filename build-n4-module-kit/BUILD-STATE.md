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
| UmbrellaDashboard | Cliente/Juarez/Umbrella | dashboard | 2026-09-16 | pass | no | no | 4 |
| chihuahua | Cliente/Honeywell/MX60 | logic | unknown | unknown | unknown | no | 1 |
| kit | niagara-tools | self | 2026-09-06 | n/a | n/a | no  | 0 |

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
retro_pending: true                # GATED — the enforcement hook: true until the owed retro exists; false here, its retros were written
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
retro_pending: true
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
retro_pending: true
last_commit: 6b9b085
last_session: 2026-09-04 · Condensadoras tab + HOA control (preview -ux); next: wire live pressure/amps data into the -ux
<!-- /build-state.v1 -->

## UmbrellaDashboard — new dashboard module (Juárez/Umbrella "Productos de Agua" 3D HMI)

<!-- build-state.v1 -->
module: UmbrellaDashboard
module_repo: Cliente/Juarez/Umbrella
module_root: /home/cristian/modulos_niagara_n4/Cliente/Juarez/Umbrella/UmbrellaDashboard
type: dashboard
profiles: rt,ux
target_version: 4.14
plugin_version: 7.6.17              # settings.gradle.kts gradlePluginVersion (settingsPluginVersion 7.6.3); coupled to the SDK family — re-check on version change (retro Δ4)
last_build: 2026-09-16             # build.sh --profiles rt,ux --target-version 4.14
bytecode_major: 52
signed: yes
verify_gate: pass                  # 2026-09-16 · verify-module 17 passed / 0 failed / 3 warned (benign ord-literal SERVICE_ORD + phantom-dep project-dep); rt 2 classes, ux 13 classes; baja 4.14 <= target 4.14
deployed: no                       # build+verify only; no station access this session
target_station: Juarez-Umbrella    # pending — station point list not yet available (B1015-G1)
pure_tests: 14                     # UmbrellaDispatchTest (router guards, 0 Baja imports) → OK (14 tests)
open_issues:
  - B1015-G1 real point wiring — Juárez/Umbrella station point list pending; live mode maps UP-0N→UnitN by ordinal; integrator links Unit1..Unit8 slots at commissioning.
  - B1015-G2 no writable config slots — WRITABLE_SLOTS empty, POST /api/setpoint always 400 until config slots added.
  - B1015-G3 unit status derived from BStatus (normal/offline/alarma) — richer states need a status slot or alarm-service.
  - B1015-G4 not deployed — ng-deploy.sh + commissioning + triage-console remain.
retro_required: false
retro_pending: true               # retro written: retros/2026-09-16-umbrelladashboard-module-creation.md (4 deltas, review-status: pending)
last_commit: unknown               # client repo; not committed this session
last_session: 2026-09-16 · created from DashboardPan exemplar (B1015); rt+ux green @4.14, 3D SPA wired to /api/equipment; next: real point wiring + deploy
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
version: 0.23.0                      # current kit VERSION (was stale at 0.18.0; CHANGELOG had reached v0.22.0)
last_change: 2026-09-20              # apply-deltas campaign #123 CLOSE: all 10 pending delta-retros folded (114Δ, PR #124-133), CHANGELOG v0.23.0
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
retro_pending: true
last_commit: TBD — feat/module-worktree-location-guard merge sha (fill post-merge)
last_session: 2026-09-18 · ODD fold campaign (branch odd/fold-kit-deltas-2026-09-18, BUILD-LOOP §7): folding the 16 corpus-mining delta-retros into kit core, one work-unit commit each; new lint checks carry bats. Progress 7/16 FOLDED — T1 corpus-index-refresh, T2 sdk-examples (+check_moduletest_present), T3 authoring-exemplars, T4 module-mechanics, T5 schema-versioning (+check_cross_module_type, schema-risk L4/fox-sync), T6 slots-flags-java8 (+check_transient_operator, +check_compact3_imports), T7 ux-wb-writesurface-rbac, T9 spa-library-integration (docs: dashboard JS-embed recipe + injection + grunt-decision table; JS-in-browser vs Java-in-JVM distinction). T8 palette (report-module dup-keys BUGFIX + check_palette root WARN V18 + palette-template/-doc-help/icon docs; verify 32/32, report 8/8). 9/16 original folded. QUEUED new retros: T17 math-control-library-layer (4Δ), T18 wiresheet (1Δ), T19 third-party-library-integration FOLDED (new types/third-party-libraries.md + build-verify pointer). T20 ml-libraries (pending gen). Original folded: T1-T9 (9/16); remaining original T10-T16 (7); new folded T19; new pending T17/T18/T20. verify-module.bats 31/31, schema-risk.bats 11/11, shellcheck clean. New verify-module checks so far: moduletest, cross-module-type, transient-operator, compact3-import. Detail in odd/tasks/fold-kit-deltas.md + retros/INDEX.md. [ev: retro corpus-index-refresh-b761-b1028] [ev: retro sdk-examples-kit-deltas] [ev: retro authoring-exemplars-deltas] [ev: retro module-mechanics-deltas] [ev: retro schema-versioning-upgrade-safety-deltas] [ev: retro slots-flags-status-units-java8-deltas] [ev: retro ux-wb-writesurface-rbac-deltas] [ev: retro spa-library-integration-deltas] [ev: retro palette-lexicon-assets-images-deltas] [ev: retro third-party-library-integration-deltas] [ev: retro wiresheet-control-library-deltas] [ev: retro watcher-subscription-livedata-deltas] [ev: retro permissions-security-model-deltas] [ev: retro factory-composition-new-types-deltas] [ev: retro ord-path-nav-deltas] [ev: retro resource-threading-consumption-deltas] [ev: retro organization-tags-grouping-deltas] [ev: retro insights-issues-catalog-deltas] [ev: retro math-control-library-layer-deltas] [ev: retro ml-libraries-deltas]
2026-09-19 · kit-improvement campaign (branch odd/apply-kit-candidates-2026-09-19) START — folding the master candidate register (odd/tasks/kit-improvement-candidates-2026-09-19.md) into kit core, one work-unit commit per cluster; new lint checks carry bats. WU1 FOLDED: new types/actions.md §1-6 (doX dispatch law; ASYNC engine-thread; flags CONFIRM_REQUIRED/NO_AUDIT; typed args defaultValue; actions-vs-@NiagaraTopic; @NiagaraRpc-vs-@NiagaraAction boundary). [ev: retro actions-authoring] WU2-6 FOLDED: new types/module-wiring.md (gradle deps + type/agent-on registration + cross-module refs), types/driver-authoring.md (network/device/BProxyExt SPI + tuning/ping/poll + KNX/Z-Wave), types/value-types.md (BFrozenEnum/BDynamicEnum/BSimple/BStruct/BFacets/BUnit), types/security.md (null-Context audit hazard + RBAC + CSRF/BPassword + anti-injection), types/observability.md (logging/spy/fault/audit-vs-log). [ev: retro module-wiring] [ev: retro driver-authoring] [ev: retro value-types] [ev: retro security-model] [ev: retro observability] WU7-10 FOLDED (wave 3, OEM+vendor classes): new types/distribution.md (OEM packaging: .dist/dist.xml, overlay+trust certs, BOG schema-safety, version floor, .ntpl, AX->N4), types/theme.md (zero-Java ux theme module), types/utility-lib.md (pure lib bundle + helper-type lib + protocol-split), types/cloud-connector.md (BCloudConnectionService SPI maps + KeyRing auth + backend plugin chain). [ev: retro distribution] [ev: retro theme-module] [ev: retro utility-lib] [ev: retro cloud-connector] WU11-14 FOLDED (wave 4, expansions to existing docs): structure.md §-se (Java-SE profile, JACE {rt,se} load, wb-dep + headless-AWT traps, install no-rollback), moduleTest.md (BTestNgStation fixture, waitFor, DataProvider, NRetryAnalyzer, @Requires, StationRunner, palette assert), issues-and-gotchas.md (Clock<=0 floor, rt lifecycle seam, Missing-class deploy, ux-Jasmine, EC-Net set(value), GET-destructive), logic-authoring.md (BSingleton/@AgentOn, lifecycle guard contract, BProgram/BBatchRoutine, BEmailService, BIRestrictedComponent, provider-in-service, Fox file-channel job). [ev: retro se-profile] [ev: retro moduletest-patterns] [ev: retro gotchas-tier-a] [ev: retro logic-authoring-spis] Docs slice merged to main via PR #109. LINTS slice (branch odd/apply-kit-lints-2026-09-19): 9 new toolbelt lints, each with bats green + shellcheck clean — no-system-out (FAIL), se-display (FAIL), agent-on-shape (FAIL), jasmine-ux/clock-zero-floor/null-context-write/bql-string-concat/uberjar-api-conflict/arbitrary-ord (WARN). [ev: retro new-lints] STRUCTURAL slice (branch odd/apply-kit-structural-2026-09-19): closed the pre-existing sweep-fold-audit --strict debt that was failing CI on main — 8 module-scoped retros (folded into CLIENT modules, out of niagara-tools kit-core) are now cited here for ledger coherence: [ev: retro per-evap-independent-control] [ev: retro air-defrost-off-cycle] [ev: retro freeze-stat-silent-surface] [ev: retro condensadoras-facade-crashsafe] [ev: retro r14-second-login] [ev: retro per-evap-config-ui-hmi-noscroll] [ev: retro two-differential-fase2] [ev: retro module-worktree-location-guard]. CI-GREEN (branch odd/green-ci-2026-09-19): shellcheck 0.10.0 exit 0 + full bats suite 571 ok / 0 not-ok (58 env-skips) + sweeps exit 0. Fixes: .shellcheckrc for pre-existing info/style/warning noise, 2 bats error-code fixes (SC2314/SC1087), `# Mutation:` guard-pin headers on 10 lints, the 9 new lints + generate-wiring-map.sh named in BUILD-LOOP.md, 2 dangling doc refs reworded. No lint runtime behavior changed. [ev: retro ci-green] ISSUE #113 (branch odd/issue-113-wire-lints): the 9 new lints wired into report-module.sh (per-artifact src §5.7-5.13 + profile-gated se-display/jasmine-ux + module-root §8-9 agent-on-shape/uberjar-api-conflict) + RM7-RM11 bats; report-module.bats 13/13, full suite 576 ok / 0 not-ok, shellcheck 0.10.0 exit 0. [ev: retro wire-lints] ISSUE #116 (branch odd/issue-116-skill): expanded the skill trigger (now names driver/network, station service, utility/library, theme/branding, cloud connector) and synced the stale installed ~/.claude copy via `install-skill.sh --force` (the sha256 diff-warn tooling already existed). install-skill.bats 5/5; full suite 576 ok / 0 not-ok. [ev: retro skill-trigger] ISSUE #115 (branch odd/issue-115-scaffold): new fixtures/MinimalDash (25 files, -rt facade + -ux BWebServlet/SPA/spec) + scaffold-module.sh `--type <logic|dashboard>` (logic default unchanged) + CI scaffold-diff (dashboard) step + TC-DASH1/2 bats. Both round-trips clean; full suite 578 ok / 0 not-ok; shellcheck 0.10.0 exit 0. [ev: retro dashboard-scaffold] ISSUE #114 (branch odd/issue-114-commissioning): new commissioning-verify.sh orchestrates the §6.b checks (lint-config-sanity + lint-status-parity + lint-recovery-path over src; bog-audit CHECK11/13-19/20 over --bog; wiring-map note; MANUAL live-only footer) into one punch-list — closes the kit's own BUILD-LOOP §6.b gap. commissioning-verify.bats 5/5; kit-links 10/10; full suite 583 ok / 0 not-ok; shellcheck 0.10.0 exit 0. ALL 4 structural issues (#113/#114/#115/#116) closed. [ev: retro commissioning-verify]
2026-09-20 · apply-deltas campaign (issue #123, ODD chained — fold ALL 114 pending kit deltas, one PR per retro). T1 FOLDED wb-field-editors-deltas (3Δ) → types/wb-widgets.md new "Field editors" § (PD-FE1 targetType facet zero-code component picker + null-ord gotcha; PD-FE2 Manager getNewTypes/promptForNew point-creation, not BComponent.add(); PD-FE3 filtered BComponentChooser subclass, never override baja:Ord). INDEX row flipped folded. [ev: retro wb-field-editors-deltas]
2026-09-20 · apply-deltas T2 FOLDED wb-manager-framework-deltas (4Δ) → types/wb-widgets.md new "Manager recipe" § (PD-WMF1 BAbstractManager 6-object anatomy + 3 boolean gates; PD-WMF2 MgrColumn taxonomy Name/Type/Prop/PropPath; PD-WMF3 BAbstractManager-vs-BWbComponentView decision table; PD-WMF4 minimal non-driver custom-manager template @AgentOn service). INDEX row flipped folded. [ev: retro wb-manager-framework-deltas]
2026-09-20 · apply-deltas T3 FOLDED our-dashboard-audit-deltas (5Δ) → types/security.md new §7 servlet-response-header checklist + post-deploy probe (PD-ODA1 X-Content-Type-Options/CSP gap + PD-ODA5 curl probe cross-ref); types/dashboard.md rt §test-intent-comment rule + facade §SERVICE_ORD portability tradeoff + Module-packaging §minimum-palette contract (PD-ODA4 + PD-ODA2 + PD-ODA3); toolbelt/commissioning-verify.sh new MANUAL servlet-response-headers step. INDEX row flipped folded. [ev: retro our-dashboard-audit-deltas]
2026-09-20 · apply-deltas T4 FOLDED module-hardening-reqexec-closed-deltas (7Δ) → types/security.md §7+§8 (UXS1 global servlet headers, UXS4 CORS recipe), types/distribution.md §5+§10 (BLD7 signing-profile build-triage, PER7 boot-recovery), types/driver-authoring.md §5 (RUN5 configFatal recovery), types/logic-authoring.md new § (PER1 dynamic-slot prune, PER4 started()-migration), types/issues-and-gotchas.md D4+G1 (BLD7 signing-profile gotcha, PER7 boot-recovery gotcha). INDEX row flipped folded. [ev: retro module-hardening-reqexec-closed-deltas]
2026-09-20 · apply-deltas T5 FOLDED module-hardening-reference-cards-deltas (8Δ) → types/logic.md (REF2 isValid/isOk control gate, REF1 BQL function catalog, REF5 BFormat pattern+security, RUN1 callback-wrap+finally rearm), types/logic-authoring.md (REF4 BEnumRange grammar+non-contiguous gotcha, REF7 SlotPath escape rules, REF3 BFacets 31-key catalogue, REF6 BRelTime raw-ms wire encoding). INDEX row flipped folded. [ev: retro module-hardening-reference-cards-deltas]
2026-09-20 · apply-deltas T6 FOLDED honeywell-wb-rt-wb-deltas (8Δ) → types/wb-widgets.md (Δ1 plugin SPI §, Δ2 per-user prefs §, Δ3 fail-closed auth rule-11+visibilityPin §, Δ4 integer-pin § , Δ8 static-State §), types/driver-authoring.md (Δ1 §1.3 plugin-SPI note), types/security.md (Δ3 §2.3 fail-closed, Δ7 §3.3 credential-redaction), types/logic-authoring.md (Δ5 chunked-OTA §, Δ6 AtomicBoolean §), types/distribution.md (Δ5 §11 OTA note), types/observability.md (Δ7 credential-redact §, Δ6 guard-log §). INDEX row flipped folded. [ev: retro honeywell-wb-rt-wb-deltas]
2026-09-20 · apply-deltas T7 FOLDED wb-vendor-ux-wave3-vendor-drivers-deltas (14Δ) → types/driver-authoring.md §9 (Δ1 obix driver-vs-servlet+local-export-discover, Δ2 lonworks NV-binding≠point-list, Δ3 OEM-on-stock-driver patterns, Δ4 BStationMgrCommand SPI+session-keyed learn, Δ7 mbus dual-address wizard, Δ10 OPC action-slot bridge+structured error decode); types/wb-widgets.md (Δ5 rung-4 vendor-env ceiling, Δ3 OEM WB notes, Δ4 session-keyed learn note, Δ6 PlatformServicePlugin target, Δ8 SNMP thin CRUD template, Δ9 multi-perspective managers, Δ10 OPC WB layer, Δ11 BDaemonSessionView platform-daemon-file, Δ12 WB target taxonomy); types/security.md §9 (Δ13 SHA-256 credential digest); types/issues-and-gotchas.md §H1 (Δ14 precision-facet-learn-mismatch lint). INDEX row flipped folded. [ev: retro wb-vendor-ux-wave3-vendor-drivers-deltas]
2026-09-20 · apply-deltas T8 FOLDED module-hardening-failure-modes-deltas (16Δ) → types/logic.md (Δ1 changed-synchronous-recursion+deadband-correctness, Δ6 dangling-link-silent-degradation+watchdog-rule, Δ8 BConversionLink-stale-converter, Δ13 slotomatic-verbatim-expression-triage); types/logic-authoring.md (Δ2 lifecycle-callback-contract, Δ3 subscribe-unsubscribe-symmetry+lint-candidate, Δ5 BJob-submission-safety+RejectedExecutionException, Δ15 @NiagaraType-required-for-type); types/security.md §2.4 (Δ4 post()-drops-RBAC-context); types/distribution.md §5b+§5-triage+§6-restore-direction (Δ9 restore-direction-floor-rule, Δ10 signature-failure-triage-pivot, Δ14 runtimeProfile-valid-set+corrupt-trap); types/issues-and-gotchas.md §D4c+§D4b+§G2 (Δ7 wb-profile-component-rt-station-drop, Δ11 bytecode-version-UnsupportedClassVersionError, Δ12 split-package-first-dep-wins); types/wb-widgets.md §-ux-toolchain (Δ16 grunt-niagara+Gradle-plugin+RequireJS-AMD). INDEX row flipped folded. [ev: retro module-hardening-failure-modes-deltas]
2026-09-20 · apply-deltas T9 FOLDED apillm-headless-servlet-rt-4.14-deltas (24Δ; Δ19 superseded→forward-ref) → types/dashboard.md (Δ1 doGet-override-contract+getServletName, Δ2 servlet-in-rt servlet-api compileOnly, Δ9 export-dedup, Δ10 absent-facet-json-null, Δ11 precision-facet-numeric, Δ12 REST-poll-citation, Δ14 dual-path-servlet-recipe); types/security.md §7 (Δ13 X-Content-Type-Options gap confirmed Apillm); types/wb-widgets.md (Δ3 -wb niagara-home-repositories plugin, Δ5 agent-registration both-required, Δ7 intuitive-UI rule, Δ8 station-space-picker+lint-wb-usability, Δ15 table-relayout-repaint, Δ17 refresh-on-activation+lint-wb-refresh, Δ19 forward-ref→Δ20/PD-FE1, Δ20+Δ21+Δ22 PD-FE1/FE2/FE3 apillm-cite, Δ23 one-step-create-referenced-component); types/structure.md (Δ4 scaffold-UnrestrictedFolder, Δ16 JDK-pinning-block-required, Δ24 slotomatic-checksum-0-not-auto); types/logic.md (Δ6 delay-floor-inline-not-variable); types/logic-authoring.md (Δ18 poller-robustness-url+enabled-guard). INDEX row flipped folded. [ev: retro apillm-headless-servlet-rt-4.14-deltas]
2026-09-20 · apply-deltas T10 FOLDED wb-vendor-ux-rt-wb-pattern-deltas (25Δ) → types/wb-widgets.md (Δ1 WB-presence-rule, Δ2 MgrColumn.Prop-per-proxy-key, Δ3 toRow-pre-population, Δ4 job-bar-xref, Δ5 prune-inherited-commands, Δ6 BOrd-station-scoped-picker, Δ7 dialog-live-RT-objects, Δ8 BWbComponentView-non-tabular, Δ10 BWbService-cross-cutting, Δ15 facet-driven-FE-selection, Δ16 addDefault/Custom-Columns, Δ17 isDefault-blank-suppression, Δ19 BTabbedPane+BLabelPane-recipe, Δ21 COV-capability-toRow, Δ22 boolean-flag-column-switch, Δ24 BTabbedPane.selection-TRANSIENT); types/driver-authoring.md §1.3+pre-§9 (Δ1 WB-presence-rule, Δ13 no-UX-explicit, Δ14 NMgrControllerUtil-native-ext-point, Δ20 getDeviceManagerSubscribeDepth); types/logic-authoring.md §PD-04-xref+§PD-09 (Δ4 long-RT-job-bar-xref, Δ9 @NiagaraTopic-topic-fire-pattern); types/security.md §10 (Δ23 outbound-enableNonDriverClients-vs-inbound-BAuthenticationScheme); METHODOLOGY.md (Δ11 WB-archetypes-decision-tree); types/dashboard.md (Δ12 RT-UX-WB-triangle); types/cloud-connector.md §7+§8 (Δ18 enableNonDriverClients-GOTCHA, Δ25 BICloudConnector-BConnectorImpl); types/issues-and-gotchas.md §H2 (Δ18 enableNonDriverClients-blocks-non-driver). INDEX row flipped folded. CAMPAIGN #123 COMPLETE — all 10 retros folded. [ev: retro wb-vendor-ux-rt-wb-pattern-deltas]
2026-09-20 · lint-candidates impl L1 (partial promotion from a folded retro): new toolbelt/lint-subscribe-without-unsubscribe.sh + tests/lint-subscribe-without-unsubscribe.bats (SWU2 guard-pin MATCH; 7/7 bats; shellcheck 0) — folds the subscribe-without-unsubscribe lint CANDIDATE from module-hardening-failure-modes-deltas Δ3 into a real WARN-only rt-src lint. report-module/BUILD-LOOP wiring deferred to a batched wire-lints PR (kit #113 precedent). [ev: retro module-hardening-failure-modes-deltas]
2026-09-20 · automation-audit PR-A: wired 9 static-source orphan lints into report-module.sh (§5.14-5.20 per-artifact rt/ux/wb + §10 lint-structure + §11 lint-write-path) + RM12-20 (report-module.bats 22/22); fixed lint-write-path exit-3=no-matrix mishandled as env-fault. Closes the 'report-module misses orphan lints' half of the audit; PR-B (build.sh auto-chain preflight+report) next. [ev: retro report-module-orphan-wiring]
2026-09-18 · ODD fold campaign COMPLETE — ALL 20 delta-retros folded (16 original + 4 operator-driven: third-party-library-integration, math-control-library-layer, ml-libraries, wiresheet-control-library). 6 new verify/lint checks shipped WITH bats (check_moduletest_present, check_cross_module_type, check_transient_operator, check_compact3_imports, check_subscription_leak, check_slot_wall + lint-timers atSteadyState-only-timer) + report-module dup-keys BUGFIX. 4 new type docs (moduleTest, third-party-libraries, math-control-libraries, ml-libraries, issues-and-gotchas) + corpus-index refreshed B761-B1028. Merged to main 289575d.
2026-09-18 · LEGACY-PENDING fold campaign COMPLETE (branch odd/fold-legacy-pending-2026-09-18) — the 8 pre-2026-09-18 pending retros all folded → kit now has ZERO pending retros. Genuinely-new: n4-client-build-config (BUILD-LOOP §1.a new-module bring-up + JDK-split governance), control-model-limits (BUILD-LOOP §6 CHECK20 doc), live-commissioning-verification-gaps (NEW toolbelt/generate-wiring-map.sh + bats 11/11). Already-folded/module-scoped (marked folded, no new content): umbrelladashboard-module-creation, tree-selection-and-schema-risk-baseline, coldroompan-indefrost-resistance-recovery, dashboard-status-state-modeling (3 dashboard state/health docs applied), gradle-properties-consistency. [ev: retro n4-client-build-config-standard] [ev: retro control-model-limits-live-commissioning] [ev: retro tree-selection-and-schema-risk-baseline] [ev: retro umbrelladashboard-module-creation] [ev: retro coldroompan-indefrost-resistance-recovery] [ev: retro dashboard-status-state-modeling] [ev: retro gradle-properties-consistency] [ev: retro live-commissioning-verification-gaps]
2026-09-11 · worktree-location-guard: orient-guard.sh (PASS/FAIL/WARN; realpath+cd-P canonicalize; trailing-slash prefix compare; fail-closed on non-1 override; BUILD_N4_LEGAL_ROOT_PREFIX test seam); BUILD-LOOP §0.a + SKILL.md step 1 routing; 11/11 orient-guard.bats GREEN; 9/9 kit-links.bats GREEN (new L9); kit retro + INDEX + BUILD-STATE filed. [ev: retro module-worktree-location-retro]
2026-09-06 · Campaign 11 CLOSE v0.22.0 — shared parser (T1 PEAK/BASH_SOURCE), DRIFT advisory (T3), client-root lib (T2), guard-pins meta-check (T4); 5 retros folded; kit-only (no client jar bump); client versions carry over 2.2.0/2.1.0/2.2.0.
2026-09-06 · Campaign 10 PR3: S23 lint-silent-protection Pattern B surface — brace_depth>=2 guard (D1b) + Pass 0b ALARM_CLASSES index + adapter→pure follow "B"+class. Depth-guard baseline: no C9 pin shifted. After Pattern B: CompPan-rt 0 (CP-1 surfaced via BIAlarmSource/newOffnormalAlarm), ColdRoomPan-rt 0 (Pattern A intact), DashboardPan 0. 12/12 bats GREEN, shellcheck 0, kit-links 8/8, full 384/384 bats. Retro campaign10-silent-protection-pattern-b pending.
2026-09-06 · Campaign 10 PR2: S22 lint-ext-writable-shape per-slot action-body exemption — @NiagaraAction harvest, action→doX, section-D parser; EW10 CompPan-rt 0→1 (faultReset by NAME); RED qa/c10-ext-writable-per-slot 452df4b (EW-s22-neg2 + EW-s22-nondo); retro campaign10-ext-writable-per-slot pending
2026-09-06 · Campaign 10 PR5: S25 lint-write-path STALE advisory + --strict. lint-write-path.sh: STALE class (per-row, [concept] exemption, matrix-root-wide covered set = all @NiagaraProperty ∪ @NiagaraAction, any flag), --strict promotes STALE to exit 1 (uncovered FAIL unchanged). Real-tree ff1b659: 5 STALE rows (hoaMode ×3 :31/:32/:52, inhibit :33, freezeEnabled :36) from both CompPan-rt and ColdRoomPan-rt (root-invariant). 22/22 lint-write-path.bats GREEN; 393/393 full suite; shellcheck 0; kit-links 8/8. Retro campaign10-write-path-stale filed (pending). Branch feat/c10-write-path-stale.
2026-09-06 · Campaign 10 PR4: S24 cwd-independent structural REDs — subshell `cd "$rt"` around java call at :62; 1 bats flip (S24-cwd FAIL→GREEN); 384/384 bats; shellcheck 0; retro_pending: true.
2026-09-06 · Campaign 9 CLOSE v0.20.0 — kit PR2/PR3/PR10/PR12 + client PR#10-#15 + tunnel PR#1-#3; 5 retros folded (demand-scope, silent-protection, ext-writable-shape, doctrine-fold, close-process-meta-lessons); #89 lint-timers FP filed → S21/S22 C10.
2026-09-06 · Campaign 8 PR15: RT doctrine fold from corpus B804/B805/B808/B822/B823/B825/B826/B828. Folded: §RT-control-logic (logic.md), §history-ext B804 (logic-authoring.md), §slot-types-for-ext-writes B823 (logic-authoring.md), dashboard.md pointer. Guards: 56/56 cited, 258/258 bats, sweep-build-state exit 0, sweep-fold-audit --strict exit 0. Retro + INDEX + BUILD-STATE filed.
2026-09-05 · Campaign 8 PR10: bog-audit.sh (CHECK1-CHECK12; python3 embedded engine). PANCCADIA smoke: 17 CHECK11 FAIL, 1 CHECK2 WARN, 0 CHECK5/7 FAIL. MX60 smoke: exit 0, 109 chihuahua components. All 13 bats green, shellcheck clean, 6/6 kit-links, sweeps clean. K19 routing: BUILD-LOOP.md §6 + skill/SKILL.md. Retro + INDEX + BUILD-STATE filed. Deviations: CHECK11 FAIL not WARN (K13 RED wins), CHECK12 WARN (advisory). Fixes: compound-type sub-slot direct-parent tracking, wsAnnotation platform-slot exclusion, Java numeric literal suffix in MIN facet regex.
2026-09-06 · Campaign 10 PR1: S21 lint-timers companion-flag scope — class-FIELD + same-method guard (closes #89). Root: @NiagaraProperty( matched candidate regex, forward-walk swallowed class body. Fix: section-D method-boundary parser + brace_depth >= 2 guard + Phase 1 field-scope check + comment strip. ColdRoomPan-rt: exit 0, anyNoHardware ABSENT. CompPan-rt: exit 0, 1 PASS timer-ticket. DashboardPan-rt: exit 0. 385/385 bats, shellcheck 0, kit-links 8/8. Retro pending.
2026-09-06 · Campaign 11 PR3: T2 centralise client-tree defaults — tests/lib/client-root.bash. 10 hardcoded Leon-Guanjuato literals → 0 outside lib; load lib/client-root at file scope in all 10 bats; LD5 retargeted to main-ff1b659 (exit 0, no FAIL; defect fixed post-C9; rule pinned by LD1/LD3/LD6); RC8 unchanged (1 FAIL host-literal); c8-close SC1-smoke root-only (assertion unchanged). shellcheck 0. 420/420 bats env-unset. OBSERVED mutations: re-hardcode → C11-T2-no-hardcode RED; delete lib → C11-T2-lib-exists RED. Retro campaign11-client-root pending.
2026-09-06 · Campaign 11 PR2: T3 DRIFT advisory in lint-write-path.sh — [concept]-marked row whose slot IS in covered set prints DRIFT (concept exemption stale); true concept rows (absent slot) stay silent; --strict promotes STALE or DRIFT to exit 1; exit 0 advisory default. 4/4 write-path-drift.bats GREEN; 22/22 lint-write-path.bats unchanged; shellcheck 0; kit-links 8/8. OBSERVED mutations: drop DRIFT emit → WP-drift-neg+WP-drift-strict RED; use _mline for concept capture → WP-drift-decoy RED; remove true-concept skip → WP-drift-true-concept RED. Real tree 00e7118: 0 DRIFT, exit 0. Real tree ff1b659: 5 STALE, 0 DRIFT, exit 1 strict. Branch feat/c11-concept-drift. Retro campaign11-concept-row-drift pending.
2026-09-06 · Campaign 11 PR1: T1 shared method-boundary parser — toolbelt/lib/method-boundary.sh (MB_AWK: mb_strip + mb_parse, PEAK depth, I1-I5); lint-timers + lint-silent-protection + lint-ext-writable-shape refactored to source fragment (3 invocation forms). RED: C11-tl-oneliner + C11-sp-oneliner + G-oneliner-timers + G-oneliner-silent; GREEN: 428/428 bats; 3×3 real-tree baselines byte-identical; shellcheck 0; I2 + I3 mutation deviations reported. Branch feat/c11-shared-method-boundary. Retro campaign11-shared-method-boundary pending.
<!-- /build-state.v1 -->
