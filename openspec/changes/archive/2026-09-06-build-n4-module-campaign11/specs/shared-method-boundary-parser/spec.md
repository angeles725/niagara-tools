# Spec: shared method-boundary parser (T1)

**Capability**: `kit-method-boundary-parser` — `toolbelt/lib/method-boundary.sh`
**PR**: PR1 (`feat/c11-shared-method-boundary`)
**QA RED (one-liner)**: `qa/c11-parser-oneliner` tip `d88af78` (base `dab0807`) — `tests/parser-oneliner.bats` — C11-tl-oneliner, C11-sp-oneliner, C11-g1-setter. RED verified RED-for-the-right-reason on `dab0807`. `[ev: explore §3.2]`
**QA RED (golden set)**: `qa/c11-golden-parser` tip `ed2088f` (base `dab0807`) — `tests/golden-parser.bats` — 7 cases: G-multiline, G-samemethod, G-adapter, G-oneliner-timers, G-oneliner-silent, G-oneliner-extwritable, G-accessor. Re-read both tips at apply. `[ev: proposal PR1 row]`
**Repo**: `niagara-tools` (kit)
**Cross-cutting**: see `../cross-cutting.md` (K11/K12/K13/K19/K20/K21/K22/K24(7)/D9b/comment-strip/observed-flip/real-tree-smoke/smoke-assertion-class/fragment-merge)
**Size**: `size:exception` granted — ~500-600 changed lines; ordered read path fragment → call site 1 → call site 2 → call site 3 → fixtures.

---

## Delta — NEW and MODIFIED requirements

### New fragment — `toolbelt/lib/method-boundary.sh`

| ID | Requirement | ev |
|----|-------------|-----|
| R-T1.1 | A new file `toolbelt/lib/method-boundary.sh` MUST exist and be sourced by `lint-timers.sh`, `lint-silent-protection.sh`, and `lint-ext-writable-shape.sh`. It is the single authoritative implementation of the section-D method-boundary detection contract. No other file in the kit may contain a second independent implementation of this detection logic. | `[ev: B832 §shipping]` `[ev: apply-package 4ef4f864c §T1]` |
| R-T1.2 | The fragment MUST use **PEAK depth**: the maximum brace depth reached at any point within a candidate method body, not the net brace depth at the end of the line. A one-liner `void arm() { flag=true; Clock.schedule(...); }` opens and closes on the same line; its peak depth inside the braces is 2 (class scope = depth 1, method body = depth 2), which satisfies the class-FIELD guard. | `[ev: B832 §invariants]` `[ev: explore §2 T1 live FN]` |
| R-T1.3 | The fragment MUST enforce a **`brace_depth >= 2` class-FIELD guard**: a method body is entered only when the PEAK brace depth during the method's brace span reaches at least 2. Depth 1 = class scope; depth 2 = method body. A class-level field initializer or annotation body that does not reach depth 2 MUST NOT be treated as a method body. | `[ev: B832 §invariants]` `[ev: apply-package 4ef4f864c §T1]` |
| R-T1.4 | The **Case-B backward scan** (scanning backward from the opening brace of a method to locate its signature line) MUST stop at any line that begins with `@`. An annotation line (e.g. `@Override`, `@NiagaraProperty(…)`) is never the method signature. The scan must not cross into the preceding annotation block. | `[ev: B832 §invariants Case-B]` `[ev: METHODOLOGY.md K24(6) @ dab0807]` |
| R-T1.5 | **Keyword exclusion**: the following Java control-flow keywords MUST NOT be identified as method names: `if`, `for`, `while`, `switch`, `catch`. A brace that opens immediately after one of these keywords is a control-flow block, not a method body. | `[ev: B832 §invariants]` `[ev: apply-package 4ef4f864c §T1]` |
| R-T1.6 | **One-line accessor skip (B832-G1)**: a method whose entire signature AND body appear on a single line AND whose name begins with `get`, `set`, or `is` (case-sensitive prefix) MUST NOT be entered as a candidate method body by any of the three lints. An accessor is not a candidate context for the companion-flag, silent-protection-trip, or ext-writable-shape checks. | `[ev: B832-G1]` `[ev: QA d88af78 C11-g1-setter]` |
| R-T1.7 | **`/* */` block-comment strip in the Case-B scan (B832-G2)**: when the Case-B backward scan processes a line looking for the method signature, any `/* */` block comment on that line MUST be stripped before the signature is extracted. A method signature inside a block comment MUST NOT be matched. | `[ev: B832-G2]` `[ev: apply-package 4ef4f864c §T1]` |

### Modified — `lint-timers.sh`

| ID | Requirement | ev |
|----|-------------|-----|
| R-T1.8 | `lint-timers.sh` MUST source `toolbelt/lib/method-boundary.sh`. The local method-boundary implementation at `:188-202` is REMOVED and replaced by the shared fragment. The PEAK-depth logic (R-T1.2) and all invariants (R-T1.3–R-T1.7) apply. | `[ev: B832]` `[ev: apply-package 4ef4f864c §T1 — replaces :188-202]` |
| R-T1.9 | A `Clock.schedule*` call inside a **one-liner method body** (net brace depth returns to the enclosing class level on the same line) MUST produce a `companion-flag` or `clock-schedule` `FAIL` row — the same as it would inside a multi-line method body. On `dab0807` such a pattern exits 0 (the false negative). After the fix it exits 1. | `[ev: QA d88af78 C11-tl-oneliner]` `[ev: explore §2 T1 live FN]` |
| R-T1.10 | Row grammar and exit codes (0 / 1 any FAIL / 3 usage) for `lint-timers.sh` are **UNCHANGED**. | `[ev: apply-package 4ef4f864c §T1]` |

### Modified — `lint-silent-protection.sh`

| ID | Requirement | ev |
|----|-------------|-----|
| R-T1.11 | `lint-silent-protection.sh` MUST source `toolbelt/lib/method-boundary.sh`. The local implementation at `:302-364` is REMOVED and replaced by the shared fragment. | `[ev: B832]` `[ev: apply-package 4ef4f864c §T1 — replaces :302-364]` |
| R-T1.12 | The NET-depth rationale comment at `:303-307` (`lint-silent-protection.sh` @ `dab0807`) — which documents the single-line method skip as intentional and correct ("preventing the method-open event from being attached to a post-close depth that spans until the class closes") — MUST be replaced. The replacement MUST accurately state the PEAK-depth invariant: single-line methods ARE entered as method bodies; runaway spans are prevented by the `brace_depth >= 2` class-FIELD guard plus the close-detection logic. No file in the kit at the end of PR1 may still document NET depth as the correct strategy. | `[ev: B832 §shipping]` `[ev: investigador1 2nd read 6f0069155]` `[ev: proposal SC-4b]` |
| R-T1.13 | A detected protection-trip write inside a **one-liner method body** MUST produce a WARN row — the same as it would inside a multi-line method body. On `dab0807` such a pattern exits 0 (the false negative). After the fix it exits 0 with a WARN row. | `[ev: QA d88af78 C11-sp-oneliner]` |
| R-T1.14 | Pattern A/B recognition, per-trip deduplification, exemptions, and row grammar for `lint-silent-protection.sh` are **UNCHANGED**. | `[ev: apply-package 4ef4f864c §T1]` |

### Modified — `lint-ext-writable-shape.sh` (canonical copy)

| ID | Requirement | ev |
|----|-------------|-----|
| R-T1.15 | `lint-ext-writable-shape.sh` sources `toolbelt/lib/method-boundary.sh`. The canonical local implementation at `:132-176` is **extracted** into the shared fragment. The existing PEAK-depth logic is the reference implementation; the extraction MUST NOT change its behavior. | `[ev: B832]` `[ev: apply-package 4ef4f864c §T1 — canonical copy :132-176]` |
| R-T1.16 | The one-line accessor skip (R-T1.6) is a NEW addition to `lint-ext-writable-shape.sh`. Before PR1, the lint has no accessor skip; after PR1, a one-line `get|set|is`-prefixed method body MUST NOT produce an ext-writable WARN. The C11-g1-setter fixture (a one-line `setInhibited` with a guarded write) MUST produce **0 WARN** after the accessor skip lands. Removing the skip MUST cause C11-g1-setter to produce a FALSE WARN (OBSERVED flip). | `[ev: B832-G1]` `[ev: QA d88af78 C11-g1-setter]` |
| R-T1.17 | Row grammar and exit codes for `lint-ext-writable-shape.sh` are **UNCHANGED**. | `[ev: apply-package 4ef4f864c §T1]` |

---

## Golden set — 7 cases from `qa/c11-golden-parser` `ed2088f`

Each case is a fixture or fixture pair run against **all three lints** in the same PR. The expected verdict per lint column describes WHAT the lint MUST produce; HOW is design territory.

| Case | Fixture (from `tests/golden-parser.bats` @ ed2088f) | Lint exercised | Expected verdict | Status on dab0807 |
|------|------|------|------|------|
| **G-multiline** | BMisparse — depth-1 `@NiagaraProperty(` with `defaultValue = "new BAlarmRecord()"` on its OWN line, FIELD flag set in `armA()`, `Clock.schedule` in `armB()` (the Case-B misparse shape) | lint-timers | exit 0, no companion-flag row (guard: `brace_depth >= 2` + Case-B `@`-stop) | GREEN (guard) |
| **G-samemethod** | `anyNoHardware` shape — a method-LOCAL boolean set true beside `Clock.schedule` in the same method | lint-timers | exit 0, does NOT FAIL (field-scope guard) | GREEN (guard) |
| **G-adapter** | CP-1 — pure class trip + `B<Pure>` adapter `implements BIAlarmSource` + `newOffnormalAlarm` | lint-silent-protection | 0 WARN (Pattern B surface via the adapter→pure follow) | GREEN (guard) |
| **G-oneliner-timers** | FIELD flag set true beside `Clock.schedule` inside a ONE-LINER method `void arm(){ … }` | lint-timers | FAIL companion-flag (exit 1) | **RED** (NET depth misses the one-liner) |
| **G-oneliner-silent** | a detected trip (`Math.min` shape) inside a ONE-LINER `step()` with no surface | lint-silent-protection | 1 WARN | **RED** (NET depth misses the one-liner) |
| **G-oneliner-extwritable** | an `@NiagaraAction` whose `do<Action>` body is a ONE-LINER writing the slot | lint-ext-writable-shape | slot exempt, 0 WARN — the PEAK reference the fragment must preserve | GREEN (canonical) |
| **G-accessor** | ONE-LINER accessor `setInhibited(...)` with a guarded boolean write | lint-silent-protection | 0 WARN (get/set/is one-line accessor skip; flips a PEAK-only fix without the skip) | GREEN on dab0807 (missed by NET), must stay GREEN under PEAK+skip |

`[ev: QA ed2088f tests/golden-parser.bats @test titles; d88af78 C11-tl-oneliner / C11-sp-oneliner / C11-g1-setter]`

Every golden case fixture file MUST strip `//` and `/* */` before any identifier or WARN-string match, and MUST include at least one comment-only decoy that does NOT satisfy any assertion. `[ev: ../cross-cutting.md — comment-strip rule]`

---

## Real-tree baselines — 3 × 3 (captured before cut, byte-identical after)

| Module root | lint-timers | lint-silent-protection | lint-ext-writable-shape |
|-------------|-------------|------------------------|-------------------------|
| CompPan-rt @ `ff1b659` | Captured verbatim before PR1 | Captured verbatim before PR1 | Captured verbatim before PR1 |
| ColdRoomPan-rt @ `ff1b659` | Captured verbatim before PR1 | Captured verbatim before PR1 | Captured verbatim before PR1 |
| DashboardPan-rt @ `ff1b659` | Captured verbatim before PR1 | Captured verbatim before PR1 | Captured verbatim before PR1 |

**MUST be byte-identical after the cut.** There are 0 one-liner methods carrying a `Clock.schedule*` / alarm trip / OPERATOR complex write in the 42 `.java` files at `ff1b659`. Therefore the expected result of the cut is 9 identical verdicts — any delta between the before and after baseline is a defect, not a win, and blocks the merge. `[ev: proposal §Intent ¶2]` `[ev: proposal SC-4]` `[ev: B832 §corpus]` `[ev: ../cross-cutting.md — real-tree-smoke]`

---

## Scenarios

### C11-tl-oneliner (RED — must FAIL after fix)

**Given** the `parser-oneliner.bats` fixture from `qa/c11-parser-oneliner` `d88af78`: a Java file containing a class-FIELD `private boolean armed;` and a one-liner method `void arm() { armed = true; Clock.schedule(this, BRelTime.make(5000L), null); }` where the opening and closing brace are on the same line, and `stopped()` does NOT clear `armed`.

**When** `lint-timers.sh` runs on that fixture (with `//` and `/* */` stripped before matching).

**Then** exits **1** with a `FAIL` row naming `arm` or `armed`. On `dab0807` this fixture exits 0 (the one-liner false negative — OBSERVED).

`[ev: QA d88af78 C11-tl-oneliner]` `[ev: R-T1.9]`

---

### C11-sp-oneliner (RED — must WARN after fix)

**Given** the `parser-oneliner.bats` fixture: a Java file containing a one-liner method `void step() { status = BStatusEnum.make(StatusEnum.FAULT); }` where the opening and closing brace are on the same line, and the trip write is not alarmed.

**When** `lint-silent-protection.sh` runs on that fixture.

**Then** exits **0** with a WARN row naming `step`. On `dab0807` this fixture exits 0 with no WARN (the one-liner false negative — OBSERVED).

`[ev: QA d88af78 C11-sp-oneliner]` `[ev: R-T1.13]`

---

### C11-g1-setter (accessor skip guard — must remain 0 WARN; flip when skip is removed)

**Given** the `parser-oneliner.bats` fixture: a one-liner method `public void setInhibited(BBoolean v) { inhibited.set(v); }` where `inhibited` is an OPERATOR `BStatusBoolean` property, and no explicit writing action is present.

**When** `lint-ext-writable-shape.sh` runs on that fixture.

**Then** exits **0** with **0 WARN rows** (the accessor skip prevents the false WARN).

**And when** the `get|set|is` accessor skip (R-T1.6) is removed from the fragment:

**Then** `lint-ext-writable-shape.sh` exits **0** with a **WARN row** naming `setInhibited` or `inhibited` (the FALSE WARN — OBSERVED flip confirming the skip is necessary). No fixture passes by absence alone.

`[ev: B832-G1]` `[ev: QA d88af78 C11-g1-setter]` `[ev: R-T1.16]`

---

### G-oneliner-all-three (cross-lint consistency)

**Given** the `golden-parser.bats` fixtures from `qa/c11-golden-parser` `ed2088f` (G-multiline, G-samemethod, G-adapter, G-oneliner-timers, G-oneliner-silent, G-oneliner-extwritable, G-accessor) run against each of the three lints.

**When** all three lints run on each case.

**Then** the verdicts match the golden-set table above exactly. In particular:
- G-oneliner-timers: lint-timers exits **1** with FAIL; lint-silent-protection and lint-ext-writable-shape exit **0** with no rows for this fixture.
- G-oneliner-silent: lint-silent-protection exits **0** with WARN; lint-timers and lint-ext-writable-shape exit **0** with no rows for this fixture.
- G-accessor: all three lints exit **0** with **0 rows** (accessor skip active across all three).

`[ev: QA ed2088f]` `[ev: proposal SC-1]`

---

### baseline-3x3-identity (real-tree smoke)

**Given** all three rt module roots (CompPan-rt, ColdRoomPan-rt, DashboardPan-rt) at `Leon-Guanjuato-worktrees/main-ff1b659` (`ff1b659`), with verbatim before-cut verdict snapshots captured.

**When** all three lints run on each module root AFTER the PR1 cut.

**Then** each of the **9 verdicts** is byte-identical to its before-cut snapshot. Exact WARN/FAIL count is unchanged; named subjects are unchanged; no new rows appear.

`[ev: proposal SC-4]` `[ev: R-T1.2 — 0 one-liner schedules/trips/OPERATOR writes in 42 .java at ff1b659]`

---

### doctrine-replaced (`:303-307` rationale comment)

**Given** the diff of `lint-silent-protection.sh` in PR1.

**When** the file at the end of PR1 is inspected for the text `preventing the method-open event from being attached to a post-close depth` or any prose claiming that single-line methods should be skipped.

**Then** that text is **ABSENT**. The replacement rationale correctly states that single-line methods ARE entered as method bodies, and that runaway spans are prevented by the `brace_depth >= 2` guard plus close detection.

`[ev: proposal SC-4b]` `[ev: R-T1.12]`

---

## Success criteria (this capability)

- [ ] `toolbelt/lib/method-boundary.sh` exists and is sourced by all three lints.
- [ ] C11-tl-oneliner: exits **1** with FAIL row (was exit 0 on `dab0807` — OBSERVED flip).
- [ ] C11-sp-oneliner: exits **0** with WARN row (was exit 0 with no WARN on `dab0807` — OBSERVED flip).
- [ ] C11-g1-setter: exits **0**, **0 WARN rows**; removing the accessor skip flips it to WARN (OBSERVED).
- [ ] All 7 golden cases produce the expected verdict on all three lints in the same PR.
- [ ] 9 real-tree baselines (3 lints × 3 modules @ `ff1b659`) are **byte-identical** before and after the cut.
- [ ] The `:303-307` NET-depth rationale comment is **replaced** — no file in the kit still documents NET depth as correct.
- [ ] Every golden fixture strips `//`/`/* */` before matching and carries a comment-only decoy.
- [ ] `kit-links.bats` green with `lib/method-boundary.sh` routed in both `BUILD-LOOP.md` and `skill/SKILL.md` (K19).
- [ ] Exit codes disjoint (K20); dot-dirs pruned (D9b); `shellcheck 0.10.0` exits 0; 0 attribution trailers (K11).
