# Changelog

All notable changes to **niagara-tools** (cross-project Niagara N4 tooling) are documented in this file.

Format based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Versioning follows [SemVer](https://semver.org/spec/v2.0.0.html) (currently `v0.x` — see CONTRIBUTING.md §Versioning policy for stability promises). Each entry references the SDD change slug and engram observation IDs for traceability.

---

## [v0.15.0] - 2026-09-04

### Added / Changed — activate the retro-enforcement gate (Campaign 5 AG-PR1)

- **`.githooks/pre-push` hardened** — the promotion exit now accepts a `Retro: promotion (folds …)` trailer
  when there is an in-range `retros/INDEX.md` diff (a full promotion flips a row) **OR** an in-range
  `BUILD-STATE.md` diff (a PARTIAL promotion folds content while its source retro stays `pending`, so it
  flips no row — it stamps the owed `open_issue` instead). This fixes the false-negative on legitimate
  partial promotions found dogfooding C4-PR2. The blanket-escape guard is kept: a promotion trailer with
  NEITHER anchor still FAILS. Either anchored path now runs `sweep-build-state.sh` for ledger coherence.
- **NEW `scripts/install-hooks.sh`** — opt-in, per-clone, reversible activation of the gate: sets
  `git config --local core.hooksPath .githooks` (idempotent); `--uninstall` restores the default; it
  REFUSES to clobber a pre-existing custom `core.hooksPath` (your own hooks are never silently overwritten)
  unless `--force`. It lives under `scripts/` (not `toolbelt/`) because it uses git — the kit-links L2 rule
  bans git only in `toolbelt/*.sh`.
- **Docs** — activation documented in `build-n4-module-kit/BUILD-LOOP.md` §7 and `CONTRIBUTING.md` §6.1;
  the §7 promotion-exit contract updated to the INDEX-OR-BUILD-STATE structural anchor.
- `tests/build-retro-sync.bats`: H8/H9/H10 (QA, RED-first) green — H10 proves a BUILD-STATE-anchored partial
  promotion PASSES. `tests/install-hooks.bats`: I1-I4 (git-real on a throwaway repo) green.
- The gate still ships **inert** — this PR makes activation a one-command step; the live activation is a
  deliberate operator action (`scripts/install-hooks.sh`), not automatic.

### References

- SDD change: `build-n4-module-continuity` Campaign 5 AG-PR1 (gate activation). New behavior + new script → MINOR.

---

## [v0.14.1] - 2026-09-04

### Added — Campaign 4 close retro (process meta-lessons)

- `retros/2026-09-04-campaign4-close-process-meta-lessons.md` (+ INDEX row, `pending`): captures the three
  genuinely-new process lessons Campaign 4 surfaced — (1) the **stale-checkout trap** (verify coverage against
  a worktree off origin/main, never a local `main` that can sit a whole campaign stale; a line-number mismatch
  is the tell); (2) the **partial-promotion 4th-gate-shape** (a fold that flips no INDEX row is legitimate but
  the §7 promotion exit false-negatives it — accept a BUILD-STATE diff as an alternative anchor, tracked as a
  gate-hardening open_issue); (3) **grep every kit file before folding** (a lesson's mined target isn't the
  only place it may live — T4 was nearly double-folded because only its target file was grepped). This closes
  Campaign 4; the 42-lesson mined corpus is exhausted.

### References

- SDD change: `build-n4-module-continuity` Campaign 4 close. New retro filed → PATCH.

---

## [v0.14.0] - 2026-09-04

### Added — corpus-index.md: a curated map of the authoring corpus (Campaign 4 C4-PR3, terminal)

- **NEW `build-n4-module-kit/corpus-index.md`** (T7a) — a curated nav of the niagara-research authoring
  corpus (B729–B760): one row per block (block → what it gives the builder → layer), grouped **priority-first**
  (P0 read-before-building → P1 → P2) with layer as the sub-sort, plus a "start here" pointer (B744 rt / B760
  our-module / B751 -wb / B752 -ux). It serves the `corpus-nav FIRST` hard rule: `corpus-nav` for a term, this
  index for what to read by layer/priority. It is a curated pointer, not a dump — each block lives in
  niagara-research and is cited, not copied.
- **Wired in-repo** (minimal, no churn): a one-line pointer from BUILD-LOOP §2, METHODOLOGY top, and
  types/logic.md start-here.
- `retros/INDEX.md`: **corpus-index-rt-authoring-and-organization-blocks → `folded`** — all eight of its
  proposed deltas are now in core (corpus-index + pointers here; WB/UX growth in C4-PR2; lexicon in C4-PR1;
  composition B737 at logic.md §Composition · L21; schema/versioning B739/B754 at METHODOLOGY §Schema/upgrade
  safety · S1/S2/S3; build-target B756 at build-verify.md). This flips the row cleanly (in-range INDEX diff).
- **This closes Campaign 4 and exhausts the 42-lesson mined corpus.** Every mined lesson (logic L, UX U,
  build/deploy/schema B/S/D, testing/process T) is now folded, and every promotion was fidelity-graded.
- Companion (out-of-repo, not gated/CHANGELOG'd): the launcher `~/.claude/skills/build-n4-module/SKILL.md`
  `corpus-nav FIRST` rule + step 2 point at `$KIT/corpus-index.md` and at B760 for our-module work.

### References

- SDD change: `build-n4-module-continuity` Campaign 4 C4-PR3 (terminal). New KB artifact + wiring → MINOR.

---

## [v0.13.4] - 2026-09-04

### Changed — T7 b/c: grow the two weakest type guides from the WB/UX corpus (Campaign 4 C4-PR2)

- **types/wb-widgets.md** (T7b, B751) — folded the "how much wb is enough" ladder (rung 0 nothing → 1
  FieldEditor → 2 Manager → 3 custom View; our components sit at rung 0), the FieldEditor recipe, and the
  "a Honeywell Wizard is a tabbed `BWbComponentView`, not a `BWizard`" rule. The seed guide now carries the
  authoring ladder.
- **types/dashboard.md** (T7c, B752/B753) — folded the three-way serving-recipe decision (servlet-SPA vs
  bajaux `@AgentOn` vs PX; our DashboardPan is servlet-SPA, don't migrate) and the vendor-contrast RBAC
  caveat (Honeywell React SPAs ship `permissions="unrestricted"` — don't copy; the browser is never the
  security boundary). The RBAC core, pure router, traversal guard, and Home-Page footgun of B752 were
  ALREADY in the guide, so only the new parts were folded (no double-fold).
- `retros/INDEX.md`: unchanged — **corpus-index-rt-authoring-and-organization-blocks stays `pending`**
  (split-retro: its T8 half folded in C4-PR1, T7 b/c here, and T7a's corpus-index nav doc is owed to C4-PR3;
  the retro flips to `folded` only when T7a lands).

### References

- SDD change: `build-n4-module-continuity` Campaign 4 C4-PR2. Doc folds → PATCH. (Partial promotion — no
  INDEX flip yet; the source retro flips in C4-PR3.)

---

## [v0.13.3] - 2026-09-04

### Changed — T-group doc folds T5/T6/T8 (Campaign 4 C4-PR1)

- **BUILD-LOOP.md §3** (T5): operator preview + explicit OK is now a REQUIRED gate before building any
  `-ux` change — seed the mock with the state that triggers the new behavior, and note the mock does NOT run
  `-rt` logic (the preview approves the DASHBOARD, not the physical rt effect).
- **METHODOLOGY.md** new "Debugging" section (T6): on a log WARNING, classify caught-and-cosmetic
  (swallowed by `changed()`/`try`) vs propagates-and-aborts before treating it as the root cause — a captured
  stack trace up to `Station.startStation` is not itself a failure.
- **METHODOLOGY.md** lexicon item (T8): reinforced — a missing `module.lexicon` key silently renders raw
  camelCase via `toFriendly`; keys are module-global (shared slot names collide).
- `retros/INDEX.md`: **self-retro-preview-gate-wsl-tests-live-verify** → `folded` — its four lessons are now
  all in core (T5 §3 here, the pure-test recipe at build-verify.md, T4 live-anchor smoke already at
  build-verify.md, T6 here). **corpus-index-rt-authoring-and-organization-blocks stays `pending`** — its T8
  half is folded but its T7 body (corpus-index + WB/serving folds) is owed to C4-PR2/PR3.

### References

- SDD change: `build-n4-module-continuity` Campaign 4 C4-PR1. Doc folds → PATCH.

---

## [v0.13.2] - 2026-09-04

### Added — Campaign 3 close retro (process meta-lessons)

- `retros/2026-09-04-campaign3-close-process-meta-lessons.md` (+ INDEX row, `pending`): captures the three
  genuinely-new process lessons Campaign 3 surfaced — (1) a new gate/verify check must be high-signal /
  low-false-positive, scoped to the reported defect (empty-palette-with-types, rc/-backup files), not a
  literal completeness rule; (2) when a lesson removes a safety guard, prefer a frictionless WARN over a
  silent removal; (3) when a retro's prose and its PROPOSED-delta marker disagree, the propose-never-apply
  marker is what promotes. This closes Campaign 3.

### References

- SDD change: `build-n4-module-continuity` Campaign 3 close. New retro filed → PATCH.

---

## [v0.13.1] - 2026-09-04

### Changed — kit doc tidy pass (Campaign 3 close, doc-only)

- **METHODOLOGY.md**: added a new "Kit maintenance — retro promotion discipline" section folding two
  process rules from the Campaign 2 close retro — the **doc-vs-script folded-completeness** rule (a
  script/gate lesson is only PARTLY folded by its prose; the impl stays owed and tracked) and the
  **adversarial fidelity-grading** rule (a promotion's correctness is faithfulness-to-source +
  fold-completeness, so budget an independent per-lesson fidelity pass — a green suite can't verify it).
- **types/dashboard.md** (U1): spelled out the detail-render height nuance — give the frame a definite
  `height:100%` (not `height:auto`), else a near-square image overflows and `overflow:hidden` clips the
  bottom zone label; the SVG `meet` then letterboxes inside a definite height.
- **Tag-convention normalization**: the two mixed-format retro citations that put a retro-local `#N` after
  the `·` separator (`comppan-fase3 · #2`, `process-timers · #2 + L16`) now use the full retro slug — neither
  lesson has a global L/U/B tag (fase3 #2 emerged during folding; process-timers #2 folds under L16), so the
  honest fix is the slug, not an invented tag. The ~16 `rt-hardening #N` / `5rooms #N` citations are
  legitimately pre-global-tag and left as-is.
- `retros/INDEX.md`: flipped **campaign2-promotion-process-meta-lessons** → `folded` (its three meta-lessons
  are now all in core: split-retro rule in BUILD-STATE, doc-vs-script + fidelity-grading in METHODOLOGY).

### References

- SDD change: `build-n4-module-continuity` Campaign 3 close (tidy pass). Doc-only → PATCH.

---

## [v0.13.0] - 2026-09-04

### Added — verify-module.sh: empty-palette gate check (Campaign 3 C3-PR4)

- `build-n4-module-kit/toolbelt/verify-module.sh` (stays VCS-free):
  - **palette check** (default-on, no `--src`) — a module that declares types but ships an **empty
    `module.palette`** (only the `b:Folder` root, zero `<p n=…>` component entries) builds, passes the whole
    gate, deploys, and then shows **nothing to drag in Workbench** — commissioning is silently broken.
    The check now WARNs (or FAILs under `--strict`), naming the empty palette and the declared type count.
    It SKIPs when the jar has no `module.palette`, and PASSes a populated one. A **typeless** module with an
    empty palette does NOT warn (the type-count guard keeps it high-signal, not a nag).
- `tests/verify-module.bats`: V13–V17 (QA, RED-first) now green; V1–V12 unchanged.
- `retros/INDEX.md`: flipped **module-palette-and-build-target** → `folded` — its last pending half (the gate
  check) is now implemented; the palette authoring rule (B5, METHODOLOGY) and the per-module build target
  (B6, build.sh) were already folded.
- This closes the owed-script-implementation backlog: B4, B6, B7, B8, B10, soft-start, and palette are all
  implemented and folded.

### References

- SDD change: `build-n4-module-continuity` Campaign 3 C3-PR4. New default-on gate check → MINOR.

---

## [v0.12.0] - 2026-09-04

### Changed — ng-deploy.sh: type-count fix, lightweight-backup default, --no-backup WARN (Campaign 3 C3-PR3)

- `scripts/ng-deploy.sh`:
  - **B8** (`ng-deploy-type-count`) — `verify_jar` counted module.xml `<type` entries with `grep "<type"`,
    which ALSO matched the `<types>` wrapper element → off-by-one (a correct jar reported N+1 and failed).
    Fixed to `grep "<type "` (trailing space): only real entries count.
  - **B10** (`ng-deploy-backup-liviano-y-autopurga`) — backup is now **lightweight by default**: it archives
    only this module's own `<MODULE>-{rt,ux,wb}.jar` present in `STATION_MODULES_DIR`, not the whole modules
    dir (the old default grew `_backups/` ~240 MB/deploy without bound). A **keep-N autopurge** (default 3,
    `--keep N`) prunes older backups after each successful one.
  - **`--no-backup` gate removed → WARN** — no longer requires `--i-know-what-im-doing`; it is a plain opt-in
    that prints a one-line rollback reminder (`backup skipped … committed to git for rollback`).
    `--i-know-what-im-doing` stays as an accepted no-op for backward compatibility.
  - **`--full-backup` / `FULL_BACKUP=1`** restores the old whole-modules-dir backup.
  - Exit code **20** now means a genuine backup failure only (no longer the removed gate).
- `tests/ng-deploy.bats`: B8 + B10 i/ii/iii/iv (QA, RED-first) now green; T1–T33 unchanged.
- `retros/INDEX.md`: flipped **ng-deploy-type-count** (B8) and **ng-deploy-backup-liviano-y-autopurga** (B10,
  promoted rule = lightweight-default + keep-N autopurge + `--no-backup` opt-in + `--full-backup`) → `folded`.
  kit self-section: owed B8/B10 → DONE; last owed = the verify-module palette check (→ C3-PR4).

### References

- SDD change: `build-n4-module-continuity` Campaign 3 C3-PR3. New ng-deploy.sh behavior → MINOR.

---

## [v0.11.0] - 2026-09-04

### Added — build.sh: auto-detect plugin, gradle-root walk-up, clean-lock message (Campaign 3 C3-PR2)

- `build-n4-module-kit/toolbelt/build.sh` (stays git-free):
  - **B6** — when no `--plugin-version` / `$NIAGARA_PLUGIN_VERSION` is given, auto-detect the module's own
    pinned `niagaraPluginVersion` from its `gradle.properties` (or `settings.gradle.kts` getOrElse) and
    forward it; an explicit flag/env still wins.
  - **B7** — if `<ROOT>/gradlew` is absent, walk UP to the gradle root (client multi-project layout) and run
    the `:MOD-p:` tasks there; profiles stay resolved under `<ROOT>/<MOD>/<MOD>-p`.
  - **soft-start** — capture gradle output; a `:clean` failure matching `Unable to delete …/modules/…jar`
    (a running-station lock) prints an actionable message (free the lock / mirror / use build/libs) and exits
    a distinct **31** (documented in the header), instead of the generic 30.
- `tests/build-sh.bats`: B8/B9/B10 (QA, RED-first) now green; B1–B7 unchanged.
- `retros/INDEX.md`: flipped **dashboardpan-ux-direct-build** (B7) and **soft-start-staggered-startup** (its
  build.sh clean-lock delta) → `folded`. **module-palette-and-build-target stays `pending`** — B6 is done, but
  its retro also owes a verify-module palette check (→ C3-PR4). kit self-section: B6/B7/soft-start owed → DONE;
  owed now B8/B10 (ng-deploy → C3-PR3) + the palette check (→ C3-PR4).

### References

- SDD change: `build-n4-module-continuity` Campaign 3 C3-PR2. New build.sh behavior → MINOR.
- Engram: #8120 (Campaign 2 complete).

---

## [v0.10.0] - 2026-09-04

### Added — verify-module.sh rc/ editor-backup check + `--strict` (Campaign 3 C3-PR1, B4 impl)

- `build-n4-module-kit/toolbelt/verify-module.sh`: new default-on `rcbackup` check — editor/backup files
  (`*~`, `*.orig`, `*.bak*`) packaged under `rc/` (bloat + servable) emit a `WARN rcbackup` row NAMING the
  offenders; the WARN does NOT increment the failure count (exit stays 0 for an otherwise-valid jar). New
  `--strict` flag promotes it to `FAIL rcbackup` (exit 1). A `WARN` severity was added to the reporter and
  a `warned` count to the summary. Implements the owed B4 (its rule was folded into build-verify.md in
  Campaign 2 PR-C).
- `tests/verify-module.bats`: V10 (WARN, exit 0) / V11 (clean rc silent) / V12 (`--strict` → FAIL), RED-first.
- `retros/INDEX.md`: `dashboardpan-detail-render-doors` → `folded` (its last owed lesson B4 is now
  implemented; U1/U4/D3 were folded in Campaign 2). kit self-section: B4 moved owed → DONE (5 impls owed).

### References

- SDD change: `build-n4-module-continuity` Campaign 3 C3-PR1. New capability (WARN check + `--strict` flag) → MINOR.
- Engram: #8120 (Campaign 2 complete).

---

## [v0.9.4] - 2026-09-04

### Added — Campaign 2 close retro (promotion-process meta-lessons)

- `retros/2026-09-04-campaign2-promotion-process-meta-lessons.md` (+ INDEX row): captures the three
  process lessons Campaign 2 proved — (a) the fold-vs-revert (split-retro) rule; (b) the doc-vs-script
  folded-completeness rule (a script/gate lesson is only partly folded by documenting its rule; impl owed
  → `pending` + open_issue); (c) conservative adversarial fidelity grading catches folded over-claims a
  green bats suite alone misses (5 caught across the promotion PRs). The mechanism documenting its own
  process — the honest Campaign 2 close.

### References

- SDD change: `build-n4-module-continuity` Campaign 2 close. New-retro exit; documentation only → PATCH.
- Engram: #8120 (Campaign 2 complete).

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
