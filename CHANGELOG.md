# Changelog

All notable changes to **niagara-tools** (cross-project Niagara N4 tooling) are documented in this file.

Format based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Versioning follows [SemVer](https://semver.org/spec/v2.0.0.html) (currently `v0.x` — see CONTRIBUTING.md §Versioning policy for stability promises). Each entry references the SDD change slug and engram observation IDs for traceability.

---

## [v0.9.3] - 2026-09-04

### Changed — Campaign 2 PR-C: fold build/deploy/schema lessons (Campaign 2 complete)

- `BUILD-LOOP.md`: §0.b free-the-lock-first (B2 — folding the last mirror-first instance) + read THIS
  module's gradle target (B6); §6 -ux-only = no restart (D1), station-on-another-device → verify live via
  oBIX (D2), power-cycle the panel after redeploy (D3), cd to the gradle root before ng-deploy (B9), the
  ng-deploy no-backup-default rule (B10), EXPECTED_*_TYPES = real+1 (B8).
- `build-verify.md`: nested/multi-project gradle layout — run :project:task from the gradle root (B7);
  keep src/rc asset-only (B4); a new §Module versioning & release (S4).
- `METHODOLOGY.md` §Schema/upgrade safety: the SAFE/LOSSY/OUTAGE saved-data survival matrix + bump
  vendorVersion / back up config.bog (S2); an additive slot change needs a version bump to auto-install (S3).
- Flipped 3 retros to `folded` (pure-doc, fully in core): ux-only-deploy, station-corre-en-atlas-snap,
  coldroompan-fan-mode-defrost (S3 + the delete-type-and-registration rule B12). soft-start stays
  `pending` (its build.sh clean-lock-message delta is owed). SIX rule-folded lessons whose IMPLEMENTATION
  is a script/gate change (B4/B6/B7/B8/B10 + soft-start build.sh) stay `pending`, tracked as kit
  self-section open_issues (future MINOR PRs); the T/process group stays pending (out of Campaign 2 scope).

### References

- SDD change: `build-n4-module-continuity` Campaign 2 PR-C (build/deploy/schema — Campaign 2 complete).
  Promotion of already-filed lessons → `Retro: promotion` exit; documentation only → PATCH.
- Engram: #8118 (Campaign 2 progress), #8101 (42-lesson mining).

---

## [v0.9.2] - 2026-09-04

### Changed — Campaign 2 PR-B: fold UX lessons U1–U9 into the core

- `types/dashboard.md`: promoted the dashboard/UX [CERT] lessons — overlay raster + vectors as ONE SVG
  (the WebView ignores CSS aspect-ratio/object-fit · U1); the 2-column no-scroll rule for the Control/HOA
  panel (U2); a reusable output row with custom buttons + special-cased prefill (U3); a per-room link-in
  boolean is SUMMARY not OPERATOR (U4); the reader's group arrays are the authority for the HMI surface
  (U6); the dashboard contract as an external port spec (U7); `dashboard-preview.py` --editor/--mock + the
  preview→Playwright→PDF loop (U9); the reusable add-a-tab + wire-data SPA recipe (U10); and the
  process/defrost timer facade+reader path — a READONLY BAbsTime
  anchor + a reader AbsTime type-reader emitting derived elapsed/remaining keys (process-timers #2 + L16).
- `METHODOLOGY.md`: the anchored-Python-`str.replace` edit method for a base64-heavy SPA (U8).
- Flipped `retros/INDEX.md` to `folded` for the 5 retros now fully in core (hmi-1280x800,
  dashboard-servlet-write-surface, dashboard-contract-port-spec, editing-base64-heavy-spa, and
  process-timers — its split completed). kit self-section open_issue narrowed to the remaining B / D / S groups.

### References

- SDD change: `build-n4-module-continuity` Campaign 2 PR-B (UX group complete: U1-U9; U5 was folded in
  Campaign 1). Promotion of already-filed lessons → `Retro: promotion` exit; documentation only → PATCH.
- Engram: #8118 (Campaign 2 progress), #8101 (42-lesson mining).

---

## [v0.9.1] - 2026-09-04

### Changed — Campaign 2 PR-A2: fold logic-group lessons L16–L22 into the core

- `types/logic.md`: promoted 7 [CERT] control-logic lessons — a polled-UI time is a READONLY `BAbsTime`
  anchor, not a stored `BRelTime` (L16); reserve `Flags.HIDDEN` for engine callbacks, expose a public
  force/run action for operator/oBIX (L17); guard a link-reachable `Clock.schedule` with
  `if(!Sys.atSteadyState())return`, set the anchor before scheduling, `period>0` (L18); an internal-only
  discrete selector is a `BFrozenEnum`, the double rule is cross-module only (L19); a time-gated auto
  control needs a run-now action + surfaced preconditions + an anchored first fire (L20); a new
  §Composition & organization — compose past ~12–15 slots, config separate from live-state (L21); the HOA
  manual-override slot pattern (L22).
- Flipped `retros/INDEX.md` to `folded` for the 3 retros now fully in core (hidden-actions,
  self-firing-timer, hoa-manual-override); process-timers stays `pending` (its logic lessons L16/L20 are
  folded here, but its dashboard/reader AbsTime lesson #2 folds in PR-B). kit self-section open_issue
  narrowed to the U / B / D / S groups.

### References

- SDD change: `build-n4-module-continuity` Campaign 2 PR-A2 (logic group complete: L3-L22). Promotion of
  already-filed lessons → `Retro: promotion` exit; documentation only → PATCH.
- Engram: #8118 (Campaign 2 progress), #8101 (42-lesson mining).

---

## [v0.9.0] - 2026-09-04

### Added — §7 close-gate third exit: `Retro: promotion` (MINOR)

- `.githooks/pre-push`: a THIRD accepted close for a build-relevant kit change, parallel to the
  new-retro path and the trivial waiver — a `Retro: promotion (folds <ids> from existing retros)`
  commit trailer, valid ONLY when the range also carries a `retros/INDEX.md` diff (the registry move).
  A promotion trailer with no INDEX diff is rejected (fail-closed), so it cannot be a blanket escape.
  Closes the gap where a lesson-promotion PR (Campaign 2) matched neither existing exit and would
  otherwise be forced into a FALSE `trivial` label. `sweep-build-state.sh` is unchanged (VCS-free,
  kit-links L2) — the trailer is a commit/diff concept the hook owns.
- `build-n4-module-kit/BUILD-LOOP.md` §7: documents exit (c).
- `tests/build-retro-sync.bats`: H8 (promotion + INDEX diff → pass) / H9 (promotion, no INDEX → fail),
  RED-first, mutation-checked. 18 cases total.
- Retro `2026-09-04-gate-exit-taxonomy-promotion.md` (+ INDEX row) captures the exit-taxonomy lesson.

### References

- SDD change: `build-n4-module-continuity` Campaign 2 (gate hardening; MINOR — a new hook behavior).
- Engram: #8114 (Campaign 1 complete). Sequenced after PR-A (0.8.2); PR-A2/B/C follow as 0.9.1/0.9.2/0.9.3.

---

## [v0.8.2] - 2026-09-04

### Changed — Campaign 2 PR-A: fold logic-group lessons L3–L14 into the core

- `types/logic.md`: promoted 11 [CERT] control-logic lessons from `retros/` into the living guide —
  invalid-reading-vs-bad-value / assume-commanded on a dead sensor (L3); latched fault + operator
  `faultReset`, never auto-retry rotating equipment (L4); proof-of-run + stuck-contactor alarms (L5);
  persist-only-survival + `resetTransient()` in `stopped()` (L7); arm the heartbeat tick in its own try
  before the first `execute()` (L8); the no-reentrancy invariant — outputs ∉ `changed()` filter (L9);
  cancel every ticket on one shared path at both edges + mode enter/exit (L10); a decoupled output needs
  its own apply method wired into every setter (L11); ship an additive phase dormant behind a default,
  old tests guard for free (L12); stub an unvalidated safety-critical data function to NaN, never invented
  numbers (L13); self-order by a guarded sibling tree-walk, not a coordinator component (L14); and the input-only-phase boundary rule (fase3 #2) — a phase changing only an INPUT pushes new logic to the adapter boundary, leaving the proven core and its tests untouched.
- Flipped `retros/INDEX.md` to `folded` for the 3 retros now fully in core (comppan-fase1/2/3); updated the
  `kit` self-section open_issue to the remaining L16-L22 / U / B / D / S groups. Excludes L6 (already folded
  in Campaign 1).

### References

- SDD change: `build-n4-module-continuity` Campaign 2 (PR-A of PR-A/A2/B/C). Documentation only → PATCH.
- Engram: #8114 (Campaign 1 complete), #8101 (42-lesson mining).

---

## [v0.8.1] - 2026-09-04

### Changed — promote proven retro lessons into the kit core (PR3)

- **Lesson folds (documentation):** promoted the highest-value [CERT]-grade lessons trapped in `retros/`
  into the living kit files —
  - `METHODOLOGY.md`: never retype a live slot / bog boot crash (S1) + NEW-safety-slot-defaults-SAFE
    carve-out (L15) under a new §Schema/upgrade safety; `0`-means-DISABLED on a protection limit (L2);
    `module.palette` one entry per `@NiagaraType` (B5); the timer `started()`+`atSteadyState()` pointer;
    the 4-layer assurance pointer.
  - `types/logic.md`: `Long.MIN_VALUE` sentinel guard (L1); the slot-default SAFE carve-out (L15); a
    §Pure-class extraction section (extract-test-then-wire, stateful `step()` adapter · T2/L6); cross-refs.
  - `types/dashboard.md`: a generic write endpoint must whitelist / check `Flags.OPERATOR` (U5).
  - `build-verify.md`: free-the-lock-first vs mirror-fallback (B2) + build/libs-still-signed (B1) +
    first-build-no-stop (B3); the `find ~/.gradle` + `run-pure-test.sh` recipe and niagaraTest-is-docs
    (B11); a §How you know it's good 4-layer assurance stack (T1/T3/T4).
- **Contradictions resolved to ONE rule each:** jar-lock (free the lock first; mirror only for a live
  production station) → `build-verify.md`; slot default (preserve current on a deployed path, but a NEW
  safety-gating slot defaults SAFE) → `METHODOLOGY.md` + `types/logic.md`.
- **`kit` self-section** added to `BUILD-STATE.md` (`module: kit`) so kit-infrastructure work has a row to
  update; `BUILD-LOOP.md` §7 notes it. Dogfood: this campaign filed TWO retros (the campaign retro + the
  run-pure-test `set -e` empty-cache lesson) with `retros/INDEX.md` rows, and marked 4 fully-folded retros
  `folded` — the mechanism recording its own evolution.

### References

- SDD change: `build-n4-module-continuity` (PR3 of 3; follows PR1 v0.6.0 ledger, PR2 v0.7.0 gate, v0.8.0
  run-pure-test). Documentation + schema-section only, no new script → PATCH.
- Engram: #8095 (campaign), #8101 (42-lesson mining), #8105 (tasks), #8111 (chain status).

---

## [v0.8.0] - 2026-09-04

### Added — one-command pure-test runner (`run-pure-test.sh`)

- `build-n4-module-kit/toolbelt/run-pure-test.sh` (NEW): runs a ZERO-Baja pure decision class +
  its JUnit test standalone in WSL, resolving junit-4.13.2 + hamcrest-1.3 from `~/.gradle` and
  compiling into a TEMP dir (never the module tree — a parallel session's working tree is
  off-limits). Removes the per-session jar hunt documented in the
  `2026-09-04-junit-standalone-cached-jar-locations` retro. Exit 0 pass / 1 test failed / 2 usage /
  3 env. Proven against the real CompPan `CompressorControlTest`: 31 tests OK in 0.015s.
- `tests/run-pure-test.bats` (NEW): 6 cases, each mutation-checked to bite — P2 (a failing pure
  test must exit non-zero, i.e. the runner does not swallow a JUnit failure) and P6 (an empty
  `~/.gradle` cache exits 3 with an actionable message) both caught real behaviour, the latter a
  `set -e`/pipefail bug in the runner itself.

---

## [v0.7.0] - 2026-09-04

### Added — retro-enforcement gate (sweep + pre-push hook + retro index)

- `build-n4-module-kit/toolbelt/sweep-build-state.sh` (NEW): content-only validator (VCS-free —
  kit-links L2) for BUILD-STATE.md + retros/INDEX.md. Exit 0 clean / 1 named integrity violation /
  3 usage. Checks column-0-anchored `build-state.v1` markers, required boolean fields
  (module/retro_required/retro_pending), a tolerated multi-line `open_issues` list, and INDEX
  integrity (every retro has a row, every row a real file, review-status ∈ {pending,folded}).
- `.githooks/pre-push` (NEW): opt-in gate (enable with `git config core.hooksPath .githooks`).
  Blocks a push that changes build-relevant kit files (`build-n4-module-kit/**` except BUILD-STATE.md
  and `retros/`, plus `scripts/**`) unless the range also carries a BUILD-STATE.md update + a pending
  retro + its INDEX row, or a `Retro: none (trivial: <reason>)` trailer. Delegates the content half
  to `sweep-build-state.sh` (the diff half owns the only VCS calls).
- `build-n4-module-kit/retros/INDEX.md` (NEW): the promotion registry — one row per retro (30
  seeded; 3 folded, 27 pending), with a `deltas` count column.
- `tests/build-retro-sync.bats` (NEW, RED-first): 16 cases (9 sweep + 7 hook), each
  mutation-checked to bite.
- `build-n4-module-kit/BUILD-LOOP.md` §7: parenthetical updated to point at the now-live gate.

### References

- SDD change: `build-n4-module-continuity` (PR2 of 3; follows PR1 v0.6.0 continuity ledger).
- Engram: #8095 (campaign), #8101 (recon), #8105 (tasks), #8107 (PR1 landed).

---

## [v0.6.0] - 2026-09-04

### Added — build-session continuity ledger (BUILD-STATE.md)

- `build-n4-module-kit/BUILD-STATE.md` (NEW): a "where did we leave off" registry for the build
  loop — one `build-state.v1` envelope per module (ColdRoomPan, CompPan, DashboardPan, chihuahua),
  seeded from evidence. The build-loop analog of research-sdd's FOCUSES + RESEARCH-STATE registries,
  collapsed into one file. Fields split GATED (`retro_required`, `retro_pending` — kit-local only)
  vs DECLARED (jar/module/src state, which lives in separate repos and is recorded, not gated).
- `build-n4-module-kit/BUILD-LOOP.md`: new `§0.a Orient from BUILD-STATE` (prints a one-line
  leave-off per module, with the meta-work exemption) and a reworded `§7` HARD close gate — update
  BUILD-STATE and write a retro, or declare `Retro: none (trivial: <reason>)`; print the `retro:`
  line in the Output Contract.
- Launcher `SKILL.md` (in `~/.claude/skills/build-n4-module/`, outside this repo): orient from
  BUILD-STATE at step 1, update it at step 6, print the retro line in the Output Contract.

### References

- SDD change: `build-n4-module-continuity` (PR1 of 3 — continuity registry; PR2 = retro-enforcement
  gate, PR3 = promote proven lessons). Tasks: `openspec/changes/build-n4-module-continuity/tasks.md`.
- Engram: #8095 (campaign kickoff), #8101 (recon: continuity design + 42-lesson mining), #8105 (tasks).

---

## [v0.5.0] - 2026-09-01

### Added — ng-deploy verify-module gate and `-ux` slotomatic

- `scripts/ng-deploy.sh`: runs the `build-n4-module-kit/toolbelt/verify-module.sh` gate on the
  built jars during the verify step, default ON for mode A/C. New `--no-gate` flag to skip it and
  `VERIFY_MODULE_BIN` env var to override the gate binary (used by the tests).
- `scripts/ng-deploy.sh`: runs `:MODULE-ux:slotomatic` (in addition to `:MODULE-rt:slotomatic`)
  under `--with-slotomatic` in mode A when the `-ux` profile source carries a
  `@Niagara(Type|Property|Action|Topic|Singleton)` annotation (presence-based scan).
- `tests/ng-deploy.bats`: 7 new cases — T27–T29 (`-ux` slotomatic) and T30–T33 (verify-module
  gate). Each was mutation-checked to fail when its target behavior is broken. 33 total.

### Changed — ng-deploy verify-module gate and `-ux` slotomatic

- `scripts/ng-deploy.sh`: exit 50 ("verify failed") now also covers a failed `verify-module.sh`
  gate, alongside the existing type-count mismatch and missing-BUILD_ID cases.
- `scripts/ng-deploy.sh`: mode-B `--with-slotomatic` warning enriched to point at
  `--mode A --with-slotomatic` or `toolbelt/build.sh` for regenerating `-ux` slots (behavior
  unchanged — mode B still warns and skips).

### References

- SDD context: build-n4-module-kit v0.2 retro §6 follow-ups P1 (ng-deploy gate) + P2 (`-ux` slotomatic).
- Engram: #7953 (this change).

---

## [v0.4.0] - 2026-09-01

### Added — `build-n4-module-kit-v0.2`

- `build-n4-module-kit`: fold-in of 41 proven lessons from 3 retros (ColdRoomPan rt-hardening,
  DashboardPan 5-rooms, DashboardPan HMI touch UX) and the concurrent session bitácora into
  7 kit files (`types/dashboard.md`, `types/logic.md`, `build-verify.md`, `METHODOLOGY.md`,
  `BUILD-LOOP.md`, `SOURCES.md`, `types/wb-widgets.md`). Evidence markers (`[CERT]`,
  `[CERT-live]`, `[INFER]`) preserved from source retros.
- `toolbelt/verify-module.sh`: THE verify gate — bytecode major 52, `NIAGARA4.SF` presence,
  `module.xml` type-to-class resolution; opt-in `--target-version`, `--stored`, `--src` checks.
  POSIX-only (no JDK). Exit codes 0/1/2/3.
- `toolbelt/build.sh`: rewrite — source-based profile selection (skips stub `-wb`), Java 8 +
  clean + slotomatic + jar, runs `verify-module.sh` on every produced jar. Exit codes 2/10/30/50.
- `toolbelt/mirror-niagara-home.sh`: safe writable mirror of a live Niagara install for builds
  against a running station (refuses real install or any non-mirror dir, exit 20).
- `toolbelt/stored-repack.sh`: STORED repackage for the Workbench re-sign path (B7 recipe);
  manifest-first ordering, `zip -0`, verifiable with `verify-module.sh --stored`.
- `tests/verify-module.bats`, `tests/build-sh.bats`, `tests/mirror-niagara-home.bats`,
  `tests/stored-repack.bats`, `tests/kit-links.bats`: 5 bats suites (27 tests; 53 total with ng-deploy.bats) with generated-jar
  fixtures helper `tests/helpers/n4-fixtures.bash`.

### Changed — `build-n4-module-kit-v0.2`

- Kit doctrine rewritten to three explicit roles: `verify-module.sh` (THE gate),
  `build.sh` (recommended WSL build, runs the gate), `ng-deploy.sh` (station deploy wrapper —
  backup → build → copy → type-count verify; slotomatic guard rt-only). Replaces the ambiguous
  primary/fallback framing in `build-verify.md` and `BUILD-LOOP.md`.
- `toolbelt/build.sh` exit codes: 2 usage error, 10 env/path error, 30 gradle failure,
  50 verify-module gate failure (supersedes previous informal codes).
- `CONTRIBUTING.md`: bats-core install step added to the test-runner prerequisites section.

### Fixed — `build-n4-module-kit-v0.2`

- Dangling kit links (`checklist-common.md` → `METHODOLOGY.md`;
  `type-dashboard.md` → `types/dashboard.md`) removed from kit files.
- `build.sh`: stub-profile selection now skips profiles with a `build.gradle` but no source files
  (was silently including them).
- `build.sh`: all-classes bytecode major check now correctly catches a later `.class` at major 65
  when the first `.class` in the jar is major 52 (false-pass fixed).

### Notes

- Launcher `~/.claude/skills/build-n4-module/SKILL.md` updated to v0.2 (three-role doctrine,
  verify-module.sh in Output Contract). This file lives outside the git repository; it is not
  included in this tag but is documented in the PR3 body and in engram.

### References

- SDD slug: `build-n4-module-kit-v0.2`
- Engram: explore #7937, proposal #7938, spec #7939, design #7940, tasks #7943.
- Tag: `v0.4.0`.

---

## [v0.3.0] - 2026-05-18

### Added — `niagara-tools-slotomatic-integration`

- `scripts/ng-deploy.sh`: `--with-slotomatic` flag — invoca `:MODULE-rt:slotomatic` con los
  3 overrides `-P` ANTES de `build_jars` (modos A/C; ignorado en modo B con WARN).
- `scripts/ng-deploy.sh`: `--strict-slotomatic` flag — aborta con exit 15 si se detectan
  cambios de anotación sin `--with-slotomatic`. No implica `--with-slotomatic` (cero magia).
- `scripts/ng-deploy.sh`: `SLOTOMATIC_DETECTION` env var (`warn`|`strict`|`off`, default `warn`) —
  controla la heurística pasiva de detección de cambios en anotaciones `@Niagara*`.
- `scripts/ng-deploy.sh`: `detect_annotation_changes()` — heurística pasiva que corre entre
  backup y build. Lee `.last-deploy-sha` como baseline (fallback `HEAD~1`); filtra diff con
  `grep -E '^[+-][[:space:]]*@Niagara(Type|Property|Action|Topic|Singleton)'`.
- `scripts/ng-deploy.sh`: `run_slotomatic()` — invoca gradlew con los mismos 3 `-P` overrides
  que `build_jars`; `die 15` si gradlew retorna distinto de cero.
- `scripts/ng-deploy.sh`: `write_last_deploy_sha()` — escribe `git rev-parse HEAD` en
  `.last-deploy-sha` post-verify exitoso; silencioso si git falla.
- `scripts/ng-deploy.sh`: `read_baseline_sha()` — lee `.last-deploy-sha`, valida con
  `git cat-file -e`, fallback a `HEAD~1` si ausente/vacío/SHA inválido.
- `scripts/ng-deploy.sh`: `warn_slotomatic_recommended()` — heredoc multi-línea a stderr.
- `scripts/ng-deploy.sh`: exit code 15 nuevo — slotomatic falló O cambios de anotación
  detectados en modo strict. Ningún code path existente (0-50) fue modificado.
- `tests/ng-deploy.bats`: 9 tests nuevos (T18–T26), total 26. git fakebin en setup().
  Refactor gradlew stub: `gradlew.calls.log` (log acumulativo) + `gradlew.args` (backward-compat).
- `.env.local.example`: sección `SLOTOMATIC_DETECTION` + nota sobre `.last-deploy-sha` en `.gitignore`.
- `docs/knowledge-base/slotomatic.md`: Card 4 — ng-deploy.sh integration (flags, detection
  mechanics, decision table, false-positive edge case, `.last-deploy-sha` gitignore note).
- `docs/GOTCHAS.md`: fila anti-pattern "Deploy with stale slotomatic".
- `tests/smoke-checklist.md`: paso opcional `--with-slotomatic` en modo A; nota no-op en modo B.

### Changed — `niagara-tools-slotomatic-integration`

- `scripts/ng-deploy.sh` `print_usage()`: documenta `--with-slotomatic`, `--strict-slotomatic`,
  `SLOTOMATIC_DETECTION` env var, y exit code 15.
- `CLAUDE.md` §1 tabla: fila "Slot/Property/Type/Action" actualizada →
  `A --with-slotomatic (or :slotomatic separately first)`.
- `scripts/ng-deploy.sh` header comment: añade `--with-slotomatic`, `--strict-slotomatic` a Usage.

### References

- SDD slug: `niagara-tools-slotomatic-integration`
- Engram: spec #1927, design #1928, tasks #1929.
- Tag: `v0.3.0` (pendiente de commit del operador).

---

## [v0.2.0] - 2026-05-18

### Added — `niagara-tools-versioning-and-contributing`

- `VERSION` file at repo root as the single source of truth for the version string.
- `CHANGELOG.md` (this file) in Keep a Changelog format, seeded with retroactive `[v0.1.0]`
  (bootstrap) and the current `[v0.2.0]` entries.
- `CONTRIBUTING.md` (solo+agents scope): release recipe, TDD gate, KB authoring rules,
  conventional commits, pre-commit checklist, versioning policy, content-boundary matrix,
  and known limitations.
- `scripts/ng-deploy.sh`: `--version` flag prints `SCRIPT_VERSION` and exits 0. Value is
  resolved CWD-agnostically via `BASH_SOURCE[0]`-relative read of `VERSION`
  (`cat "${SCRIPT_DIR}/../VERSION"`), with an `"unknown"` fallback when the file is missing.
- `tests/ng-deploy.bats`: 2 new tests (17 total) — `--version` exit/output contract and
  anti-drift regression guard for the path resolution.
- `CLAUDE.md` §9 "Release process": one-liner bump rule + pointer to `CONTRIBUTING.md`
  §Release process. (Decision tree only — no policy duplication; see proposal-decisions #1846.)
- `README.md`: "Versioning" section linking to this changelog.
- Git tags: `v0.1.0` retroactive on bootstrap commit `8d9a396`; `v0.2.0` on this commit.

### Changed — `niagara-tools-versioning-and-contributing`

- `scripts/ng-deploy.sh` header Usage comment: added `--version` next to `--help` in the
  metadata-flags Usage line.

### References

- SDD slug: `niagara-tools-versioning-and-contributing`
- Engram: explore #1843, proposal #1844, proposal-decisions #1846, spec #1848, design #1847.
- Tag: `v0.2.0` (this commit).

---

## [v0.1.0] - 2026-05-17

### Added — `niagara-tools-bootstrap` (retroactive entry)

This release documents the initial bootstrap of the `niagara-tools` repo. The actual commit
(`8d9a396`) shipped before any versioning infrastructure existed; this entry is the retroactive
`v0.1.0` record, tagged after the fact on the bootstrap commit.

- `scripts/ng-deploy.sh`: bash wrapper for the Niagara N4 module deploy cycle
  (backup → build → copy → verify). Flag API: `--mode A|B|C`, `--env-file PATH`,
  `--no-deploy`, `--no-backup --i-know-what-im-doing`, `--help`. Exit codes: 0 (success),
  10 (env/path), 20 (backup), 30 (build), 40 (copy), 50 (verify).
- `.env.local.example`: config schema with all 8 required env vars (`MODULE_NAME`,
  `GRADLEW_PATH`, `NIAGARA_HOME`, `NIAGARA_USER_HOME`, `JAVA_HOME`, `STATION_MODULES_DIR`,
  `EXPECTED_RT_TYPES`, `EXPECTED_UX_TYPES`) and optional `BUILD_ID` cache-buster.
- `tests/ng-deploy.bats`: 15 bats-core unit tests using PATH-injected fakebin stubs for
  `gradlew`, `unzip`, `tar`. No real station or build dependency.
- `tests/smoke-checklist.md`: manual integration checklist for modes A, B, C.
- `docs/GOTCHAS.md`: cross-project anti-patterns index linking the KB topic files.
- `docs/knowledge-base/`: four topic files seeded from chihuahua learnings —
  `bql-gotchas.md` (BQL N4.14 confirmed bugs + persistent-ack pattern),
  `wsl-build-gotchas.md` (WSL build overrides, gradlew path, slotomatic-in-WSL myth),
  `hot-reload-rules.md` (Java = station restart; JS/CSS = browser hard-reload),
  `slotomatic.md` (when to run, slot removal coordinated edit, AUTO region rules).
- `CLAUDE.md`: agent guide with deploy decision table, invariants
  (backup → build → copy → verify), onboarding, gitignore exception note, test runner setup,
  engram `topic_key` conventions, cross-project search hints, KB index.
- `README.md`: human-first quick-start with consumer-module integration pattern.
- `.gitignore`: `_backups/`, `.env.*` with explicit `!.env.local.example` exception.

Bootstrap commit: `8d9a396` (retroactively tagged `v0.1.0`).

### References

- SDD slug: `niagara-tools-bootstrap`
- Engram: init #1806, proposal #1811, delivery decision #1817, apply #1820, verify #1823,
  archive #1824.
