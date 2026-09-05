# build-n4-module-campaign6 — harvest (feeds the SDD proposal)

**Author**: investigador1 (Opus 4.8) · **Status**: COMPLETE. **Repo**: /home/cristian/modulos_niagara_n4/niagara-tools · **Kit**: build-n4-module-kit/ · main 48f3736 v0.15.1. NOT committed.

**Method**: mined the 8 pending kit retros + 9 named niagara-research module retros + the stale openspec change; excluded anything already present in a core file (cited `file:line`) or `folded` in `retros/INDEX.md` / the `BUILD-STATE.md` kit self-section. The kit is heavily folded (5 campaigns exhausted the 42-lesson mined corpus), so the genuine remainder is small and high-signal. Research-sdd/engram loop mechanics were excluded as belonging to the research kit, not the build kit. Evidence strength ∈ {proven live, proven by test, proven by code, opinion}.

---

## Group: `METHODOLOGY.md` — two NEW sections (highest-value picks)

| id | destination | source retro | one-line delta | evidence | est. lines |
|---|---|---|---|---|---|
| M1 | METHODOLOGY.md (new §Multi-session coordination) | dashboardpan-2d-to-3d-port Δ3 + module-authoring-mega-campaign Δ5 | Before editing any file in a shared client repo, `git status`/`git diff` the target first — a dirty tree is a peer live session's uncommitted work and is off-limits; coordinate or deliver read-only analysis, never clobber. | proven live (retro shows a peer's uncommitted CompPan work) | ~8 |
| M2 | METHODOLOGY.md (new §Live-verify safety) | live-cutover-and-authenticated-control Δ4 + obix-and-loginless-dashboard-runbooks D2 | During verification NEVER do a state-changing write on a production station; do read-only prod checks + an out-of-band negative check (no-token→401); take test creds from a file OUTSIDE the repo (`chmod 600`), never in a channel/artifact; cite a secret's STRUCTURE, never its value. | proven live | ~10 |

## Group: `METHODOLOGY.md` §"Kit maintenance — retro promotion discipline" — deferred meta-lesson cluster

The five pending *meta* retros each say "consider a one-line pointer here next time this section is edited — do NOT add speculatively." They are un-folded BY DESIGN, batched for the next edit. All LOW/MED value, LOW cost, evidence = opinion (process rules) unless noted.

| id | source | one-line delta |
|---|---|---|
| K1 | gate-exit-taxonomy-promotion | A gate's exit set must cover every legitimate change class; add a typed exit with its own proof-of-work guard, never stretch a label. |
| K2 | campaign3-close-process L1 | A new gate/verify check must be high-signal/low-false-positive — scope to the reported defect, mutation-prove the guard fires only on a real case. |
| K3 | campaign3-close-process L2 | When a lesson removes a safety guard, prefer a frictionless WARN that leaves a trace over silent removal. |
| K4 | campaign3-close-process L3 | When a retro's prose and its PROPOSED-delta marker disagree, the MARKER promotes; read the source verbatim, not from memory. |
| K5 | campaign4-close-process L1 | Run kit-coverage checks against a worktree off `origin/main`, never the local `main` (a campaign stale); ff-only sync after origin merges. |
| K6 | campaign4-close-process L3 | Grep EVERY kit file (and its `[ev:]` tag) for a lesson before folding — the mined target file may not be where it already lives. |
| K7 | campaign5-gate-activation L2 | A pure feature PR uses the close-gate NEW-RETRO exit (a); it is not a 4th unclassified gate shape. |
| K8 | ci-server-side-enforcement L3 | A gate/check resolves references against the repo (or a declared external), never `$HOME`/dev state; prove it bites under `HOME=/nonexistent`. |
| K9 | run-pure-test-set-e-empty-cache | A `set -e` toolbelt script that must REPORT a specific exit code on a guarded failure must isolate the probe (`\|\| true`) so its `die <code>` runs, not a bare abort. |
| K10 | ci-server-side-enforcement L2 | (dest CONTRIBUTING.md) Pin any linter/tool a CI gate depends on; an unpinned tool drifts newer and turns green→red with no code change. — MED/LOW |

## Group: `types/logic.md`

| id | source | one-line delta | evidence | est. lines |
|---|---|---|---|---|
| L1 | freeze-stat-leds Δ2 | §Regenerating slots — annotations-ONLY (no hand AUTO stub) is sufficient in the `build.sh`/slotomatic flow (slotomatic runs before javac); the AUTO stub is only for compiling WITHOUT slotomatic. Keep "annotation+stub" as belt-and-suspenders, not the requirement. | proven by test | ~6 |
| L2 | dashboardpan-2d-to-3d-port Δ4 | A control `BComponent`'s station MOUNT/ORD is integrator-placed config, NOT derivable from module source; state it's unknown and get it from live oBIX nav / the operator, don't fabricate a path. | proven live | ~4 |

## Group: `types/logic.md` (from CLOSED kitControl corpus — no new blocks, cite B536–B557/B552)

Per lead: angles 1+4 convert to kit-delta harvests. These promote already-verified control-programming findings into exemplar-backed `types/logic.md` patterns so it stops being "growing." All evidence = proven by code (decompiled Tridium source, cited block carries file:line).

| id | source block | one-line delta | evidence | est. lines |
|---|---|---|---|---|
| LC1 | B536 + B544 | Add a "writable-point priority array" pattern: 16-level first-valid-wins arbitration (WritableSupport.onExecute), relinquish-default = ordinal 17, null-status = relinquished; only levels 1/8 raise OVERRIDDEN; to make a config oBIX-writable add a NEW slot + link, never re-type (ties schema S1). | proven by code (B536/B544) | ~12 |
| LC2 | B539 | Add a "BLoopPoint / PID" exemplar: ramp = rate limiter + ramp-aware anti-windup, executeTime clamp [100ms,60s], direct=cooling/reverse=heating, disableAction bumpless pre-load, official tuning kP=(maxOut−minOut)/throttlingRange, PI recommended / PID seldom. | proven by code (B539) | ~12 |
| LC3 | B537 | Add the "multi-input null contract" rule: nulls are SKIPPED not zeroed (BQuadMath nonNullCount, BAnd nullOnInactive); latch = rising-edge vs latch-action both-edges; switch/select invalid → hold + invalid-flag. | proven by code (B537) | ~8 |
| LC4 | B543 | Add a "control-logic fail-safe checklist" (refines the existing rt-hardening fault-posture rule): the 6 unsafe-unless-configured defaults — disableAction=hold freezes last command, propagateFlags=0 lets a bad sensor drive the loop with no fault, rampTime=0 = no anti-slam, alarm-ext is notification-only (no interlock), 999 sentinel is convention not framework, clHVAC strips Baja status. Operator recs: set disableAction/Fallback safe, propagateFlags=fault\|stale\|down, rampTime>0, emergency L1 for interlocks. | proven by code (B543) | ~14 |
| LC5 | B552 | Add the "point-extension chain (alarm+history)" pattern (covers angle 4): BAlarmSourceExt→offnormal/fault algorithm (BOutOfRange/FloatingLimit/TwoState) + AlarmService; BIntervalHistoryExt (timer, default 15min) vs BCovHistoryExt (COV changeTolerance deadband); alarm-ext NEVER writes value/priority-array — it is notification-only. | proven by code (B552) | ~10 |

## Group: `types/dashboard.md`

| id | source | one-line delta | evidence | est. lines |
|---|---|---|---|---|
| D1 | dashboardpan-2d-to-3d-port Δ2 | Off-station consumer nuance: servlet-DERIVED keys (`*ElapsedMs`/`*RemainingMs`) are computed `now−anchor`, NOT oBIX slots — an off-station poller must read the ANCHOR slots (`coolingSince`/`defrostStart`/`defrostDuration`/`nextDefrostTime`) and recompute. Read `DashboardReader` to tell a real oBIX slot from a reader-derived value. | proven by code | ~6 |

## Group: `build-verify.md`

| id | source | one-line delta | evidence | est. lines |
|---|---|---|---|---|
| BV1≡N1 | this audit + v0.2-retro P2 (verified vs ng-deploy.sh:493-500,552-553,60) | RETITLE §"Known gap" to "mode B ignores `--with-slotomatic`": modes A/C regenerate `-ux` slots when `-ux` is annotated (run_slotomatic 493-500); mode B never runs slotomatic (552-553), so a `-ux` annotation edit needs `--mode A --with-slotomatic` or `toolbelt/build.sh`. The blanket "-rt only" wording is stale for A/C, still true for B. | **proven by code** | ~4 |
| V1 | dashboardpan-2d-to-3d-port Δ1 | Consumer-absence delta proof: to prove "what changed since X" when same-day timestamps don't discriminate, `grep -c <symbol>` the CONSUMER artifact — 0 hits = genuine delta. | proven by test | ~4 |
| V2 | live-cutover Δ2 + rt-authoring-campaign Δ5 | Live-vs-doc precedence: for behavior the live system arbitrates (is a point writable?), verify live and let it OVERRIDE the doc, then fix the doc (PORT-SPEC said setpoint writable; live oBIX PUT → 400). | proven live | ~4 |
| V3 | live-cutover Δ1 | Verify freshness before labeling "live": probe max `ts` is advancing ≈ now; if frozen, label honestly (SNAPSHOT / última lectura `<ts>`). | proven live | ~4 |
| V4 | live-cutover Δ3 | Headless-QA/CORS boundary: a headless-from-localhost e2e can't cross a browser CORS origin by design — confirm the backend with `curl` + verify allow-origin separately; a CORS block ≠ a code bug. | proven live | ~4 |

## Group: `toolbelt/verify-module.sh`

| id | source | one-line delta | evidence | est. lines |
|---|---|---|---|---|
| T1 | v0.2-retro P3/D4 | Add `--plano <index.html>`: assert exactly one frame `aspect-ratio` declaration AND the four plano values agree (rule is doc-only at dashboard.md:54 today). | opinion (rule proven, check new) | ~20 |

## Group: `toolbelt/` (new `preflight.sh`) or `build-verify.md`

| id | source | one-line delta | evidence | est. lines |
|---|---|---|---|---|
| P1 | v0.2-retro P4 | Add `preflight.sh <niagara_home>` automating BUILD-LOOP §0.b: JDK 8 present, `etc/m2` plugin vs `settings.gradle.kts` pin, station lock on the target jar, WSL `/mnt/c` path form — §0.b is prose only and every item was a real failure. | opinion (each item proven live) | ~60 |

## Group: `SOURCES.md` / `corpus-index.md` (research-tooling — marginal to building)

| id | source | one-line delta | evidence | est. lines |
|---|---|---|---|---|
| S1 | rt-authoring-campaign Δ4 + mega-campaign #6 | Tool-zero ≠ absence: a `corpus-nav`/`mem_search` zero is not proof of absence — run a control query or fall back to `rg`/`mem_context`/the code before concluding "not found." | proven live | ~4 |
| S2 | rt-authoring-campaign Δ3 | Never `[CERT]` a claim off an `ln`/`n`-mangled decompiled body; prefer the vineflower/procyon tree, else mark `[INFER]` or decline. | opinion | ~3 |

---

## CONTRADICTIONS flagged

- **BV1≡N1 (build-verify.md:115-116 vs ng-deploy.sh):** the doc asserts a blanket "-rt only" slotomatic gap that the code has CLOSED for modes A/C (real only for mode B). Doc-vs-code drift — retitle to the mode-B-scoped gap. This is exactly the "doc drifted from script" defect the kit's own promotion discipline targets.
- **FOCUSES.md vs RESEARCH-STATE-kitControl.md (fixed this session, commit bbc2f7968 in niagara-research):** FOCUSES said "planned (0/12)"; RESEARCH-STATE said stopped/investigable=0. Kit-side lesson for M-cluster: a sweep should flag FOCUSES-vs-RESEARCH-STATE drift (candidate K-lesson / research-kit delta, not build-kit).
- No delta-vs-delta contradictions within the backlog (all additive; LC-group cites closed research, does not conflict with existing logic.md content).

---

## Input (4) — stale openspec change `niagara-tools-slotomatic-integration` (May 2026, 40 tasks)

**VERDICT: 100% SUPERSEDED. ARCHIVE, do NOT apply. Zero live tasks.** Every proposed flag/function/file/exit-15 is live in `scripts/ng-deploy.sh`; `.env.local.example`, `docs/knowledge-base/slotomatic.md`, `docs/GOTCHAS.md`, `tests/smoke-checklist.md` all exist; `ng-deploy.bats` has 37 tests (change wanted 26); repo is v0.15.1 (change targeted v0.3.0). Recommend a one-line proposal task: move `openspec/changes/niagara-tools-slotomatic-integration/` → `openspec/changes/archive/` with a supersession note. (Lead confirmed; he moves it under the campaign6 hybrid-store commit.)

---

## Math-models section (separate + short) — quantitative models worth prototyping for THIS build kit

Build-kit models only (gate/coverage/schema-risk); research-corpus models (embeddings/PageRank/bandits) are OUT of scope here. Each is a **pure, deterministic function** → biteable with exact-value mutation tests (QA's constraint), separated from I/O.

| # | Model | Pure-function shape | Why (one sentence) | Priority |
|---|---|---|---|---|
| MM1 | Verify-gate coverage % | `coverage(covered:int, total:int) -> pct` + classifier of what counts as "covered" (checks run AND passed / applicable) | Turns the gate's binary PASS/FAIL into a graded number so a jar that skipped opt-in checks (`--src`/`--stored`/`--target-version`) is visibly less assured. | **HIGH — RED first (QA)** |
| MM2 | Exposed-type set-coverage (palette/lexicon/type-registration) | `set_coverage(declared:set, required:set) -> pct` over the exposed `@NiagaraType` set | The three silent-deploy footguns — empty palette passes the gate (B5), missing lexicon key → camelCase (T8), dangling module-include type (B12) — are all `\|declared ∩ required\|/\|required\|`; one metric catches all three before deploy. | HIGH |
| MM3 | Schema-change survival risk classifier | `schema_risk(slot_diff) -> {SAFE,LOSSY,OUTAGE}` = max severity over add/remove/rename/retype | The SAFE/LOSSY/OUTAGE saved-data matrix (S1/S2/S3) is a decision table over a slot diff; a deterministic pre-deploy risk class from two module.xml snapshots prevents the ClassCastException boot-loop that killed a live rack. | HIGH |
| MM4 | 4-layer assurance-stack completeness | `assurance(pure_junit, gate, live_smoke, adversarial) -> fraction` (gated on "has decision/safety logic") | Makes the qualitative "done = all four layers ran" auditable, surfacing e.g. DefrostController's 742 lines with 0 pure tests as a hard 25%. | MED |
| MM5 | Risk-weighted checklist / expected-defect-escape | `escape(unchecked_items, severity_map) -> Σ severity` | The ~20-item checklist maps each item to a defect class with a known live severity (0-limit killed a rack, slot-retype = outage, missing lexicon = cosmetic); weight by historical severity, not flat count. | MED (parse-fuzzier → 2nd per QA) |
| MM6 | Retro-fold burndown / staleness | `fold_debt = Σ over pending retros of age_days × delta_count` | One number for un-folded lesson debt (today: 8 pending in INDEX.md); mirrors research-sdd's sweep ESCALATED aging. | LOW |

**MM1 contract decision (answering QA):** `coverage(covered,total)` returns a float in [0,100] for `total>0`; for `total==0` it returns a distinct **N/A sentinel, NOT 100** — "no applicable checks" and "all applicable passed" are different states, and reporting 100% for an empty denominator manufactures false confidence and would hide a scaffold-only module (the empty-palette footgun MM2 targets). A mutation returning 100 for 0/0 must flip the assertion.

Prototype order: **MM1 → MM2 → MM3** (each directly catches a documented live footgun), then MM4/MM5. MM6 nice-to-have.

---

## Appendix — excluded as ALREADY-FOLDED (dedupe audit)

- **freeze-stat-leds** (INDEX pending, 3/4 folded): Δ1 lock/mirror → BUILD-LOOP.md:20 + build-verify.md:65-66; Δ3 BDouble.make + import → METHODOLOGY.md:9,11; Δ4 junit glob → build-verify.md:106 + run-pure-test.sh:37-40. Only Δ2 (L1) un-folded.
- **v0.2-retro**: P1 → build-verify.md:13; P5 → .githooks/pre-push + install-hooks.sh; P6 → INDEX.md + kit-links.bats; §1 → build-verify.md:5-13; D3 → build-verify.md:45,51-57; D4 → dashboard.md:54; D6 → build-verify.md:10,80,82; D7 → build-verify.md:72; D2 → logic.md:13. Un-folded: P2(BV1/N1), P3/D4(T1), P4(P1).
- **rt-authoring-campaign Δ6** → logic.md:58-61 + 4-layer stack. **mega-campaign #8** ("pin cached junit path") → build-verify.md:106.
- **kit-continuity-and-retro-gate-campaign** (INDEX pending) — its deltas (fold L3-L22/U1-U9/B4-B10/D1-D3/S2-S4 + corpus-index T7) all folded; BUILD-STATE says CAMPAIGN 4 COMPLETE (42-lesson corpus exhausted). Content-complete.
- **campaign4 L2** (partial-promotion BUILD-STATE anchor) → BUILD-LOOP.md:61 exit (c). **run-pure-test-set-e / campaign5 / ci-server-side** structural deltas → shipped code; only generalized RULES (K7,K8,K9,K10) remain.
- **Out-of-scope (research-kit loop mechanics, not build-kit):** mega-campaign Δ1/2/3/6/7; obix-runbooks engram-mirror + `#id`-not-CERT-live; module-best-practices + module-dev-workflow focus retros (ZERO in-scope rows — both are PROMPT-LOOP synthesis conventions); coldroom-build Δ1/2/4/5. Parked low-value process: v0.2 P7 (role template), D5 (fixture escape-byte), coldroom B721-B723 (mostly folded via build-verify §Signing / METHODOLOGY:15), rt-campaign Δ7 (incident-mining tier).

**Highest-value un-folded picks:** M1 (multi-session git-status guard) + M2 (live-verify write/secret safety) — HIGH value, LOW cost, zero footprint in the kit today; plus the LC1–LC5 kitControl harvest (turns logic.md from "growing" to exemplar-backed) and BV1 (doc-vs-code fix).
