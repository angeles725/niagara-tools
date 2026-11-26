---
name: build-n4-module
description: >-
  Trigger: build/create/extend a Niagara N4 module — rt control logic, ux dashboard/servlet, wb widget/view,
  driver/network (BDeviceNetwork/BProxyExt), station service, utility/library module, theme/branding module, or cloud
  connector (ColdRoomPan/DashboardPan/chihuahua-style and beyond). Enforces the layered checklist, the Java-8 + slotomatic
  build, and a verify gate.
license: Apache-2.0
metadata:
  author: angeles725
  version: "0.9"
---

Thin launcher. The real content lives in an EXTERNAL kit (methodology, per-type guides, toolbelt, retros) — read it,
don't restate it from memory.

## Working profile — Excavador Técnico

Improve the kit and the modules as an "Excavador Técnico" (R&D engineer + system architect + deep-tech investigator +
full-stack systems engineer), each stance bound to a kit mechanism:
- **first-principles** (dismantle to physics / binary, rebuild) → why `lint-delays.sh` exists: `Math.max(x,0L)` "fixed"
  the defrost timer but Niagara rejects a delay `≤ 0`, so the bug lived — root-cause to the framework rule, not the
  symptom. `[ev: corpus B801]`
- **obsessive rigor** — do NOT stop when it works; stop only when you know exactly WHY it worked AND how to make sure it
  never fails → every check must BITE, mutation-proven on a real module (K2). `[ev: retro campaign7-plano]`
- **systems thinking** (one bit in a protection latch moves the whole industrial process) → trace each finding to its
  process consequence. `[ev: corpus B805]`
- **only what RAN** — report a build/test result, never "should work" (B815: build GREEN, run blocked — both recorded).
  `[ev: corpus B815 §815.12]`

## Resolve the kit (do this first)

Before any command with a `toolbelt/` path (a kit script run directly), load this skill: a toolbelt run without it
skips the steps below. `[ev: issue 1127]`

`KIT` = the directory holding `METHODOLOGY.md`, `BUILD-LOOP.md`, `types/`, `toolbelt/`. Resolve it with the fail-closed
resolver, never by hand:
`$HOME/modulos_niagara_n4/niagara-tools-worktrees/kit-main/scripts/kit-resolve.sh` (no kit-main yet: the same
`scripts/kit-resolve.sh` in `$HOME/modulos_niagara_n4/niagara-tools`).
- It tries `$BUILD_N4_KIT`, then the `kit-main` worktree (`$BUILD_N4_KIT_MAIN`), then the default checkout
  `$HOME/modulos_niagara_n4/niagara-tools/build-n4-module-kit`, and accepts the first one only if it is at `origin/main`
  and clean.
- Exit 0: its first line `KIT=<path> SHA=<sha> (origin/main, clean)` is your first output; use that `KIT`.
- Then materialize the generated state on that resolved checkout: `$KIT/toolbelt/kit-state.sh --write`. The generated
  BUILD-STATE.md, retros/INDEX.md and toolbelt/INDEX.md are not tracked.
- A clean kit-main strictly behind origin/main is fast-forwarded automatically (stderr
  `kit-resolve: refreshed <old>→<new>`, then exit 0); `--refresh` is no longer needed.
- Exit 2 (stale `$BUILD_N4_KIT` or default checkout, dirty, ahead of or diverged from origin/main, origin unreachable):
  do not build. Run the command it prints, or `--offline-ok` when offline is known and accepted (never auto-refreshes).
  Exit 3: no kit or a bad `$BUILD_N4_KIT`; ask. `[ev: retro r10-stale-kit-resolution Δ1]`

Then read `$KIT/METHODOLOGY.md` + `$KIT/BUILD-LOOP.md` + the type guide, and run the loop.

## Hard Rules (non-negotiable — full detail in the kit)

- **corpus-nav FIRST** (`cnav find "<topic>"`) — ~90% is already documented. For WHAT TO READ by layer/priority, use
  **`$KIT/corpus-index.md`** (the curated map of the corpus for module builders; P0 blocks before building; its title
  states the block range it reaches and this skill hard-codes none) `[ev: retro doc-coverage-audit Δ7]`; for our-module
  work start from B760's punch-list.
- **Read the REAL source** (ColdRoomPan/chihuahua), never from memory.
- **Preview before compiling** (`~/.local/lib/niagara-go/bin/dashboard-preview`, open `/hmi` at 1280×800).
- **A dashboard ships its own login** (a `BLoginTemplate` in its `-rt`, set as `WebService.loginTemplate` at
  station start, never over another custom one): `$KIT/types/dashboard.md` § The module ships its own login.
- **Verify gate = `$KIT/toolbelt/verify-module.sh`** (major 52 on every class outside `META-INF/versions/`, a
  `META-INF/<ALIAS>.SF` signature with its key file, module.xml types resolve; `--target-version`, `--stored`, `--src`
  opt-in). A jar that has not passed it does not go to a station. `--src` sub-checks: `typecount`, `facets` (raw-number
  MIN/MAX), `facets-req` (OPERATOR numeric without facets key; setpoint/count-like without UNITS/PRECISION — WARN),
  `ord-literal` (hardcoded station:|slot:/ string — WARN) [ev: retro campaign8-facets-lint]. **Build (WSL) =
  `$KIT/toolbelt/build.sh`** (Java 8 + clean + slotomatic for every profile with sources + jar, then runs the gate).
  **Deploy = `niagara-tools/scripts/ng-deploy.sh`** (backup → build → prechecks → type-count verify → verify-module gate
  → copy; a failing check copies nothing). Modes A/C: annotation-change guard + opt-in `--with-slotomatic` (-rt, plus
  -ux in mode A when -ux is annotated); mode B runs no slotomatic. Prechecks before the copy: version bump against the
  deployed jar, every mode (exit 51 same version, 52 unverifiable, 54 downgrade), and schema-risk SAFE on
  `SCHEMA_BASELINE_DIR`, modes A/C (exit 12 with no baseline unless `--first-deploy`, 53 on OUTAGE or LOSSY without
  `--accept-lossy`). The verify-module gate runs on modes A/C before the copy (exit 50 failed or missing; `--no-gate`
  skips it); mode B has none — run `verify-module.sh` on `build/libs` before a mode-B deploy. Exit 40: the copy failed
  and is all-or-nothing, so read the ERROR line: either the station holds the jars it held before (the copy is undone
  from the jars it moved aside, backup or not), or it names the jar that is NEW or MISSING and the `<jar>.ng-deploy-old`
  file to rename back before any restart. A `.ng-deploy-old` file a deploy that did not finish left in the station is
  never deleted by the tool: the next deploy is exit 40 with nothing changed until you rename it over its jar (go back)
  or delete it (keep what is there); a SIGINT or SIGTERM during the copy undoes it and exits 130 or 143
  `[ev: retro go-ng-deploy-port Δ2]` `[ev: retro go-ng-deploy-port Δ9]`. Exit 21: the backup exists but an older one
  could not be purged, nothing was copied; the backup and its purge run just before the copy, so a run that stops
  earlier leaves `_backups/` as it was (coordination-stretch retro D4). Exit 41: the jars are copied but
  `.last-deploy-sha` was not recorded, so restart the station and then fix the record; it is not a failed deploy
  `[ev: retro go-ng-deploy-port Δ3]` `[ev: retro go-ng-deploy-port Δ4]`; a deploy from a directory that is not a git
  repository has no commit to record and ends 0 with a NOTICE (coordination-stretch retro D9); an exit 40 or 10 with
  `Device or resource busy`, or `Permission denied` in a modules directory this user may write (a Windows jar held
  open), adds `a running station or Workbench holds its module jars open; stop it and run again`; in one this user may
  not write the line names its permissions instead (coordination-stretch retro D1); a `.last-deploy-sha` that is a
  directory or cannot be written is exit 10 before the copy, with nothing copied `[ev: retro go-ng-deploy-port Δ10]`.
  Exit 15 in strict mode also covers a `.last-deploy-sha` that is malformed or names a commit git does not know (a
  NOTICE otherwise) `[ev: retro go-ng-deploy-port Δ11]`. A flag wins over the same option variable in the env file, and
  `NO_GATE=1` / `NO_BACKUP=1` in the env file are exit 10 unless `--no-gate` / `--no-backup` is given; `MODE=B` (a mode
  with no gate) and `VERIFY_MODULE_BIN` in the env file are refused the same way (exit 10 with file and line) unless
  `--mode` or `--no-gate` is given `[ev: retro go-ng-deploy-port Δ1]` `[ev: retro go-ng-deploy-port Δ12]`. Never
  `gradle :jar` with the default JDK.
- **`@NiagaraProperty` edits** go in the annotation AND the generated region AND the imports; prove with slotomatic.
- **Ask, don't assume — Behavior decisions gate:** before the first source write of any rt control, protection,
  automatic action or operator-facing UI change, every behavior the user has not stated becomes a question with options
  and an answer recorded in the feature doc's `## Behavior decisions` table; nothing on that list is coded unanswered
  (`$KIT/BUILD-LOOP.md` §1). `[ev: retro behavior-decisions-ask-dont-assume Δ1]`
- **Classify before writing:** pick the blast-radius tier P0-P3 (`$KIT/BUILD-LOOP.md` §0.c) before the first write; the
  non-skippable floor (§5) applies to every tier. `[ev: retro change-tier-time-budgets Δ1]`
- **Kit tooling (when you change the kit itself)** — procedure in `$KIT/BUILD-LOOP.md` §5 "Kit tooling doctrine":
  - a new tool or check is written in Go in `niagara-tools/go/` (`go/cmd/<tool>` + `go/internal/<pkg>` + a thin `.sh`
    wrapper over `go/go-tool.sh`); bash gets small fixes only; a check not ported yet is an owed `Go lint` item in a
    `ledger.d/` fragment. `[ev: retro r8-t22-retro-deltas Δ1]`
  - port every tool to Go, first what PARSES structured input (Java, XML, jars, ledgers) or has had parsing bugs;
    orchestration glue is ported too (`build.sh` is a launcher over `go/cmd/build-module`, `preflight.sh` over
    `go/cmd/preflight`) and a bash script stays only as a launcher; a port proves byte-identical stdout/stderr/exit on
    real `Cliente/` trees (reported as counts only, the repo is public), records before/after timing, keeps its bats.
    `[ev: retro r8-t22-retro-deltas Δ8]` `[ev: retro r10-preflight-go Δ1]`
  - Go is layered: `internal/javasrc` (scanner) → `internal/lint` (Rule, registry, Finding, one formatter, one walker) →
    `internal/lint/rules/<lint>` → thin `cmd/`; a new lint = one rule file + registration, never a copied
    walker/formatter/stripper; bash reuses `toolbelt/lib/*`. `[ev: retro r8-t22-retro-deltas Δ9]`
  - a bats assertion is one command per line: errexit ignores every element of an `&&` list but the last and a bare
    `! cmd`, so neither fails its test mid-body; `$KIT/toolbelt/lint-bats-errexit.sh` FAILs both
    (`tests/lint-bats-errexit.bats` BEX-kit runs it on the kit's `tests/`). `[ev: retro r10-bats-list-hazard Δ1]`
  - a lint never forks a process per input line: a Go rule on `go/internal/lint` (the shared runner reads each file
    once), or one awk per file in a bash script that must stay; `tests/lint-fork-per-line.bats` fails a new one.
    `[ev: retro r8-t22-retro-deltas Δ2]`
  - Java text is scanned literal-aware and scope-aware through the Go scanner `go/internal/javasrc` (`Strip`/`Methods`),
    never a private stripper. `[ev: retro r8-t22-retro-deltas Δ3]`
  - a review candidate stays under ~400 changed lines per commit, reviewed with `--base-ref <exact parent SHA>`; after
    review merge origin/main in, never rebase. `[ev: retro r8-t22-retro-deltas Δ6]`

## Decision Gates — pick the module type

| Building | Read | Scaffold `--type` |
|----------|------|------|
| Pure logic (rt control) | `$KIT/types/logic.md` | `logic` |
| Framework extension (service, ORD scheme, point ext, analytics node, job, watchdog, provider) | `$KIT/types/logic-authoring.md` | `logic` (then replace the component's base class) |
| Station service (a `BAbstractService`: registration, lookup, lifecycle order, licence behaviour, tests) | `$KIT/types/service.md` `[ev: retro doc-coverage-audit Δ17]` | `logic` (then replace the component's base class) |
| Driver / device network (network, device, proxy ext, comm stack, discovery, licensing) | `$KIT/types/driver-authoring.md` (§ 1.8 picks the framework, § 12 testing and the gate's `driver-*` rows) `[ev: retro r10-dr1-driver-doctrine Δ1]` `[ev: retro r10-dr4-verify-driver Δ1]` | `driver` |
| Dashboard (facade + servlet + SPA) | `$KIT/types/dashboard.md` (+ `$KIT/types/frontend-standard.md` for the SPA side) | `dashboard` |
| Logic + dashboard | both | `logic` + `dashboard` (two modules) |
| Utility library (a wrapped OSS jar, or helper types others depend on; no palette) | `$KIT/types/utility-lib.md` (§ 6: `type: library` in the module map) | `library` (Shape A: no Java, `<types/>`; Shape B adds its types by hand) |
| Theme (CSS, NSS, fonts, icons; zero Java) | `$KIT/types/theme.md` | no scaffold (a theme module's name starts with `theme`, which `--type` names never do; declare `type: theme`, `types/theme.md` § 7) |
| Cloud / IoT connector (transport, authenticator, channel, MQTT, RDBMS export) | `$KIT/types/cloud-connector.md` | no scaffold |
| Workbench widget / PX (wb) | `$KIT/types/wb-widgets.md` | no scaffold (only `driver` emits a `-wb` part) |
| Hx view, Px graphic or Px widget (hx vs Px vs bajaux, binding ORDs and their audit, Hx theming, events, permissions) | `$KIT/types/hx-px.md` | no scaffold |
| Reading data from module code (BQL, NEQL, history, alarm, lease) | `$KIT/types/queries.md` | no scaffold |
| Data from a browser or off-station client (BOX limits, gzip, file cache, audio, oBIX Watch, `baja.rpc`) | `$KIT/types/data-access.md` | no scaffold |
| New module scaffold / source layout audit | `$KIT/types/structure.md` | `logic` + `dashboard` + `driver` + `library` (the one of the module's row) |
| Lexicons, translation modules, runtime and browser text | `$KIT/types/lexicon.md` | no scaffold |
| Station diagnostics (engine stats, log levels, console messages, thread dumps, remote debugger, launcher flags) | `$KIT/types/diagnostics.md` | no scaffold |
| JSON export or a jsonToolkit extension | `$KIT/types/json-toolkit.md` | no scaffold |
| Runtime profiles, Software Manager install, NRE memory on a JACE | `$KIT/types/platform.md` | no scaffold |
| N4 capabilities our modules do not use yet | `$KIT/types/horizons.md` | no scaffold |
| A shipped module's dependencies, reverse deps, java-permissions; a type's class and agents (`nh module`, `nh type`) | `$KIT/types/module-wiring.md` §7 | no scaffold |

The Scaffold column is the `toolbelt/scaffold-module.sh --type` that starts a new module of that row (`BUILD-LOOP.md` §
1.a); "no scaffold" means the kit has no skeleton for it. `[ev: retro build-kit-bench-session Δ17]`

**wb builders**: `types/wb-widgets.md` was a seed and is now exemplar-backed for the -wb model seam, but the guide is
still thin — start from corpus B751–B753 and B762/B780 before the type guide. The Predicate-injection seam (DWB1) is the
critical entry point.

`$KIT/METHODOLOGY.md` (common checklist) applies to ALL.

**Design checklist item — automatic protective actions:** for any auto-lockout, auto-trip or auto-mode-change, the
design names the latch slot, its operator-owned clear condition and its HMI rendering, or explicitly waives them with a
reason (item 6 of the design-shard checklist, `$KIT/ORCHESTRATION.md` §3.b).
`[ev: retro comppan-auto-lock-indicator Δ1]`

## Execution Steps

1. **Orient guard (FIRST):** run `$KIT/toolbelt/orient-guard.sh <module-root>` — asserts that the module root is under
   the sole legal build prefix `$HOME/modulos_niagara_n4/Cliente/`; exit 1 = unauthorized location, stop; exit 0 =
   proceed. Set `BUILD_N4_CLIENTE_OVERRIDE=1` only for an explicitly authorized exceptional path.
   `[ev: retro module-worktree-location-retro]`
1a. Orient from `$KIT/toolbelt/kit-state.sh build-state` (BUILD-LOOP §0.a; it prints the generated `BUILD-STATE.md`,
which is not committed — never trust a local copy you did not just regenerate) — state the module's one-line leave-off;
then read `METHODOLOGY.md` + `BUILD-LOOP.md` + the type guide; skim `$KIT/corpus-index.md` for the P0 blocks of the
layer you are building.
1b. Explore shard: when the change touches > 3 files, open a `sdd-explore` shard (sonnet) to map the affected symbol set
before writing; audit-first prevents fold errors on renamed or moved files. `[ev: METHODOLOGY.md K6]`
1c. Design shard: when the change introduces a schema annotation, a new slot, or a new façade type, open a `sdd-design`
shard (opus) before writing; these are high blast-radius changes that need architectural scoping (`gentle-sdd-ff` path).
`[ev: ORCHESTRATION.md §2; live read of the agent definitions model: frontmatter, 2026-10-24]` **HARD gate for a new
control state machine or a new facet/unit:** produce the design-shard checklist of `$KIT/ORCHESTRATION.md` §3.b in the
feature doc (states + guards, invalid-sensor behavior, min-on/min-off interplay, config bounds, boot-critical
expressions, automatic-action latch) — or record `Design shard: waived (<reason>)`; a silent skip fails the step.
`[ev: retro panccadia-commissioning-lessons Δ7]`
1d. Behavior questions (interaction format): ask with the native multiple-choice question UI when it is available — at
most 4 related questions per round, each with 2-4 options, the recommended one first and labeled "(Recomendado)" (or the
equivalent in the user's language), each option's field consequence in its description, free-text answer always allowed.
Otherwise use a numbered plain-text list with the same content. Write the question in the user's language and in field
terms (valves, compressors, pressure, rooms), not slot names alone. `[ev: retro behavior-decisions-ask-dont-assume Δ6]`
1e. **Module map first:** read the module's `MODULE-MAP.md` (module root, `$KIT/types/structure.md` § Module map) before
opening a source file; find the concern the change belongs to and edit only the files of that concern's row (its `Files`
and `Tests`); a change that spans concerns names each one. A new file adds its row entry in the SAME change, and a new
concern adds a row with at least one test. No map: add one (`modularity: legacy` + `legacy-reason:` for an existing
module) before the first edit, or record why not in the feature doc. `[ev: retro modularity-module-map Δ4]` Tools:
`$KIT/toolbelt/module-map.sh where <file|concern> --root <module-root>` names a file's concern, its files and tests;
`module-map.sh tests <concern>` runs only that concern's pure tests; `module-map.sh add-concern <Name>` (or
`scaffold-module.sh add-concern`) writes a new concern's folder, pure class, test and map row.
Before you report done, run `$KIT/toolbelt/lint-module-map.sh --base <base-ref> <module-root>`: MM8 names every
file your change added (a test, a vendored file) that no row lists (WARN in `legacy`, FAIL in `strict`); add its row.
`[ev: retro modularity-structure Δ1]` `[ev: retro modularity-structure Δ4]` Restructuring an existing module toward
strict happens only on request, in the order of `$KIT/types/structure.md` § Legacy → strict migration.
`[ev: retro modularity-structure Δ5]`
2. corpus-nav the topic; read the exemplar source. A user-supplied reference (course lab, vendor sample, their own code)
   comes first: `BUILD-LOOP.md` § 0, reference-first. `[ev: retro n4-course-as-exemplar-source Δ2]`
3. Build the layers; preview the UI.
4. Build with `toolbelt/build.sh` — ONE command runs the full automatic chain: preflight (env checks) → gradle (Java 8 +
   clean + slotomatic + jar) → verify gate → report-module punch-list. `niagara_home` must be a mirror
   (`toolbelt/mirror-niagara-home.sh`): a real install exits 10 unless you pass `--install-into-home`, because gradle's
   `:jar` replaces its `modules/` jars. On that stop (or no `niagara_home`) it lists the `~/niagara-mirror-*`
   it finds, those made from the given installation first, with a `try:` line: reuse one, never make a second.
   Use `--no-preflight`/`--no-report` for quick inner rebuild loops; pass `--feature-doc <file>` to put the feature
   doc's ADVISORY rows in the punch-list (report-module also runs `lint-vendored` on a `-ux` part's `src/rc`); pass
   `--ui-profile <ui_profile>` (+ `--legacy`) for a `-ux` module so the punch-list uses its browser floor. Before the
   first build, `scripts/check-client-source.sh` confirms the checkout is the module's declared `source_of_truth`.
   Confirm: bytecode major 52, signed jars — signed by the module's own alias: with `niagaraSigningAlias` configured
   `build.sh` passes `verify-module.sh --expected-signer` and another signer FAILs; a
   `WARN — no niagaraSigningAlias is configured` line means any signature (the development key included) was accepted.
   The envelope's optional `expected_signer: <alias>` records it once: `build.sh` uses it when no alias is configured
   and refuses (exit 10) a configured alias that differs. Its `niagara_target: <version>` and `plugin_version:
   <version>` (in `<module-root>/BUILD-STATE.md`) tie the module to its station: `build.sh` refuses a
   `niagara_home` or `--target-version` of another major.minor, and `ng-deploy.sh` a station of one (exit 55).
   Lost track of which signing certificates exist, where each lives, when each expires or which Niagara version a
   built jar targets? `toolbelt/signing-inventory.sh` (next to `signing-doctor.sh`) is the read-only inventory:
   `[--home <niagara_home>]... [--profile <profile.xml>]... [--module <root>]`. Per profile and keystore alias it
   prints the subject, issuer, days left (WARN under 60, FAIL expired), code-signing or CA, and the file; a store it
   cannot open is `locked`, path only; it never prints a password or key. With `--module` it adds the recorded
   `expected_signer` and `niagara_target`, each built jar's signer and compiled-for Niagara version, and FAILs when
   the recorded alias is in no listed store, a jar carries another signer, or a jar was compiled for another
   major.minor than the recorded target.
   On WSL, sign with the module's alias through the explicit profile: `--gradle-prop niagaraSigningAlias=<alias>
   --gradle-prop niagaraSigningProfile=$HOME/.config/n4-signing/<alias>.signing.xml`. Never report signing as
   unavailable (or as needing Windows) before running it: the only valid "unavailable" is a missing profile file, and
   the report quotes that path. `[ev: issue #917]` (retro R11 ΔB1)
   In a fresh `git worktree add` checkout run `chmod +x gradlew` first: a client repo that stores `gradlew` as 100644
   (`core.fileMode=false`) checks it out non-executable, and `build.sh` fails fast with
   `gradlew exists but is not executable … fix: chmod +x <path>` — run that command, or store the wrapper as 100755 in
   the client repo. `[ev: external retro niagara-research/retros/2026-10-06-panccadia-comppan-hmi-support.md Δ1, Δ4]`
5. Pre-gate checks before `verify-module.sh`: the authoritative lint list (usage, exit contract, evidence, and whether
   `report-module.sh` runs it) is the generated lint index, printed by `$KIT/toolbelt/kit-state.sh lint-index` (not
   committed) — run every lint whose row matches the profiles you touched (an Auto `no` lint by hand); a lint header is
   its row (`$KIT/toolbelt/gen-lint-index.sh --check` fails on a broken one); never enumerate lints in this step
   `[ev: retro kit-meta-hygiene-2026-10-01 Δ3]`. After a Niagara upgrade,
   `toolbelt/check-facets-card.sh $KIT/types/logic-authoring.md <BFacets.java>` diffs the kit's BFacets key card against
   the new source (exit 1 = the card needs regenerating) [ev: retro devguide415-runtime-api-corrections Δ8]. The lint
   rows carry the evidence that used to be listed here [ev: retro campaign8-lint-delays]
   [ev: retro campaign10-lint-timers-scope] [ev: retro campaign8-wb-audit] [ev: retro campaign9-demand-scope]
   [ev: retro campaign8-structure] [ev: retro campaign8-write-path] [ev: retro campaign10-write-path-stale]
   [ev: retro campaign11-concept-row-drift] [ev: retro campaign9-silent-protection]
   [ev: retro campaign9-ext-writable-shape] [ev: retro campaign10-ext-writable-per-slot]
   [ev: retro live-commissioning-verification-gaps]. Non-lint pre-gate tools (BUILD-LOOP §5): run
   `toolbelt/slot-coverage.sh [--strict] <module-include.xml> <module.lexicon>` (type-set lexicon coverage; empty or
   missing lexicon with declared types → exit 1) and `toolbelt/schema-risk.sh <before-dir> <after-dir>` before deploy
   (two-snapshot slot diff; verdict SAFE/LOSSY/OUTAGE — OUTAGE means a deployed change breaks saved data)
   [ev: retro campaign7-tool-integration]; a client repository carries the split-package gate
   (`toolbelt/split-package-check.sh --strict`, one Java package per module) as a pre-push hook — install it once per
   clone with `niagara-tools/scripts/install-client-hooks.sh <client-repo>` and commit `.githooks/pre-push`; it blocks a
   split package and any check it cannot run (issue #142) [ev: retro module-hardening-failure-modes-deltas Δ12]; then
   run the verify gate (`toolbelt/verify-module.sh`) with `--plano <index.html|jar>` when available
   [ev: retro campaign7-plano]; the aggregated punch-list (`toolbelt/report-module.sh`) runs automatically at the end of
   `build.sh` — exit 1 = FAILs block hand-off [ev: retro campaign7-report-module]; `$KIT/toolbelt/triage-console.sh`
   (own-module exception triage over station console logs; three attribution channels C1/C2/C3; exit 0 clean / 1 rows /
   3 usage; run after reload — see step 6) [ev: retro campaign8-triage-console]. Field commissioning of DIP-addressed
   MS/TP controllers: plan MAC, switches and device IDs with `toolbelt/mstp-plan.sh` (conflict checks;
   `types/driver-authoring.md` §11.2) [ev: retro mstp-plan Δ1].
5a. Deploy with `niagara-tools/scripts/ng-deploy.sh` from the gradle root (`BUILD-LOOP.md` § 6; `--help` lists options,
env variables and exit codes). `NIAGARA_HOME` must be a mirror, as for `build.sh`: a real installation is exit 10 before
gradle runs unless `--install-into-home`, because gradle's `:jar` step replaces the jars of its `modules/` before any
check; when a check fails the deploy copies nothing and puts `<NIAGARA_HOME>/modules` back as it was before the build.
`[ev: retro build-kit-bench-session Δ1]` A mirror build gets the same `user.home` isolation and `--no-daemon` as
`build.sh` (one rule, `go/internal/gradleenv`), and `--gradle-prop k=v` is forwarded as `-Pk=v` as `build.sh` forwards
it (`niagaraSigningAlias=`, `niagaraSigningProfile=`) (coordination-stretch retro D3). Its gate is told the signer
`build.sh` expects (`niagaraSigningAlias`, else the signer of the jars deployed in `<NIAGARA_HOME>/modules`): a jar
signed with another key is exit 50, and with no signer known the deploy prints
`WARN no niagaraSigningAlias is configured …`. `[ev: retro build-kit-bench-session Δ2]` A deploy can be repeated with no
source change: its version-bump precheck ignores the build time and host of `module.xml`, the manifest digests and the
signature files; any other difference under the same vendorVersion is exit 51, an unreadable jar exit 52.
`[ev: retro build-kit-bench-session Δ3]` Backups go to `_backups/` under the directory the deploy is run from; `--help`
says how to restore one. `[ev: retro build-kit-bench-session Δ12]` A deploy that went through ends with `done:` /
`signer:` / `next:` / `undo:` lines: it restarts nothing and does not look at whether a station is running; first deploy
of a new module and removal: `BUILD-LOOP.md` § 6. `[ev: retro build-kit-bench-session Δ13]` One deploy at a time per
modules directory: a run holds `.ng-deploy.lock` there, a second run is exit 10, and a lock a killed run left is removed
by hand after `ps -p <pid>` (`BUILD-LOOP.md` § 6). `[ev: retro build-kit-bench-session Δ22]`
6. After a reload (rt or full station restart), triage the console:
   `toolbelt/triage-console.sh --package com.vendor <station-dir>/console*.txt` surfaces own-module exceptions and
   load-time failures invisible to C1/C2 alone (three channels: own frame, own logger tag, [sys]/[sys.xml] load-fail
   shape); exit 1 = rows found — investigate before calling the deploy clean. [ev: retro campaign8-triage-console]
7. Retro + close (HARD gate): run `toolbelt/new-retro.sh <module|kit> <slug>` and fill the stub (What happened /
   Evidence / Proposed kit deltas / Lessons); a kit-check/doctrine defect also opens
   `toolbelt/kit-ticket.sh "<one line>"` (gh absent: SKIP + file, exit 0; gh present but no host/auth/repo context:
   LOCAL + file in `retros/tickets/`, exit 4; other gh failure: file kept, exit 3 — for 3/4 file it by hand, it is not
   on GitHub). Edit `$KIT/BUILD-STATE.src.md` (envelope + `retro_pending`; kit work adds a `ledger.d/` fragment; never
   commit or hand-edit the generated `BUILD-STATE.md`, the generated `retros/INDEX.md` or the generated
   `toolbelt/INDEX.md`, which `toolbelt/kit-state.sh` writes) — or declare `Retro: none (trivial: <reason>)`. Before a
   kit PR run `niagara-tools/scripts/ci-local.sh -j 3` (the exact CI replay with the pinned shellcheck 0.10.0; `-j 3` is
   also the cap on a host other sessions share; it refuses to start, exit 4, while a cgroup is above 50% of `pids.max`).
   On a shared host: `$KIT/toolbelt/host-budget.sh` reads pids, threads, GOCACHE and disk in one run (exit 1 on a FAIL
   row) `[ev: retro r8-t47-host-budget Δ1]`; wait while `/sys/fs/cgroup/init.scope/pids.current` is above 20,000 before
   any heavy suite, at most two writers per session run suites at once, `go clean -cache` + `df -h` at round close, poll
   `gh pr checks` instead of `--watch`, absolute paths before `cd`, no helpers in `/tmp` (BUILD-LOOP §7).
   `[ev: retro r8-t22-retro-deltas Δ4]` `[ev: retro r8-t22-retro-deltas Δ5]` `[ev: retro r8-t22-retro-deltas Δ7]`
   `[ev: retro r8-t41-host-safety Δ1]` `[ev: retro r8-t41-host-safety Δ2]` `[ev: retro r8-t41-host-safety Δ3]`
   `[ev: retro r8-t41-host-safety Δ4]` `[ev: retro r8-t41-host-safety Δ5]` `[ev: retro r8-t41-host-safety Δ6]`
   `sweep-build-state.sh --age` at orient shows outstanding retro debt. [ev: retro campaign8-retro-loop]

## Steps no tool checks (the reviewer does)

Eight steps of this skill are judged by the reviewer, not by a gate: nothing under `toolbelt/`, `scripts/` or `go/`
reads the feature document, the console of a session, or the operator's backup. Each is marked here as UNCHECKED so
a green gate is never read as proof of it (checked 2026-11-23 by searching those trees for each step's wording;
the 2026-11-15 audit listed them). `[ev: retro doc-coverage-audit Δ21]` Three of them (the tier line, the Behavior
decisions heading, the Checks skipped heading) are ADVISORY rows of `report-module.sh --feature-doc <file>`: they
never change the verdict, and they are only there when the option is passed (issue #1127), to `build.sh` too, which
forwards it (issue #1145).

| Step | Where it is required | Check today |
|---|---|---|
| Behavior decisions table, every unstated behavior answered before the first write | Hard Rules, `BUILD-LOOP.md` § 1 | ADVISORY row of `report-module.sh --feature-doc` if heading or content is missing; the reviewer reads the answers |
| Design-shard checklist, or its waiver with a reason | step 1c, `ORCHESTRATION.md` § 3.b | UNCHECKED: the reviewer reads the design shard or the waiver |
| `## Checks skipped` section for every check not run | `BUILD-LOOP.md` § 5, `METHODOLOGY.md` | ADVISORY row of `report-module.sh --feature-doc` if the heading is missing; a SKIPped floor check reads INCOMPLETE |
| Priority tier P0-P3 recorded before the first write | Hard Rules, `BUILD-LOOP.md` § 0.c | ADVISORY row of `report-module.sh --feature-doc` if no tier line is found; the reviewer reads a P0 commit body |
| Preview before compiling (UI types) | Hard Rules, step 3 | UNCHECKED: `dashboard-preview` is an external tool, no gate runs it |
| `cnav` / `corpus-index.md` search before deriving anything | Hard Rules, step 2 | UNCHECKED: the feature document names the blocks read; nothing verifies the search ran |
| Console triage after a reload | step 6 | UNCHECKED: `triage-console.sh` exists (`go/internal/triageconsole`) but no gate requires that it ran |
| Full station backup before a runtime install | `BUILD-LOOP.md` § 6 | UNCHECKED: `ng-deploy.sh` backs up the module's own jars (lightweight); the platform backup is an operator step |

A step moves out of this table in the change that gives it a check, with its test.

## Recipe — add a read-only indicator (rt → dashboard)

The P1 path (`$KIT/BUILD-LOOP.md` §0.c), mapped once so a writer does not re-trace it. First find the SIBLING indicator
you are mirroring (an existing link-in status slot that already reaches the dashboard) and copy its flags and hops;
per-client file paths live in the client repo's feature docs, the kit holds only the generic hops:
1. Pure model field + a focused pure test of the latch, run with the feature ENABLED (and the near-miss case).
2. rt adapter slot (flags as the sibling: `SUMMARY`; READONLY only when the owning component itself writes it).
3. Facade/panel link-in slot — same flags as the working sibling; NEVER `READONLY` on a link target
   (`lint-link-target-flags.sh` LTF1).
4. ux reader field (fail-visible when unlinked: `{v,st}` with a non-ok status, never an ok/false default).
5. rc render + CSS class.
6. Version bumps in every touched group.
7. Generated wiring-map row (`$KIT/toolbelt/generate-wiring-map.sh`) with the Workbench display name, for the
   commissioning link.
`[ev: retro change-tier-time-budgets Δ2]`

## Output Contract

Report: kit path, type chosen, layers built, preview + build outcome (bytecode 52 + signed), verify-module.sh result
(PASS/FAIL per check), proposed kit deltas, and the retro line — `retro: <path> (N deltas, review-status: pending)` or
`retro: none (trivial: <reason>)`.

## References

- Orchestration: `$KIT/ORCHESTRATION.md` (model table, delegation triggers, escalation gate, per-run retro/ticket loop).
  `[ev: retro campaign8-orchestration]`
- `$KIT/METHODOLOGY.md`, `$KIT/BUILD-LOOP.md`, `$KIT/build-verify.md`, `$KIT/types/logic.md`,
  `$KIT/types/logic-authoring.md`, `$KIT/types/dashboard.md`, `$KIT/types/wb-widgets.md`, `$KIT/SOURCES.md`; the skill
  installer is `scripts/install-skill.sh` of the repository (beside the kit, not in it).
- Toolbelt: `$KIT/toolbelt/build.sh`, `$KIT/toolbelt/verify-module.sh` (+ coverage, + --plano
  <ux-profile>/src/rc/index.html), `$KIT/toolbelt/run-pure-test.sh`, `$KIT/toolbelt/mirror-niagara-home.sh`,
  `$KIT/toolbelt/stored-repack.sh`, `$KIT/toolbelt/run-niagara-test.sh` (`verdict [<log>|-]`: PASS only when every
  `Total tests run: N` has N > 0 and 0 failures; exit 0 PASS / 1 FAIL / 2 usage / 3 unreadable log)
  [ev: retro r10-run-niagara-test-go], `$KIT/toolbelt/preflight.sh`, `$KIT/toolbelt/slot-coverage.sh` (type-set +
  per-slot modes; per-slot: `per-slot <xml> <lex> <src>`, exit 1 = MISSING) [ev: retro campaign8-slot-per-slot],
  `go/internal/javasrc` (the shared method-boundary scanner of the Go lints, `Strip` and `Methods`; it replaced the awk
  library toolbelt/lib/method-boundary.sh, C11 T1; fragment rule: never give a lint a parser of its own — edit the
  shared scanner) [ev: retro campaign11-shared-method-boundary], `$KIT/toolbelt/lint-timers.sh`
  (timer-ticket/discarded-ticket/companion-flag — class-FIELD + same-method scope; PEAK-depth method bounds from
  go/internal/javasrc; fragment rule: never overwrite) [ev: retro campaign10-lint-timers-scope]
  [ev: retro campaign11-shared-method-boundary], `$KIT/toolbelt/lint-delays.sh` (Clock.schedule delay-floor lint; exit 0
  clean / 1 any FAIL / 3 usage; run in step 5 before lint-timers) [ev: retro campaign8-lint-delays],
  `$KIT/toolbelt/lint-wb-threading.sh` (Swing-thread traversal + agent-breadth WARN over -wb src; exit 0 clean / 1 under
  --strict / 3 usage) [ev: retro campaign8-wb-audit], `$KIT/toolbelt/lint-demand-scope.sh` (demand-in-scope lint; WARN
  rows for a control-decision method reading a process variable with no demand-shaped input in scope; exit 0 clean or
  WARN-only / 1 under --strict / 3 usage or no sources) [ev: retro campaign9-demand-scope],
  `$KIT/toolbelt/lint-silent-protection.sh` (silent-protection-trip lint; WARN rows for unalarmed trips; exit 0 clean or
  WARN-only / 1 under --strict / 3 usage; run on any -rt profile with Java sources; PEAK-depth method bounds from
  go/internal/javasrc; fragment rule: never overwrite) [ev: retro campaign9-silent-protection]
  [ev: retro campaign11-shared-method-boundary], `$KIT/toolbelt/lint-ext-writable-shape.sh` (ext-writable-shape lint;
  WARN rows for OPERATOR complex (BStatusNumeric/BStatusBoolean/BStatusEnum) properties with no writing action; exit 0
  clean or WARN-only / 1 under --strict / 3 usage; PEAK-depth method bounds from go/internal/javasrc; fragment rule:
  never overwrite) [ev: retro campaign9-ext-writable-shape] [ev: retro campaign11-shared-method-boundary]; S22 (C10):
  per-slot body-scan, EW10 0→1 faultReset [ev: retro campaign10-ext-writable-per-slot],
  `$KIT/toolbelt/sweep-build-state.sh` (+ --age), `$KIT/toolbelt/sweep-fold-audit.sh`,
  `$KIT/toolbelt/scaffold-module.sh` (a new module skeleton,
  `<ModuleName> <out-dir> --vendor <v> [--type logic|dashboard|driver]`, from `fixtures/`; run once, before the first
  build, then `BUILD-LOOP.md` § 1.a) [ev: retro build-kit-bench-session Δ17], `$KIT/toolbelt/schema-risk.sh`
  (two-snapshot slot diff; verdict=SAFE/LOSSY/OUTAGE; exits 0/1/2/3/4) [ev: retro campaign7-tool-integration],
  `$KIT/toolbelt/report-module.sh` (aggregated conformance report per module; `--console-dir <dir>` for triage-console
  integration; exit 0 CLEAN / 1 FAILs / 3 env; run last in step 5 before hand-off) [ev: retro campaign7-report-module]
  [ev: retro campaign8-report-integration], `$KIT/toolbelt/rc-scan.sh` (browser-resource lint over rc/ assets; exit 0
  clean / 1 any FAIL / 3 usage; run before hand-off on any -ux profile with rc/) [ev: retro campaign8-rc-scan],
  `$KIT/toolbelt/bog-audit.sh` (station config.bog auditor; CHECK1-CHECK12; exit 0 clean / 1 any FAIL / 3 usage, not a
  file, no file.xml in a zip, a malformed bog or no component of the named modules; a Go command since round 10, no
  python3; add --source-dir for source-coupled checks CHECK2-7; proxy-link-safety CHECK11 is FAIL)
  [ev: retro campaign8-bog-audit], `$KIT/toolbelt/station-snapshot.sh` (audit-surface snapshot: copies config.bog +
  console*.txt, writes manifest.json with sha256; exit 0 ok / 1 copy fail / 3 usage; run before/after deploy; source
  never written) [ev: retro campaign8-station-snapshot], `$KIT/toolbelt/lint-servlet.sh` (BWebServlet security lint:
  auth gate, input-400, unbounded-set, cache-nofinger, log-in-handler, csrf-xrw-only; exit 0 clean or WARN-only / 1 any
  FAIL / 3 usage; run on any -ux profile with BWebServlet subclasses) [ev: retro campaign8-lint-servlet]; source never
  written) [ev: retro campaign8-station-snapshot].`$KIT/toolbelt/lint-structure.sh` (module source-tree structure lint,
  rules L1–L18 in `types/structure.md`; exit 0 clean / 1 FAIL / 3 usage; run at start of step 5 pre-gate)
  [ev: retro campaign8-structure],`$KIT/toolbelt/lint-write-path.sh` (OPERATOR-slot write-path matrix coverage; every
  @NiagaraProperty with Flags.OPERATOR must have a row in docs/write-path-matrix.md; --bog adds link-traced dashboard
  slots; STALE advisory: matrix row naming a slot absent from ALL @NiagaraProperty/Action names (any flag) prints STALE
  per-row ([concept] exempts the marked row only; exit 0); DRIFT advisory (C11 T3): a [concept]-marked row whose slot IS
  in the covered set prints DRIFT — the exemption is stale; true concept rows (slot absent) stay silent; --strict
  promotes STALE or DRIFT to exit 1; uncovered FAIL always exits 1; exit 3 usage) [ev: retro campaign8-write-path]
  [ev: retro campaign10-write-path-stale] [ev: retro campaign11-concept-row-drift]; source never written)
  [ev: retro campaign8-station-snapshot], `$KIT/toolbelt/lint-guard-pins.sh` (guard-pin meta-check: scans
  toolbelt/lint-*.sh headers for # Mutation: <id> -- <desc> declarations (the leading comment block) and resolves each
  id against tests/*.bats @test names; MATCH rows confirm declared pins; WARN rows flag missing declarations or
  unresolvable ids; scope = lint-*.sh only (D4b); exit 0 WARN-only / 1 under --strict / 3 usage)
  [ev: retro campaign11-lint-guard-pins], `$KIT/toolbelt/lint-lexicon-ascii.sh` (ASCII-purity lint for *.lexicon files;
  Niagara reads lexicons as Latin-1 so UTF-8 accents mojibake on-station; exit 0 clean / 1 any FAIL / 3 usage; wired
  into report-module.sh step 4b) [ev: retro umbrelladashboard-module-creation]
  [ev: retro dashboard-status-state-modeling Δ3] [ev: corpus B1305 §1305.1]
