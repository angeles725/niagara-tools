# Proposal: build-n4-module-campaign11

**Status**: proposal · **Phase**: propose (post-explore, pre-proposal handoff CONFIRMED)
**Source**: niagara-tools `v0.21.0` (tag `dab0807`, C10 archive `154803c`) · **Target**: kit `v0.22.0` (MINOR)
**Client**: `angeles725/niagara-panccadia-leon` — main `00e7118` (= `ff1b659` + PR6); read tree = `Leon-Guanjuato-worktrees/main-ff1b659` (CompPan-rt 2.2.0, ColdRoomPan-rt 2.1.0, DashboardPan rt/ux 2.2.0)
**Inputs**: `openspec/changes/build-n4-module-campaign11/explore.md` (gate-passed; §1.1 items 1-6 are constraints; investigador1 second read PASS `6f0069155`) · research **B832** (`593019540`, investigador1) · apply-packages `4ef4f864c` / `8ad4bb36e` / T2 re-cut `0ad09c658` (tip `4ef9726e7`) · QA C11 RED branches · `METHODOLOGY.md` **K24** @ `dab0807`
**Topic key**: `sdd/build-n4-module-campaign11/proposal`
**Delivery**: auto-chain, ONE wave (W1), 5 PRs · review budget 400 changed lines/PR (**PR1 requests `size:exception`**)

---

## 1. Intent

C10 made three lints precise. C11 pays the bill for having *three copies* of the code that decides where a Java method begins and ends, plus three smaller ways the kit can be wrong without anyone noticing. WSL-only, nothing from Cristian.

**First, one false negative is shared by two lints.** The section-D method-boundary parser exists three times: `lint-timers.sh:188-202`, `lint-silent-protection.sh:302-364`, `lint-ext-writable-shape.sh:132-176`. The first two gate the method open on **NET** brace depth and therefore never see a one-liner — `void arm(){ flag=true; Clock.schedule(...); }` closes on the same line, net depth returns to 0, the body is never entered. The third uses **PEAK** depth and catches it. Same rule, three implementations, two verdicts. `[ev: B832 593019540]` `[ev: QA qa/c11-parser-oneliner d88af78]`

**Second, the divergence is silent today and will not stay silent.** Across the **42 `.java` files at `ff1b659`** there are **zero** one-liner methods carrying a schedule / alarm / trip write, so nothing flips and no gate goes red — the defect is invisible until an operator writes one line of ordinary Java. That also means the one-liner behaviour can only ever be pinned by golden fixtures, never by a real-tree count. Worse, `lint-silent-protection.sh:303-307` documents the NET-depth skip as *intentional and correct*, so the divergence currently reads as doctrine. Three copies also drift independently: every future refinement must be applied three times, and K24(6) had to be written because one copy misparsed `@NiagaraProperty(` as a method. `[ev: B832 §shipping]` `[ev: investigador1 2nd read 6f0069155]` `[ev: METHODOLOGY K24(6) @ dab0807]`

**Third, ten test sites hardcode an absolute client path, and three of them read a stale tree.** `tests/*.bats` carries the literal `…/Cliente/Leon-Guanjuato…` at 10 places: 5 × `C9_CLIENT_ROOT` (ext-writable-shape:26, demand-in-scope:27, lint-silent-protection:30, lint-timers:418, lint-write-path:338), 2 × `C9_CLIENT_REPO` (c9-close:108, c10-close:90), and 3 that read the **live checkout** — `c8-close:107` (`C8_CLIENT_REPO`), `lint-delays:53` and `rc-scan:75` (no override at all, a bare `$HOME/…/Leon-Guanjuato/…`). The live checkout is `4f5f1c7`, a pre-C9 tree with Cristian's uncommitted files. Verified by grep at the C11 base. `[ev: lead grep 2026-09-07, 10 sites]` `[ev: QA qa/c11-client-root 54078f6]`

**Fourth — the new finding — one of those smokes pins a BUG, not a rule.** `lint-delays` smoke **LD5** (and `c8-close` SC1-smoke) assert `exit 1 + FAIL BDefrostController`. That is true only on the stale `4f5f1c7`: it is the ColdRoomPan `defrost time <= 0` defect, fixed after C9. On `main-ff1b659` the lint is **clean, exit 0**. A real-tree smoke that asserts a FAIL rots the moment the bug is fixed — and until then it reports green for the wrong reason. The delay-floor **rule** survives the retarget because `LD1`/`LD3`/`LD6` already pin it on synthetic fixtures, so LD5 simply flips to clean and no new fixture is owed. `RC8` (rc-scan, DashboardPan-ux `:701` host literal) is 1 FAIL on both trees, no delta. `[ev: lead re-measure 2026-09-07 on main-ff1b659]` `[ev: investigador1 2nd read 6f0069155]` `[ev: memory coldroompan-defrost-time-le-0-bug]`

**Fifth, two markers can go stale in silence.** A `[concept]` row in `docs/write-path-matrix.md` is skipped unconditionally by the S25 pass (`lint-write-path.sh:441`, `case "$_row" in *'[concept]'*) continue`), so once that concept becomes a real slot the exemption is a lie nobody sees. And every lint header names the OBSERVED mutation that proves it bites (K24(7)), but nothing checks that the named fixture still exists — a renamed or deleted fixture leaves a guard pin pointing at nothing. `[ev: lint-write-path.sh:422-458 @ dab0807]` `[ev: 8ad4bb36e §T4]` `[ev: K24(7)]`

---

## 2. Scope

### 2.1 In scope — W1, kit-only, WSL-only, needs nothing from Cristian

| R-id | Item | Repo | Why now |
|---|---|---|---|
| **R1** | **T1 — shared method-boundary parser (keystone, ONE PR).** NEW `toolbelt/lib/method-boundary.sh` (awk-in-shell-var, the kit idiom), sourced by all three lints. Canonical body = `lint-ext-writable-shape.sh:132-176` (PEAK depth). Replaces `lint-timers.sh:188-202` and `lint-silent-protection.sh:302-364` — **including the NET-depth rationale comment at `lint-silent-protection.sh:303-307`**, which claims the single-line skip is correct ("preventing the method-open event from being attached to a post-close depth that spans until the class closes"). That fear is already handled by the `brace_depth >= 2` guard plus close detection, so the comment must be rewritten, not left contradicting the code. Invariants: `brace_depth >= 2` class-FIELD guard; Case-B backward scan stops at any line starting with `@`; keyword exclusion (`if/for/while/switch/catch`); PEAK depth; **NEW** `get\|set\|is` one-line accessor skip (B832-G1); `/* */` strip in the Case-B scan (B832-G2) | kit | One live false negative in two of three copies, and a 3× cost on every future refinement. Must cut all three at once or the toolbelt is internally inconsistent. `[ev: B832]` `[ev: 4ef4f864c §T1]` `[ev: investigador1 2nd read 6f0069155]` |
| **R2** | **T3 — concept-row-drift advisory.** NEW verdict class `DRIFT` in `lint-write-path.sh`, at the S25 row pass `:422-458`: a `[concept]`-marked row whose backtick-inner slot name **IS** now in the matrix-root-wide covered set (`@NiagaraProperty` ∪ `@NiagaraAction` ∪ `--bog`) is a stale marker. STATUS-first row grammar + matrix line, default exit **0**, `--strict` → **1**, same flag as STALE; STALE rows and the uncovered **FAIL exit 1 untouched**; exit 3 preserved | kit | The exact inverse of STALE, in the same lint, same file, same flag. A true concept row (name absent from source) stays silent. `[ev: 4ef4f864c §T3]` `[ev: K24(2)(3)]` |
| **R3** | **T2 — centralise client-tree defaults.** NEW `tests/lib/client-root.bash` owning **ONE** default `CLIENT_READ_ROOT` = `Leon-Guanjuato-worktrees/main-ff1b659`, exported as `C9_CLIENT_ROOT`, `C9_CLIENT_REPO` and `C8_CLIENT_REPO`; env override wins. All **10** sites converted; `lint-delays:53` and `rc-scan:75` gain the override form they lack. **LD5 and c8-close SC1-smoke are re-pinned to the ff1b659 clean state**; no new fixture is owed — the delay-floor rule is already pinned by `LD1`/`LD3`/`LD6` | kit/test | Three suites currently read a pre-C9 tree with uncommitted files. That is a rule violation, not a feature. `[ev: 0ad09c658]` `[ev: QA 54078f6]` `[ev: investigador1 2nd read 6f0069155]` |
| **R4** | **T4 — guard-pins meta-check.** NEW `toolbelt/lint-guard-pins.sh`: every OBSERVED mutation named in a lint header (`# Mutation:` contract) must map to an existing bats fixture name in `tests/*.bats`. WARN-only default (exit 0), `--strict` → 1, usage → 3 (K20 disjoint); D9b dot-dir prune; K19 routing in `BUILD-LOOP.md` + `skill/SKILL.md` | kit | K24(7) is doctrine but unenforced; C10 caught three unpinned guards by hand. Exact header grammar is fixed in `sdd-design`. `[ev: 8ad4bb36e §T4]` `[ev: K24(7)]` |
| **R5** | **C11 close.** `tests/c11-close.bats` (`C11_CLOSE=1`), `VERSION` `0.21.0` → **`0.22.0`** + CHANGELOG, retro folds to pending = 0, `BUILD-STATE.md` envelope, `retros/INDEX.md` rows | kit | Hard close gate (BUILD-LOOP §7). BASE pin `dab0807`; tool-pins gain `lib/method-boundary.sh` + `lint-guard-pins.sh` |

### 2.2 Out of scope — explicit, with the gate that would open each

| Item | Class | Gate that must resolve first |
|---|---|---|
| **P1** viewer per-user re-auth + configurator role list | tunnel product | Tunnel PRs #1/#2/#3 merged by Cristian + Supabase operator re-check `[ev: explore §4 W2]` |
| **P2** HMI per-operator screens (RBAC/VIEW) | client -ux product | Cristian confirms the need — attribution already shipped as R14 — + harness session green `[ev: 29d203e3d]` |
| **P3** `airDefrost` flag (rooms 1/2/4) | client -rt product | Defrost-trial green light `[ev: explore §4 W2]` |
| **P4** intercambiador Cuarto 3 control point | station + client | Cristian confirms it sits on a Niagara output `[ev: explore §4 W2]` |
| **P5** `coolOnSensorFault` station-side link | station (Workbench) | Cristian approves the link `[ev: explore §4 W2]` |
| **Deploy chain** (2.0.7 / 2.0.3 / 2.1.1, then the C9/C10 jars) | prerequisite | Cristian executes the runbook; the station is four repo versions behind. NOT C11 work `[ev: before-c11-checklist 4ef4f864c]` |
| **niagaraTest harness session** (`qa/c9-harness-runsheet.md`) | prerequisite | Runs on Cristian's Windows box; closes C9/C10 CLOSE-harness-run. NOT C11 work `[ev: before-c11-checklist]` |
| **Tunnel PR #1/#2/#3 merge** | prerequisite | Cristian merges; C11 authors no tunnel code `[ev: explore §6.2]` |
| **Any live station WRITE or jar deploy** | — | Only with Cristian's direct authorization. No C11 slice performs one |

---

## 3. Capabilities

> `openspec/specs/` does not exist in this repo. Per the campaign-6…10 convention, `sdd-spec` writes one change-local `spec.md`.

### New Capabilities

- `kit-method-boundary-parser`: the single shared section-D method-boundary contract (`toolbelt/lib/method-boundary.sh`) — PEAK depth, `brace_depth >= 2` class-FIELD guard, Case-B `@`-stop, keyword exclusion, one-line accessor skip, `/* */` strip. Consumed by three lints; the verdict of each is defined against this one implementation. `[ev: B832]`
- `module-guard-pin-lint`: `lint-guard-pins.sh` — every OBSERVED mutation named in a lint header maps to an existing bats fixture. WARN-only default, `--strict` → 1, usage → 3. `[ev: 8ad4bb36e §T4]`

### Modified Capabilities

- `module-timer-lint`: method-boundary detection moves from local NET depth to the shared PEAK-depth fragment. A companion flag + `Clock.schedule*` inside a **one-liner** method body is now FAIL (was silently clean). Row grammar and exit codes (0 / 1 any FAIL / 3 usage) unchanged. `[ev: B832]` `[ev: QA d88af78 C11-tl-oneliner]`
- `module-silent-protection-lint`: same replacement; a detected trip inside a one-liner body now WARNs (was clean). Case-B backward scan additionally strips `/* */`. Pattern A/B recognition, exemptions and per-trip dedupe unchanged. `[ev: B832-G2]`
- `module-ext-writable-shape-lint`: keeps its verdict (it is the canonical copy) but gains the accessor skip — a one-line `get|set|is` accessor is not a method body for this rule. `[ev: B832-G1]`
- `module-write-path-matrix`: gains the advisory `DRIFT` class (a `[concept]` row whose slot IS in the covered set), promoted to exit 1 by the existing `--strict`. STALE and the uncovered FAIL exit 1 are byte-identical to `v0.21.0`. `[ev: lint-write-path.sh:441]`
- `kit-test-harness`: client-tree roots resolve through `tests/lib/client-root.bash` (one default, env override wins); no bats file carries an absolute client path; real-tree smokes pin the **current clean state**, never a known defect. `[ev: QA 54078f6]`

---

## 4. Approach — one wave, RED-first, T1 → T3 → T2 → T4 → close

One kit PR per slice, each branching from kit `main` (`dab0807`). The kit is **ff-only**: every code PR **cherry-picks its QA RED commit** (never merges the RED branch), re-reads the tip at apply, and workers NEVER edit a QA RED (K13). Fixtures strip `//` and `/* */` before any identifier or WARN-string assertion and carry a comment-only decoy. Real-tree work runs on `Leon-Guanjuato-worktrees/main-ff1b659` — the live `Cliente/Leon-Guanjuato` checkout (`4f5f1c7`) is never read (K21). Every rule change records an **OBSERVED** mutation flip (verbatim RED-then-GREEN) and, per K24(7), **names the exact fixture it flips, with QA confirming the flip** — a "would flip" claim is not evidence. Process invariants: K11 no attribution trailers, K12 workers write only inside their own worktree (tasks.md ticks there too), K19 every routed script named in `BUILD-LOOP.md` + `skill/SKILL.md`, K20 disjoint exit codes, K22 exact-count + subject + absence smoke pins, D9b every scanner prunes dot-dirs, one `[ev:]` per paragraph/row. The four always-conflict files (`BUILD-LOOP.md`, `skill/SKILL.md`, `retros/INDEX.md`, `BUILD-STATE.md`) **fragment-merge**: append, keep both rows, dedupe by script name, never overwrite. Each PR passes **lead gate → QA verify → investigador1 second read** before merge.

| # | R-id | Branch / work unit | RED branch + tip | Est. changed | Lead gate (all must pass before merge) |
|---|---|---|---|---|---|
| **PR1** | R1 | `feat/c11-shared-method-boundary` | `qa/c11-parser-oneliner` **`d88af78`** + `qa/c11-golden-parser` **`ed2088f`** (7 cases: G-multiline, G-samemethod, G-adapter, G-oneliner-timers, G-oneliner-silent, G-oneliner-extwritable, G-accessor) — re-read both tips at apply | **~500-600 → `size:exception`** | Golden set green on **all three** lints in the same PR; C11-tl-oneliner FAIL (dab0807: exit 0) and C11-sp-oneliner WARN (dab0807: 0) — both OBSERVED flips naming their fixture; C11-g1-setter stays **0 WARN** and OBSERVED-flips to a false WARN when the accessor skip is removed; **real-tree baselines: 3 lints × 3 modules (CompPan-rt, ColdRoomPan-rt, DashboardPan-rt) on `main-ff1b659`, captured verbatim BEFORE the cut and byte-identical after — identical is the EXPECTED result** (0 one-liner methods with a schedule/alarm/trip write in the 42 `.java` at `ff1b659`), so any delta is a defect, not a win; the `:303-307` NET-depth rationale comment is **replaced**, not left contradicting the code; comment-decoy pins; existing suites of all three lints unchanged; `kit-links.bats`; `shellcheck` 0; 0 trailers |
| **PR2** | R2 | `feat/c11-concept-row-drift` | `qa/c11-concept-drift` (being authored on `dab0807`) — PR does not open until blessed | ~120 | Synthetic `[concept]` row whose slot IS in the covered set → exactly **1 DRIFT** row, exit **0**; `--strict` → exit **1**; a true concept row (slot absent) → silent; `[concept]` inside an HTML comment (decoy) → no DRIFT; STALE rows unchanged; uncovered **FAIL exit 1 identical with and without `--strict`**; exit 3 preserved (K20); real tree at client `00e7118` → **0 DRIFT** (the five PR6 concept rows have no source slot); covered set harvested matrix-root-wide (K24(3)) with the multi-line `name = "X"` harvest (K24(5)); OBSERVED flip |
| **PR3** | R3 | `feat/c11-client-root-lib` | `qa/c11-client-root` **`54078f6`** (path-pattern, 10 offenders on `dab0807` → 0) | ~140 | C11-T2-lib-exists + C11-T2-no-hardcode green: **10 → 0** absolute-path literals in `tests/*.bats` outside `tests/lib/client-root.bash`; full suite green with `C9_CLIENT_ROOT` / `C9_CLIENT_REPO` / `C8_CLIENT_REPO` **all unset**; env override still wins (one pin per variable); the three retargeted tail smokes re-pinned on `main-ff1b659`: **LD5 flips from `exit 1 + FAIL BDefrostController` to clean exit 0** (the delay-floor rule stays pinned by the existing `LD1`/`LD3`/`LD6` synthetic fixtures — no new fixture owed), c8-close SC1-smoke likewise, **RC8 stays 1 FAIL** (`:701` host literal, no delta); no toolbelt script touched |
| **PR4** | R4 | `feat/c11-lint-guard-pins` | `qa/c11-guard-pins` (being authored) — PR does not open until blessed | ~220 | Positive fixture (header names a mutation with no fixture) → exactly 1 WARN, exit **0**; `--strict` → **1**; negative (every named mutation has a fixture) → 0 WARN; usage → **3** (K20); D9b dot-dir prune pinned; run over the kit at `dab0807` + PR1..PR3 → expected **0 WARN**, else the printed list IS the finding and is fixed in the same PR; K19 rows added to `BUILD-LOOP.md` + `skill/SKILL.md`; `kit-links.bats`; `shellcheck` 0 |
| **PR5** | R5 | `chore/c11-close` | `qa/c11-close-checklist` (skeleton; QA freezes `TODO(freeze)` pins at close) | ~180 | `tests/c11-close.bats` green under `C11_CLOSE=1`; BASE pin `dab0807`; tool-pins include `lib/method-boundary.sh` + `lint-guard-pins.sh`; `sweep-build-state.sh` + `sweep-fold-audit.sh --strict` green; `retros/INDEX.md` pending = **0**; `VERSION` **`0.22.0`** + CHANGELOG per CONTRIBUTING §4-5; `shellcheck` 0; **no attribution trailer in the whole PR range** (K11) |

**Order and dependencies.** T1 first (it is the keystone; T3/T2/T4 are cheaper against a settled parser). T3 before T2 because T3 touches a toolbelt script and T2 touches only `tests/`, so they cannot conflict once ordered. T4 last of the code slices — it measures the headers the first three PRs leave behind. PR5 opens only after PR1-PR4 merge. Each branch rebases onto `main` before the QA ping, and the lead verifies `git log -1` equals the blessed tip before settling the ledger.

**Size.** PR1 is the one slice that cannot be split: the shared fragment replaces three call sites and the golden set must run across all three lints in the same diff, otherwise the toolbelt is inconsistent between merges and the golden set cannot prove cross-lint equivalence. Estimated **~500-600 changed lines** (fragment ~120, three call-site cuts ~120, golden + one-liner fixtures ~300). It requests an explicit **`size:exception`** with a reviewer note routing the diff as: fragment → call site 1 → call site 2 → call site 3 → fixtures. PR2-PR5 are each under 400 by construction.

**Versions.** Kit `VERSION` + `CHANGELOG.md` → **`0.22.0`** in PR5 (MINOR: two new capabilities, two additive verdict/behaviour widenings, no CLI removal). No client `vendorVersion` bump — C11 ships no jar. Every kit-changing push takes close-gate exit (a) NEW RETRO.

---

## 5. Alternatives considered

| Option | Verdict | Why |
|---|---|---|
| **Three separate parser PRs** (one lint per PR) | **REJECTED** | Between merges the toolbelt would hold two parsers with two verdicts for the same rule, and the golden set — whose whole purpose is proving the three lints agree — could not run in any single PR. The reviewer cost is real; it is paid once with a `size:exception` and an ordered read path, not three times with an inconsistent tree in between. `[ev: B832 §shipping]` `[ev: explore §5 risk 1]` |
| **Keep NET depth** (documented as intentional at `lint-silent-protection.sh:303-307`) and patch only the one-liner symptom | **REJECTED** | The documented rationale is wrong, not merely outdated: the runaway span it fears is already prevented by the `brace_depth >= 2` guard plus close detection, and `lint-ext-writable-shape` has run PEAK depth in production without it. Keeping NET also keeps three copies, so the next refinement is applied three times again — K24(6) exists because one copy misparsed an annotation. The comment is replaced in the same PR. `[ev: B832]` `[ev: K24(6)]` `[ev: investigador1 2nd read 6f0069155]` |
| **A second client-tree default** (`main-00e7118`) for the three tail tests | **REJECTED** | Two defaults reintroduce exactly what T2 removes: a per-test opinion about which tree is blessed. The three tail tests read the live checkout by accident, not by design, and `00e7118` differs from `ff1b659` only by a `.gitignore` and five matrix rows — nothing those three smokes measure. ONE default, env override for anything else. `[ev: 0ad09c658]` |
| **Keep LD5 asserting `exit 1 + FAIL BDefrostController`** after the retarget | **REJECTED** | That assertion pins a *bug* (ColdRoomPan defrost `time <= 0`), not a *rule*: it was only ever true on the stale `4f5f1c7`, and it went green for the wrong reason once the bug was fixed. A rule is pinned by a synthetic fixture that the lint must always flag; a real-tree smoke pins the tree's **current clean state**. LD5 is re-pinned clean, and the rule keeps its existing synthetic carriers `LD1`/`LD3`/`LD6` — so nothing is lost by dropping the bug assertion. `[ev: lead re-measure 2026-09-07]` `[ev: investigador1 2nd read 6f0069155]` |
| **Open W2 (P1-P5) in parallel with W1** | **REJECTED** | All five depend on gates only Cristian can resolve (tunnel merge, deploy chain, harness session, three station answers). Queuing preserves W1's WSL-only autonomy. `[ev: explore §5 risk 7]` |

---

## 6. Affected Areas

| Area | Impact | Description |
|---|---|---|
| `build-n4-module-kit/toolbelt/lib/method-boundary.sh` | **New** | R1 — the single section-D parser (PEAK depth, depth guard, `@`-stop, keyword exclusion, accessor skip, `/* */` strip) |
| `build-n4-module-kit/toolbelt/lint-timers.sh` | Modified | R1 — `:188-202` replaced by a `source` of the fragment |
| `build-n4-module-kit/toolbelt/lint-silent-protection.sh` | Modified | R1 — `:302-364` replaced (including the NET-depth rationale comment at `:303-307`); Case-B scan inherits the `/* */` strip |
| `build-n4-module-kit/toolbelt/lint-ext-writable-shape.sh` | Modified | R1 — canonical `:132-176` extracted into the fragment; gains the accessor skip |
| `build-n4-module-kit/toolbelt/lint-write-path.sh` | Modified | R2 — `DRIFT` class in the row pass `:422-458`; `[concept]` skip at `:441` becomes a covered-set test |
| `build-n4-module-kit/toolbelt/lint-guard-pins.sh` | **New** | R4 — header-mutation → fixture meta-check |
| `tests/lib/client-root.bash` | **New** | R3 — one default (`main-ff1b659`), three exported names, env override wins |
| `tests/{lint-timers,lint-silent-protection,ext-writable-shape,demand-in-scope,lint-write-path,lint-delays,rc-scan,c8-close,c9-close,c10-close}.bats` | Modified | R3 — 10 sites converted; R1/R2 golden + drift cases; LD5 + SC1-smoke re-pinned |
| `tests/{parser-oneliner,golden-parser,lint-guard-pins}.bats` + `tests/fixtures/**` | **New** | R1/R4 RED suites with comment decoys |
| `tests/c11-close.bats` | **New** | R5 close gate, `C11_CLOSE=1` |
| `build-n4-module-kit/{BUILD-LOOP.md, METHODOLOGY.md, skill/SKILL.md}` | Modified | K19 routing for the fragment + `lint-guard-pins.sh`; fragment-merge only |
| `build-n4-module-kit/{retros/INDEX.md, BUILD-STATE.md}`, `VERSION`, `CHANGELOG.md` | Modified | One retro per kit-changing push; `0.21.0 → 0.22.0` |

---

## 7. Risks

| # | Risk | Likelihood | Mitigation |
|---|---|---|---|
| RK1 | **PR1 exceeds the 400-line budget** and a reviewer skims the keystone | High | Explicit `size:exception` requested up front with a ~500-600 estimate and an ordered read path (fragment → 3 call sites → fixtures); no other slice is merged into it; the golden set is the review's own checklist. Splitting is rejected in §5 with its cost stated |
| RK2 | **Partial T1 cut** leaves two parsers live in one tree | Med | Single PR, single merge; the golden set runs against all three lints in that PR; a lead-gate row per lint must be green simultaneously `[ev: explore §5 risk 1]` |
| RK3 | **Accessor skip is too broad** — a real `isDirty()` that schedules gets skipped | Med | The skip is one-line accessors only, `get|set|is` prefix; C11-g1-setter is the guard and its OBSERVED flip (remove the skip → false WARN) is mandatory, not optional `[ev: B832-G1]` `[ev: QA d88af78]` |
| RK4 | **`/* */` strip shifts silent-protection counts** | Med | Real-tree baselines for all three lints × three modules captured verbatim before and after the cut; any delta is a BLOCKER, not a note. B832-G2 stays a **header NOTE** — QA could not build a biting fixture and a vacuous pin is worse than none `[ev: B832-G2]` |
| RK5 | **LD5-class rot repeats** — another smoke silently pins a defect | Med | PR3 audits every real-tree smoke's assertion class while converting the 10 sites; any smoke asserting a FAIL must either have its rule covered by an existing synthetic fixture (LD5 does: `LD1`/`LD3`/`LD6`) or be documented as an intentional known-defect pin with its ticket. Recorded as a K-lesson candidate at close `[ev: lead re-measure 2026-09-07]` `[ev: investigador1 2nd read 6f0069155]` |
| RK5b | **The T1 cut is unprovable on the real tree** — 0 one-liners in the 42 `.java` at `ff1b659`, so the fix changes no real-tree count and the golden fixtures are the ONLY proof | Med | Accepted and stated as the expected result: real-tree identity is the regression pin, the 7 golden cases are the behaviour pin. Because the fixtures carry the whole proof, PR1 also replaces the `:303-307` comment that currently asserts the opposite doctrine — a wrong rationale left in place is what makes the next author restore NET depth `[ev: investigador1 2nd read 6f0069155]` |
| RK6 | **Stale live-checkout reads** — the campaign's recurring defect class | High | T2 removes the last three; until PR3 merges, every cite names worktree + commit (K21) and `4f5f1c7` is never a smoke target `[ev: memory client-reads-use-a109249-worktree]` |
| RK7 | **T4 header grammar is undefined** — a loose parser yields 0 WARN vacuously | Med | `sdd-design` fixes the exact `# Mutation:` grammar before apply; the RED must include a positive fixture that WARNs, so a parser that finds nothing fails the suite `[ev: 8ad4bb36e §T4]` |
| RK8 | **T3 covered-set miscount** → false DRIFT on real matrices | Med | Reuse the existing matrix-root-wide harvest (`lint-write-path.sh:144-149`, K24(3)) and the multi-line `name = "X"` field-line match (K24(5), which undercounted 56 vs 177 at `ff1b659`); the real-tree pin is 0 DRIFT at `00e7118` `[ev: K24(3)(5)]` |
| RK9 | **Three REDs not yet blessed** (golden/T3/T4) | Med | RED-first is a hard precondition: PR1 waits on `qa/c11-golden-parser` `ed2088f` being blessed, PR2 on `qa/c11-concept-drift`, PR4 on `qa/c11-guard-pins`. An unblessed slice rolls to C12 without holding the version bump `[ev: explore §3.2]` |
| RK10 | **Always-conflict files across parallel kit PRs** | High | Fragment-merge (append, keep both rows, dedupe by script name, never overwrite); rebase onto `main` before the QA ping; the pre-push hook catches a dropped BUILD-STATE line `[ev: K24]` `[ev: C10 PR2 merge]` |
| RK11 | **P-slice scope creep** if a station gate resolves mid-wave | Med | P1-P5 queue for W2 after W1 closes; a resolved gate updates the queue, never interrupts the wave `[ev: explore §5 risk 7]` |

---

## 8. Rollback Plan

| Slice | Rollback |
|---|---|
| PR1 | `git revert` restores the three in-script parsers and the `v0.21.0` verdicts exactly (the one-liner FN returns). Because the golden set asserts the shared behaviour, the revert must include the RED cherry-picks in the same commit, otherwise the suite is red against a green tree |
| PR2 | `git revert` — additive verdict class behind the existing `--strict`; STALE and the uncovered FAIL are untouched, so nothing downstream depends on DRIFT |
| PR3 | `git revert` — test-only; restores the 10 literals and the LD5/SC1 stale-tree assertions. No toolbelt script is touched, so no lint verdict changes |
| PR4 | `git revert` — a new WARN-only script plus its routing rows; no existing verdict depends on it |
| PR5 | `git revert` — doc/version only; retro files are never deleted (propose-never-apply); `retros/INDEX.md` rows return to `pending`, `VERSION` returns to `0.21.0` |

No station write, no operator data mutation, and no live jar deploy is performed by any C11 slice. Deploys stay with the operator under BUILD-LOOP §6.

---

## 9. Dependencies

- `bats-core`, `shellcheck 0.10.0` (pinned in `ci.yml`), the live pre-push hook — installed.
- Research landed: **B832** (`593019540`) — the single source-backed block for T1, `[CERT]`. Open gaps carried as gates, not blockers: **B832-G1** (accessor skip → the C11-g1-setter OBSERVED flip), **B832-G2** (`/* */` strip → header note + before/after baselines). T2/T3/T4 are kit-internal; no further `research-sdd` iteration before spec.
- QA RED branches on kit origin, base `dab0807`: `qa/c11-parser-oneliner` **`d88af78`** (RED-for-the-right-reason verified), `qa/c11-golden-parser` **`ed2088f`** (7 cases), `qa/c11-client-root` **`54078f6`** (final, path-pattern, 10 → 0). **Being authored**: `qa/c11-concept-drift`, `qa/c11-guard-pins`, `qa/c11-close-checklist`. Cite by branch, re-read the tip at apply (K13).
- Apply-packages (niagara-research, tip `4ef9726e7`): `2026-09-07-c11-t1-t4-apply-packages.md` (T1 `4ef4f864c`, T3 `4ef4f864c` §T3, T4 `8ad4bb36e`, T2 re-cuts `17306e9b9` / `0ad09c658`) and `2026-09-07-c11-explore-draft.md`.
- Real tree for every smoke: `Leon-Guanjuato-worktrees/main-ff1b659` (three module roots: CompPan-rt, ColdRoomPan-rt, DashboardPan-rt/-ux); the T3 real-tree pin additionally reads client `00e7118`.
- Not dependencies of any C11 PR, but blocking W2: tunnel PRs #1/#2/#3 merge, the deploy chain, the niagaraTest harness session, and Cristian's four product answers.

---

## 10. Success Criteria

- [ ] **SC-1 (T1 golden)** — the 7 golden cases on `qa/c11-golden-parser` `ed2088f` are green against **all three** lints in one PR: G-multiline, G-samemethod, G-adapter, G-oneliner-timers, G-oneliner-silent, G-oneliner-extwritable, G-accessor.
- [ ] **SC-2 (T1 one-liner flip)** — C11-tl-oneliner goes **exit 0 → FAIL** and C11-sp-oneliner goes **0 → WARN** versus `dab0807`, each recorded as a verbatim OBSERVED flip naming its fixture, QA confirming (K24(7)).
- [ ] **SC-3 (T1 accessor)** — C11-g1-setter reports **0 WARN**; removing the `get|set|is` accessor skip makes it FALSE-WARN (OBSERVED flip). No fixture passes by absence alone.
- [ ] **SC-4 (T1 baselines)** — `lint-timers`, `lint-silent-protection`, `lint-ext-writable-shape` × CompPan-rt, ColdRoomPan-rt, DashboardPan-rt on `main-ff1b659`: **9 verdicts captured verbatim before the cut and byte-identical after**. Identity is the EXPECTED result (0 one-liner methods with a schedule/alarm/trip write in the 42 `.java` at `ff1b659`); any delta blocks the merge.
- [ ] **SC-4b (T1 doctrine)** — the NET-depth rationale comment at `lint-silent-protection.sh:303-307` is **replaced** by the shared fragment's PEAK-depth rationale (single-line methods ARE bodies; runaway spans are prevented by the `brace_depth >= 2` guard + close detection). No file in the kit still documents NET depth as correct.
- [ ] **SC-5 (T3)** — a synthetic `[concept]` row whose slot IS in the covered set → exactly **1 DRIFT** row, exit **0**; `--strict` → exit **1**; a true concept row → silent; a `[concept]` decoy inside an HTML comment → no DRIFT; STALE rows unchanged; the uncovered **FAIL exit 1 is identical with and without `--strict`**; exit **3** preserved (K20); real tree at client `00e7118` → **0 DRIFT**.
- [ ] **SC-6 (T2 sites)** — absolute `Leon-Guanjuato` path literals in `tests/*.bats` outside `tests/lib/client-root.bash`: **10 → 0**; the full suite is green with `C9_CLIENT_ROOT`, `C9_CLIENT_REPO` and `C8_CLIENT_REPO` **all unset**; each of the three variables has one override pin proving env wins.
- [ ] **SC-7 (T2 retarget)** — the three retargeted smokes are re-pinned on `main-ff1b659`: **LD5 = clean exit 0** (was `exit 1 + FAIL BDefrostController` on the stale `4f5f1c7`) and c8-close SC1-smoke likewise, with the delay-floor **rule** still pinned by the existing `LD1`/`LD3`/`LD6` synthetic fixtures (no new fixture owed, and those three stay green); **RC8 = 1 FAIL** (DashboardPan-ux `:701` host literal, unchanged on both trees). No real-tree smoke left in the suite asserts a FAIL without a named ticket.
- [ ] **SC-8 (T4)** — positive fixture (header names a mutation with no fixture) → **1 WARN**, exit 0; `--strict` → **1**; negative → 0 WARN; usage → **3**; run over the kit at `dab0807` + PR1..PR3 reports **0 WARN**, or the printed list is the finding and is closed in the same PR.
- [ ] **SC-9 (method)** — every rule-changing PR records an OBSERVED mutation flip that **names the exact fixture it flips**, confirmed by QA (K24(7)); no PR merges on a "would flip" claim; every smoke pin asserts exact count + subject + absence (K22); a smoke that cannot run is a BLOCKER.
- [ ] **SC-10 (fixtures)** — every new fixture strips `//` and `/* */` before matching and carries a comment-only decoy that does NOT satisfy the pin.
- [ ] **SC-11 (routing)** — `kit-links.bats` green with `lib/method-boundary.sh` and `lint-guard-pins.sh` routed in **both** `BUILD-LOOP.md` and `skill/SKILL.md` (K19); exit codes disjoint (K20); every scanner prunes dot-dirs (D9b).
- [ ] **SC-12 (process)** — each PR passes lead gate → QA verify → investigador1 second read; REDs are **cherry-picked, never merged** (ff-only kit, K13); K12 holds; the four always-conflict files were fragment-merged.
- [ ] **SC-13 (close)** — `tests/c11-close.bats` green under `C11_CLOSE=1` with BASE `dab0807`; `sweep-build-state.sh` + `sweep-fold-audit.sh --strict` green; `retros/INDEX.md` pending = **0**; `VERSION` = **`0.22.0`** with a CHANGELOG entry per CONTRIBUTING §4-5; `shellcheck` 0; **no attribution trailer in the whole PR range** (K11).
- [ ] **SC-14 (evidence)** — one `[ev:]` token per paragraph/row across every doc the campaign touches; every load-bearing client cite names its worktree and commit (K21).

---

## 11. Open Gates for Cristian (do NOT block W1)

| Gate | Unblocks | Status |
|---|---|---|
| Merge tunnel PRs #1/#2/#3 (viewer must send `x-config-token`) | P1, P2 | Blessed, awaiting merge |
| Execute the deploy chain (2.0.7 / 2.0.3 / 2.1.1, then the C9/C10 jars) | every future client jar | Station is four repo versions behind (2.0.3 / 2.0.1 / 2.0) |
| Run the niagaraTest harness session (`qa/c9-harness-runsheet.md`) | C9/C10 CLOSE-harness-run; alarm REDs | Runsheet owed, on `qa/c9-verify-runbooks` |
| P2 need: attribution (R14, already shipped) **or** per-operator screens (new RBAC)? | P2 | One question, two answers |
| Defrost trial green light (rooms 1/2/4) | P3 | — |
| Intercambiador Cuarto 3 on a Niagara output? YES / NO | P4 (YES) or demote the panel control (NO) | — |
| `coolOnSensorFault` station-side link approval | P5 | — |

---

## 12. Review Workload Note

`delivery_strategy` = **auto-chain**.
`Decision needed before apply: No` — **`size:exception` GRANTED for PR1 with a CEILING of 700 changed lines** (design D1k measured ≈663 changed / ~353 authored; above 700 the lead re-requests, never splits) by the lead under Cristian's standing delegation (2026-09-06 "sigan trabajando … preparar el terreno"): the three-lint cut is atomic by design (§Alternatives), the review load is mitigated by the 7-case golden set + 3×3 real-tree baselines, and investigador1's second read is mandatory before merge. Cristian may revoke; if revoked, PR1 rolls to C12 as a whole, never split. `[ev: proposal §Size; explore §5.1]`
`Chained PRs recommended: Yes` — one wave, 5 slices, ordered T1 → T3 → T2 → T4 → close; PR5 opens only after PR1-PR4 merge.
`400-line budget risk: High` — PR1 is estimated at **~500-600 changed lines** and cannot be split without leaving two parsers live between merges (§5). PR2 ~120, PR3 ~140, PR4 ~220, PR5 ~180 are all under budget.

---

## 13. Next Phases

- `sdd-spec` — the shared method-boundary contract as a single verdict domain (PEAK depth, `brace_depth >= 2` class-FIELD guard, Case-B `@`-stop, keyword exclusion, one-line `get|set|is` accessor skip, `/* */` strip) and what each of the three lints inherits from it; the `DRIFT` row grammar and its exit contract beside the untouched STALE/FAIL; the `client-root.bash` resolution contract (one default, three exported names, env override) and the smoke-assertion class rule; the `lint-guard-pins.sh` header grammar and exit contract.
- `sdd-design` (parallel with `sdd-spec`) — the awk-in-shell-var fragment layout and its sourcing seam in each of the three lints, with the exact line ranges replaced (`:188-202`, `:302-364` including the `:303-307` rationale comment, `:132-176`); the accessor-skip and `/* */`-strip algorithms; the DRIFT covered-set reuse at `lint-write-path.sh:144-149` and the `:441` `[concept]` branch inversion; the `# Mutation:` header grammar for T4; fixture layout with comment decoys; the 5-PR chain, the ff-only cherry-pick mechanics, and fragment-merge.
- Then `sdd-tasks` → `sdd-apply` (RED-first on every code slice) → `sdd-verify` → `sdd-archive`.
