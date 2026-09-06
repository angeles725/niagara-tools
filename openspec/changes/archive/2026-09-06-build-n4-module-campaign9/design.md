# Design: build-n4-module-campaign9

**Phase**: design · **Source**: kit `v0.19.0` (main `ba3432c`, C8 close `1109c0f`) · **Target**: kit `v0.20.0`
**Inputs**: `proposal.md` (R1-R13, §5 audit-sink decision, §6 doctrine deltas, §9 rollback) · `explore.md` ·
campaign-8 `design.md` (shape) · research **B820** `16b635f0f`, **B823** , **B824** `b5060f60b`, **B827** `ff0ce3f5b`,
**B828 §828.7**, **B829** `d26305d21` · S12 plan `1ecdf437c`/`80adc279e` · S20 seed (`campaign9-research-candidates.md:235-261`) ·
C8 write-path retro (`build-n4-module-kit/retros/2026-09-06-campaign8-write-path.md`).

**Read-tip discipline (K13/K21) — and one correction this design had to make about itself.** Client cites are read
from the worktree **`/home/cristian/modulos_niagara_n4/Cliente/Leon-Guanjuato-worktrees/main-a109249`** (client
`a109249`, the tip the chain builds on). An earlier draft cited the working checkout
`Cliente/Leon-Guanjuato`, which is **stale at `4f5f1c7`**; three of its findings were wrong and are corrected inline
rather than silently overwritten — D4's "B824 has drifted" table (B824 was right), D8a's "a non-numeric writes 0.0
today" (the guard already exists at `a109249`), and D11a's "the matrix is absent" (it exists, 113 lines / 20 rows).
This is exactly the failure K21 names, committed by the design that cites K21.
`[ev: corpus B821 re-anchor]` `[ev: kit METHODOLOGY.md:85 (K21)]`

**Tunnel cites** are read from `/home/cristian/tunnel/clientes/Leon-Guanajuato/Pancaddia` (read-only) — note
`Guanajuato` **with** the `a`, distinct from the client module tree `Leon-Guanjuato`.

**Unread inputs, marked honestly.** This executor has **no shell tool**, so no `git show` was possible: every QA RED
pin text is `[INFER]` from the proposal, the explore and the coordinator's relay — `origin/qa/c9-demand-in-scope`
`2916954`, `origin/qa/c9-silent-protection` `e38e503`, `origin/qa/c9-ext-writable-shape` `3726722`,
`origin/qa/c9-s12-write-server` **`55d6797`** (re-pinned), client `qa/c9-s12-servlet` `4c18837`, and client
`qa/c9-comppan-rotation` **`cf28572`** (R1, 17 pins), `qa/c9-alarm-cr3` **`70a357b`** (R8), `qa/c9-alarm-cp1`
**`8b43488`** (R9) and `qa/c9-s12-audit-mirror` **`0a14df8`** (R7, MIR1-MIR5). Their test files are absent from the
kit worktree — a glob of
`niagara-tools/tests/*.bats` returns 32 files, none of them the three new lints. **Where this design and a landed RED
disagree, the executable RED wins** — it already did, four times: the `Cfg` symbol names (D1b), the lint CLI shape
(D2), the WARN-with-exit-0 contract (D2b), and the rotation clock basis (ROT16). `[ev: proposal §10]` `[ev: explore §3.3]`

---

## Technical Approach

Three fixed waves of chained PRs, one destination-file group each, RED-first. The kit half (R2/R3/R10/R11/R12/R13)
reuses the campaign-5/6/7/8 script shape verbatim: header usage block, `set -u`, `printf` rows, typed disjoint exits
(K20), VCS-free, `shellcheck 0.10.0` clean, dot-dirs pruned (D9b). The client half (R1/R6/R8/R9) is **additive-only**
Java: every new slot is `add_slot`, so `schema-risk.sh` must read SAFE and no retype is proposed anywhere (B800 §800.8
is an OUTAGE class). The tunnel half (R4/R5/R7) is grounded in the **read** `9acb47c` source (`[CERT]` baseline in
D6a/D7); only the unwritten seam, token store and mirror stay `[INFER]`. One new client slice (D8c) came from a user
decision mid-design and is `[INFER]` pending research block **B830**. `[ev: corpus B795]` `[ev: corpus B800 §800.8]`

---

## Architecture Decisions

### D1 — S20 time-slice rotation: a two-cycle make-before-break state machine inside `CompressorControl.step`

**Insertion point (anchors verified at client `a109249`).** Two new blocks in `step(...)`, both in the PURE model
(`CompressorControl.java`, no Baja): **step 2b** (rotation COMPLETION) after the target clamp ends at
`:217 if (target > N) target = N;` and before `:219 // 3) Move ONE stage`; **step 3b** (rotation ARM) after the
stage-move block closes at `:246` and before `:248 // 4) Manual HOA override`. Both therefore sit **before the
`:255 boolean[] cmdPreHoa = cmd.clone();` snapshot**, so the HOA loop (`:256-271`) and the step-5 safety envelope
(`:273-306`) still run last and still win — a rotation write is an ordinary AUTO command as far as precedence is
concerned (B805 §805.11: OFF > safety > sequence > HAND > AUTO). Rejected: a single in-cycle pair of writes — zero
`stageDelay` between them, which is what "make-before-break with `stageDelay`" forbids, and `lastStageMs` would lie.
Rejected: placing either block after `:255` — it would be invisible to `cmdPreHoa` and could bypass the LP-floor
envelope. `[ev: client a109249 CompressorControl.java:217, :219, :246, :248, :255, :256-271, :273-306]` `[ev: corpus S20]`

**Do not confuse the two shed mechanisms.** `pickMostHoursOn` (`:365-376`, called at `:238`) is the ordinary
one-per-cycle AUTO stage-down. The **LP-floor HAND envelope** at `:288-305` (`lpFloor` computed at `:300`, loop
`:301-306`) is a different mechanism that sheds a *running HAND* unit only, scoped by `cmdPreHoa[k] && modes[k] ==
MODE_ON`. Rotation interacts with the first and must never be reasoned about through the second.
`[ev: client a109249 CompressorControl.java:288-306, :365-376]`

**Why COMPLETION runs before the stage move, and the real hazard it closes.** During the pending window `onCount` is
`target + 1`. The ordering is **enforced by the completion action setting `lastStageMs = now`**, which makes
`stageReady` (`:221`) false for the rest of that cycle so the stage move (`:222-246`) cannot fire behind it. Without
that reset, when `stageDelayMs >= minOnMs` the incoming unit is already past `minOn` by the time the stage move runs
and `pickMostHoursOn` becomes reachable.

**The hazard is NOT "it might drop the incoming unit" — it is that `pickMostHoursOn` selects by lifetime `hours[]`
and has no knowledge of `rotOut` at all** (`:365-376`: the only filters are `cmd[k]`, `minOnMs`, and max `hours`). In
an N=3 rack at `target = 2`, the THIRD running unit can carry more hours than the outgoing one, so the stage move
sheds **that** unit: the outgoing unit keeps running, `rotOut` dangles, and the rack silently holds the wrong pair.
This is **invisible in any 2-compressor fixture**, which is why ROT1 and ROT4 need an explicit N=3 case. Two
consequences are design requirements: (1) completion drops `rotOut` **explicitly**, never by delegating to the hours
shed; (2) gate 10 (`hours[in] < hours[out]`) is a wear-direction guard, **not** the barrier that protects the swap
window — the `lastStageMs` reset is.
`[ev: client a109249 CompressorControl.java:221, :222-246, :365-376]` `[ev: RED qa/c9-comppan-rotation 5955a89 ROT1/ROT4]`

**D1c — swap-window edge cases, mapped to the RED's pin IDs (`cf28572`, 17 pins).**

| Pin | Situation inside the pending window | Required behaviour |
|---|---|---|
| **ROT11** (N=3) | the ordinary shed could pick a unit other than `rotOut` | the BREAK targets **`rotOut` explicitly**; never delegate the drop to `pickMostHoursOn` |
| **ROT12** | demand RISES (`target` becomes `onCount`) | **CANCEL** the swap — clear `rotOut`, keep both units on; **`swaps` is NOT incremented** (it counts completions only) |
| **ROT13** (E2) | demand FALLS (`target < onCount - 1`) | drop **`rotOut`** first (explicit completion), then let the normal shed take the next unit on a later cycle |
| **ROT14** | `dischargeHigh` trips mid-window | still drop `rotOut`; and **never re-arm under high head** — gate 5 blocks the next arm for as long as it holds |
| **ROT15** | `rotOut` flips to HAND (`MODE_ON`) or OFF (`MODE_OFF`) | **skip and clear** — no completion write; the operator now owns that unit |
| **ROT16** | rotation enabled, or station restart / component re-enable | wait a **FULL** `rotationInterval` — requires `rotSinceMs[]`, see the clock decision above |

**D1d — lifecycle.** `resetTransient()` (`:328-335`) must additionally clear `rotOut`, `rotArmedMs`, `rotSinceMs[]`
and `swaps`; `seedRestart(now)` (`:346-349`) must re-seed `rotSinceMs[]` alongside `cmdSince[]`. The pin is **"no swap
on the first step after re-enable"**, and it verifies `seedRestart` runs before the first `step`.
`[ev: client a109249 CompressorControl.java:328-335, :346-349]` `[ev: RED qa/c9-comppan-rotation cf28572 ROT11-ROT16]`

**Per-compressor clock — `cmdSince[]` alone is the WRONG basis (ROT16). Correcting an earlier draft of this design.**
An earlier draft used `now - cmdSince[k]` as the continuous-run clock with "no new state field". `cmdSince[k]` (`:71`)
is stamped by *every* command change — including the HOA OFF edge (`:261`) and the LP-floor envelope (`:305`) — and,
decisively, it is **not** stamped when `rotationInterval` transitions from `0` to non-zero. So an operator enabling
rotation on a rack whose lead unit has already run five hours would get an **immediate** swap, when ROT16 requires
enabling to wait a **full interval**. **Chosen**: a dedicated per-unit `rotSinceMs[k]`:

- stamped when a unit is commanded ON by staging or by a rotation arm;
- **re-seeded for every unit** on `seedRestart(now)` (`:346-349`) and on the `rotationInterval` `0 → non-zero` edge;
- zeroed with the rest of the transient block in `resetTransient()` (`:328-335`).

The continuous-run clock is then `now - rotSinceMs[k]`. Outgoing candidate = the running unit with the LARGEST
`now - rotSinceMs[k]`, tie-broken by most `hours`. Rejected: reusing `cmdSince[]` (ROT16, above). Rejected: selecting
the outgoing unit by hours alone — that is what `pickMostHoursOn` already does at stage-down, and it is not a
*time-slice*. `[ev: client a109249 CompressorControl.java:71, :261, :305, :328-335, :346-349]` `[ev: RED qa/c9-comppan-rotation cf28572 ROT16]`

**Incoming selection — a new `pickLeastHoursOffAuto`, NOT the existing picker.** `pickLeastHoursOff(now, minOffMs)`
(`:352-363`) skips on `if (cmd[k] || modes[k] == MODE_OFF) continue;` — it excludes HOA **OFF** but **not HOA HAND**
(`MODE_ON`). A HAND unit that is currently off and sitting inside its `minOff` window is therefore an eligible
incoming candidate, which violates spec **ROT7** ("a compressor in HAND MUST be left untouched by rotation"). ROT7's
fixture is built as exactly that trap: modes `{AUTO, OFF, ON}` through the 10-arg `step` with the HAND unit off inside
`minOff`. **Chosen**: add `pickLeastHoursOffAuto(now, minOffMs)` whose skip condition is
`if (cmd[k] || modes[k] != MODE_AUTO) continue;`, used **only** by rotation. It inherits **min-off + HOA-OFF +
HOA-HAND** by construction, and leaving `pickLeastHoursOff` untouched keeps the ordinary stage-up path byte-identical,
which is what ROT5's golden requires. **Rejected**: adding a `MODE_ON` term inside `pickLeastHoursOff` — the stage-up
path would change behaviour and ROT5 would fail for a reason unrelated to rotation. **Rejected**: filtering the result
at the call site — a caller-side filter returns the wrong unit rather than the next-best one, because the picker has
already collapsed the candidate set to a single index.
`[ev: client CompressorControl.java:352-363]` `[ev: RED qa/c9-comppan-rotation 5955a89 ROT5/ROT7]`

**Arm gates (ALL must hold, evaluated in this order).**

| # | Gate | Source of truth |
|---|---|---|
| 1 | `c.rotationIntervalMs > 0` | `0 = disabled` sentinel (SC-1) |
| 2 | no pending rotation (`rotOut < 0`) | one swap in flight at a time |
| 3 | `onCount == target` (steady demand only) | rotation never substitutes for staging `[ev: client a109249 :222-246]` |
| 4 | `stageReady` — `(lastStageMs == Long.MIN_VALUE) \|\| (now - lastStageMs) >= c.stageDelayMs` | reuse the same expression as `:221`, not a copy of the constant |
| 5 | `!dischargeHigh` | a swap adds a stage; `:213` already forbids adding on high head |
| 6 | `!(suctionValid && c.suctionLowLimit > 0d && suction < c.suctionLowLimit)` | LP floor at `:215` sheds, never adds — **note the `> 0d` term**: a limit of 0 means the LP safety is DISABLED, and omitting it makes the gate always-true on real R404A readings |
| 7 | `available >= 2` | `available` computed at `:173-174`; a single available unit cannot rotate |
| 8 | outgoing `out >= 0`, `modes[out] == MODE_AUTO`, `(now - rotSinceMs[out]) >= rotationIntervalMs && (now - cmdSince[out]) >= minOnMs` (second read: the interval MUST be measured on the rotation clock — with `cmdSince` a lead unit already running 5 h would pass the gate the instant rotation is enabled, the exact ROT16 failure) | HAND/`MODE_ON` untouched — a `MODE_ON` unit is re-forced ON at `:264-270` anyway, so swapping it is a guaranteed no-op write |
| 9 | incoming `in = pickLeastHoursOffAuto(now, c.minOffMs) >= 0` | inherits **min-off + HOA-OFF + HOA-HAND** by construction (skip on `cmd[k] \|\| modes[k] != MODE_AUTO`) — ROT7 |
| 10 | `hours[in] < hours[out]` | a swap into a unit with MORE hours increases divergence — the opposite of the ask |

**Arm action** (one write): `cmd[in] = true; cmdSince[in] = now; rotSinceMs[in] = now; lastStageMs = now; rotOut = out; rotArmedMs = now;`. The ordinary stage-up write (`:229`) also stamps `rotSinceMs[k] = now` (rule: stamped whenever a unit is commanded ON, by staging or by arm). Neither `rotSinceMs` nor `rotArmedMs` is in the trace line, so ROT5 (golden) still holds.
**Completion action** one or more cycles later, when `rotOut >= 0 && (now - rotArmedMs) >= c.stageDelayMs`:
`cmd[rotOut] = false; cmdSince[rotOut] = now; lastStageMs = now; rotOut = -1; swaps++;`. The two pending-state fields
(`rotOut`, `rotArmedMs`) are private internals — the RED does not name them, so they may be renamed freely — but they
join the transient block at `:66-75` and MUST be cleared in `resetTransient()` (`:328-344`), otherwise a disable→enable
leaves a phantom pending swap. `swaps` is **not** private: see the public contract below.
`rotationMode = ROTATION_BREAK_BEFORE_MAKE` inverts the two actions (drop first, add after `stageDelay`);
`ROTATION_MAKE_BEFORE_BREAK` is the default because break-before-make surrenders a stage for a full `minOff` on a rack
whose entire purpose is holding suction. `[ev: client CompressorControl.java:66-75, :328-344]` `[ev: corpus S20]`

**D1b — the pure-core public contract is fixed by the RED, not by this design.** QA authored `qa/c9-comppan-rotation`,
tip **`cf28572`** (superseding `5955a89`; parent `a109249`), test file
`CompPan-rt/srcTest/.../CompressorRotationTest.java`, now pinning **17 cases (ROT1-ROT16 + the golden)**. The symbol
contract is **unchanged across both tips**. It fails `javac` on these names, and those exact names are the contract —
an earlier draft of this design used `cfg.rotationMakeBeforeBreak` (a boolean), which is **wrong and corrected here**:

| Symbol | Kind | Contract |
|---|---|---|
| `CompressorControl.Cfg.rotationIntervalMs` | `long` | continuous-run threshold in ms; **`0` = disabled** |
| `CompressorControl.Cfg.rotationMode` | `int` | compared against `CompressorControl.ROTATION_MAKE_BEFORE_BREAK` |
| `CompressorControl.ROTATION_MAKE_BEFORE_BREAK` | `static final int` | the default mode constant (sibling `ROTATION_BREAK_BEFORE_MAKE`) |
| `CompressorControl.swaps` | `int` field, package-visible like `cmd`/`hours` | count of **COMPLETED** swaps; incremented in the completion action only, never on arm |

`swaps` counts completions rather than arms so an armed-but-never-completed rotation (the component stopped mid-swap)
never inflates the count; `resetTransient()` zeroes it with the rest of the transient block. The mode constants are
plain `int` in the pure core — the `BRotationMode` frozen enum lives only in the Baja adapter and is converted at the
`Cfg` boundary, which is what keeps the core Baja-free.
`[ev: RED qa/c9-comppan-rotation 5955a89 (compile contract; NOT read here — no shell)]`

**Hours ledger unaffected.** `hours[k]` integrates on the COMMANDED state at `:158` (`if (cmd[k]) hours[k] += dtH;`),
inside the feedback loop that runs BEFORE both new blocks. A swap therefore changes which unit accrues from the NEXT
cycle onward and never rewrites accrued hours — `condenserNHours` stay monotonic (slot decls `:1372`/`:1395`, setter `:1384`; the `ctl.hours` write-back call site is located at apply — `:1975-1977` is the relay output `getCondenserN().setValue(ctl.cmd[k])`, not the ledger).
`[ev: client CompressorControl.java:145-168]` `[ev: client BCompressorControl.java:1372, :1384, :1395]`

**New slots on `BCompressorControl` (additive, schema-risk SAFE).**

```java
@NiagaraProperty(
  name = "rotationInterval",
  // Continuous run after which a running compressor is swapped for the idle
  // least-hours unit. 0 (default) = time-slice rotation DISABLED (today's behaviour).
  type = "BRelTime",
  defaultValue = "BRelTime.make(0)",
  facets = @Facet("BFacets.make(BFacets.MIN, BRelTime.make(0), BFacets.MAX, BRelTime.makeHours(24))"),
  flags = Flags.SUMMARY | Flags.OPERATOR
)
@NiagaraProperty(
  name = "rotationMode",
  type = "BRotationMode",            // NEW frozen enum: makeBefore(0) | breakBefore(1)
  defaultValue = "BRotationMode.makeBefore",
  flags = Flags.SUMMARY | Flags.OPERATOR
)
```

`powerOnDelay` at `:383-388` is the shape precedent (`BRelTime`, `defaultValue = "BRelTime.make(0)"`,
`SUMMARY | OPERATOR`). **MIN and MAX are both supplied** because `verify-module.sh --src` `facets-req` (C8 D5) FAILs a
numeric OPERATOR slot carrying only one of them; `MIN = 0` is correct here because 0 IS the disabled sentinel. This does
**not** collide with `lint-delays`' `facet-min-zero` rule: `getRotationInterval()` is never an argument to
`Clock.schedule*` — the only schedule call sites in the adapter are `:1764` (`getPowerOnDelay()`), `:1839` and `:1856`
(the `TICK_PERIOD` constant). `[ev: client BCompressorControl.java:383-388, :1764, :1839, :1856]` `[ev: retro campaign8-lint-delays]`

**`rotationMode` is a NEW, non-linked frozen enum — B828 §828.7-safe.** The existing HOA slots are
`condenser1Mode`/`condenser2Mode`/`condenser3Mode`, declared `type = "double"` at `:392-409` and read back as
`(int)Math.round(getCondenserNMode())` at `:1965-1967`; they are dashboard-linked and MUST NOT be retrofitted to a
frozen enum (that is exactly the live "Missing class" trap). `rotationMode` is new, has no link, and therefore takes the
frozen-enum form legally. `[ev: client BCompressorControl.java:392-409, :1965-1967]` `[ev: corpus B828 §828.7]`

**Adapter wiring** (two lines beside the `cfg.minOnMs/minOffMs/stageDelayMs` trio at `:1907-1909`, block `:1903-1940`): `cfg.rotationIntervalMs = getRotationInterval().getMillis();` and
`cfg.rotationMode = getRotationMode().getOrdinal();` — the `BRotationMode` ordinals are declared to match
`ROTATION_MAKE_BEFORE_BREAK = 0` / `ROTATION_BREAK_BEFORE_MAKE = 1` so the conversion is an ordinal read, not a switch.
No change to the `ctl.step(...)` call at `:1971` — both fields ride the existing `Cfg` object (`:2054`). A third
optional line surfaces the counter as a READONLY status slot (`setRotationSwaps(ctl.swaps)`); the RED does not require
the slot, only the `ctl.swaps` field, so the slot is a design add and costs one more matrix row (D11).
`[ev: client BCompressorControl.java:1907-1909, :1971, :2054]` `[ev: RED qa/c9-comppan-rotation 5955a89]`

**D1a — the byte-identical golden (SC-1 / ROT5), as QA actually captured it.** The oracle is **already embedded in RED
`5955a89`**, generated from the **unmodified `a109249`** class before PR1 touches the file — so the apply worker
**regenerates nothing**. Shape: helpers `trace()` / `demandSeq()`, neutral cfg, `stageDelay` 60 s, **120 steps**, demand
blocks `D = 1×20, 2×15, 3×10, 2×10, 1×25, 0×5, 2×15, 1×20`. Each step emits one canonical line
`now|cmd[]|stagesOn|lastStageMs`. `lastStageMs` is in the trace on purpose: it is the field the completion action
mutates, so a rotation that fires when it should not is visible even when the command vector happens to match. The
post-change run with `cfg.rotationIntervalMs = 0` must reproduce it **byte-for-byte**. The trace must exercise
stage-up, hold, stage-down, HOA, LP floor, `dischargeHigh` and restart — a golden that only covers steady demand
proves nothing about the paths rotation sits between. Rejected: asserting final state only (a swap that reverts inside
the trace would pass). Rejected: regenerating the oracle from the post-change class — circular. Text oracle, so the
no-binary-fixture rule holds (CONTRIBUTING §2).
`[ev: proposal SC-1]` `[ev: RED qa/c9-comppan-rotation 5955a89 ROT5]` `[ev: retro campaign8 D11 fixtures]`

### D2 / D3 / D10 — the three new kit lints: one shared skeleton, three rules

**Shared skeleton — `lint-delays.sh`, NOT `lint-write-path.sh`. Correcting an earlier draft.** An earlier version of
this design specified the `lint-write-path.sh` module-root/profile-discovery skeleton. **Both REDs pin the
`lint-delays` shape instead**: usage `[--strict] <java-src-dir>` — a **single Java source directory**, `--strict` as a
leading flag — with fixtures written as **inline heredocs into `$BATS_TEST_TMPDIR`**, so neither PR adds a
`tests/fixtures/` directory. The REDs win, and the shape is already the kit's: `lint-delays.sh:13` documents
`lint-delays.sh <java-src-dir>`, `:18` the row `FAIL|WARN  lint-delays  <file>:<line>  <reason>`, `:19` the exits
`0 no FAIL · 1 any FAIL · 3 usage/env`, `:21-22` the D9b dot-dir prune. Copy that file's skeleton verbatim.

**Module-root/profile discovery becomes a LATER additive flag**, not part of R2/R3. The C8 lesson 11c convention still
stands and is still folded as doctrine (D12), but forcing it into these two scripts now would break their landed REDs
for zero evidence gain. A follow-up slice adds `--module-root` to all four lints at once, which is also the only way
the convention lands consistently rather than three-scripts-three-ways.
`[ev: kit toolbelt/lint-delays.sh:13, :18, :19, :21-22]` `[ev: RED qa/c9-demand-in-scope 2916954 + qa/c9-silent-protection e38e503 (relayed, not read here)]` `[ev: retro campaign8-close-process-meta-lessons lesson 11c]`

**D2a — row column ORDER is `STATUS check subject detail`, not lint-servlet's order.** Two grammars exist in the kit
today: `lint-delays.sh:18` and `lint-write-path.sh:374` emit `FAIL|WARN  <script>  <subject>  <detail>`, while
`lint-servlet.sh:26` documents `<check>  FAIL|WARN  <file>:<line>  <detail>`. `report-module.sh`'s aggregate parser
reads `st=$1; chk=$2` (C8 D1), so only the first order aggregates correctly. **All three new lints use
`WARN  <script-name>  <file>:<line>  <detail>`** with a whitespace-free subject — the same row `lint-delays.sh:18`
already emits. `[ev: kit toolbelt/lint-delays.sh:18]` `[ev: kit toolbelt/lint-servlet.sh:26]` `[ev: retro campaign8 D1]`

**D2b — exit contract (K20), and why 2 and 4 are unused.** All three lints are **WARN-only advisory** by research
decision (B820 §820.3 and B824 §824.3: the *absence* is decidable, the *presence* case is a data-flow judgment a static
scan cannot settle). Map: **0** = no FAIL rows — **WARN rows still exit 0**, which the REDs pin directly (`DS2` = WARN
row **plus exit 0**) · **1** = any FAIL row, reachable **only** under `--strict` (`DS5` = `--strict` → exit 1) ·
**3** = usage, not-a-directory, or no Java source under `<java-src-dir>` · **2 and 4 are deliberately NOT emitted** —
K20 requires the verdict range `{0,1}` and the fault range `{3}` to be disjoint, not that every code be used, and the
legacy `2 = usage` map (`lint-timers.sh`) is not retrofitted (C8 D1). An empty or missing source dir exits **3, never a
silent 0**. `[ev: corpus B820 §820.3]` `[ev: corpus B824 §824.3]` `[ev: RED DS2/DS5 (relayed)]` `[ev: retro campaign8 D1]`

**D3 — `lint-demand-scope.sh <module-root> [--strict]` (R2).** Pass 1 finds *control-decision methods*: a method body
that assigns `target`, `cmd[`, `setBool(`, `.setValue(` or `set<Out>(` from an expression referencing a
process-variable-shaped identifier (`suction|pressure|temp|discharge|cv|psi`). Pass 2 collects *demand-shaped inputs*
in scope = the method's parameter list PLUS the enclosing class's fields, matching
`demand|.*[Cc]all.*|enable|.*[Cc]ount|loopEnable|BStatusBoolean in`. Zero demand-shaped input in scope →
`WARN  lint-demand-scope  <file>:<line>  <method> stages on <pv> with no demand/enable input in scope`. Presence is
never an automatic PASS row — it is silence. **Real-tree smoke expectation**: `CompressorControl.step` is CLEAN on all
four roots — `demandCount` is a parameter at `:114`/`:129`, the FASE-2 gate is `if (demandCount <= 0) target = 0;` at
`:201`, and the FASE-1 fallback is `target = demandCount;` at `:207`. The named mutation deletes the `demandCount`
parameter and both uses; the WARN then appears. `[ev: corpus B820 §820.2/§820.4]` `[ev: client CompressorControl.java:114, :129, :201, :207]`

**D4 — `lint-silent-protection.sh <module-root> [--strict]` (R3).** Two-part scan per B824 §824.2. TRIPS =
`setBool(<out>, false)` / `<out>.setValue(false)` / `set<Out>(…false)` under an interlock/limit/mode guard, or
`target = Math.min(target, …)` / `cmd[k] = false` / a `continue` inside a stage-pick under a limit/timer guard, or a
boolean returned by a `*Inhibited()`/`*Trip()`/`*High`/`*Low` decision feeding an output force. SURFACES, in the same
method or one level of pure-model-field → adapter-slot follow = a write to a slot whose name matches
`{*Alarm,*Fault,*Skip*,*Reason,*Status,*Mismatch,*Stuck,*Available,*Fallback}` **and** whose declaration carries
`Flags.SUMMARY` or `Flags.OPERATOR`, or a `BAlarmSourceExt`/`BIAlarmSource`. Three false-positive controls are design
requirements, not optimisations: **(a) effect-slot exemption** — the output the trip forces (`valveOut`, `evapOut`,
`condenserN`) is a SUMMARY slot that IS written but is the tier-3 effect, never the reason; **(b) one-level
field→adapter follow**; **(c) the name allowlist is the advisory seam** and is why the verdict is WARN.
`[ev: corpus B824 §824.2/§824.4]`

**Smoke expectations, verified at client `a109249`. B824's cites are CORRECT — an earlier draft of this design
reported "drift" against them, and that was this design's own error**, produced by reading a stale local checkout
(`4f5f1c7`) instead of the tip the chain builds on. The correction is recorded rather than silently overwritten,
because it is the exact failure K21 exists to prevent.

| Subject | Expected | Anchor at `a109249` (verified) |
|---|---|---|
| CP-1 low suction | **WARN** | `if (suctionValid && c.suctionLowLimit > 0d && suction < c.suctionLowLimit) target = Math.min(target, onCount - 1);` at `CompressorControl.java:215`; no named field; `suctionLowAlarm` absent from the whole property list (`stuckAlarm`, `dischargeHighAlarm`, `suctionMismatch` exist; no low-suction slot) |
| CR-3 freeze | **WARN** | `freezeTripped` private field at `BEvaporatorUnit.java:1287`, assigned in `recomputeFreeze()` at `:1092`, consumed by `valveInhibited()` at `:1102-1106` to force `valveOut` OFF; no status/reason slot, no alarm |
| CP-2 high discharge | **CLEAN (absence pin)** | named field `this.dischargeHigh` at `CompressorControl.java:82`/`:140`; adapter slot `dischargeHighAlarm` declared at `BCompressorControl.java:361` and **written at `:1994`** — the `:1994` write is what the one-level follow (SP2) must reach; declaration alone is not a surface |
| defrost-skip | **CLEAN (absence pin)** | `getDefrostSkipped()` + `setLastSkipReason()` SUMMARY writes at `BDefrostController.java:746-747` |

**Per-pin obligations from the RED (`e38e503`, relayed).** **SP1 / SP3 / SP8 each emit exactly ONE WARN** — the lint
**dedupes per trip site**, so a trip reached by several guard branches is one row, not one per branch; a naive
per-match emitter fails all three. **SP2** is the cross-file one-level field→slot follow, resolved **within the same
source directory** (consistent with the single-`<java-src-dir>` CLI: no walking out of the given tree). **SP3** is the
effect-slot exemption. **SP8** pins that a **private field is never a surface** — which is precisely why CR-3
(`freezeTripped`, private at `:1287`) FLAGs while CP-2 (named field → adapter slot write at `:1994`) does not.
`[ev: RED qa/c9-silent-protection e38e503 SP1/SP2/SP3/SP8 (relayed, not read here)]` `[ev: client a109249 BCompressorControl.java:361, :1994]`

All four subjects are still re-read at the blessed tip before the smoke pin is written; a pin asserting a stale line
number is a fixture-green/real-red failure of exactly the class C8 lesson 11 names.
`[ev: corpus B824 §824.4]` `[ev: client a109249 CompressorControl.java:215, BEvaporatorUnit.java:1092, :1102, :1287]` `[ev: retro campaign8-close-process-meta-lessons lesson 11a]`

**D5 — `lint-ext-writable-shape.sh <module-root> [--strict]` (R10).** Parse parens-balanced `@NiagaraProperty` blocks
(the `lint-write-path.sh:310-343` awk technique, unchanged) and WARN a property that carries `Flags.OPERATOR` (or
`"o"`) **and** a complex `type` in `{BStatusNumeric, BStatusBoolean, BStatusEnum}` **and** whose declaring class has no
`@NiagaraAction` that writes that slot. Clean by construction: a plain `double`/`boolean`/`BRelTime` type, or a complex
type with a writing action. **Consequence for CompPan-rt**: every OPERATOR slot there is plain (`double` at `:104-129`,
`BRelTime` at `:132-187`), so CompPan is a natural **absence pin** — the lint must emit ZERO rows for it. EW10 is the
real `BRoomPanel.setpoint` WARN. `[ev: corpus B823 §823.7]` `[ev: client BCompressorControl.java:104-187]` `[ev: kit toolbelt/lint-write-path.sh:310-343]`

**D5a — `report-module.sh` integration.** Three appended member invocations per artifact, each guarded by the existing
`src/` presence gate, each SKIPping silently when the profile has no source. Because all three are WARN-advisory, a
member WARN does **not** flip the aggregate; only a `--strict` FAIL does. The aggregate mapping is the C8 D9 table
extended by three rows — no new parser. `[ev: retro campaign8 D9]`

### D6 / D7 — tunnel: `/config/login` step-up and the canonical `change_log` — **`[CERT]` where read**

The tunnel repo **is** on this machine, read-only, at `/home/cristian/tunnel/clientes/Leon-Guanajuato/Pancaddia`
(note: `Guanajuato` **with** the `a`; the client module tree is `Leon-Guanjuato` **without** it — two different trees,
and a design that conflates them cites the wrong file). All cites below are from
`instalacion/pipeline/write-server.mjs` in that tree.

**D6a — what already exists, verified by reading `[CERT]`.**

| Fact | Anchor | Consequence for R4/R5 |
|---|---|---|
| Auth is **JWT bearer only** (`verifyJwt`, 401 on failure) | `:247-251` | the config step-up is genuinely new; nothing to migrate |
| **No seam** — `main()` builds the server inline with `http.createServer`, and `main()` is invoked unconditionally at module scope | `:231-233`, `:306-309` | importing the module today **starts a listener** on `127.0.0.1:WRITE_PORT`; `buildServer(cfg, deps)` guarding `main()` is exactly what makes `node:test` possible |
| **Server-held ORD allowlist already exists** — `^Cuarto([1-5])/(.+)$` then `WRITABLE[m[2]]` | `:258-261`, `:81-89` | R4 does not add the allowlist; it re-homes it **behind** the token gate |
| A non-allowlisted ORD returns **400**, not 403 | `:261` | proposal SC-4 says 403 — decided in D6b |
| **Pre-write GET already exists** — `oldVal` from a GET of the same `putOrd`, try/caught | `:266-268` | R5's `old` requirement is already satisfied; R5's real work is the columns, `result` and the spool |
| setpoint writes the **child** `${ord}/value`, everything else the slot directly | `:264-265` | confirms B826 `[CERT-live]`; unchanged by C9 |
| `auditChange` is best-effort by construction — early return with no service key, and every throw swallowed | `:154-169` | audit-fail-never-fails-the-write already holds **structurally**; the RED protects it rather than introducing it |
| The audit row is written **only inside the success branch**, `ok: true` hardcoded | `:270-283` | there is **no error row today** — the `ok=false` row is new work |
| Row shape today `{user_email, user_id, room, slot, label, old_value, new_value, area, ok}` | `:272-282` | matches the viewer echo exactly; the five new columns are purely additive |

**D6b — the 400-vs-403 divergence, decided.** SC-4 asks for 403 on a non-allowlisted ORD; the shipped code returns
400 (`:261`) and reserves 403 for a *station* permission error (`:285`). **Chosen: keep 400 for "not in the
allowlist", leave 403 meaning "the station refused".** The two conditions are genuinely different — a malformed
request against a published contract versus an authorisation outcome — and collapsing them would make the new `result`
column ambiguous. **SC-4's wording is amended; the code is not.** `[ev: tunnel write-server.mjs:261, :285]`

**D6c — the seam and the token model (`[INFER]` — not yet written).** `buildServer(cfg, deps)` returns the handler
without binding a port; `main()` keeps `server.listen` (`:306`) behind an `import.meta.url` guard.
`deps = { changeLog, station, clock, spool }` — the RED pins `deps.changeLog` as the injection point, so `auditChange`
moves from a module-scope function to a `deps`-provided collaborator. `POST /config/login` re-authenticates
`(email, password)` against Supabase over TLS and mints a **server-held** random token bound to
`(email, purpose="config-write")` with a 2-5 min absolute TTL AND a sliding inactivity window; the client receives an
opaque handle only. `POST /config/logout` deletes it immediately. `/write` (`:256`) and `/alarms/ack` (`:290`) gate on
it; `/health` (`:239`) and read paths do not. **No user+password store anywhere.**
`[ev: corpus B803 §803.6]` `[ev: RED qa/c9-s12-write-server 55d6797 (not read here — no shell)]`

**D7 — canonical schema, and what actually changes `[CERT]` for the baseline, `[INFER]` for the delta.**
`public.change_log` gains `ts`, `config_session`, `result`, `surface` (`'write-server' | 'servlet'`) and `client_ip`
**beside** the nine columns written today at `:272-282` — additively: no drop, no retype. `schema-risk.sh` has no
jurisdiction over Postgres, so R5 carries its own additive-only migration pin. `old_value` keeps coming from the
pre-write GET at `:266-268` (already correct); the RED pins that it stays a GET and never the request body. The audit
call moves **out of** the success-only branch (`:270-283`) so a failed write records an `ok=false` row carrying the
station status in `result`. **S12A-4** = exactly ONE `change_log` row per write, through `deps.changeLog`, carrying
`ts`/`config_session`/`result`/`surface`/`client_ip`. **S12A-6** = the sink throws → the endpoint still returns **200**
and exactly ONE JSON-lines row lands at `cfg.AUDIT_SPOOL` — a **new** config key, absent from `loadConfig` (`:24-59`)
today. The JSON-lines file is a failure spool only, drained on the next successful insert.
`[ev: tunnel write-server.mjs:24-59, :154-169, :266-268, :270-283]` `[ev: RED qa/c9-s12-write-server 55d6797 S12A-4/S12A-6]`

**D7a — the mirror (R7), contract fixed by RED `qa/c9-s12-audit-mirror` `0a14df8`** (parent `9acb47c`,
`instalacion/pipeline/test/audit-mirror.test.mjs`, MIR1-MIR5). A **new module** `audit-mirror.mjs` exports:

```js
async runMirror(cfg, deps) -> { read, inserted, skipped }
```

| Element | Contract | Pin |
|---|---|---|
| `cfg.MIRROR_ENABLED` | default **OFF** — flag absent ⇒ `readAuditHistory` is **never called** and `inserted = 0` | MIR1 |
| `deps.readAuditHistory()` | returns `[{ts, user, target, old, new}]`; the real job maps `/PANCCADIA/AuditHistory` records into that shape | — |
| `deps.changeLog` | `{ rows, insert(row), has(key) }`, `has` = dedupe lookup on `(ts, user, target, old, new)` | MIR2 |
| idempotent replay | replaying the same fixture inserts nothing the second time | MIR2 |
| **ts is not the key** | two records sharing a `ts` must **NOT** collapse — the key is the full 5-tuple | MIR3 |
| inserted row | `{ ts, user, target, old_value, new_value, surface: 'servlet', config_session: null }` | MIR4/MIR5 |

**MIR1's shape is the real teeth**: "flag off" is pinned as *`readAuditHistory` was never called*, not merely as
*zero rows inserted* — a mirror that reads AuditHistory and then discards would pass a row-count-only assertion while
still touching the station on every tick. **MIR3 is the one a naive implementation fails**: keying on `ts` alone (or
on `(ts, user)`) silently drops a second legitimate change made in the same clock tick — dedupe must hash the whole
tuple. `runMirror` returning `{read, inserted, skipped}` is what makes both observable.

**`config_session` is NULL for surface B in this campaign (MIR5), and that is the CURRENT pin.** D8c/R14's
re-authenticated station username is a **later RE-PIN of MIR5**, not a supersession of D7a: MIR5 stands as written
until R14 lands, and R14's own change includes updating MIR5. Enabling the mirror live is the B829-live gate, never a
PR gate. `[ev: RED qa/c9-s12-audit-mirror 0a14df8 MIR1-MIR5 (relayed, not read here)]` `[ev: proposal §5]` `[ev: corpus B829]`

### D8 — R6 servlet: `DashboardWriteGuards.evaluate` + the real-Context `set`

**Seam.** A new `DashboardWriteGuards` class exposing
`static Verdict evaluate(HttpServletRequest req, String relOrd, String rawValue)` returning a typed
`Verdict{ int status; String errorJson; Double parsed; }`. `handleSetpointWrite` calls it once and returns early on any
non-200 verdict. This makes the guard order testable without a servlet container — today the guards are scattered
across three files. `[ev: client a109249 BDashboardServlet.java:195-330]`

**Guard order, every cite re-read at `a109249`** (the S12 plan read `fbe9009` and reported `:291`/`:357`/`:195`/`:312`;
all four **match** at `a109249`, so the plan's servlet cites are sound — the stale numbers were this design's earlier
draft, from checkout `4f5f1c7`).

| # | Guard | Status | Where it lives at `a109249` |
|---|---|---|---|
| 1 | `X-Requested-With` missing on `/api/*` | **302** redirect home (not 4xx) | `DashboardDispatch.java:141-144` |
| 2 | no authenticated station user | **401** | `DashboardRbacHelper.java:40-41` |
| 3 | user lacks `OPERATOR_WRITE` (fail-closed) | **403** | `DashboardRbacHelper.java:55-56`, `:98` |
| 4 | missing / empty / non-numeric / NaN / Infinity value | **400** | **already present** at `BDashboardServlet.java:274-288` |
| 5 | unresolvable ORD, traversal, missing property | **400** | `BDashboardServlet.java:214-269` |

**D8a — guard4 is REGRESSION PROTECTION, not a new fix. Correcting this design's own earlier claim.** An earlier draft
asserted that `{"ord":"Cuarto1/setpoint","value":"abc"}` writes 0.0 and answers 200 today. **That is false at
`a109249`**: the PR#7 numeric guard is already there at `:274-288` — for `BStatusNumeric`/`BDouble`/`BRelTime` slots it
rejects with 400 `{"error":"invalid value"}` unless `JsonUtil.parseFiniteDouble(value).isPresent()`, explicitly "so we
never write an accidental 0". The claim came from the stale `4f5f1c7` checkout. What remains true is that the **latent
hazard is still in the file**: `parseDouble` at `:403-407` returns `0.0` for `null` and for any
`NumberFormatException`, and `coerceValue` at `:357`/`:361` feeds it into `new BStatusNumeric(parseDouble(rawValue))`
— unreachable for numeric slots *only because* guard 4 runs first. So guard4/guard4b pin `""`, `"abc"`, `"NaN"`,
`"Infinity"`, `null` each → 400 with no `parent.set`, and the named mutation **deletes the `:274-288` block** to prove
the silent zero returns. Moving the guard into `evaluate` must preserve its type-scoping: boolean/enum/string slots
have no `parseDouble` path and must not start failing.
`[ev: client a109249 BDashboardServlet.java:274-288, :357, :361, :403-407]`

**D8b — the real Context. `[CERT]` via B830, and the finding is BIGGER than audit.** `parent.set(prop, toSet, null)`
at **`:291`** becomes `parent.set(prop, toSet, user)`. B829 established that `ComplexSlotMap.set:662` gates the
`AuditEvent` on `context != null && context.getUser() != null`. **B830 read the enclosing block (`:655-672`) and found
the same `if` also wraps `user.checkWrite(base.component, base.propertyPath[0])`** plus the `BIProtected` old-value
check. So today's null-Context servlet write is **neither audited NOR permission-checked by Niagara** — the *only*
write gate in production right now is the module's own `DashboardRbacHelper.resolveOperatorWrite` (`:90-98`).

Passing the user as Context switches on **two** framework behaviours at once:

| | Mechanism | Result |
|---|---|---|
| **enforcement** | `user.checkWrite(...)` inside `:655-672` | `PermissionException` without `OPERATOR_WRITE` (`BUser.java:1659-1671`) → the servlet maps it to 403 |
| **attribution** | `audit(base, user, "Changed", old, new)` (`:813-814`) → `new AuditEvent(op, path, slot, old, new, user.getUsername())` (`:1687`) | the `/PANCCADIA/AuditHistory` row names the operator; dispatch needs `Nre.auditor != null` + an `@AuditableSpace` space (`:1682-1685`), both already satisfied (B829-G1 CLOSED) |

**No `BasicContext` wrapper is needed**: `BUser` **implements `Context`** (`BUser.java:300-303`; `getUser()` returns
`this` at `:1049`), and the client **already relies on this** — `DashboardRbacHelper.java:97` reads
`user.getPermissions((javax.baja.sys.Context) user)`. Pass the `BUser` directly. If a wrapper is ever wanted it is
`new BasicContext(user)`; **`BasicContext.make(user)` does not exist** (`BasicContext.java:27-60`).

**Consequence for guard 3.** Once a Context is passed the framework enforces `OPERATOR_WRITE`, so the servlet-side
`resolveOperatorWrite` pre-check is **no longer load-bearing** — harmless and worth keeping for a friendlier 403
raised *before* the station is touched, but the authoritative 403 now comes from the framework. The RED pins the
framework path, not the helper. Schema-neutral; `schema-risk.sh` SAFE. The fire-and-forget module audit
`svc.appendAudit(...)` at **`:312-313`** inside `catch (Exception auditEx)` is preserved unchanged.
`[ev: corpus B830 §830.4]` `[ev: corpus B829 §829.1/§829.2]` `[ev: client a109249 BDashboardServlet.java:291, :312-313, DashboardRbacHelper.java:90-98, :97]`

**Decision (guard order 1→2→3→6→4→5).** Guard 3 evaluates the CONFIG-SESSION user's `OPERATOR_WRITE` when a config session is present (falls through to guard 6 → 403 `config_login_required` when absent), never the kiosk user's — so the kiosk account can be deployed VIEWER-ONLY (the safer deployment) and operators still write; CL10 (framework `checkWrite` on the write path) remains the authoritative gate. `[ev: corpus B830 §830.4]`

### D8c — surface-B **in-module config login** (NEW user decision; work-unit **R14**)

> Numbered **D8c** because D8b (the real-Context `set`) already exists above and is cross-referenced; renumbering it
> silently would break those references. This is the item relayed as "new D8b" and carried as work unit **R14**;
> both aliases refer to this section.

**The problem.** The HMI panel runs on **one shared kiosk login**, so `req.getRemoteUser()` returns the kiosk account
for every operator standing at the panel. Passing that as the real Context (D8b) makes the write *audited and
permission-checked* but still attributes it to the kiosk, not to a person. The user's decision: a **second login
inside DashboardPan** so an operator identifies themself before writing.

**Flow.** SPA modal → `POST /dashboardpan/config/login` (username + password over the station's HTTPS) → the servlet
re-authenticates against the **station user database** → a **server-held config session** (TTL + sliding expiry,
keyed by the container session id) → every write goes through `DashboardWriteGuards.evaluate` carrying that session,
and `parent.set(prop, toSet, op)` runs with **that operator's** `BUser` as Context (B829-G2) →
`POST /dashboardpan/config/logout` revokes immediately.

**The call path — `[CERT]`, settled by B830** (`niagara-research 778d3b64b`, `niagara-mental-model-bloque830.md`
§830.2/§830.7). This was `[INFER]` in the previous revision; it is now a read call path:

```java
// POST /dashboardpan/config/login {username, password} — the kiosk request is already authenticated
BUserService svc = (BUserService) Sys.getService(BUserService.TYPE);
BUser u = svc.getUser(unescape(username));            // BUserService.java:590-602 — null on unknown
if (u == null || !svc.canLogin(u)) return 401;        // :662-684 — enabled / lockOut / expired
BAbstractAuthenticator a = u.getAuthenticator();      // BUser.java:517
if (!(a instanceof BPasswordCache)) return 401;       // unsupported scheme — no cast, no validate
boolean ok = ((BPasswordCache) a).validate(password); // BPasswordCache.java:87-95 — public final, ZERO side effects
if (!ok) { u.authenticateFailed(svc); return 401; }   // BUser.java:1130-1158 — the MODULE owns lockout accounting
u.authenticateOk(svc);                                // :1120-1123
token = configSessions.issue(req.getSession().getId(), u.getUsername());  // WebOp.java:185-188
// later, on a write carrying the token:
BUser op = svc.getUser(configSessions.userFor(token));   // re-resolve per request; never cache a BUser
try { parent.set(prop, toSet, op); }                     // BUser IS a Context
catch (PermissionException e) { return 403; }
```

**`validate()` has zero side effects, so lockout accounting is the module's duty** — `authenticateOk` /
`authenticateFailed` must be called explicitly (defaults: 5 bad logins / 30 s window / 10 s lockout). A module that
calls `validate` and forgets `authenticateFailed` builds a password oracle with no lockout at all.

**This is where D8b's B830 finding pays off.** Writing with the re-authenticated `BUser` makes
`ComplexSlotMap.set:662` run **both** `checkWrite` (`:655-672` → `PermissionException` without `OPERATOR_WRITE`,
`BUser.java:1659-1671`) **and** the `AuditEvent` naming the second operator (`:813-814`/`:1687`). Today's
`parent.set(prop, toSet, null)` at `BDashboardServlet.java:291` **bypasses Niagara permission enforcement as well as
audit** — B830 extends B829 on exactly this point.

**Unsupported / ruled out** (do not attempt these at apply): `BUserService.authenticateBasic()` and `getAuthAgent()`
were **removed in N4** (devguide `security/authentication.txt:212-222`); `BAuthenticationScheme.login(...)` is the
**JAAS** entry and yields a `LoginContext`, not a web session — wrong tool for an in-servlet re-check;
LDAP / SAML / OAuth / gauth users are **not locally re-verifiable** through `validate()` and fall out at the
`instanceof BPasswordCache` guard (`[INFER]`, B803-G1, gauth → B830-G3).

**Session.** `WebOp.getRequest().getSession()` (`WebOp.java:185-188`) supplies the container session id; the config
token lives in a **module-held map** keyed by that id, with the module's **own** short absolute TTL and sliding
inactivity — shorter than the station auto-logoff (`BUserService.defaultAutoLogoffPeriod` = 15 min, MIN 2 min,
MAX 4 h). **Never touch `WebOp.getUser()`**: the kiosk identity must stay the kiosk's, and the second identity must
never be installed into the container session as "the user".

**D8c-1 — the pure-core contract, fixed by RED `qa/c9-s12-config-login` `cc1c948`** (parent `a109249`,
DashboardPan-ux). The decision layer is Baja-free and exhaustively unit-testable off-station; the servlet keeps only
the adapter. These names are the contract:

```java
ConfigLoginGuard(users, sessions)
  login(...)          -> 200 | 401
  logout(...)
  requireSession(...) -> 200 (RENEWS the session) | 403
  reason(403)         -> "config_login_required"
  static statusForWrite(boolean permissionDenied, boolean auditFailed) -> 403 | 200

interface Clock      { now(); }
interface UserLookup { find(name); }                       // = svc.getUser, null on unknown
interface UserHandle { username(); canLogin(); isPasswordCache(); validate(pw);
                       authenticateOk(); authenticateFailed(); }

ConfigSession(Clock, ttlMs)
  issue(httpSessionId, username)
  userFor(httpSessionId)   // null if absent or expired; TOUCHES on hit -> sliding window
  revoke(httpSessionId)
```

**Adapter mapping** (the only Baja-aware code): `UserLookup.find` → `svc.getUser`; `UserHandle.canLogin` →
`svc.canLogin(u)`; `isPasswordCache` → `authenticator instanceof BPasswordCache`; `validate` →
`BPasswordCache.validate`; `authenticateOk` / `authenticateFailed` → `u.authenticateOk(svc)` /
`u.authenticateFailed(svc)`. The injected `Clock` is what makes CL8's TTL expiry and sliding renewal testable without
sleeping.

**`ConfigSession` stores the USERNAME; the servlet re-resolves the `BUser` per request and never caches it.** A cached
`BUser` goes stale against an account disabled, locked or deleted mid-session and would keep writing with a Context
the station no longer honours. `requireSession` **renews on success** — `userFor` touching on hit is where the sliding
window actually lives, a mechanism rather than an optimisation.

**`statusForWrite(permissionDenied, auditFailed)` folds CL10 and CL11 into one pure function**, so
audit-fail-never-fails-the-write cannot drift on this surface: `permissionDenied → 403`, `auditFailed → 200`. That
asymmetry is the contract, now a two-argument truth table instead of scattered `catch` blocks.

**Decision — CL6 answers 401, NOT B830 §830.7's 400.** A non-`BPasswordCache` authenticator (LDAP / SAML / OAuth /
gauth) returns the **same 401** as a wrong password. B830 sketched `400 "unsupported scheme"`, but a distinct status
or message tells an attacker **which accounts use which authentication scheme** — a free enumeration oracle over the
user base. Uniform 401 leaks nothing and costs nothing operationally, since that operator cannot use the panel login
either way. **This design deliberately diverges from B830 here**, and the RED encodes 401.
`[ev: RED qa/c9-s12-config-login cc1c948]` `[ev: corpus B830 §830.7 (400 — not adopted)]`

**Non-negotiables.** **No credential storage in the module** — it never persists a username/password pair; the JSON
user+password store is retracted on this surface exactly as on the tunnel (D6c). The session holds a station
**username** and timestamps, nothing else. The password is never cached, never logged, never echoed.

**Relationship to D7a/MIR5 — a re-pin, not a supersession.** `config_session` for surface B is **NULL today**, pinned
by **MIR5**, and that pin is correct as long as no operator-level identity exists on the panel path. When D8c lands as
**R14**, the session's station username becomes the value the R7 mirror writes, and **R14's own change updates MIR5**.
So MIR5 is not "wrong pending D8c"; it is right now, and R14 re-pins it. Sequencing R7 before R14 is therefore safe
and needs no placeholder column.

**Pins — QA's CL1-CL11.**

| Pin | Asserts |
|---|---|
| CL1 | a write with **no** config session → **403** (guard6; the RBAC 401/403 pair is unchanged — guard6 is an additional gate, not a replacement) |
| CL2 | **unknown** user → 401, and `validate` is **never called** (`getUser` returned null) |
| CL3 | **wrong** password → 401 **and `authenticateFailed` called exactly once** |
| CL4 | correct credentials → 200 **and `authenticateOk` called** |
| CL5 | **locked** user → 401 with **no `validate` call** (`canLogin` rejects first) |
| CL6 | non-`BPasswordCache` authenticator → 401 "unsupported scheme", **no cast, no validate** |
| CL7 | `/config/logout` → the next write is 403 |
| CL8 | TTL **expiry** and **sliding**-inactivity behaviour |
| CL9 | the Context passed to `set` **is the re-authenticated `BUser`**, and the kiosk `WebOp.getUser()` is **unchanged** |
| CL10 | a user missing `OPERATOR_WRITE` → the **framework** raises `PermissionException`, surfaced as 403 (not the helper) |
| CL11 | an **audit failure never fails the write** — same contract as R5/R6 |

CL1-CL11 run against the pure core (D8c-1) with fakes. Guard order becomes 1 XHR-302 → 2 auth-401 → 3 OPERATOR-403 →
**6 config-session-403** → 4 value-400 → 5 ORD-400 — the session check sits with the other authorisation gates,
before any body interpretation.

**Wiring pins CLW1-CLW5 (`ConfigLoginWiringTest`) — structural, over the servlet source.** The pure core (D8c-1) can
be perfect and the feature still absent if nothing calls it; these assert the seam is actually wired:

| Pin | Asserts |
|---|---|
| CLW1 | `DashboardDispatch` routes `POST /api/config/login` **and** `POST /api/config/logout` |
| CLW2 | the legal-path tokens are present (`getUser` → `canLogin` → `instanceof BPasswordCache` → `validate` → `authenticateOk`/`authenticateFailed`) |
| CLW3 | `parent.set(prop, toSet, null)` at `:291` is **REMOVED**, and `parent.set(prop, toSet, <BUser>)` exists |
| CLW4 | an **explicit** `catch (PermissionException …) → SC_FORBIDDEN` — the named mutation is a catch-all that swallows it |
| CLW5 | `requireSession` / `config_login_required` is on the **write** path, not only on the routes |
| SC-13 | Dashboard `defaultModuleVersion("2.2.0")` (group file — see the version-location correction above) |

**CLW3 is a REMOVAL pin, which is rarer and stronger than an addition pin**: asserting only that
`parent.set(..., <BUser>)` *exists* would pass a servlet that still carries the null-Context call on some other
branch — the exact bypass D8b exists to close. **CLW4 exists because the framework's 403 arrives as an exception**: a
`catch (Exception e)` mapping everything to 500 would turn a permission denial into a server error and silently lose
CL10.

**Harness-only (Windows `niagaraTest`), never green from WSL**: the `AuditEvent` naming the second operator, and real
lockout after 5 bad logins. Off-station, CL3/CL4 assert that `authenticateFailed`/`authenticateOk` **were called on
the fake `UserHandle`**; that the station then enforces lockout is B830-read behaviour confirmed in the harness, not a
WSL assertion. `[ev: RED qa/c9-s12-config-login cc1c948 CLW1-CLW5]` `[ev: retro campaign7 D9 skip-is-not-pass]`

**CL3, CL5 and CL6 are the ones a plausible implementation fails**: calling `validate` before `canLogin` leaks whether
a locked account's password is right (CL5); casting before the `instanceof` guard throws instead of answering
"unsupported" (CL6); and skipping `authenticateFailed` removes lockout entirely (CL3).

**Schema-risk = SAFE.** D8c touches no slot — servlet routes, a session map and a guard only. Like D8b it is
schema-neutral, so `schema-risk.sh` must read SAFE and the Dashboard group bump stays additive.
`[ev: corpus B830 §830.2/§830.4/§830.5/§830.6/§830.7 (niagara-research 778d3b64b)]` `[ev: corpus B829 §829.2]` `[ev: corpus B803 §803.6]`

### D9 — R8 alarm Pattern A (CR-3 freeze, declarative)

`BAlarmSourceExt extends BPointExtension` and `BPointExtension.isParentLegal` is a hard
`return parent instanceof BControlPoint`; `BEvaporatorUnit extends BComponent`, so the ext **cannot** mount on it.
Pattern A adds a **child `BBooleanPoint freezeAlarmPt`** with a child `BAlarmSourceExt` whose `offnormalAlgorithm` is a
`BBooleanChangeOfStateAlgorithm` (`alarmValue = true`). The point's `out` is driven from the existing latch:
`recomputeFreeze()` assigns `freezeTripped` through `ColdRoomControl.freezeTrip(...)` at `:1092`, so the single added
line writes `freezeAlarmPt.out` from that field on change. The ext raises and clears on the point's edge — no alarm
Java, no edge state machine to get wrong. Schema-risk = SAFE (`add_slot`, frozen child). Making `freezeTripped`
visible also **closes the CR-3 WARN that D4's lint raises**, so R3 and R8 must not be reviewed as independent: R8
changes R3's smoke expectation, and whichever merges second updates the pin.
`[ev: corpus B827 §827.2/§827.3/§827.6]` `[ev: client a109249 BEvaporatorUnit.java:1092, :1102, :1287]` `[ev: corpus B795]`

**D9a — the RED (`qa/c9-alarm-cr3` `70a357b`) is a STRUCTURAL JUnit that reads the source, not a runtime test.** That
shape is forced by D9's own finding: the ext only becomes real inside a station, so an off-station test can assert the
*authoring*, not the *firing*. Pins:

| Pin | Asserts | Kind |
|---|---|---|
| CRA1s | child `freezeAlarmPt` declared as a `BBooleanPoint` | structural |
| CRA2s | `recomputeFreeze` drives the point from `freezeTripped` | structural |
| CRA3s | `BAlarmSourceExt` + `BBooleanChangeOfStateAlgorithm` with `alarmValue = true` | structural |
| CRA4 | **additive-only** against the embedded `a109249` **21-slot baseline** | structural |
| CRA6 | `defaultModuleVersion("2.1.0")` | build |
| CRA1/CRA2/CRA3 **live routing** | the alarm actually routes | **HARNESS-ONLY** — Windows `niagaraTest` |

**The harness split is load-bearing**: the live-routing halves of CRA1/2/3 cannot run in WSL and must never be
reported as green from a WSL run. They are `skip`-gated and belong to the Windows `niagaraTest` harness; a SKIP is not
a PASS (campaign-7 D9). What gates PR8 off-station is the structural set plus CRA4's additive-only proof.
`[ev: RED qa/c9-alarm-cr3 70a357b (relayed, not read here)]` `[ev: retro campaign7 D9 skip-is-not-pass]`

CRA5 (WSL mutation pin): remove the `BAlarmSourceExt` → CRA3s flips OBSERVED. `[ev: RED qa/c9-alarm-cr3 70a357b]`

### D10 — R9 alarm Pattern B (CP-1 low suction, programmatic) and its edge state machine

`BCompressorControl` implements `BIAlarmSource`, declares `@NiagaraAction BBoolean ackAlarm(BAlarmRecord)`, builds a
**transient** `AlarmSupport support = new AlarmSupport(this, "defaultAlarmClass")` in `started()`, and fires on the
EDGE only. One source scales to CompPan's ~5 trips via different `alarmData`, which is why Pattern B and not Pattern A
here. `sourceState = offnormal` is what the DashboardPan alarm bql already selects — `LowLimit`/`HighLimit` are NOT
sourceState values, the low/high detail is a key in `alarmData`. `[ev: corpus B827 §827.4/§827.5]`

**Edge state machine (B827-G1), stated so it cannot be implemented level-triggered.** Per-trip index `t`:

```
boolean now_t = <trip condition>;                       // CP-1: suctionValid && suction < suctionLowLimit
if (now_t && !wasOffnormal[t]) { support.newOffnormalAlarm(dataFor(t)); wasOffnormal[t] = true; }
else if (!now_t && wasOffnormal[t] && <recovered past deadband>) { support.toNormal(BFacets.DEFAULT, null); wasOffnormal[t] = false; }
// started(): re-seed wasOffnormal[t] from the CURRENT condition, do NOT fire on the seed
```

`wasOffnormal[]` is transient (a station restart re-seeds, never re-fires). Recovery uses a deadband
(`suction >= suctionLowLimit + deadband`, ~1 psi) so a sensor hovering on the limit cannot chatter the console. The
named pin: repeated `execute()` cycles with the condition held true produce **exactly one** routed record.
`[ev: corpus B827 §827.4/§827.6]` `[ev: proposal RK5]`

**D10a — the RED (`qa/c9-alarm-cp1` `8b43488`) extracts the edge machine into a PURE nested class. The RED wins over
the inline sketch above.** The contract is `CompressorControl.AlarmEdge`:

```java
new AlarmEdge(trips)                                             // trips = number of trip channels
int decide(int trip, boolean nowOffnormal, boolean recoveredPastDeadband)   // -> FIRE | CLEAR | NONE (static final int, per the RED's compile contract)
void     reseed(boolean[] current)                               // started(): seed, never fire
boolean  wasOffnormal(int trip)
```

The adapter maps `FIRE → support.newOffnormalAlarm(data)` and `CLEAR → support.toNormal(BFacets.DEFAULT, null)`.
**This is the better design and supersedes the inline `if/else` sketch**: the three-way `Decision` makes "no
transition" an explicit value instead of a fall-through, and it puts the whole edge rule in the Baja-free core where
JUnit can exhaust it — the level-triggered re-fire (B827-G1, RK5) is a *pure* bug and now has a *pure* test.

**Wiring pins** (structural, off-station): `implements BIAlarmSource`; a **transient** `new AlarmSupport(` built in
`started()`; every `newOffnormalAlarm` call **guarded by `FIRE`**; `started()` calls `reseed(`; Compresores
`defaultModuleVersion("2.2.0")`. **CPB5 (`sourceState` on the routed record) is HARNESS-ONLY** — Windows
`niagaraTest`, `skip`-gated in WSL, never reported as green from a WSL run.
`[ev: RED qa/c9-alarm-cp1 8b43488 (relayed, not read here)]` `[ev: corpus B827 §827.4]`

### D11 — R11 write-path matrix: EXTEND an existing client-repo file, sized by measurement

**D11a — the matrix EXISTS, in the CLIENT repo root.** `docs/write-path-matrix.md` is present at the client repository
root at `a109249` — **113 lines, 20 data rows** — measured by companero on a clean worktree and recorded in
`niagara-research 19e756062` (`2026-09-06-c9-r11-write-path-matrix-measurement.md`). An earlier draft of this design
reported the file as **absent**; that reading came from the stale local checkout `4f5f1c7` and is **withdrawn**. What
survives from that draft is the repo location: the kit has no `docs/` directory, and proposal §7's
`build-n4-module-kit/docs/write-path-matrix.md` row is still wrong — **R11 is a client-repo doc PR**.
`[ev: niagara-research 19e756062 c9-r11-write-path-matrix-measurement]` `[ev: kit retros/2026-09-06-campaign8-write-path.md:104, :108]`

**D11b — R11 EXTENDS, and the SC-9 pin is exit 1 → 0.** Because the matrix already has 20 rows, `lint-write-path.sh`
exits **1** (FAIL rows for uncovered slots) on the client roots today, not 3. So the SC-9 pin is **exit 1 → exit 0
(FAIL rows → clean)**, not 3 → 0; an exit-3 pin would be asserting a missing-matrix condition that does not exist.

**D11c — measured sets at `a109249` (companero, clean worktree).**

| Module root | OPERATOR slots | Uncovered |
|---|---|---|
| ColdRoomPan-rt | 10 | **6** |
| CompPan-rt | 20 | **15** |
| DashboardPan-rt | 46 | **41** |
| DashboardPan-ux | 0 | 0 |
| **total** | **76** | **62** |

**62 rows to author**, +2 from S20 (`rotationInterval`, `rotationMode`) = **64** — not the nine `W14-W22` the proposal
assumed. DashboardPan-rt is the bulk (41 of 62) and was never measured in C8, which is why the earlier estimate was
low. **Sequencing decision**: either land PR11 **after** PR1 and author all 64 rows there, or let PR1 carry its own 2
rows and PR11 author 62 — **not both**, or the two slots are double-counted and the row IDs collide. Recommended:
PR1 carries its own 2 (a slot and its matrix row belong in one reviewable change), PR11 authors 62.
`[ev: niagara-research 19e756062 c9-r11-write-path-matrix-measurement]`

### D12 — doctrine fold: PR12 VERIFIES what C8 already folded, and adds only what is genuinely missing

**Three of the proposal's §6 deltas are ALREADY IN THE KIT.** This design's earlier draft claimed K22 did not exist and
that K21 was the last entry — **that was wrong**, produced by a grep pattern (`K2[01]`) that structurally could not
match `K22`. Grepping for the rule text instead of the number is the K6 discipline this row exists to enforce, and it
was not applied. Verified present at `ba3432c`:

| Already folded | Where | Action for PR12 |
|---|---|---|
| **K22** — real-tree smoke on every client module root is part of the lead gate; a smoke pin asserts exact counts AND subjects (never mere presence); module root = the directory containing the profiles; every lint ITERATES the profiles; a root with no sources or no matrix is ERROR exit 3, never a silent 0 | `METHODOLOGY.md:86` | **K22 present at :86 — do NOT duplicate.** Add an idempotent presence guard only (the C8 CHECK12-pin pattern): `tests/c9-close.bats` asserts exactly ONE `**K22 —` line in `METHODOLOGY.md` |
| Slot types for externally written values | `types/logic-authoring.md:62` (§ heading) + the one-liner already at `types/dashboard.md:33` | **Present — do NOT re-fold.** C8 PR15 promoted it (`METHODOLOGY.md:91`). PR12 adds at most a cross-reference from the new lint's header |
| `BAlarmSourceExt` needs a control-point parent | `types/dashboard.md:113` | Present in the *dashboard* context only. PR12 cross-references it from `types/logic.md` rather than restating the parent-legality fact |

**What PR12 actually folds (the genuinely missing set).**

| Delta | Target § | Evidence token |
|---|---|---|
| Alarm authoring **Patterns A and B** in a NEW `types/logic.md` §"Protection anatomy" — §"Protection anatomy" does not exist today, and `BIAlarmSource`/`AlarmSupport` appear **nowhere** in the kit (grep = 0), so Pattern B is entirely new; both route `sourceState=offnormal`; both `add_slot`-SAFE | `types/logic.md` §"Protection anatomy" (new) | `[ev: corpus B827]` `[ev: corpus B821 §821.4]` |
| A protection trip that forces an output or sheds a stage MUST write a named SUMMARY/OPERATOR status-or-reason slot (tier 2); a SAFETY trip should raise tier 1; the lint WARNs the tier-4 gap | same new § + a "folded as code: `lint-silent-protection.sh`" prose line | `[ev: corpus B824]` |
| ONE module-root/profile convention as a **BUILD-LOOP §5 routing line that cross-references K22**, not a restatement of it — §5 carries no module-root/profile/exit-3 text today | `BUILD-LOOP.md` §5 | `[ev: retro campaign8-close-process-meta-lessons lesson 11c]` `[ev: kit METHODOLOGY.md:86]` |
| Unified write-audit reconciliation (one canonical sink, two writers, dedupe key, audit-fail never fails the write, a null-Context servlet write is SUPPRESSED not merely unattributed) | `types/dashboard.md` + `BUILD-LOOP.md` §6 note | `[ev: corpus B829]` `[ev: S12 plan 1ecdf437c]` |
| Mutation tables record OBSERVED flips (verbatim RED-then-GREEN), never "would flip" prose | `METHODOLOGY.md` §Kit maintenance | `[ev: retro campaign8-lint-timers-ext]` |
| Four always-conflict files + the fragment-merge rule (append, keep both rows, dedupe by script name, never overwrite) | `METHODOLOGY.md` §Multi-session (extends K12) | `[ev: retro campaign8-close-process-meta-lessons lesson 2]` |
| Lead merge/settle order — merge ff-only → verify `git log -1` equals the blessed tip → THEN settle the ledger | `BUILD-LOOP.md` §7 | `[ev: retro campaign8-close-process-meta-lessons lesson 10]` |
| K19 routing for `lint-demand-scope.sh`, `lint-silent-protection.sh`, `lint-ext-writable-shape.sh` — **ships in the PR that lands each script**, never deferred to PR12 | `BUILD-LOOP.md` + `skill/SKILL.md` | `[ev: retro campaign8 D8 K19/CD5]` |

Before writing any line: `grep -rn "<rule keyword>" build-n4-module-kit/` — grep the **rule text**, never the rule
number; the mined target file is a suggestion and the rule may already live elsewhere (K6). This design violated that
rule once already, which is the whole argument for the idempotent presence guards above.
`sweep-fold-audit.sh --strict` refuses to credit a row with no `[ev:]` token.

### D13 — close gate (R13)

`tests/c9-close.bats`, gated on `C9_CLOSE=1`, mirroring `tests/c8-close.bats`: `VERSION` = `0.20.0`; a `CHANGELOG.md`
`## [0.20.0]` section with a `### References` block; `retros/INDEX.md` pending = 0; `sweep-build-state.sh` green;
`sweep-fold-audit.sh --strict` green; `kit-links.bats` naming all three new scripts in **both** `BUILD-LOOP.md` and
`skill/SKILL.md`; `shellcheck` exit 0 over `toolbelt/*.sh`; and **zero attribution trailers** across the whole PR range.
`VERSION`/`CHANGELOG` land in PR13 only, so a rolled wave-3 client slice (RK12) never holds the release.
`[ev: proposal SC-12]` `[ev: proposal RK12]`

---

## CLI Contracts

| Script | Usage | Exits |
|---|---|---|
| `lint-demand-scope.sh` | `[--strict] <java-src-dir>` (lint-delays shape; `--module-root` is a later additive flag) | `0` no FAIL — **WARN rows exit 0** · `1` FAIL under `--strict` · `3` usage/env/no-source |
| `lint-silent-protection.sh` | `[--strict] <java-src-dir>` | `0` · `1` · `3` |
| `lint-ext-writable-shape.sh` | `[--strict] <java-src-dir>` | `0` · `1` · `3` |
| `lint-write-path.sh` | unchanged — `<module-root> [--bog <config.bog>] [--matrix <path>]` | `0` · `1` · `3` (unchanged) |
| `report-module.sh` | unchanged usage + three appended member rows | `0` · `1` · `3` |

---

## PR Matrix

| PR | R | Branch | Repo | RED (unread here) | Files | Named mutation |
|---|---|---|---|---|---|---|
| 1 | R1 | `feat/c9-comppan-rotation` | client | **`qa/c9-comppan-rotation` `cf28572`** (parent `a109249`), `CompressorRotationTest.java`, **17 pins ROT1-ROT16 + golden** | `CompressorControl.java` (2b/3b + `pickLeastHoursOffAuto` + `rotSinceMs[]` + `swaps` + mode constants), `BCompressorControl.java` (2 slots + 2 cfg lines + generated block), `BRotationMode.java`, matrix rows ×2 | drop the `!= MODE_AUTO` term in `pickLeastHoursOffAuto` → **ROT7** flips on the HAND-in-`minOff` fixture; delete the completion `lastStageMs = now` → **ROT4/ROT11** fail on the N=3 case; base the clock on `cmdSince` → **ROT16** fails |
| 2 | R2 | `feat/c9-demand-scope` | kit | `qa/c9-demand-in-scope` `2916954` (DS1-DS7 + DS-smoke) | `toolbelt/lint-demand-scope.sh`, `tests/lint-demand-scope.bats` (**inline heredoc fixtures — no `tests/fixtures/` dir**), `ci.yml`, K19 routing ×2 | DS2 — drop the field scan → a class-field demand input stops counting |
| 3 | R3 | `feat/c9-silent-protection` | kit | `qa/c9-silent-protection` `e38e503` (SP1-SP8 + SP-smoke) | `toolbelt/lint-silent-protection.sh`, `tests/lint-silent-protection.bats` (inline heredocs), `ci.yml`, K19 routing ×2 | SP8 — treat a private field as a surface → CR-3 stops WARNing; drop the field→slot follow → CP-2 starts WARNing |
| 4 | R4 | `feat/c9-s12-config-login` | tunnel | `qa/c9-s12-write-server` **`55d6797`** (re-pinned; rebase → `9acb47c`) | `write-server.mjs` `buildServer(cfg,deps)` seam + token store, tests | drop the config-token check → S12A-1/S12A-5 flip |
| 5 | R5 | `feat/c9-s12-audit-schema` | tunnel | extends the same RED (S12A-4, S12A-6) | migration, `deps.changeLog` writer, `cfg.AUDIT_SPOOL` | make the sink throw → **S12A-6**: still 200 + exactly ONE spool row |
| 6 | R6 | `feat/c9-s12-servlet-guards` | client | `qa/c9-s12-servlet` `4c18837` | `DashboardWriteGuards.java` (new), `BDashboardServlet.java`, `srcTest` | delete the existing `:274-288` numeric guard → `"abc"` reaches `parseDouble` (`:403-407`) and writes 0.0 again — a **regression** pin |
| 6b | **R14** (D8c) | `feat/c9-config-login-servlet` | client | **`qa/c9-s12-config-login` `cc1c948`** — CL1-CL11 (pure core) + CLW1-CLW5 (`ConfigLoginWiringTest`) + SC-13 | `ConfigLoginGuard` / `ConfigSession` / `Clock` / `UserHandle` / `UserLookup` (new, Baja-free), `BDashboardServlet` `/api/config/login` + `/logout`, guard6, SPA modal, `Dashboard/build.gradle.kts:33` → `2.2.0`, **MIR5 re-pin** | replace the explicit `catch (PermissionException)` with a catch-all → **CLW4** flips (403 becomes 500); leave the null-Context `parent.set` on any branch → **CLW3** flips; drop `authenticateFailed` → **CL3** flips |
| 7 | R7 | `feat/c9-s12-audit-mirror` | tunnel + kit doc | **`qa/c9-s12-audit-mirror` `0a14df8`** (MIR1-MIR5) | new `instalacion/pipeline/audit-mirror.mjs` (`runMirror(cfg, deps)`), dedupe on the 5-tuple, kit reconciliation doc | key the dedupe on `ts` alone → **MIR3** fails (two same-tick records collapse); default `MIRROR_ENABLED` on → **MIR1** fails (`readAuditHistory` called) |
| 8 | R8 | `feat/c9-alarm-cr3` | client | **`qa/c9-alarm-cr3` `70a357b`** (structural; CRA1s/2s/3s/CRA4/CRA6 + harness-only live routing) | `BEvaporatorUnit.java` child point + ext, `Paccadia/build.gradle.kts:33` → `2.1.0` | drop `alarmValue = true` → CRA3s fails; add a slot outside the ext → CRA4 additive-only fails |
| 9 | R9 | `feat/c9-alarm-cp1` | client | **`qa/c9-alarm-cp1` `8b43488`** (pure `AlarmEdge` contract + wiring pins; CPB5 harness-only) | `CompressorControl.AlarmEdge` (new nested class), `BCompressorControl.java` `BIAlarmSource` + transient `AlarmSupport`, `Compresores/build.gradle.kts:33` → `2.2.0` | make `decide` level-triggered (return `FIRE` whenever `nowOffnormal`) → the once-only pin fails; drop the `started()` `reseed(` → restart re-fires |
| 10 | R10 | `feat/c9-ext-writable-shape` | kit | `qa/c9-ext-writable-shape` `3726722` | `toolbelt/lint-ext-writable-shape.sh`, tests, fixtures, `ci.yml`, K19 routing ×2 | drop the `@NiagaraAction` exemption → a clean complex-with-action starts WARNing |
| 11 | R11 | `docs/c9-write-path-rows` | **client** (D11a) | `qa/c8-write-path` `5e357d1` | `<client-root>/docs/write-path-matrix.md` — EXTEND from 20 rows by **62** (D11c); PR1 carries its own 2 | delete one row → that slot FAILs again; SC-9 pin is exit **1 → 0** |
| 12 | R12 | `docs/c9-doctrine` | kit | none | `types/logic.md` §Protection anatomy (new), `types/dashboard.md`, `BUILD-LOOP.md` §5/§6/§7, `METHODOLOGY.md` — **K22 already at :86, presence-guard only** | none; guard = `sweep-fold-audit.sh --strict` + `kit-links.bats` + the exactly-one-K22 pin |
| 13 | R13 | `chore/c9-close` | kit | none | `tests/c9-close.bats`, `VERSION`, `CHANGELOG.md`, `retros/INDEX.md`, `BUILD-STATE.md` | none; the gate is the test |

---

## Threat Matrix

| Boundary | Applicability | Design response | RED |
|---|---|---|---|
| Untrusted content decode | **Applicable** — the three lints read customer Java; the mirror reads an AuditHistory export | `LC_ALL=C` byte reading, `grep`/`awk` only, no `eval`, no interpolation of file content into a command; malformed input → exit 3, never a crash | a malformed-source fixture per lint |
| Routing / auth boundary | **Applicable** — `/config/login`, `/config/logout`, the token gate on `/write` and `/alarms/ack`, and the servlet's four guards | Server-held token bound to `(email, purpose)`, absolute TTL + sliding window, server-held ORD allowlist, fail-closed guards in a fixed order, no user+password store | S12A-1..7; guard1-5 + guard4b |
| Input validation / silent coercion | **Applicable** — `parseDouble` returns 0.0 on any parse failure (`BDashboardServlet.java:403-407`) | Guard 4 rejects before `coerceValue`; a non-numeric is 400, never a 200 that writes 0.0 | guard4 / guard4b no-silent-zero |
| Filesystem write | **Applicable** — the tunnel failure spool | Appends only to a configured spool path; drained, never executed; never `+x` | spool-append pin |
| Subprocess / external tool | **Applicable** — `python3` only in the pre-existing `--bog` helper | The three new lints add no subprocess; `command -v python3 \|\| exit 3` stays where it is | unchanged |
| Git / VCS automation | **N/A** — every toolbelt script is VCS-free; `kit-links.bats` L2 fails the suite if any names `git` | — | L2 (existing) |
| Executable-file classification | **N/A** — no new artifact is marked executable except the three `toolbelt/*.sh`, matching every existing script | — | — |

---

## Migration / Rollout

**Schema-risk expectation: SAFE on every client slice.** PR1 adds two frozen properties and one new enum type; PR8 adds
a frozen child point + ext; PR9 adds a frozen action + a transient field. All are `add_slot`, and an old `.bog` with no
entry takes the new default; PR8 and PR9 each prove it against the **embedded `a109249` slot baseline** their RED
carries (CRA4 = 21 slots for `BEvaporatorUnit`). **No retype and no slot removal appears anywhere in C9** — B800
§800.8 makes retype an OUTAGE class, and the user rejected the setpoint retype explicitly. PR6 and PR6b touch no slot
at all (schema-neutral). `[ev: corpus B795]` `[ev: corpus B800 §800.8]` `[ev: proposal §2.4]` `[ev: RED qa/c9-alarm-cr3 70a357b CRA4]`

**D-id correction — the version lives in the GROUP `build.gradle.kts`, not the module's.** Any earlier line here
implying a per-module `vendorVersion` edit is wrong. `defaultModuleVersion("X.Y.Z")` is declared once per group and
**stamps the `vendorVersion` attribute on every module in that group**, verified at `a109249`:

| Group file | Value at `a109249` | Stamps | C9 bumps |
|---|---|---|---|
| `Compresores/build.gradle.kts:33` | `2.0.3` | CompPan-rt (`module.xml` `vendorVersion="2.0.3"`) | PR1 → **2.1.0** (CRA6-equivalent), PR9 → **2.2.0** |
| `Paccadia/build.gradle.kts:33` | `2.0.7` | ColdRoomPan-rt | PR8 → **2.1.0** (CRA6) |
| `Dashboard/build.gradle.kts:33` | `2.1.1` | DashboardPan-**rt** AND DashboardPan-**ux** (both `module.xml` read `2.1.1`) | PR6 → **2.2.0** |

Two consequences the per-module reading would have hidden. **(1)** One group bump moves every profile in the group —
the proposal's "DashboardPan 2.1.1 → 2.2.0" is correctly *one* edit covering both `-rt` and `-ux`, and a
Compresores bump moves any future CompPan profile whether or not that PR touched it. **(2)** `Compresores/build.gradle.kts:33`
is edited by **both PR1 and PR9**, and `Dashboard/build.gradle.kts:33` by **both PR6 and PR6b** — so these are
**client-side always-conflict files**, and the fragment-merge discipline (D12, C8 lesson 2) applies to the client
chain too, not only to the four kit files. Sequence them, or the second PR silently reverts the first's bump.
`[ev: client a109249 Compresores/build.gradle.kts:32-33, Paccadia/build.gradle.kts:32-33, Dashboard/build.gradle.kts:32-33]` `[ev: client a109249 CompPan-rt/…/module.xml:2, DashboardPan-ux/…/module.xml:2]`

**Rollback.** Kit PRs (2/3/10/12/13) revert cleanly — new paths nothing references until the `report-module` rows land;
retro files are never deleted and `retros/INDEX.md` rows return to `pending`. Client slot-touching PRs (1/8/9) revert as
a slot REMOVE against a station bog that already holds the new slots, so a revert-after-deploy runs `schema-risk.sh`
first and, on a LOSSY/OUTAGE verdict, **the safe rollback is a disabled-by-default value — `rotationInterval = 0`, the
alarm ext disabled — NOT a jar downgrade.** That is precisely why `rotationInterval = 0` must be byte-identical (SC-1).
PR6 reverts to the null-Context write. Tunnel PRs revert the server code while the additive migration is **left in
place** (dropping a column is the destructive direction); flags off = the mirror stops and the gate falls back to
JWT-bearer only. No station write, no operator data mutation and no live jar deploy is performed by any slice.
`[ev: proposal §9]`

---

## Fresh-context validator checklist

**Rule 0 — read `Cliente/Leon-Guanjuato-worktrees/main-a109249`, never `Cliente/Leon-Guanjuato`.** The latter is stale
at `4f5f1c7` and already produced three wrong findings in an earlier draft of this file.

1. `git show origin/qa/c9-comppan-rotation:…/CompressorRotationTest.java` — confirm the four compile symbols
   (`Cfg.rotationIntervalMs`, `Cfg.rotationMode`, `ROTATION_MAKE_BEFORE_BREAK`, `ctl.swaps`) and that **ROT1/ROT4
   include an N=3 case**. A 2-compressor-only ROT4 cannot see the `pickMostHoursOn`-sheds-the-wrong-unit hazard.
2. `git show origin/qa/c9-silent-protection:tests/lint-silent-protection.bats` — the SP-smoke subjects must be
   `CompressorControl.java:215` and `BEvaporatorUnit.java:1287`/`:1102`, matching D4 **and** B824.
3. `git show origin/qa/c9-demand-in-scope:tests/lint-demand-scope.bats` and `…:tests/lint-ext-writable-shape.bats` —
   confirm the row column ORDER matches D2a (`STATUS check subject detail`), not `lint-servlet.sh`'s order.
4. In the `main-a109249` worktree, confirm D1's anchors: `:217` clamp end · `:219` step-3 comment · `:221` stageReady ·
   `:238` `pickMostHoursOn` call · `:246` block close · `:248` step-4 comment · `:255` `cmdPreHoa`. If the stage-move
   block does not close immediately before `:248`, D1's insertion points move.
5. `grep -n 'K22' build-n4-module-kit/METHODOLOGY.md` — **must return `:86`**. K22 exists; PR12 guards it, never
   re-folds it. Grep the rule TEXT, not the number.
6. `git -C <client> show a109249:docs/write-path-matrix.md | grep -c '^|'` — expect ~20 data rows (D11a), and confirm
   `lint-write-path.sh` exits **1**, not 3, on the client roots (D11b).
7. In the tunnel tree, confirm D6a: `:231-233`/`:306-309` (no seam), `:247-251` (JWT only), `:258-261` (allowlist,
   400), `:266-268` (pre-write GET), `:270-283` (success-only audit, `ok:true` hardcoded).
8. Cross-check every proposal ID (R1-R13, SC-1..SC-13) against a D-number; a requirement with no design home is a gap.
   Confirm no line proposes a station write or a live jar deploy.

## Open Questions

- [x] **K22** — RESOLVED, against this design's earlier claim. **K22 already exists at `METHODOLOGY.md:86`**, folded by the C8 close. PR12 verifies presence (idempotent guard); it does not author it.
- [x] **D11a** — RESOLVED. `docs/write-path-matrix.md` **exists** at the CLIENT repo root (113 lines / 20 rows at `a109249`). The "absent" reading came from the stale checkout. The kit still has no `docs/`, so R11 remains a client PR.
- [x] **D11b sizing** — RESOLVED by measurement: 62 uncovered slots (CRP 6 · CompPan 15 · DashboardPan-rt 41 · ux 0), +2 from S20.
- [x] **D8a** — RESOLVED, against this design's earlier claim. The numeric guard **already exists** at `BDashboardServlet.java:274-288`; guard4 is regression protection, not a new fix.
- [x] **R1 RED** — RESOLVED. Authored at `qa/c9-comppan-rotation` **`cf28572`** (17 pins, superseding `5955a89`); its compile contract is binding and the symbol set is unchanged across both tips.
- [x] **ROT11+ numbering** — RESOLVED by the RED: the swap-window edge cases are **ROT11-ROT16** (D1c), not design-invented IDs.
- [x] **Lint CLI shape** — RESOLVED against this design's earlier draft: both REDs pin `[--strict] <java-src-dir>` (lint-delays shape, inline heredoc fixtures). Module-root discovery is a later additive flag.
- [x] **R8 / R9 REDs** — RESOLVED. Authored at `qa/c9-alarm-cr3` `70a357b` (structural) and `qa/c9-alarm-cp1` `8b43488` (pure `AlarmEdge` contract). RK12's "a wave-3 RED is not authored before close" is now closed for both alarm slices.
- [x] **Version location** — RESOLVED. `defaultModuleVersion` is GROUP-scoped in `<Group>/build.gradle.kts:33`, not per module; `Compresores` and `Dashboard` become client-side always-conflict files (PR1/PR9 and PR6/PR6b).
- [ ] **Windows `niagaraTest` harness** — CRA1/2/3 live routing and CPB5 `sourceState` cannot run in WSL. The harness run is a lead gate for PR8/PR9 and must be recorded as a real run, never as a SKIP counted green. Who runs it and when is unsettled.
- [x] **B830** — RESOLVED (`niagara-research 778d3b64b`). D8c's call path is `[CERT]`: `getUser` → `canLogin` → `getAuthenticator` → `instanceof BPasswordCache` → `validate` → module-owned `authenticateOk`/`authenticateFailed`. B830 also **extends B829**: the null Context bypasses `checkWrite` (permission enforcement), not only the audit — folded into D8b.
- [x] **R14 RED** — RESOLVED. Authored at `qa/c9-s12-config-login` `cc1c948`; the `ConfigLoginGuard` / `ConfigSession` / `Clock` / `UserHandle` / `UserLookup` shape and CLW1-CLW5 are contract, not design preference. R14 no longer waits on anything.
- [x] **CL6 status code** — RESOLVED as a **deliberate divergence from B830 §830.7**: 401, not 400. A distinct code for "unsupported scheme" is a user-base enumeration oracle over authentication schemes.
- [ ] **B830-G2 / B830-G3** — the `getLogoffPeriod()` resolution chain inside `NiagaraSuperSession` was not read (G2), and gauth second-factor behaviour under `validate()` is unsettled (G3). Neither blocks R14: the module holds its own shorter TTL, and non-`BPasswordCache` schemes fall out at CL6.
- [x] **D7a vs D8c** — RESOLVED as a **re-pin, not a supersession**. MIR5 (`config_session = null`) is the correct current pin; when D8c lands as **R14** it updates MIR5 to the station username as part of its own change. R7 may therefore ship before R14 with no placeholder column.
- [x] **R7 RED** — RESOLVED. Authored at `qa/c9-s12-audit-mirror` `0a14df8` (MIR1-MIR5), so the `runMirror(cfg, deps)` shape and the 5-tuple dedupe are contract, not design preference.
- [ ] **SC-4 amendment** — the proposal says a non-allowlisted ORD returns 403; the shipped code returns 400 and D6b keeps it. The proposal text needs the one-line correction.
- [ ] **R3 ↔ R8 interaction** — landing R8 (CR-3 alarm) removes the CR-3 WARN that R3's smoke pin asserts. Whichever merges second updates the pin; they must not be reviewed as independent.
- [ ] **D6c / D7a remain `[INFER]`** — the seam, token store and mirror are not yet written; the RED (`55d6797`) was not readable here. The `9acb47c` baseline in D6a/D7 **is** `[CERT]`.
- [ ] **All QA RED pin texts** are `[INFER]` — none was readable here (no shell). Where this design and a landed RED disagree, **the executable RED wins**.
