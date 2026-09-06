# Design: build-n4-module-campaign9

**Phase**: design · **Source**: kit `v0.19.0` (main `ba3432c`, C8 close `1109c0f`) · **Target**: kit `v0.20.0`
**Inputs**: `proposal.md` (R1-R13, §5 audit-sink decision, §6 doctrine deltas, §9 rollback) · `explore.md` ·
campaign-8 `design.md` (shape) · research **B820** `16b635f0f`, **B823** , **B824** `b5060f60b`, **B827** `ff0ce3f5b`,
**B828 §828.7**, **B829** `d26305d21` · S12 plan `1ecdf437c`/`80adc279e` · S20 seed (`campaign9-research-candidates.md:235-261`) ·
C8 write-path retro (`build-n4-module-kit/retros/2026-09-06-campaign8-write-path.md`).

**Read-tip discipline (K13/K21).** Every client `file:line` below was read from the working checkout
`/home/cristian/modulos_niagara_n4/Cliente/Leon-Guanjuato` on 2026-09-06. **This executor had no shell tool**, so the
checkout's SHA could not be verified and no `git show` of a QA RED branch or of the tunnel repo was possible. Client
cites are therefore **re-anchor targets at apply**, not blessed tips. Where a line differs from a research block's cite
the divergence is stated inline. `[ev: corpus B821 re-anchor]` `[ev: retro campaign8-close-process-meta-lessons]`

**Unread inputs, marked honestly.** `origin/qa/c9-demand-in-scope` `2916954`, `origin/qa/c9-silent-protection` `e38e503`,
`origin/qa/c9-ext-writable-shape` `3726722`, `origin/qa/c9-s12-write-server` `24adcba`, client `qa/c9-s12-servlet`
`4c18837` and the tunnel repo at `9acb47c` were **NOT readable** here (no shell; tunnel not on this machine). Their
`tests/*.bats` / `test/*.mjs` files are absent from the kit worktree — a glob of `niagara-tools/tests/*.bats` returns 32
files, none of them `lint-demand-scope.bats`, `lint-silent-protection.bats` or `lint-ext-writable-shape.bats`. Every
statement about a RED's pin text below is `[INFER]` from the proposal/explore and **the executable RED wins at apply**.
`[ev: proposal §10]` `[ev: explore §3.3]`

---

## Technical Approach

Three fixed waves of chained PRs, one destination-file group each, RED-first. The kit half (R2/R3/R10/R11/R12/R13)
reuses the campaign-5/6/7/8 script shape verbatim: header usage block, `set -u`, `printf` rows, typed disjoint exits
(K20), VCS-free, `shellcheck 0.10.0` clean, dot-dirs pruned (D9b). The client half (R1/R6/R8/R9) is **additive-only**
Java: every new slot is `add_slot`, so `schema-risk.sh` must read SAFE and no retype is proposed anywhere (B800 §800.8
is an OUTAGE class). The tunnel half (R4/R5/R7) is designed from the proposal §5 decision plus the viewer echo only,
and is marked `[INFER]` end-to-end. `[ev: corpus B795]` `[ev: corpus B800 §800.8]`

---

## Architecture Decisions

### D1 — S20 time-slice rotation: a two-cycle make-before-break state machine inside `CompressorControl.step`

**Insertion point.** Two new blocks in `step(...)`, both in the PURE model (`CompressorControl.java`, no Baja):
**step 2b** (rotation COMPLETION) inserted between the target clamp (`:212-216`) and the stage move (`:218-233`), and
**step 3b** (rotation ARM) inserted between the stage move's closing brace (`:233`) and the HOA override loop
(`:235-245`). Rejected: a single in-cycle pair of writes — it writes two commands in one cycle with zero `stageDelay`
between them, which is exactly what "make-before-break with `stageDelay`" forbids, and it makes `lastStageMs` lie.
Rejected: placing the whole thing after the HOA loop — the operator override at `:238-245` must remain the LAST word.
`[ev: client CompressorControl.java:212-245 (read 2026-09-06)]` `[ev: corpus S20]`

**Why COMPLETION runs before the stage move.** During the pending window `onCount` is `target + 1`. If ordinary staging
ran first it would see `onCount > target` and shed through `pickMostHoursOn` (`:230`, `:304-314`), which selects by
hours and could drop the INCOMING unit instead of the outgoing one. Evaluating the pending completion first drops the
intended outgoing unit and leaves step 3 seeing `onCount == target`. This ordering is a correctness requirement, not a
style choice. `[ev: client CompressorControl.java:223-232, :304-314]`

**Per-compressor clock.** `cmdSince[k]` (`:71`) already records the ms of the last command change and is seeded at
`atSteadyState` by `seedRestart` (`:285-288`), so a station restart cannot make a just-stopped unit look eligible. The
continuous-run clock is therefore `now - cmdSince[k]` with **no new state field**. Outgoing candidate = the running unit
with the LARGEST `now - cmdSince[k]` (longest continuous run), tie-broken by most `hours`. Rejected: selecting the
outgoing unit by hours alone — that is what `pickMostHoursOn` already does at stage-down, and it does not implement a
*time-slice*. `[ev: client CompressorControl.java:71, :285-288]` `[ev: corpus S20]`

**Incoming selection reuses `pickLeastHoursOff(now, c.minOffMs)` verbatim** (`:291-301`). That single reuse inherits
three required gates for free: HOA OFF exclusion (`modes[k] == MODE_OFF continue`, `:296`), min-off respect (`:297`),
and least-hours lead selection (`:298`). Rejected: a new picker — a second selection function is a second place for the
HOA-OFF rule to drift. `[ev: client CompressorControl.java:291-301]`

**Arm gates (ALL must hold, evaluated in this order).**

| # | Gate | Source of truth |
|---|---|---|
| 1 | `c.rotationIntervalMs > 0` | `0 = disabled` sentinel (SC-1) |
| 2 | no pending rotation (`rotOut < 0`) | one swap in flight at a time |
| 3 | `onCount == target` (steady demand only) | rotation never substitutes for staging `[ev: client :223-232]` |
| 4 | `stageReady` — `(lastStageMs == Long.MIN_VALUE) \|\| (now - lastStageMs) >= c.stageDelayMs` | same expression as `:220`, not a copy of the constant |
| 5 | `!dischargeHigh` | a swap adds a stage; `:213` already forbids adding on high head |
| 6 | `!(suctionValid && suction < c.suctionLowLimit)` | LP floor at `:214` sheds, never adds |
| 7 | `available >= 2` | `available` computed at `:173-175`; a single available unit cannot rotate |
| 8 | outgoing `out >= 0`, `modes[out] == MODE_AUTO`, `(now - cmdSince[out]) >= max(rotationIntervalMs, minOnMs)` | HAND/`MODE_ON` untouched — a `MODE_ON` unit is re-forced ON at `:241` anyway, so swapping it is a guaranteed no-op write |
| 9 | incoming `in = pickLeastHoursOff(now, c.minOffMs) >= 0` | inherits min-off + HOA-OFF |
| 10 | `hours[in] < hours[out]` | a swap into a unit with MORE hours increases divergence — the opposite of the ask |

**Arm action** (one write): `cmd[in] = true; cmdSince[in] = now; lastStageMs = now; rotOut = out; rotArmedMs = now;`.
**Completion action** one or more cycles later, when `rotOut >= 0 && (now - rotArmedMs) >= c.stageDelayMs`:
`cmd[rotOut] = false; cmdSince[rotOut] = now; lastStageMs = now; rotOut = -1;`. Two new transient fields (`rotOut`,
`rotArmedMs`) join the transient block at `:66-75` and are cleared in `resetTransient()` (`:267-274`) — otherwise a
disable→enable leaves a phantom pending swap. `rotationMode = breakBefore` inverts the two actions (drop first, then
add after `stageDelay`); `makeBefore` is the default because break-before-make surrenders a stage for a full `minOff`
on a rack whose entire purpose is holding suction. `[ev: client CompressorControl.java:66-75, :267-274]` `[ev: corpus S20]`

**Hours ledger unaffected.** `hours[k]` integrates on the COMMANDED state at `:158` (`if (cmd[k]) hours[k] += dtH;`),
inside the feedback loop that runs BEFORE both new blocks. A swap therefore changes which unit accrues from the NEXT
cycle onward and never rewrites accrued hours — `condenserNHours` (`:1975-1977` in the adapter) stay monotonic.
`[ev: client CompressorControl.java:145-168]` `[ev: client BCompressorControl.java:1975-1977]`

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
`(int)Math.round(getCondenserNMode())` at `:1952-1956`; they are dashboard-linked and MUST NOT be retrofitted to a
frozen enum (that is exactly the live "Missing class" trap). `rotationMode` is new, has no link, and therefore takes the
frozen-enum form legally. `[ev: client BCompressorControl.java:392-409, :1952-1956]` `[ev: corpus B828 §828.7]`

**Adapter wiring** (one line each, beside `:1892-1902`): `cfg.rotationIntervalMs = getRotationInterval().getMillis();`
and `cfg.rotationMakeBeforeBreak = getRotationMode().getOrdinal() == 0;`. No change to the `ctl.step(...)` call at
`:1959-1960` — the two fields ride the existing `Cfg` object (`:2042`). `[ev: client BCompressorControl.java:1892-1902, :1959-1960, :2042]`

**D1a — the byte-identical golden (SC-1), and how it is actually proven.** The RED commits a fixed input trace
(`~40` steps of `{now, demandCount, amps[], ampsValid[], suction, suctionValid, discharge, dischargeValid, modes[]}`)
and a **committed oracle file** `srcTest/resources/rotation-golden.txt` produced by running that trace against the
**pre-change** `CompressorControl` on the RED branch. Each step emits one canonical line
`<stepIdx>:<cmd0><cmd1><cmd2>|<stagesOn>|<demand>|<pressureFallback>`. The post-change test runs the same trace with
`cfg.rotationIntervalMs = 0` and asserts `assertEquals(golden, actual)` on the joined string. Rejected: asserting final
state only — a swap that reverts inside the trace would pass. Rejected: regenerating the golden from the post-change
class — circular, proves nothing. The oracle file is text, so it satisfies the no-binary-fixture rule (CONTRIBUTING §2).
`[ev: proposal SC-1]` `[ev: retro campaign8 D11 fixtures]`

### D2 / D3 / D10 — the three new kit lints: one shared skeleton, three rules

**Shared skeleton, copied from `lint-write-path.sh`** (the newest and the one whose module-root convention C8 lesson 11c
promotes to doctrine): usage guard → exit 3 when no operand (`lint-write-path.sh:40-43`); `[ -d "$MODULE_ROOT" ]` →
exit 3 (`:75-78`); profile discovery — if `<root>/src` exists scan it as one profile named `basename $root`, else
`find "$MODULE_ROOT" -maxdepth 1 -mindepth 1 -type d \( -name '*-rt' -o -name '*-ux' -o -name '*-wb' -o -name '*-se' \)`
and scan each `<profile>/src`, and **ERROR + exit 3 when no source is found anywhere** (`:134-153`); Java file
enumeration with dot-dirs pruned — `find "$src" -type d -name '.*' -prune -o -name '*.java' -print` (`:351-354`, D9b).
Reusing this file verbatim is the point: C8 lesson 11c exists because three scripts each invented their own root
handling. `[ev: kit toolbelt/lint-write-path.sh:40-43, :75-78, :134-153, :351-354]` `[ev: retro campaign8-close-process-meta-lessons lesson 11c]`

**D2a — row column ORDER is `STATUS check subject detail`, not lint-servlet's order.** Two grammars exist in the kit
today: `lint-write-path.sh:374` emits `FAIL  lint-write-path  <module>  <detail>`, while `lint-servlet.sh:26` documents
`<check>  FAIL|WARN  <file>:<line>  <detail>`. `report-module.sh`'s aggregate parser reads `st=$1; chk=$2` (C8 D1), so
only the first order aggregates correctly. **All three new lints use `WARN  <script-name>  <file>:<line>  <detail>`**
with a whitespace-free subject. `[ev: kit toolbelt/lint-write-path.sh:374]` `[ev: kit toolbelt/lint-servlet.sh:26]` `[ev: retro campaign8 D1]`

**D2b — exit contract (K20), and why 2 and 4 are unused.** All three lints are **advisory** by research decision
(B820 §820.3 and B824 §824.3 both say WARN, never a hard FAIL, because the *presence* case is a data-flow judgment a
static scan cannot settle). Map: **0** = no FAIL rows (WARN rows still exit 0) · **1** = any FAIL row, reachable only
under `--strict` · **3** = usage, not-a-directory, or no Java source under the module root · **2 and 4 are deliberately
NOT emitted** — the K20 requirement is that the verdict range `{0,1}` and the fault range `{3}` stay disjoint, not that
every code be used, and the legacy `2 = usage` map (`lint-timers.sh`) is not retrofitted (C8 D1). A root with no
sources exits **3, never a silent 0**. `[ev: corpus B820 §820.3]` `[ev: corpus B824 §824.3]` `[ev: retro campaign8 D1]`

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

**Smoke expectations, re-anchored to the checkout I read (they differ from B824's cites — flag at apply).**

| Subject | Expected | Evidence at my read | B824's cite |
|---|---|---|---|
| CP-1 low suction | **WARN** | `if (suctionValid && suction < c.suctionLowLimit) target = Math.min(target, onCount - 1);` at `CompressorControl.java:214`; no named field; `suctionLowAlarm` absent from the whole property list (`BCompressorControl.java:43-409` — `stuckAlarm`, `dischargeHighAlarm`, `suctionMismatch` exist, no low-suction slot) | `:215` |
| CR-3 freeze | **WARN** | `freezeTripped` is a **private field** at `BEvaporatorUnit.java:1173`, set in `recomputeFreeze()` `:1033-1039`, consumed by `valveInhibited()` `:1047-1052`; no status/reason slot, no alarm | `:1287` field, `:1106` method |
| CP-2 high discharge | **CLEAN (absence pin)** | named field `this.dischargeHigh` at `CompressorControl.java:82` / `:140`, adapter slot `dischargeHighAlarm` declared at `BCompressorControl.java:361` | `:82,140` — matches |
| defrost-skip | **CLEAN (absence pin)** | `getDefrostSkipped()` + `setLastSkipReason()` SUMMARY writes in `BDefrostController.java` | `:746-747` |

The CR-3 divergence is ~114-235 lines, which means `BEvaporatorUnit.java` moved materially since B824 read `fbe9009`.
The apply worker re-anchors these four subjects at the blessed client tip BEFORE writing the smoke pin; a smoke pin
asserting a stale line number is a fixture-green/real-red failure of exactly the class C8 lesson 11 names.
`[ev: corpus B824 §824.4]` `[ev: retro campaign8-close-process-meta-lessons lesson 11a]` `[ev: corpus B821 re-anchor]`

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

### D6 / D7 — tunnel: `/config/login` step-up and the canonical `change_log` — **`[INFER]` end to end**

The tunnel repo is **not present on this machine** and was not read. Everything in D6/D7 is designed from proposal §5,
the S12 plan, and the viewer echo recorded in `explore.md` §3.4. `[ev: S12 plan 1ecdf437c]` `[ev: explore §3.4]`

**D6 — the seam and the token model `[INFER]`.** `buildServer(cfg, deps)` returns the HTTP handler without binding a
port; `main()` calls it behind `if (import.meta.url === ...)` so the RED can construct a server with injected fakes
(`deps = { supabase, station, clock, spool }`) and `node:test` never opens a socket. Token model: `POST /config/login`
re-authenticates `(email, password)` against Supabase over TLS, then mints a **server-held** random token bound to
`(email, purpose="config-write")` with an absolute TTL of 2-5 min AND a **sliding inactivity window** refreshed on each
mutating call; the client receives an opaque handle only. `POST /config/logout` deletes the entry immediately.
`/write` and `/alarms/ack` require it (401 when missing or expired); `/equipment` and `GET /alarms` do not. The ORD
allowlist is **server-held**, never client-supplied — a write to a non-allowlisted ORD is 403 before the station is
touched. **No user+password store exists anywhere in the diff**; the JSON store path is retracted. `[ev: corpus B803 §803.6]` `[ev: corpus S12]`

**D7 — canonical schema, pre-write GET, and the spool `[INFER]`.** `public.change_log` is extended **additively only**
— `ts`, `config_session`, `result`, `surface` (`'write-server' | 'servlet'`), `client_ip` are ADDED beside the existing
`user_email, user_id, room, slot, label, old_value, new_value, area, ok`. The migration never drops and never retypes a
column; `schema-risk.sh` has no jurisdiction over Postgres, so R5 carries its own additive-only migration pin.
`old_value` comes from a **pre-write GET of the slot**, never from the request body — a client-supplied `old` is a claim,
not a reading. Order per write: GET old → PUT/POST to the station → capture `result` → insert exactly ONE row. On insert
failure the write **still returns the station's outcome** and the row is appended to a local JSON-lines **failure
spool**, drained on the next successful insert; the JSON-lines file is demoted from a parallel trail to that spool only.
The named pin is `audit-append failure never fails the write`. `[ev: proposal §5]` `[ev: S12 plan 1ecdf437c]`

**D7a — the mirror (R7) `[INFER]`.** A flag-gated job (default **OFF**) reads `/PANCCADIA/AuditHistory` and inserts
`surface='servlet'` rows. Idempotence key = `(ts, user, target, old, new)` enforced as a unique index so a replay is a
no-op insert, proven over a recorded fixture replayed twice with a row-count assertion. `config_session` is **NULL** for
surface B this campaign — surface-B step-up (S12 plan Part 3) is not in C9 and a faked session id would be worse than an
empty column. Enabling the mirror live is the B829-live gate, never a PR gate. `[ev: proposal §5]` `[ev: corpus B829]`

### D8 — R6 servlet: `DashboardWriteGuards.evaluate` + the real-Context `set`

**Seam.** A new `DashboardWriteGuards` class exposing
`static Verdict evaluate(HttpServletRequest req, String relOrd, String rawValue)` returning a typed
`Verdict{ int status; String errorJson; Double parsed; }`. `handleSetpointWrite` calls it once and returns early on any
non-200 verdict. This makes the guard order testable without a servlet container — today the four guards are scattered
across three files. `[ev: client BDashboardServlet.java:195-311]`

**Guard order, each cite read at my checkout.**

| # | Guard | Status | Where it lives today |
|---|---|---|---|
| 1 | `X-Requested-With` missing on `/api/*` | **302** redirect home (not 4xx) | `DashboardDispatch.java:141-144` |
| 2 | no authenticated station user | **401** | `DashboardRbacHelper.java:40-41` |
| 3 | user lacks `OPERATOR_WRITE` (fail-closed) | **403** | `DashboardRbacHelper.java:55-56`, `:98` |
| 4 | missing / empty / non-numeric / NaN value | **400** | must move INTO `evaluate` — see D8a |
| 5 | unresolvable ORD, traversal, missing property | **400** | `BDashboardServlet.java:214-269` |

**D8a — guard4 no-silent-zero is a REGRESSION pin against live code.** `parseDouble` at
`BDashboardServlet.java:386-391` returns **`0.0`** for null input and for any `NumberFormatException`, and
`coerceValue` at `:344-347` feeds it straight into `new BStatusNumeric(parseDouble(rawValue))`. So a body of
`{"ord":"Cuarto1/setpoint","value":"abc"}` today writes a setpoint of **0.0** and answers `200 {"ok":true}`. Guard 4
must reject BEFORE `coerceValue` is reached (`:273`), and guard4b pins the boundary values (`""`, `"abc"`, `"NaN"`,
`null`) each returning 400 with no `parent.set` call. `[ev: client BDashboardServlet.java:344-347, :386-391, :273]`

**D8b — the real Context.** `parent.set(prop, toSet, null)` at `:274` becomes `parent.set(prop, toSet, cx)` where `cx`
carries the authenticated request user. `ComplexSlotMap.set:662` gates the whole `AuditEvent` construction on
`context != null && context.getUser() != null`, and dispatch on `Nre.auditor != null` (`:1685`) — the second gate is
already satisfied on PANCCADIA (`AuditHistoryService` installed, B829-G1 CLOSED). So this one argument converts the
servlet write from **suppressed** to Niagara-audited. The change is schema-neutral (no slot touched) and
`schema-risk.sh` must read SAFE. The existing fire-and-forget module audit at `:287-302` is preserved unchanged —
it already implements audit-fail-never-fails-the-write, and the RED pins that it stays that way.
`[ev: corpus B829 §829.1/§829.2]` `[ev: client BDashboardServlet.java:274, :287-302]`

### D9 — R8 alarm Pattern A (CR-3 freeze, declarative)

`BAlarmSourceExt extends BPointExtension` and `BPointExtension.isParentLegal` is a hard
`return parent instanceof BControlPoint`; `BEvaporatorUnit extends BComponent`, so the ext **cannot** mount on it.
Pattern A adds a **child `BBooleanPoint freezeAlarmPt`** with a child `BAlarmSourceExt` whose `offnormalAlgorithm` is a
`BBooleanChangeOfStateAlgorithm` (`alarmValue = true`). The point's `out` is driven from the existing latch:
`recomputeFreeze()` (`:1033-1039`) already computes `freezeTripped` through `ColdRoomControl.freezeTrip(...)`, so the
single added line writes `freezeAlarmPt.out` from that field on change. The ext raises and clears on the point's edge —
no alarm Java, no edge state machine to get wrong. Schema-risk = SAFE (`add_slot`, frozen child). Making
`freezeTripped` visible also **closes the CR-3 WARN that D4's lint raises**, so R3 and R8 must not be reviewed as
independent: R8 changes R3's smoke expectation, and whichever merges second updates the pin.
`[ev: corpus B827 §827.2/§827.3/§827.6]` `[ev: client BEvaporatorUnit.java:1033-1039, :1047-1052, :1173]` `[ev: corpus B795]`

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

### D11 — R11 write-path matrix: two corrections the proposal needs

**D11a — the matrix is in the CLIENT repo, not the kit.** `build-n4-module-kit/docs/` **does not exist**; a glob of
`**/write-path-matrix.md` under `niagara-tools` returns only the two lint fixtures
(`tests/fixtures/lint-write-path/{covered,uncovered}/docs/write-path-matrix.md`). The C8 write-path retro records the
real layout: the smokes ran `git -C <client> archive origin/main Paccadia/ColdRoomPan-rt Compresores/CompPan-rt docs`
and the walk-up found **`<client-repo-root>/docs/write-path-matrix.md`**, 2-3 levels above each module root. R11 is
therefore a **client-repo doc PR**, and proposal §7's `build-n4-module-kit/docs/write-path-matrix.md` row is wrong.
`[ev: kit retros/2026-09-06-campaign8-write-path.md:104, :108]` `[ev: glob niagara-tools/**/write-path-matrix.md 2026-09-06]`

**D11b — nine rows (W14-W22) cannot reach SC-9's exit 0.** The same retro records the measured uncovered sets at client
`origin/main a109249`: **ColdRoomPan-rt = 13** with `--bog` (`coolOnSensorFault, duration, evapHighAlarmLimit,
evapLowAlarmLimit, fanMode, freezeDiffStop, freezeProtect, interval, powerOnDelay, resistanceTempThreshold,
staggerDelay, startDelay, valveMode`) and **CompPan-rt = 15** (`condenser1Mode, condenser2Mode, condenser3Mode,
faultReset, floatingSuction, minOn, powerOnDelay, runningAmpsThreshold, stageDelay, stageDownDelay, stageUpDelay,
startProveDelay, suctionBand, suctionLowLimit, suctionMismatchTol`). Deduplicated (`powerOnDelay` is shared) that is
**27 distinct slots**, plus whatever DashboardPan-rt/ux contribute. PR1 then adds **two more** (`rotationInterval`,
`rotationMode`). **Decision**: R11's row set is `W14…W40+` sized by the measured set, not a fixed nine, and PR1 carries
its own two rows in its own PR so the matrix never lags the slot. SC-9's "exits 0 on every client module root" holds
only under that full set. `[ev: kit retros/2026-09-06-campaign8-write-path.md:110-166]`

### D12 — doctrine fold targets, one `[ev:]` per row

| Delta | Target § | Evidence token |
|---|---|---|
| Slot types for externally written values (SIMPLE value or a writing `@NiagaraAction`; a bare `Flags.OPERATOR` complex rejects the write or silently writes a default) | `types/logic-authoring.md` §"Slot types for externally written values" + one-liner in `types/dashboard.md` | `[ev: corpus B823]` `[ev: corpus B826]` |
| Alarm authoring Pattern A (child `BControlPoint` + `BAlarmSourceExt`) and Pattern B (`BIAlarmSource` + `AlarmSupport` on the offnormal EDGE); both route `sourceState=offnormal`; both `add_slot`-SAFE; the ext needs a `BControlPoint` parent | `types/logic.md` §"Protection anatomy" | `[ev: corpus B827]` `[ev: corpus B821 §821.4]` |
| A protection trip that forces an output or sheds a stage MUST write a named SUMMARY/OPERATOR status-or-reason slot (tier 2); a SAFETY trip should raise tier 1; the lint WARNs the tier-4 gap | `types/logic.md` §"Protection anatomy" + "folded as code: `lint-silent-protection.sh`" | `[ev: corpus B824]` |
| ONE module-root/profile convention — module root contains the profiles; every lint ITERATES `<name>-(rt\|ux\|wb\|se)`; a root with no sources or no matrix is ERROR exit 3, never a silent 0 | `BUILD-LOOP.md` §5 | `[ev: retro campaign8-close-process-meta-lessons lesson 11c]` |
| **K22** — a real-tree smoke on every client module root is part of the lead gate; a smoke pin asserts exact count AND subject AND absence; a skipped smoke BLOCKS | `METHODOLOGY.md` §Kit maintenance (K21 is the current last entry at `METHODOLOGY.md:85`, so **K22 is confirmed**, no longer `[INFER]`) | `[ev: retro campaign8-close-process-meta-lessons lesson 11a/11b]` `[ev: kit METHODOLOGY.md:85]` |
| Mutation tables record OBSERVED flips (verbatim RED-then-GREEN), never "would flip" prose | `METHODOLOGY.md` §Kit maintenance | `[ev: retro campaign8-lint-timers-ext]` |
| Four always-conflict files + the fragment-merge rule (append, keep both rows, dedupe by script name, never overwrite) | `METHODOLOGY.md` §Multi-session (extends K12) | `[ev: retro campaign8-close-process-meta-lessons lesson 2]` |
| Lead merge/settle order — merge ff-only → verify `git log -1` equals the blessed tip → THEN settle the ledger | `BUILD-LOOP.md` §7 | `[ev: retro campaign8-close-process-meta-lessons lesson 10]` |
| Unified write-audit reconciliation (one canonical sink, two writers, dedupe key, audit-fail never fails the write, a null-Context servlet write is SUPPRESSED not merely unattributed) | `types/dashboard.md` + `BUILD-LOOP.md` §6 note | `[ev: corpus B829]` `[ev: S12 plan 1ecdf437c]` |
| K19 routing for `lint-demand-scope.sh`, `lint-silent-protection.sh`, `lint-ext-writable-shape.sh` — **ships in the PR that lands each script**, never deferred to PR12 | `BUILD-LOOP.md` + `skill/SKILL.md` | `[ev: retro campaign8 D8 K19/CD5]` |

Before writing any line: `grep -rn "<rule keyword>" build-n4-module-kit/` — the mined target is a suggestion, the rule
may already live elsewhere (K6). `sweep-fold-audit.sh --strict` refuses to credit a row with no `[ev:]` token.

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
| `lint-demand-scope.sh` | `<module-root> [--strict]` | `0` no FAIL (WARN=0) · `1` FAIL under `--strict` · `3` usage/env/no-source |
| `lint-silent-protection.sh` | `<module-root> [--strict]` | `0` · `1` · `3` |
| `lint-ext-writable-shape.sh` | `<module-root> [--strict]` | `0` · `1` · `3` |
| `lint-write-path.sh` | unchanged — `<module-root> [--bog <config.bog>] [--matrix <path>]` | `0` · `1` · `3` (unchanged) |
| `report-module.sh` | unchanged usage + three appended member rows | `0` · `1` · `3` |

---

## PR Matrix

| PR | R | Branch | Repo | RED (unread here) | Files | Named mutation |
|---|---|---|---|---|---|---|
| 1 | R1 | `feat/c9-comppan-rotation` | client | to author | `CompressorControl.java` (2b/3b + 2 transient fields), `BCompressorControl.java` (2 slots + 2 cfg lines + generated block), `BRotationMode.java`, `CompressorControlTest`, `rotation-golden.txt` | set `rotationIntervalMs = 0` internally → the swap pins fail; delete gate 4 (`stageReady`) → the make-before-break ordering pin fails |
| 2 | R2 | `feat/c9-demand-scope` | kit | `qa/c9-demand-in-scope` `2916954` | `toolbelt/lint-demand-scope.sh`, `tests/lint-demand-scope.bats`, fixtures, `ci.yml`, K19 routing ×2 | DS2 — drop the field scan → a class-field demand input stops counting |
| 3 | R3 | `feat/c9-silent-protection` | kit | `qa/c9-silent-protection` `e38e503` | `toolbelt/lint-silent-protection.sh`, tests, fixtures, `ci.yml`, K19 routing ×2 | SP8 — drop the field→slot follow → CP-2 starts WARNing |
| 4 | R4 | `feat/c9-s12-config-login` | tunnel | `qa/c9-s12-write-server` `24adcba`, rebase → `9acb47c` | `write-server.mjs` seam + token store, tests | drop the config-token check → S12A-1/S12A-5 flip |
| 5 | R5 | `feat/c9-s12-audit-schema` | tunnel | extends the same RED | migration, audit writer, spool | make the insert throw → the write must still return the station outcome |
| 6 | R6 | `feat/c9-s12-servlet-guards` | client | `qa/c9-s12-servlet` `4c18837` | `DashboardWriteGuards.java` (new), `BDashboardServlet.java`, `srcTest` | remove guard 4 → `"abc"` writes 0.0 again (no-silent-zero regression) |
| 7 | R7 | `feat/c9-s12-audit-mirror` | tunnel + kit doc | to author | mirror job, dedupe index, kit reconciliation doc | replay the fixture twice → row count must not double |
| 8 | R8 | `feat/c9-alarm-cr3` | client | to author | `BEvaporatorUnit.java` child point + ext | remove the ext → no `offnormal` record |
| 9 | R9 | `feat/c9-alarm-cp1` | client | to author | `BCompressorControl.java` `BIAlarmSource` + `AlarmSupport` | make it level-triggered → the once-only pin fails |
| 10 | R10 | `feat/c9-ext-writable-shape` | kit | `qa/c9-ext-writable-shape` `3726722` | `toolbelt/lint-ext-writable-shape.sh`, tests, fixtures, `ci.yml`, K19 routing ×2 | drop the `@NiagaraAction` exemption → a clean complex-with-action starts WARNing |
| 11 | R11 | `docs/c9-write-path-rows` | **client** (D11a) | `qa/c8-write-path` `5e357d1` | `<client-root>/docs/write-path-matrix.md` — the measured set (D11b) | delete one row → that slot FAILs again |
| 12 | R12 | `docs/c9-doctrine` | kit | none | `types/{logic,logic-authoring,dashboard}.md`, `BUILD-LOOP.md`, `METHODOLOGY.md` (K22) | none; guard = `sweep-fold-audit.sh --strict` + `kit-links.bats` |
| 13 | R13 | `chore/c9-close` | kit | none | `tests/c9-close.bats`, `VERSION`, `CHANGELOG.md`, `retros/INDEX.md`, `BUILD-STATE.md` | none; the gate is the test |

---

## Threat Matrix

| Boundary | Applicability | Design response | RED |
|---|---|---|---|
| Untrusted content decode | **Applicable** — the three lints read customer Java; the mirror reads an AuditHistory export | `LC_ALL=C` byte reading, `grep`/`awk` only, no `eval`, no interpolation of file content into a command; malformed input → exit 3, never a crash | a malformed-source fixture per lint |
| Routing / auth boundary | **Applicable** — `/config/login`, `/config/logout`, the token gate on `/write` and `/alarms/ack`, and the servlet's four guards | Server-held token bound to `(email, purpose)`, absolute TTL + sliding window, server-held ORD allowlist, fail-closed guards in a fixed order, no user+password store | S12A-1..7; guard1-5 + guard4b |
| Input validation / silent coercion | **Applicable** — `parseDouble` returns 0.0 on any parse failure (`BDashboardServlet.java:386-391`) | Guard 4 rejects before `coerceValue`; a non-numeric is 400, never a 200 that writes 0.0 | guard4 / guard4b no-silent-zero |
| Filesystem write | **Applicable** — the tunnel failure spool | Appends only to a configured spool path; drained, never executed; never `+x` | spool-append pin |
| Subprocess / external tool | **Applicable** — `python3` only in the pre-existing `--bog` helper | The three new lints add no subprocess; `command -v python3 \|\| exit 3` stays where it is | unchanged |
| Git / VCS automation | **N/A** — every toolbelt script is VCS-free; `kit-links.bats` L2 fails the suite if any names `git` | — | L2 (existing) |
| Executable-file classification | **N/A** — no new artifact is marked executable except the three `toolbelt/*.sh`, matching every existing script | — | — |

---

## Migration / Rollout

**Schema-risk expectation: SAFE on every client slice.** PR1 adds two frozen properties and one new enum type; PR8 adds
a frozen child point + ext; PR9 adds a frozen action + a transient field. All are `add_slot`, and an old `.bog` with no
entry takes the new default. **No retype and no slot removal appears anywhere in C9** — B800 §800.8 makes retype an
OUTAGE class, and the user rejected the setpoint retype explicitly. PR6 touches no slot at all (schema-neutral).
`[ev: corpus B795]` `[ev: corpus B800 §800.8]` `[ev: proposal §2.4]`

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

1. `git show origin/qa/c9-silent-protection:tests/lint-silent-protection.bats` — confirm the SP-smoke subject lines
   against D4's table. **If the RED pins `CompressorControl.java:215` or `BEvaporatorUnit.java:1287`, the RED and this
   design were read at different tips — re-anchor before apply, do not "fix" one to match the other blindly.**
2. `git show origin/qa/c9-demand-in-scope:tests/lint-demand-scope.bats` and `…:tests/lint-ext-writable-shape.bats` —
   confirm the row column ORDER matches D2a (`STATUS check subject detail`), not `lint-servlet.sh`'s order.
3. `git -C <client> show <tip>:…/CompPan-rt/src/com/angeles/CompPan/CompressorControl.java` — confirm D1: the target
   clamp block, the stage-move block, the HOA loop, `cmdSince[]`, `seedRestart`, `pickLeastHoursOff`. If the stage-move
   block is not immediately followed by the HOA loop, D1's insertion points move.
4. `ls build-n4-module-kit/docs/` — must be **absent**, confirming D11a; then
   `git -C <client> show origin/main:docs/write-path-matrix.md | grep -c '^|'` to size D11b's row set.
5. `sed -n '80,90p' build-n4-module-kit/METHODOLOGY.md` — confirm K21 is the last K-number, so K22 is free.
6. Cross-check every proposal ID (R1-R13, SC-1..SC-13) against a D-number; a requirement with no design home is a gap.
   Confirm no line proposes a station write or a live jar deploy.

## Open Questions

- [x] **K22 number** — RESOLVED. `METHODOLOGY.md:85` ends at K21, so K22 is the correct next number (was `[INFER]` in proposal §10).
- [x] **D11a** — RESOLVED. `write-path-matrix.md` lives in the CLIENT repo root; the kit has no `docs/`. R11 is a client PR.
- [ ] **D11b** — the exact W-row count depends on the DashboardPan-rt/ux uncovered set, which was not measured in C8. Size it at apply from a real `lint-write-path.sh` run on all four roots.
- [ ] **R3 ↔ R8 interaction** — landing R8 (CR-3 alarm) removes the CR-3 WARN that R3's smoke pin asserts. Whichever merges second updates the pin in its own PR; they must not be reviewed as independent.
- [ ] **All tunnel design (D6/D7/D7a)** is `[INFER]` — the repo was unreadable from this machine. Re-derive against `9acb47c` at apply before writing a line.
- [ ] **All QA RED pin texts** are `[INFER]` — none was readable here. Where this design and a landed RED disagree, **the executable RED wins**.
