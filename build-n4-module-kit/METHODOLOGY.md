# Common checklist — every N4 module (the verify gate)

Applies to all module types. Each item is proven from real builds (DashboardPan/ColdRoomPan/chihuahua, 2026-08). Run this against the built module before "done".

> Background reading, by layer and priority: **`corpus-index.md`** — the curated map of the niagara-research authoring corpus (B729–B760). `corpus-nav FIRST` for a term; the index for what to read (P0 before building).

## rt (components)
- [ ] **Slot flags** on every `@NiagaraProperty`: `Flags.SUMMARY` for visible; `SUMMARY|OPERATOR` for operator-writable config; `TRANSIENT|SUMMARY|READONLY` for computed outputs; `HIDDEN` for internal. (control-rt `BNumericWritable` is the exemplar.)
- [ ] **Facets / units.** Temperatures carry a Celsius facet: `BFacets.make(BFacets.UNITS, BUnit.getUnit("celsius"))`. A MIN/MAX facet wraps its number in `BDouble.make(...)` — `BFacets.make(BFacets.MIN, 0d)` does NOT compile (no `make(String,double)` overload).
  - **delta vs absolute:** a *difference* (e.g. a hysteresis `differential`, a deadband) is in *degrees*, not absolute Celsius (offset 273.15). Show it as a band, label it as such.
- [ ] **@NiagaraProperty edits in 3 places:** the annotation `@Facet(...)`, the generated `newProperty(...)` region, AND the imports. Prove with `slotomatic` (it regenerates from the annotation — a fix only in the generated region reverts on the next regen). `import javax.baja.sys.BDouble` if the annotation uses `BDouble.make`.
- [ ] **Status flags.** Read/propagate `BStatus` (fault/down/stale/disabled/null/overridden/alarm) — do not collapse everything to null. Guard control decisions on `getStatus().isValid()`.
- [ ] **module.lexicon** populated (type + user-facing slot display names); format `key=value` (see control-rt.lexicon). A MISSING key silently renders the raw camelCase via `toFriendly` — every exported type + slot needs one, or the slot shows its bare name in Workbench/the property sheet. Keys are module-global, so shared slot names collide. [ev: retro corpus-index · T8]
- [ ] **Icon:** a 16×16 PNG in `src/rc/` + `getIcon()` returning `BIcon.make(BOrd.make("module://<mod>/rc/icon16.png"))`.
- [ ] **Permissions:** delete the empty `type="all"` wizard scaffold in module-permissions.xml → `<permissions/>` (a component/dashboard module needs the base grant). [module-anatomy B636 deviation #1]
- [ ] **`module.palette` has one `<p n=… t=…>` entry per exposed `@NiagaraType`:** a scaffold-only palette (just `<p t="b:Folder">`) passes the ENTIRE verify gate and deploys, yet Workbench shows nothing to drag — commissioning is silently broken. [ev: retro module-palette · B5]
- [ ] **A self-armed timer arms in BOTH `started()` and `atSteadyState()`** — never only `atSteadyState()` (it skips a late commissioning mount and the timer never fires; the anti-pattern has ZERO hits in the entire Tridium first-party corpus). Full idiom in `types/logic.md` §Safety fail-modes. [ev: retro self-firing-timer]

## Domain correctness
- [ ] Compare a value only against a limit that APPLIES to it: setpoint/deviation/alarm belong to zone sensors, NOT to evaporator/resistance temps (those alarm against their own high/low limits). Wrong comparisons = false alarms.
- [ ] Times shown legibly (hours/minutes), not raw seconds/ms.
- [ ] **`0` on a protection limit means DISABLED, not "worst case":** a limit slot defaulting to `0`, compared `valid && value > limit` against a sensor whose physical range sits entirely on one side of `0`, makes the interlock permanently block (rack never starts) or permanently inert (protection never trips). It appeared 3× and killed a live rack. Write `trip = limit != 0 && valid && value > limit`, keep symmetric limits consistent, and add an explicit `limit == 0` test case. [ev: retro default-zero-limit · L2]

## Schema / upgrade safety
- **NEVER retype an existing slot on a live-instantiated module** (a complex `BStatusNumeric` → a simple `double` especially): a `.bog` binds to the class BY NAME and ungated, so retyping a frozen slot desalinates decoding — `ClassCastException` / `Cannot load station` / `App Failed`, a real refrigeration-production outage. ADDING a slot is safe; RETYPING one is not — to make a config oBIX-writable, add a NEW `double` slot (e.g. `setpointCmd`) and link it, do not re-type the original. [ev: retro slot-type-change · S1]
- **Know the saved-data survival matrix before a schema-change deploy (a `.bog` binds to the class BY NAME, ungated):** SAFE (boots, data kept) = add a slot, reorder, change default/flags/facets, add an enum tag; LOSSY (boots, data dropped with only a warning count) = remove/rename a slot; OUTAGE (won't boot) = retype a frozen simple slot, remove/rename a stored enum tag. There is no per-module migration hook and "it booted" ≠ "data survived" — bump `vendorVersion` and back up `config.bog` on every schema-change deploy. [ev: retro corpus-index · S2]
- **An additive slot change preserves instances/links (keyed by name), BUT the JACE won't auto-install it without a version bump:** new frozen slots load at default on the existing bog; Software Manager compares versions, so a same-version jar shows "Up to Date" and is never offered — bump the module version (or force Re-Install). [ev: retro coldroompan-fan-mode-defrost · S3]
- **When you DELETE a `@NiagaraType` class, delete its `module-include.xml` registration line in the SAME change:** the gate's `types` check catches a dangling registration only on an actual REBUILD, so a stale committed jar hides it — a leftover surfaces live as `Missing class <Module>:<Type>`. [ev: retro coldroompan-fan-mode-defrost · B12]
- **A NEW safety/startup slot defaults to the SAFE posture, not to the pre-fix behavior:** a brand-new slot has no operator setting to protect, and "preserve current behavior" is wrong when the current behavior IS the hazard (e.g. every evaporator energizing at once). Default it SAFE via a sentinel (`0 = AUTO` computes a safe value) + an explicit override; reserve "default to current behavior" for a change to an already-commissioned control path. [ev: retro soft-start · L15 — refines the rt-hardening sensor-fault-posture rule in `types/logic.md`]

## Editing technique — asset-laden single-file artifacts
- **A dashboard `index.html` with embedded base64 art (photos/planos on multi-MB lines) defeats the Read tool — navigate with range-scoped `sed` + a line-length filter from the start:** `sed -n 'A,Bp' file | cut -c1-160` for a known range; to search, skip the giant lines with `awk 'length<300'` or a tiny python `for i,l in enumerate(f): if len(l)<300 and pat.search(l)`. Re-verify every edit with `node --check` on the extracted `<script>` blocks. [ev: retro hmi-touch-ux Δ8]
- **When a prior version renders correctly and the new one doesn't, diff the two to isolate the one stray value — same technique, hunt the inconsistency:** a known-good reference build localizes the single desynced value faster than re-deriving it. [ev: retro hmi-touch-ux Δ11]
- **To EDIT (not just read) an asset-laden single-file SPA, use an anchored Python `str.replace` that asserts the anchor occurs exactly once before writing the whole file back — `Read`/`Edit` fail even with offset/limit because one base64 line is enormous:** `sed -n 'A,Bp'` to view a region, then the all-or-nothing Python replace (`assert content.count(anchor)==1`), then `node --check` on the extracted `<script>`. [ev: retro editing-base64-heavy-spa · U8]

## Build (see build-verify.md)
- [ ] Built with **Java 8** + **clean + slotomatic + jar**. Bytecode major version **52**.
- [ ] Both/all profile jars **signed** (`META-INF/NIAGARA4.SF`).
- [ ] Tests: unit-test the pure-Java model (e.g. a pure router) with JUnit — `niagaraTest` does not run in WSL.
- [ ] `toolbelt/verify-module.sh` passed on the built jars.
- [ ] **The 4-layer assurance stack ran** for any decision/safety logic: pure JUnit (`toolbelt/run-pure-test.sh`) → the verify gate → a live cold-boot smoke → an adversarial pure-logic review. Pure tests are mandatory for decision/safety logic; `niagaraTest` is documentation, not a WSL gate. Detail in `build-verify.md`. [ev: retro qa-stack · T1]
- **What to test, where — by module type** [ev: corpus B743/B12]:

  | Module type | Pure JUnit (WSL) | Verify gate | Live smoke |
  |---|---|---|---|
  | rt (logic) | Math seam (time as param); scheduler-seam DI for arm-path + cancel (`Sched{at(delayMs);cancel(t)}`) | `verify-module.sh` | Cold-boot: anchor populates, timer fires |
  | dashboard (ux) | Pure model/router (no Baja runtime needed) | `verify-module.sh` | oBIX probe; RBAC write smoke |
  | wb-widget | `BTestNgStation` needs kernel+license — not WSL-runnable; no pure seam equivalent today | `verify-module.sh` | WB mount + property-sheet smoke |

## Tradeoffs to state, not hide
- Adding alarm sources / control points to a "pure display" facade makes it an alarm SOURCE — a real change of role. Flag it.

## Debugging
- **On a log WARNING, first classify caught-and-cosmetic vs propagates-and-aborts before treating it as the root cause:** does a `changed()`/`try` swallow it (log-and-continue, not re-thrown)? A captured stack trace shows the full path up to `Station.startStation` even when the throw is caught — that depth alone is NOT evidence of a failure. Example: an `applyRunCmd` `NotRunningException` during `activateLinks` looked like the dead-defrost cause but was cosmetic (caught by `changed()`); the real cause was a missing `started()`. Separate noise from cause before investing effort. [ev: retro self-retro-preview-gate · T6]

## Kit maintenance — retro promotion discipline (not a per-module build step)
- **Doc-vs-script folded-completeness:** a retro lesson whose whole content is a rule/checklist item is FULLY folded by documenting it (INDEX row → `folded`). A lesson that asks for a SCRIPT or GATE behavior change is only PARTLY folded by its prose — the implementation is still owed, so its row STAYS `pending` and the impl is logged as a `kit` self-section `open_issue` in `BUILD-STATE.md` (a tracked future MINOR PR). Never mark a retro `folded` just because its prose landed. [ev: retro campaign2-promotion-process-meta-lessons · meta-lesson 2]
- **Adversarial fidelity grading beats a green suite:** a promotion is a documentation act whose correctness is faithfulness-to-source + fold-completeness — neither is a runtime property, so a passing bats gate cannot verify it. Budget an independent per-lesson fidelity pass on every promotion PR (diff each fold against its source retro); it catches folded over-claims a green suite misses. [ev: retro campaign2-promotion-process-meta-lessons · meta-lesson 3]
- **K1 — Gate exits cover every change class:** a gate's exit set must cover every legitimate change class it will see; when a real workflow fits no exit, add a typed exit with its own proof-of-work guard — never stretch an existing label. [ev: retro gate-exit-taxonomy-promotion]
- **K2 — High-signal, low-FP gate checks:** a new gate/verify check is scoped to the reported defect (not a blanket completeness rule); mutation-prove it bites only on a real case. [ev: retro campaign3-close-process-meta-lessons L1]
- **K3 — WARN over silent guard removal:** when a lesson removes a safety guard, prefer a frictionless WARN that leaves a trace over silent removal. [ev: retro campaign3-close-process-meta-lessons L2]
- **K4 — Marker over prose:** when a retro's prose and its PROPOSED-delta marker disagree, the marker promotes; read the source retro verbatim, not from memory or a mining note. [ev: retro campaign3-close-process-meta-lessons L3]
- **K5 — Coverage checks against origin/main worktree:** run kit-coverage checks in a worktree off `origin/main`, never against a stale local `main`; ff-only sync after each origin merge. [ev: retro campaign4-close-process-meta-lessons L1]
- **K6 — Grep every kit file before folding:** grep every kit file (and its `[ev:]` tag) for a rule before folding — the mined target file is a suggestion; the rule may already live elsewhere. [ev: retro campaign4-close-process-meta-lessons L3]
- **K7 — Feature PR = exit (a), not a 4th shape:** a pure feature PR uses the close-gate NEW-RETRO exit (a); it is not a 4th unclassified gate shape. [ev: retro campaign5-gate-activation L2]
- **K8 — No $HOME coupling in gate checks:** a gate/check resolves references against the repo (or a declared external), never dev-machine state (`$HOME`); prove it bites under `HOME=/nonexistent`. [ev: retro ci-server-side-enforcement L3]
- **K9 — set -e probe isolation:** a `set -e` toolbelt script that must report a specific exit code must isolate its probe (`|| true`) so `die <code>` runs instead of a bare abort. [ev: retro run-pure-test-set-e-empty-cache]

## Multi-session coordination
- **Before editing a file in a shared repo, check the tree first:** run `git status`/`git diff` on the target — a dirty working tree is a peer's live uncommitted work and is off-limits; coordinate with the owning session instead of duplicating or clobbering. [ev: retro dashboardpan-2d-to-3d-port, retro research-sdd-module-authoring-mega-campaign]

## Live-verify safety
- **Never perform a state-changing write on a production station during verification:** do read-only prod checks + an out-of-band negative check (no-token → 401). [ev: retro live-cutover-and-authenticated-control, retro obix-and-loginless-dashboard-runbooks]
- **Test credentials from a file outside the repo** (`chmod 600`), never pasted in a channel or embedded in an artifact; cite a secret's structure (filename, format, purpose), never its value. [ev: retro live-cutover-and-authenticated-control]
