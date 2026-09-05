# Design: build-n4-module-campaign8

**Phase**: design · **Source**: `v0.18.0` (main `48fb210`, 179 bats) · **Target**: `v0.19.0`
**Inputs**: `explore.md`, `proposal.md`, `spec.md` (incl. R7.10/SC11), campaign-7 `design.md` (shape),
research B775 §775.6 / B800 / B801 / B802 / B803 §803.6 / B804 / B805 (`ccd8ab95d`, landed mid-design) /
B806, consolidated fold retro D1-D8, audit shards (comppan, coldroompan, dashboardpan, chihuahua, logs,
wb, **bog — landed, PR10 now bound**).
**QA REDs bound** (K13 — branch is the durable reference, tip read 2026-09-05):
`qa/c8-lint-delays` `c96b2ad` (LD1-LD9) · `qa/c8-triage-console` `6492b2d` (TR1-TR9) ·
`qa/c8-lint-timers-ext` `ce6ee5c` (TC-A/B/C + 4 companions). PR4-PR13 REDs pending.
**Precedence**: where `spec.md` and a landed RED disagree, the **executable RED wins**. Four divergences
exist — D2a, D6a, D7a, D9a — each carrying a one-line spec correction, all **resolved** (Open Questions).

## Technical Approach

Two waves of chained PRs, one destination-file group each, RED-first. Every new script follows the
campaign-5/6/7 shape verbatim: header usage block, `set -u`, `row()` printf, typed disjoint exits (K20),
VCS-free (`kit-links.bats` L2), `shellcheck 0.10.0` clean, no `$HOME` (K8). Behaviour is fixture- or
table-driven so the *data* is the contract and each script stays thin. Checks that fit an existing tool
extend it; only the new evidence surfaces (station console, saved `config.bog`) become their own scripts.

## Architecture Decisions

### D1 — one row grammar for new scripts; the legacy exit map is NOT retrofitted

**Rejected**: normalising every script to one exit map — it breaks `lint-timers` TL1-TL4 (`2`=usage today)
and `report-module`'s 3-column lint-timers parser for zero evidence gain. **Chosen**: new scripts get the
4-column row and `0/1` verdict with `3` for usage *and* env; existing scripts are untouched and the two
conventions are stated in each header.

New standalone scripts (`lint-delays`, `triage-console`, `rc-scan`, `station-snapshot`, `bog-audit`) emit
`printf '%-4s  %-14s  %s  %s\n' <STATUS> <check> <subject> <detail>` and exit `0` no FAIL · `1` any FAIL ·
`3` usage/env. This is forced by the REDs (LD9, TR4 both pin `3` for a missing operand) and is K20-legal:
the verdict range `{0,1}` and the fault range `{3}` stay disjoint. `lint-timers.sh` keeps its 3-column
`STATUS  check  <file>: <detail>` row and `2`=usage/`3`=env (R3.4, four green pins). `verify-module.sh`
keeps `%-4s %-9s` 4-column rows. **Consequence for PR8**: `report-module.sh`'s existing 4-column awk
(`st=$1; chk=$2; detail=$4..NF`) ingests the new rows unchanged — so `<subject>` MUST be a single
whitespace-free token (`<file>:<line>`, or the console basename).

### D2 — lint-delays: two-pass grep/awk over Java, fail-loud on anything unresolved

No Java parser: the kit is JDK-free by contract (`verify-module.sh`: "needs only unzip, od, grep, awk,
sort") and a parser jar would make this the only check that cannot run on a station box. Also rejected:
matching literal arguments only — the real bug binds `Math.max(delayMs, 0L)` to a local, so a literal-only
lint has zero bite. Pass 1 collects per file: (a) `static final BRelTime <NAME> = BRelTime.make*(<lit>)` constants;
(b) `@NiagaraProperty` blocks joined until parens balance (the schema-risk D4 technique) → slot → facet
`BFacets.MIN, BRelTime.makeSeconds(n)` map, plus the `newProperty(..., BFacets.make(BFacets.MIN, …))`
slotomatic form; (c) getter → slot (`get<Slot>()` ↔ `name = "<slot>"`, plus a literal `get("slot")` body).
Pass 2 classifies the `BRelTime` argument of every `Clock.schedule*(` call site:

| Argument shape | Verdict |
|---|---|
| `BRelTime.make*(k)` literal `k ≥ 1` · identifier resolving to such a constant (`POLL`) | PASS |
| `BRelTime.make(0)` / `makeSeconds(0)` | FAIL `literal-zero` |
| `Math.max(x, k)` with `k ≥ 1` (inline or via a same-method `long <id> = …` binding) | PASS |
| `Math.max(x, k)` with `k ≤ 0` | FAIL `zero-floor` |
| `get<Slot>()` whose facet `MIN ≥ 1` | WARN `facet-floor` (exit stays 0) |
| `get<Slot>()` whose facet `MIN = 0` or has no MIN | FAIL `facet-min-zero` |
| anything else (unresolved identifier/expression) | FAIL `unfloored` — never silently exempt (campaign-7 D6/L3 precedent) |

Local binding resolution scans backwards from the call site to the enclosing method's opening brace only
— one method, one file, no cross-file inference.

**D2a — the spec's LD-FAIL line list is wrong; the real pre-fix tree is the oracle.** `spec.md` says FAIL
at `BDefrostController.java:566,622,664`. The pre-fix tree at
`~/modulos_niagara_n4/Cliente/Leon-Guanjuato/Paccadia/ColdRoomPan/ColdRoomPan-rt/src` actually carries
`:555-556` (`Math.max(delayMs, 0L)` → `Clock.schedule`), `:566` `getInterval()`, `:620` `getDuration()`,
`:664` `getStaggerDelay()` — those three getters all carrying facet `MIN = makeSeconds(0)` at
`:170/:216/:285` — while `:622`/`:641` schedule the constant `POLL = BRelTime.make(5000)` (`:731`) and MUST
**not** FAIL. Corrected FAIL set: `{556, 566, 620, 664}`; `:622/:641` are the false-positive control that
forces constant resolution into the design. LD5 bites today because this tree is still pre-fix.

### D3 — triage-console: byte-safe scan, level normalisation, three attribution channels

`triage-console.sh [--package com.angeles] [--console-dir <dir>] <console.txt>…`. All reading is
`LC_ALL=C` with `grep -a`/`awk`; bytes are never decoded (`console-es.txt` carries a raw `0xD3`), so
Spanish levels are matched by ASCII prefix: `GRAVE`→SEVERE, `ADVERTENCIA`→WARNING, `INFORMACI`→INFO
(covers `INFORMACIÓN` and its mojibake). Normalisation happens **before** grouping (R2.4/TR9). Rejected:
`iconv`/python re-encoding — the bytes are genuinely invalid UTF-8; B800 §800.5 specifies byte reading.

| Channel | Trigger | Key |
|---|---|---|
| C1 own frame | a stack line `at <--package>.…` under an exception | `LEVEL·class·norm(msg)·own-frame` |
| C2 own logger | level ≥ WARNING and the `[tag]` is **not** in the framework denylist (`sys`, `sys.xml`, `web*`, `fox`, `box`, `driver`, `station`, `alarm`, `history`, `jetty`) | `LEVEL·tag·norm(msg)` |
| C3 load-fail shape | `SEVERE [sys] Cannot load station` (+ the following exception line) and `[sys.xml]` lines matching `Cannot set property\|Missing frozen property\|Cannot decode slot` | `LEVEL·C3·class·norm(msg)` |

`norm(msg)` = strip the bracketed timestamp, then `s/[0-9]+/N/g` (TR5's two `controlTick N` lines collapse to
count 2). Timestamps parse from `[HH:MM:SS DD-Mon-YY TZ]` via a month table carrying EN and ES abbreviations
(`ene feb mar abr may jun jul ago sep|set oct nov dic`) into a sortable `YYMMDDhhmmss`; first/last = min/max.
Row: `FAIL  triage-console  <console-basename>  <count>x <first> -> <last> <LEVEL> <Class>: <msg> @ <frame>`
(C2/C3 carry `@ [tag]` instead). Exit `1` any row · `0` none · `3` usage/unreadable. An exception matching no
channel (the jetty NPE) produces no row — that is TR1's and TR3's whole bite.

### D4 — lint-timers extension: one script, three appended passes, TL1-TL4 code untouched

The new checks are appended after the existing per-file loop body and reuse `row()`/`FAILED`; no existing
line is edited, so the four green pins cannot regress. Rejected: a fourth script — `report-module.sh`
already invokes `lint-timers.sh` per artifact, so extending it routes the three rules into the aggregate
for free and needs no new K19 entry.

- **`companion-flag`** — a `<ident> = true;` within ±3 lines of a `Clock.schedule*` assignment in the same
  method is a candidate; the check then extracts the `stopped()` **and** `started()` bodies (brace-counted,
  the existing `found_cancel` awk technique) and requires `<ident> = false` **inside one of them**. A clear
  anywhere else does not count — the RED's `BStaggerHold` clears `startingUp` in the expiry handler and MUST
  still FAIL, so a bare `grep 'startingUp = false'` fails the pin. Row names the field (`startingUp`).
- **`jdk-thread`** — file declares a Niagara component (`class B\w+ extends B\w+`) **and** matches
  `ScheduledExecutorService|Executors\.|new Thread\(` → FAIL naming the class. `PlainPool` (no `B` prefix,
  no `extends B*`) is the false-positive control.
- **`changed-sched`** — extract the `changed(`/`started(` bodies; collect direct `Clock.schedule*` calls and
  bare one-level callee names (`foo();`); for each callee extract its body. FAIL unless
  `isRunning()|atSteadyState()` appears **in the same body that contains the schedule**. TC-C's `changed()`
  *does* carry `if (!isRunning()) return;` and must still FAIL; the fix puts `!Sys.atSteadyState()` inside
  `applyRunCmd()`. Guard-in-the-scheduling-body is therefore the contract, not guard-anywhere-upstream.
  Following is exactly one level deep (the named mutation removes it).

### D5 — facet presence (and the Java ORD literal) are `verify-module.sh --src` sub-checks, not scripts

**Rejected**: a `facets-lint.sh` — it would need its own profile-dir resolution, report row, K19 routing and
CI step for evidence `--src` already walks. **Chosen**: a `check_facet_presence` function beside
`check_raw_double_facets`, so rows and aggregation come free. Check label `facets-req`. It parses parens-balanced `@NiagaraProperty` blocks carrying `flags = "o"` /
`Flags.OPERATOR` and applies: numeric type (`BDouble|BFloat|BInteger|BLong|BRelTime|BStatusNumeric`)
without both MIN and MAX → FAIL; name matching `[Ss]etpoint|Temp|Limit|Band|Psi` without `UNITS` → FAIL;
name matching `count|Count|stages|demand` without `PRECISION` → FAIL; enum/range type without `RANGE` →
FAIL. **False-positive control (R4.2)**: this check tests *presence only* and never reads a facet's value,
so a valid `MIN = 0` is invisible to it. MIN-value semantics belong to `lint-delays` alone — splitting them
between two checks is what produces contradictory verdicts on the same slot.

PR4 also lands a second `--src` sub-check, **`ord-literal` (WARN)**: a Java string literal matching
`"(station:\|local:\|slot:/)` under `<profile>/src`, because a hardcoded ORD is a Java-source defect and
`rc-scan` scans `rc/` only (D7a). Three exemptions keep it low-FP (K2): a literal inside a
`@NiagaraProperty` `defaultValue`, anything under `srcTest/**`, and a literal in a class whose name matches
`*OrdConstants*` carrying a justification comment on or above the declaration — the sanctioned way to hold
a deployment ORD in one reviewable place. Real-shape fixture: `DashboardReader.java:75` `SERVICE_ORD`
(sanitised). WARN not FAIL: a wrong-but-working ORD is a portability defect, not a broken build (K3).

### D6 — slot-coverage: additive `per-slot` subcommand, plus one deliberate pin change

`slot-coverage.sh per-slot <module-include.xml> <module.lexicon> <src-dir>` is dispatched **before** the flag
loop (the `set-coverage` pattern) so it is never read as a file positional. It emits `pct=<n.n> (per-slot)`
(K14 — the label is the metric's meaning), `missing=<Type.slot,…>` and a new `stale=<Type.slot,…>`:
required = every `@NiagaraProperty(name="x")` per type from `<src-dir>`, declared = lexicon `Type.slot` keys,
stale = a key with no matching annotation. Type-level parse output is byte-unchanged.

**D6a — R5.3 contradicts a green pin.** SC6-parse pins *empty lexicon + 3 types → exit 0*; R5.3 requires
exit 1. Decision: escalate in **parse** mode (not only `--strict`) and amend SC6-parse in PR5, recorded in
`CHANGELOG.md` as a behaviour change — `chihuahua` ships an empty lexicon and passes the aggregate today, so
a `--strict`-only gate leaves the very module that motivated E green. `report-module.sh` renders it as a
`FAIL  slot-coverage  empty lexicon` row (member exit 1 is a verdict, not the `ERROR`/exit-3 path).

### D7 — rc-scan: one file class (`rc/` assets), vendored assets excluded

`rc-scan.sh <artifact-dir> [--strict]` scans **`**/rc/**` `*.html *.js *.css` only**. **D7a — resolved**:
the spec's ORDFAIL shape (`DashboardReader.java:75`) is Java and does not belong here; a `.java` ORD literal
is caught by the new `verify-module.sh --src` `ord-literal` WARN sub-check (D5, PR4), so each file class is
scanned by exactly one tool and neither reimplements the other's walk. `index.html:701` is `rc-scan`'s own
real host-literal shape.

| Check | Pattern (`rc/` assets only) | Verdict |
|---|---|---|
| `ord-literal` | a string literal matching `(station:\|local:\|slot:/\|h:)` | FAIL |
| `host-literal` | `http://` or a bare `([0-9]{1,3}\.){3}[0-9]{1,3}` | FAIL |
| `bare-catch` | `catch\s*\(\s*\(\s*\)\s*=>\s*\{\s*\}\s*\)` | WARN (`--strict` FAIL) |
| `null-branch` | `\?\s*null\s*:` or a `= null` feeding `textContent\|innerHTML` on the same line | WARN (`--strict` FAIL) |

Exclusions: `rc/ext/**`, `*.min.js`, `srcTest/**`, and comment-only lines. Rejected: scanning vendored
`ext/` (three.js + chart.js are megabytes of third-party JS full of IPv4 literals — guaranteed noise,
K2 demands low false positives).

### D8 — PR7 doctrine placement: one token per line, grep before folding (K6)

| Delta | Lands in | Token |
|---|---|---|
| 8-layer timer SM defense (R7.1) + fold D1 (`BTimeTrigger` layers 5/6, PARTIAL 4, monitor tick `[INFER]`) | `types/logic.md` §Safety-fail-modes-&-timers | `[ev: corpus B775]` `[ev: corpus B801]` |
| D2 delay-floor rule + "folded as code: `lint-delays.sh`" prose line | `types/logic.md` timer § | `[ev: corpus B801]` |
| D3 inter-module comms (R7.3, B802-G1 OPEN) | `types/logic-authoring.md` §"talking to another module" (new) | `[ev: corpus B802]` |
| D4 step-up auth (R7.4) + D5 CsrfUtil correction to DWS1 gate 2 + #49 client guidance (R7.5) | `types/dashboard.md` | `[ev: corpus B803]` |
| D7 triple-attribution + locale contract (R2.6) + "folded as code: `triage-console.sh`" | `triage-console.sh` header **and** `METHODOLOGY.md` §Conformance rules | `[ev: corpus B800]` |
| D8 cert-chain trust note (R7.6) + "folded as code:" line | `verify-module.sh` header §Checks | `[ev: corpus B800]` |
| Mandatory `schema-risk.sh` pre-deploy step, exit 2 blocks deploy (R7.2) | `BUILD-LOOP.md` §5 (pre-deploy), one line + the exit-2 rule | `[ev: corpus B795]` + CERT-live |
| Decompiled line numbers are build-specific (R7.7) | `METHODOLOGY.md` §Conformance rules | `[ev: corpus B801]` |
| "Excavador Técnico" working profile (R7.9) | `METHODOLOGY.md` §0 **and** `skill/SKILL.md` | `[ev: corpus B801]` |
| Station load budget table + §806.9 8-count checklist (R7.10/SC11) | `METHODOLOGY.md` §"Station load budget" (new) | `[ev: corpus B806]` |
| K19 routing for `lint-delays` / `triage-console` / `rc-scan` | `BUILD-LOOP.md` + `skill/SKILL.md` | — |

Before writing any line: `grep -rn "<rule keyword>" build-n4-module-kit/` — the mined target file is a
suggestion, the rule may already live elsewhere (K6). Every folded row needs its `[ev:]` token in kit text
or `sweep-fold-audit.sh --strict` refuses to credit it; the three code-folded rows (D2/D7/D8) also need an
explicit "folded as code: `<script>`" prose line.

**K19/CD5 materialisation — routing ships with the script, never in PR7.** Every PR that lands a new
toolbelt script adds its routing lines to **both** `BUILD-LOOP.md` and `skill/SKILL.md` **in that same PR**,
each tagged `[ev: retro <token>]`: PR1 `lint-delays.sh`, PR2 `triage-console.sh`, PR6 `rc-scan.sh`, PR9
`station-snapshot.sh`, PR10 `bog-audit.sh`, PR11 `lint-wb-threading.sh`, PR12 the `rc-scan --servlet` mode.
Deferring them to PR7 would leave `kit-links.bats` L4/L5 RED on every merged tree between the script's merge
and PR7's — the exact gap K19 exists to close. PR7 only **consolidates doctrine**; it adds no routing line
for a script that already shipped one.

### D9 — report-module: two new member invocations, one explicit row, SKIP semantics

`report-module.sh <module-root> [--target-version x.y] [--console-dir <dir>]`. Per artifact, appended
after the existing four steps: `lint-delays.sh <artifact>/src` (skipped silently when `src/` is absent —
the existing `lint-timers` gate) and `schema-risk.sh`. Once per run, not per artifact: `triage-console.sh
--console-dir <dir>`.

| Member | Row | SKIP condition |
|---|---|---|
| `lint-delays` | its own rows, re-emitted with the artifact prefix (verify-module parse path) | no `<artifact>/src` |
| `triage-console` | one row per group, subject = console basename | `--console-dir` absent → `SKIP  triage-console  no --console-dir` |
| `schema-risk` | one explicit row: `PASS\|WARN\|FAIL  schema-risk  <artifact>  verdict=<V>` | no `<artifact>/.deploy-baseline` snapshot → `SKIP` |

**D9a — `schema-risk.sh` exits `0/1/2` as verdicts, `3/4` as faults.** `report-module.sh` maps `0`→PASS,
`1`→WARN (LOSSY), `2`→**FAIL** (OUTAGE), `3|4`→`ERROR` row + exit 3 — exit 2 must not reach the existing
`ERROR` branch. The B798 `--src` contract, row shape and exit set are otherwise untouched.

### D10 — wave 2, at contract level

- **`station-snapshot.sh <station-dir|mounted-copy> <out-dir>`** — copies `config.bog`, `console*.txt` and
  history/alarm db *pointers* (paths + sizes, never the db files) into `<out-dir>/<station>-<UTCstamp>/`
  plus a sorted `manifest.txt` (`sha256  relpath  bytes`). Read-only by construction: no station
  connection, `cp` only, source never opened for write; the named mutation adds a source write and the
  `chmod a-w` pin turns RED. Exits `0/1/3`.
- **`bog-audit.sh <config.bog> --module <MOD>… [--source-dir <DIR>] [--strict]`** — **now concrete** (shard
  landed). Row `<CHECK_ID>  PASS|FAIL|WARN  <component-path>  <detail>`; exits `0` clean / `1` any FAIL /
  `3` parse error or prefix not found. CHECK1 inventory INFO · CHECK2 flag-drift **WARN, FAIL only under
  `--strict`** (lead decision, overriding the shard's FAIL: an un-hidden action is a station-config choice,
  e.g. the PANCCADIA operator workaround — K3, a trace over a hard stop) · CHECK3 out-of-facet WARN ·
  CHECK4 transient-persisted WARN · CHECK5 schema-drift bog-extra FAIL · CHECK6 src-missing WARN · CHECK7
  link-dangling FAIL · CHECK8 hoa-leftover WARN · CHECK9 orphan-handle FAIL · CHECK10 duplicate-handle
  FAIL. CHECK2-7 need `--source-dir` (annotations); without it they emit SKIP rows while CHECK1/8/9/10 still
  run from the bog alone. Parser: **python3 stdlib wrapped by bash** — the single deliberate exception to
  the awk-only rule, justified by a 68k-line bog parsed in ~0.1 s and by handle-graph resolution that awk
  would encode as a second hash table; `command -v python3 || exit 3` guards it. Grammar per the shard:
  `m='PREFIX=MODULE'` registry, `<p n h t f v>`, `<a n f>` present only when flags differ from the class
  default, links live **inside the target component** (`sourceOrd h:<handle>`, `sourceSlotName`,
  `targetSlotName`), flag letters `h/o/r/s/t` with `L` = link-locked (ignored), depth by one-element-per-line
  counting.
- **`wb` checks (P)** — WB-LEX1 / WB-DEP1 extend the tools that already own lexicons and dependencies
  (`slot-coverage` over every `-wb` include+lexicon; `verify --src` diffing built `module.xml`
  `<dependency>` against `api()`/`nre()`); WB-SCAFFOLD1 is a `verify-module.sh` check closing the
  `:245-257` dead angle (0 classes AND 0 palette entries → WARN, `--strict` FAIL); WB-THREAD1 + WB-AGENT1
  become one new `lint-wb-threading.sh` (`doInvoke` bodies calling `getNavChildren|getNavNodes|BQL`
  without `invokeLater|BJobService` → FAIL; `@AgentOn(types="baja:Component")` with no justification
  comment → WARN) because Swing thread affinity shares nothing with the other two.
- **ux servlet lint (Q)** — an opt-in `--servlet` mode of `rc-scan.sh` over `BWebServlet` subclasses (the
  default scan stays `rc/`-only, D7): an API branch
  with no `getRemoteUser()`/RBAC call → FAIL; a write reachable without a parsed-and-validated value → FAIL;
  CsrfUtil (`x-niagara-csrfToken`/`csrfToken`) required, `X-Requested-With` alone → FAIL; a critical write
  with no fresh short-TTL step-up token bound to session+user+target ORD → FAIL `[ev: corpus B803 §803.6]`;
  missing cache headers → WARN.
- **Post-deploy checklist (R)** — `BUILD-LOOP.md` §6: snapshot → `triage-console` → `bog-audit` →
  `report-module` → keep the snapshot as the deploy baseline. `kit-links.bats` L4/L5 name each script.
- **PR14 (S)** — `BUILD-LOOP.md` §4 gradle task matrix + version-bump checklist. **R14.3 is already
  satisfied at `48fb210`**: `build.sh:84-88` already prints the `mirror-niagara-home.sh` recipe to stderr
  before `exit 31`. PR14 therefore carries no code change; it adds a `tests/build-sh.bats` pin asserting
  the recipe text on exit 31 so the behaviour cannot regress silently.
- **PR15 (T)** — **unblocked: B805 landed** (`ccd8ab95d`); B804 was already available. Doc-only, after PR13.
  `types/logic.md` takes the §805.10 deltas: PID anti-windup as an `errorSum` clamp to the output range,
  NaN → fault + hold + bumpless transfer (`BLoopPoint`), deadband latch-on-cross (`BTstat`), the
  `execute()`/`changed()` split, and the §805.9 flowchart template (STATES / INPUTS / TIMERS / CONTROL /
  PROTECTIONS / OUTPUTS / FEEDBACK); the §805.5 one-bit alarm trace (limit algorithm → `BStatus`
  alarm/fault/unacked → `BAlarmRecord` → `BAlarmService.routeAlarm` → console → ack) backs the R15.2
  health/feedback checklist (a silent FAIL with no status update is forbidden). Two boundaries get their own
  lines: `BLatch` is an **edge D-latch only** (no SR/interlock latch ships in kitControl — a protection latch
  is author-built) and `[CERT-negative]` Tridium has no ODE/matrix/state-space facility; `BTest` is the
  in-process off-station harness. R15.3 (`BHistoryExt`, Interval vs COV, `BHistoryConfig` capacity +
  `fullPolicy`, one ext per logged slot, B804-G1 OPEN) lands in `types/logic-authoring.md`.
  **Campaign-9 candidate**: a tested pure protection-latch seam as a `scaffold-module.sh` template.

### D11 — fixtures: real shapes, sanitised names, oracle files, no binaries

Every new rule is proven against a fixture copied from an operator module with names sanitised
(`BDefrostController` → `Overdue`, `BChiDashboardService` → `BThreadSvc`, `BEvaporatorUnit` → `BUnitPre`),
never an invented snippet (CD4). Committed fixtures are text only — no binary test fixture (CONTRIBUTING
§2); the `bog-audit` fixture is a hand-written **60-line synthetic XML bog** (orphan property → CHECK5,
`differentialUp=-1.5` → CHECK3, dangling `sourceSlot` → CHECK7, `intervalExpired f='o'` → CHECK2) plus
`clean.bog`, never a copy of a customer `config.bog`. Enumerable domains (B806 table, bog check IDs) are
quoted heredocs byte-diffed against an oracle file as a named mutation. Real-tree smokes are `skip`-gated on
path presence, local run pasted into the PR body (campaign-7 D9: a SKIP is never a PASS; the zero-SKIP CI
guard is not extended here).

Fixture files the landed REDs add: `tests/fixtures/lint-delays/{Overdue,Floored,LiteralZero,LiteralPos,
Periodic,SlotGetterMinZero,SlotGetterMinPos}.java` (7) and `tests/fixtures/triage-console/{console,
console-es,console-load-fail,console-load-fatal-only,console-load-fail-es}.txt` (5). `qa/c8-lint-timers-ext`
adds **none** — its seven cases write heredocs into per-test `$BATS_TEST_TMPDIR` dirs, which is what keeps
TL1-TL4's shared `$SRC` untouched.

**Real-tree access.** The pre-fix ColdRoomPan tree is the live working copy under
`Cliente/Leon-Guanjuato/…/ColdRoomPan-rt/src` (verified pre-fix at `:555`), so LD5 bites today. The
**fixed** tree has no worktree on this machine: PR1 creates one from client `main` `c66e412` at
`~/modulos_niagara_n4/Leon-Guanjuato-worktrees/fix-defrost/` (never `/tmp`; lead-authorised) and adds pin
**LD10** — fixed tree → exit 0, no FAIL rows, `skip`-gated on `$C8_CRP_FIXED`.

### D12 — chain mechanics

Two waves, stacked to `main`, ff-only. PR1 targets `main`; each later PR branches only after its predecessor
merges and rebases until the child diff shows only its own work unit (CD8). Wave 2 (PR9-PR15) appends after
PR8, never interleaves. **One ledger attempt per PR** (CONTRIBUTING §9 — the ledger is serial), with
`--max-changed-lines` sized to real content: the merged RED and its fixtures inflate the diff, so the budget
column below is the declared size, not the authored estimate; `--evidence-goal` stays one terse sentence and
the untracked-inventory sha is read from the error output, never cached. Because routing now ships with each
script (D8), `scripts/install-skill.sh --force` runs after every merged PR that touches `skill/SKILL.md` —
**PR1, PR2, PR6, PR7, PR9, PR10, PR11, PR12** — and each of those records the launcher's before/after text
in its PR body, since `skill/SKILL.md` is outside git. `VERSION`/`CHANGELOG.md` → `0.19.0` in
**PR8, the last wave-1 slice** (earlier PRs append under `## [Unreleased]`; PR8 renames it and adds
`### References`), so a delayed wave-2 slice never holds the release. Every kit-changing push range carries
its retro + `retros/INDEX.md` row + `BUILD-STATE.md` self-envelope (CD1); pending = 0 at close.

## CLI Contracts

| Script | Usage | Exits |
|---|---|---|
| `lint-delays.sh` | `<src-dir>` | `0` no FAIL · `1` any FAIL · `3` usage/env |
| `triage-console.sh` | `[--package <pkg>] [--console-dir <dir>] <console.txt>…` | `0` no rows · `1` any row · `3` usage/env |
| `rc-scan.sh` | `<artifact-dir> [--strict] [--servlet]` — scans `rc/` assets only | `0` · `1` · `3` |
| `lint-timers.sh` / `verify-module.sh --src` (+`facets-req`, `ord-literal`) / `slot-coverage.sh` | unchanged usage + the new checks/subcommand above | `0` · `1` · `2` usage · `3` env (legacy map, kept) |
| `report-module.sh` | `<module-root> [--target-version x.y] [--console-dir <dir>]` | `0` · `1` · `3` |
| `station-snapshot.sh` | `<station-dir> <out-dir>` | `0` · `1` · `3` |
| `bog-audit.sh` | `<config.bog> --module <MOD>… [--source-dir <DIR>] [--strict]` | `0` · `1` · `3` |
| `lint-wb-threading.sh` | `<wb-src-dir>` | `0` · `1` · `3` |

## PR Matrix

| PR | Branch | RED (tip 2026-09-05) | Files | Est. authored | Ledger budget | Real smoke | Named mutation |
|---|---|---|---|---|---|---|---|
| 1 | `feat/c8-lint-delays` | `qa/c8-lint-delays` `c96b2ad` | `toolbelt/lint-delays.sh`, `tests/lint-delays.bats`, `tests/fixtures/lint-delays/**`, `ci.yml`, **`BUILD-LOOP.md` + `skill/SKILL.md` routing (K19)**, `BUILD-STATE.md` | ~215 | 470 | CRP pre-fix FAIL `{556,566,620,664}` + fixed-tree worktree PASS (LD10) | accept `Math.max(x,0L)` → LD1/LD3 stop failing |
| 2 | `feat/c8-triage-console` | `qa/c8-triage-console` `6492b2d` | `toolbelt/triage-console.sh`, `tests/triage-console.bats`, fixtures, `ci.yml`, **routing (K19)** | ~195 | 440 | PANCCADIA `console_backup_260903_1704.txt`, REFLOW, MX60 | drop C1 own-frame filter → jetty NPE row appears (TR1); drop C3 → TR8 exits 0 |
| 3 | `feat/c8-lint-timers-ext` | `qa/c8-lint-timers-ext` `ce6ee5c` | `toolbelt/lint-timers.sh`, `tests/lint-timers.bats` | ~160 | 350 | CompPan `:1799-1805`, chihuahua `:305`, CRP `BEvaporatorUnit` pre-fix | accept any `stopped()` cancel (TC-A); whitelist `ScheduledExecutorService` (TC-B); drop 1-level following (TC-C) |
| 4 | `feat/c8-facets-lint` | needs RED (`facets-req` + `ord-literal`) | `toolbelt/verify-module.sh` (2 sub-checks), `tests/verify-module.bats`, `tests/fixtures/facets/**` + a sanitised `DashboardReader`-shaped ORD fixture | ~150 | 320 | CompPan 12 OPERATOR doubles; DashboardPan `DashboardReader.java:75` `SERVICE_ORD` | strip one facet annotation → FAIL; enum `MIN=0` must stay clean; drop the `OrdConstants`+comment exemption → the sanctioned holder starts WARNing |
| 5 | `feat/c8-slot-per-slot` | needs RED | `toolbelt/slot-coverage.sh`, `tests/slot-coverage.bats` (SC6-parse amended) | ~160 | 340 | CRP 19 missing per-slot keys; chihuahua empty lexicon | drop one slot key → FAIL; empty lexicon back to exit 0 |
| 6 | `feat/c8-rc-scan` | needs RED | `toolbelt/rc-scan.sh` (`rc/` assets only), `tests/rc-scan.bats`, fixtures, `ci.yml`, **routing (K19)** | ~215 | 440 | DashboardPan `index.html:701` (host) + `:1298` (bare catch) + `:863` (null branch) | (a) remove the ORD rule → ORDFAIL exits 0; (b) restore the bare `.catch(() => {})` in the fixture → the WARN row flips |
| 7 | `docs/c8-doctrine` | none (CD2) | `types/{logic,logic-authoring,dashboard}.md`, `BUILD-LOOP.md`, `METHODOLOGY.md`, `skill/SKILL.md` (doctrine + working profile only — routing already shipped with PR1/2/6) | ~85 | 300 | — | none; guard = `kit-links.bats` + `sweep-fold-audit.sh --strict` |
| 8 | `feat/c8-report-integration` | needs RED | `toolbelt/report-module.sh`, `tests/report-module.bats`, `VERSION`, `CHANGELOG.md` | ~80 | 220 | CRP pre-fix tree → aggregate ISSUES | drop lint-delays aggregation → report exits 0 on a delay-FAIL module |
| 9 | `feat/c8-station-snapshot` | needs RED | `toolbelt/station-snapshot.sh`, `tests/station-snapshot.bats`, **routing (K19)** | ~165 | 340 | PANCCADIA station dir copy | make the snapshot mutate the source → read-only pin FAILs |
| 10 | `feat/c8-bog-audit` | needs RED (**now bound**, shard landed) | `toolbelt/bog-audit.sh` (+ python3 helper), `tests/bog-audit.bats`, `tests/fixtures/bog/{synthetic,clean}.bog`, **routing (K19)** | ~265 | 620 `size:exception` | PANCCADIA bog → exactly one CHECK2 WARN (`DefrostController.intervalExpired f='o'`), 0 CHECK5/7/9/10; MX60 clean | skip handle resolution → CHECK7 dangling fixture stops FAILing; per-check mutation for CHECK2/3/5 |
| 11 | `feat/c8-wb-audit` | needs RED | `toolbelt/{verify-module,slot-coverage}.sh`, `toolbelt/lint-wb-threading.sh`, tests, `types/wb-widgets.md`, **routing (K19, new script)** | ~215 | 440 | chihuahua-wb (no lexicon), DashboardPan-wb (empty scaffold), `BBatchLinkEditor.java:684` | empty `wb` scaffold stops being detected; `doInvoke` DFS stops FAILing |
| 12 | `feat/c8-ux-servlet` | needs RED | `toolbelt/rc-scan.sh --servlet`, tests, fixtures, **routing (K19, new mode)** | ~195 | 420 | `BDashboardServlet.handleSetpointWrite:239-274`, `BChiServlet:613` | remove the auth check from one API branch → must FAIL; `X-Requested-With`-only must FAIL |
| 13 | `docs/c8-post-deploy` | none (CD2) | `BUILD-LOOP.md` §6, `tests/kit-links.bats` | ~40 | 160 | — | none; `kit-links.bats` names every step's script |
| 14 | `docs/c8-build-pipeline` | none (CD2) | `BUILD-LOOP.md` §4, `tests/build-sh.bats` (exit-31 recipe pin) | ~60 | 200 | exit-31 recipe already live at `build.sh:84-88` | delete the recipe lines → the new pin FAILs |
| 15 | `docs/c8-rt-doctrine` | none (CD2) | `types/logic.md`, `types/logic-authoring.md` | ~90 | 280 | — | none; guard = `sweep-fold-audit.sh --strict` credits `[ev: corpus B805]`/`[ev: corpus B804]` |

## CI Changes (`.github/workflows/ci.yml`)

Appended after the existing `lint-timers` step, each mirroring its shape (never `|| true`), each expected to
exit 0: **PR1** a per-fixture loop over `tests/fixtures/lint-delays` asserting the mapped exit per file;
**PR2** `triage-console.sh --package com.angeles tests/fixtures/triage-console/console.txt` asserting exit 1
plus the grouped row; **PR6** `rc-scan.sh tests/fixtures/rc-scan/clean`; **PR10** `bog-audit.sh
tests/fixtures/bog/clean.bog --module Demo`. `shellcheck` and `bats tests/*.bats` already glob new files.

## Threat Matrix

| Boundary | Applicability | Design response | RED |
|---|---|---|---|
| Untrusted content decode | **Applicable** — console files (invalid UTF-8), `config.bog` XML from a customer station | `LC_ALL=C` byte reading, no decoding, no `eval`; the bog parser is python3 stdlib `xml`-free line scanning with no entity/DTD expansion and no network; malformed input → row/exit 3, never a crash | TR6, TR9, a malformed-bog case in PR10 |
| Subprocess / external tool | **Applicable** — `python3` (PR10), `cp`/`sha256sum` (PR9) | `command -v` guard → exit 3; fixed argv, no shell interpolation of file content | PR9/PR10 env-missing pins |
| Filesystem write / executable-file classification | **Applicable** — `station-snapshot.sh` writes a capture set from station files | Writes only under `<out-dir>`; source opened read-only; refuses when `<out-dir>` is inside the station dir; copies never set `+x` and the manifest records mode | read-only-source pin (`chmod a-w`) + manifest pin |
| Git repository selection / commit / push / PR commands | **N/A** — every toolbelt script is VCS-free; `kit-links.bats` L2 fails the suite if any names `git`; chain mechanics are orchestrator-owned | — | L2 (existing) |

## Migration / Rollout

Additive throughout. Rollback is `git revert` per slice: PR1/2/6/9/10 are new paths nothing references until
PR8/PR13 (their routing lines revert with them); PR3/4/5/8/11/12 lose only their new rules/rows; doc PRs
revert cleanly (retro files are never deleted and INDEX rows return to `pending`). Two behaviour changes
need a CHANGELOG line each: the empty-lexicon WARN→FAIL escalation (D6a) and the new `--console-dir`.
No station, deployed jar or operator file is touched.

## Fresh-context validator checklist

A second agent validates this design against `spec.md` and the three landed REDs. Run, in order:

1. `git show qa/c8-lint-delays:tests/lint-delays.bats` — confirm LD9 pins exit **3** (not 2) and that D1's
   exit map matches; confirm LD8 requires WARN **with** exit 0.
2. Read `…/ColdRoomPan-rt/src/com/angeles/ColdRoomPan/BDefrostController.java` — confirm D2a: `Math.max`
   at `:555`, getters at `:566/:620/:664` with facet `MIN = makeSeconds(0)` at `:170/:216/:285`, and
   `POLL = BRelTime.make(5000)` at `:731` scheduled at `:622/:641`. If `:622` appears in the FAIL set, the
   constant-resolution rule is missing.
3. `git show qa/c8-lint-timers-ext:tests/lint-timers.bats` — confirm three things D4 depends on: TC-A's
   `BStaggerHold` clears the flag in the expiry path (a bare `grep 'startingUp = false'` must NOT pass);
   the `started()` companion must PASS; TC-C's `changed()` already contains `isRunning()` yet must FAIL.
4. `grep -n 'SC6-parse' tests/slot-coverage.bats` (must currently expect exit 0, so D6a is a real amendment);
   `sed -n '84,90p' toolbelt/build.sh` (R14.3 already satisfied); `sed -n '86,100p' toolbelt/report-module.sh`
   (the 4-column awk parser consumes D1's row shape, so `<subject>` must be whitespace-free).
5. Confirm every PR landing a script also edits `BUILD-LOOP.md` **and** `skill/SKILL.md` (K19/CD5); a script
   whose routing is deferred to PR7 leaves `kit-links.bats` L4/L5 RED on the merged tree.
6. Cross-check every spec ID (R1.1-R15.4, SC1-SC11) against a D-number; a requirement with no design home is
   a gap, not an omission to fill silently. Confirm no line proposes editing a client repo or a station.

## Open Questions — all RESOLVED

- [x] **D2a**: spec corrected — LD-FAIL set is `{556, 566, 620, 664}`; `:622/:641` (the `POLL` constant)
      PASS, and constant resolution is a design requirement, not an optimisation.
- [x] **D6a**: escalate in parse mode; PR5 amends the SC6-parse pin to exit 1 and records the behaviour
      change in `CHANGELOG.md`.
- [x] **D7a**: `rc-scan` scans `rc/` assets **only**; the Java ORD literal moves to a `verify --src`
      `ord-literal` WARN sub-check in PR4 (D5).
- [x] **D9a**: `schema-risk` exit 2 maps to a FAIL row, never to the `ERROR`/exit-3 path.
- [x] **PR10 python3**: accepted, `command -v python3 || exit 3`-guarded and declared in the header.
- [x] **LD10**: lead-authorised — the fixed-tree worktree is taken from client `main` `c66e412` at
      `~/modulos_niagara_n4/Leon-Guanjuato-worktrees/fix-defrost/`, so SC1's PASS half is provable.

**Validation**: fresh-context validator FAIL → 6 fixes applied → re-validated by lead.
