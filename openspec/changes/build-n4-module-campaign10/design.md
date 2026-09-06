# Design: build-n4-module-campaign10

**Phase**: design · **Source**: kit `v0.20.0` (main `20c2360`, C9 archive `df8c7ec`) · **Target**: kit `v0.21.0`
**Inputs**: `proposal.md` (R1-R7, §4 PR chain, §7 RK1-RK11, §10 SC-1..SC-12) · `explore.md` (§1.1 constraints 1-5) ·
campaign-9 `design.md` (shape) · research **B831** `686b54ac5` · apply-package `2f710626d` (S21/S22/S23) ·
investigador1 S22 reuse-vs-new note `685e7981e` (relayed) · coordinator S25 amendments (2026-09-07, two messages).
**Topic key**: `sdd/build-n4-module-campaign10/design`

**Read-tip discipline (K13/K21/C9 lesson 1).** Client cites are read from the worktree
**`/home/cristian/modulos_niagara_n4/Cliente/Leon-Guanjuato-worktrees/main-ff1b659`** (client `ff1b659`). The working
checkout `Cliente/Leon-Guanjuato` is **never** read by this design and must never be a smoke target. Kit cites are read
from the `niagara-tools` working tree; line anchors below were **read**, the tree SHA (`20c2360`) is relayed, not
verified — this executor has **no shell tool**. `[ev: retro campaign9-close lesson 1]`

**A trap this design found and names.** `niagara-tools-worktrees/c10-lints/` is **QA's live worktree** (it moves
between QA's `qa/c10-*` branches; at design time it carried the C9 pins for three of the four bats, at coordinator
read time it sits on `qa/c10-write-path-strict` `a56a72e`). It is never a contract: a worker who reads any worktree
other than its own as "the RED" ships whatever that session happened to have checked out — the C9-lesson-1 defect,
pre-staged on the filesystem. Rule: the RED is the tip on **origin** (`git show origin/qa/c10-<x>:tests/<file>`);
nobody reads or deletes another session's worktree; the C10 worktree map is §Worktree/branch map.
`[ev: git worktree list 2026-09-07 — c10-lints on qa/c10-write-path-strict a56a72e]`

**Unread inputs, marked honestly.** No shell ⇒ no `git show`. Every **QA RED pin text is `[INFER]`**, relayed from the
proposal/explore/coordinator. What IS `[CERT]` here: the four remote-tracking tips read from
`.git/refs/remotes/origin/qa/` match the proposal exactly — `c10-lint-timers-fp` **`52ebd11`**,
`c10-ext-writable-per-slot` **`954ebd7`**, `c10-silent-protection-surfaces` **`f981754`**, `c10-structural-cwd`
**`a792d7a`**. **Where this design and a landed RED disagree, the executable RED wins.** `[ev: refs read 2026-09-07]`

---

## Technical Approach

Six independent slices, one PR each, RED-first, plus a close PR. Five of the six are **precision narrowings of an
existing awk heuristic**; none adds a script, a slot or a CLI verb (S25 adds one flag). The unifying design move is the
one B831 §831.4 names: **stop proxying at the wrong granularity**. Three of the four kit lints today decide at file or
class scope what is really a per-method, per-slot or per-class-pair fact. Rather than invent three new parsers, this
design **promotes ONE already-proven parser — `lint-silent-protection.sh`'s section-D method-boundary parser
(`:250-320`) — to the shared primitive**, adds one guard it is missing, and reuses the Pass-0 paren-balanced
`@Niagara*` join (`:60-95`, mirrored at `lint-ext-writable-shape.sh:78-102`) for annotation harvesting. Scripts stay
`set -u`, `printf` rows, VCS-free, dot-dir pruned (D9b), `shellcheck 0.10.0` clean, exit codes disjoint (K20).
`[ev: corpus B831 §831.4]`

---

## Architecture Decisions

### D1 — S21 `lint-timers.sh`: replace the forward brace-walk with a real method parser + a class-scope FIELD pass

**D1a — the root cause is bigger than "no field check", and it decides the fix.** Pass 1 (`:143-186`) picks a
*candidate signature* with `match(line, /[a-zA-Z_][a-zA-Z0-9_]*[[:space:]]*\(/)` and then brace-walks **forward**. On
`BDefrostController.java` the candidate that fires is not a method at all: `@NiagaraProperty(` matches the regex with
`cand = "NiagaraProperty"`, which is not in the keyword list (`:138`); the forward walk then finds no `;` and no `{`
until the **class body's** opening brace, so `body` becomes the WHOLE CLASS. That body contains `Clock.schedule` at
`:808`/`:810`/`:850` **and** the first `X = true;` in the class, which is `anyNoHardware = true;` at `:726` — a
method-LOCAL declared at `:718` inside `requestDefrostCycle()` (`:713`), a method with no `Clock.schedule` and whose
only `Clock` call is `Clock.time()` at `:715`. Both guards the apply-package asks for are therefore necessary, and
neither is achievable by patching the forward walk. `[ev: client BDefrostController.java:713,:715,:718,:726 @ ff1b659]`
`[ev: kit lint-timers.sh:138,:143-186,:212]` `[ev: apply-package S21 2f710626d]` `[ev: B831 §831.1]`

**D1b — Chosen: port section D (`lint-silent-protection.sh:250-320`) into `lint-timers.sh` and add a depth guard.**
Section D detects a method only when the line's **net** brace change is `> 0` (`:273`), resolves the name by Case A
(single-line signature) or Case B (`{` alone → backward scan, stopping at `@`, `;$` or `{`, excluding
`if|for|while|switch|catch|try|else|do|new|class|interface|enum` at `:297`), and records `meth_start/meth_end/meth_name`
(`:313-316`). It rejects `NiagaraProperty(` outright because that line's net brace change is 0.

**The one guard section D is missing — and it must be added, not inherited.** Case B's backward scan can still name a
**class body** as a method: at `BCompressorControl.java:448` the `{` is alone, the scan walks up through `:447`
`implements BIAlarmSource`, `:446`, `:445` (`public class …` — no parens, no match), `:444` `)`, and stops at `:442`
`defaultValue = "new BAlarmRecord()"`, yielding `mname = "BAlarmRecord"`. **Chosen guard: accept a method open only when
the post-brace `brace_depth >= 2`.** A top-level class body opens at depth 1; any real method opens at depth ≥ 2. This
is one comparison, it kills the class-body-as-method case by construction, and it is the same counter that answers the
field question. **Rejected**: extending Case B's exclusion regex — it is a token blacklist chasing an unbounded set
(`BAlarmRecord` today, the next annotation default tomorrow). **Rejected**: keeping Pass 1 and adding a field check
only — the schedule would still be pulled from another method.
`[ev: client BCompressorControl.java:442-448 @ ff1b659]` `[ev: kit lint-silent-protection.sh:250-320]`

**D1c — the FIELD pass is the SAME brace walk, not a second one.** A declaration matching
`^[[:space:]]*(private|protected|public|static|final|volatile|transient|[[:space:]])*\b(boolean|int|long)\b[[:space:]]+<name>[[:space:]]*[;=]`
seen while `brace_depth == 1` (inside the class body, outside every method) is a **FIELD**; the same shape at depth ≥ 2
is a **LOCAL**. Deriving both facts from one counter in one pass is a design requirement, not tidiness: two independent
walks can disagree about where a method starts, and a disagreement here is exactly the class of bug S21 is fixing.
`anyNoHardware` is emitted by this pass as a LOCAL and never enters the candidate set.

**D1d — "same enclosing method" binding.** For each method `m` (from D1b) and each assignment `X = true;` at line `i`
with `meth_start[m] <= i <= meth_end[m]` and `X ∈ fields`: FAIL only when **some** `Clock.schedule` also lies in
`[meth_start[m], meth_end[m]]`. The binding is by **line-range containment**, never by proximity — the header at
`:127-130` already records why (`startingUp` `:1760` and `powerOnTicket = Clock.schedule` `:1764` are 4 lines apart, so
a ±3 window was wrong). Pass 2 (`:190-208`, the `stopped()`/`started()` clear check) is **untouched**, and the FAIL row
at `:212` is **byte-identical** — S21 changes which flags reach it, never its text or the exit map (0 / 1 any FAIL /
3 usage). `[ev: kit lint-timers.sh:127-130,:190-208,:212]`

**D1e — what happens to the existing POS fixture.** The C8 companion-flag fixtures are a **regression asset, not
collateral**: they encode the real `startingUp`/`powerOnTicket` shape (a class FIELD, same method as the schedule, never
cleared). Under D1b-D1d that shape still parses as one method at depth 2, `startingUp` still resolves as a field, and
the schedule is still in range ⇒ **the existing POS fixtures keep FAILing, unedited**. Any C8 fixture that only FAILs
because of the forward-walk overrun would be a *fixture-green/real-red* artefact and must be re-cut inside
`qa/c10-lint-timers-fp`, never silently deleted by the apply worker (K13). The OBSERVED flip is recorded on **S21-pos**
(must stay FAIL) as well as S21-neg — RK5's whole point. `[ev: proposal RK5]`

### D2 — S22 `lint-ext-writable-shape.sh`: a NEW `@NiagaraAction`-aware pass, not a reuse of silent-protection

**D2a — where it lives, and why "reuse" is the wrong word.** `lint-silent-protection.sh` never reads `@NiagaraAction`;
its Pass-1 slot→writer follow (`:124-165`) harvests the *argument* of a `setX(`/`getX().setValue(` call to build
`SURF_WRITE_FIELDS`, which is a different question. **Chosen: a NEW action-aware pass built in-script inside
`lint-ext-writable-shape.sh`**, borrowing two *idioms* — the section-D method-boundary parser (`lint-silent-protection.sh:250-320`,
with the D1b depth guard) and the Pass-0 paren-balanced annotation join already present at
`lint-ext-writable-shape.sh:78-102` — and nothing else. **Rejected**: sourcing or shelling out to
`lint-silent-protection.sh` (K20 exit codes would collide and the two lints would stop being independently runnable).
`[ev: investigador1 685e7981e (relayed)]` `[ev: kit lint-silent-protection.sh:124-165]`

**D2b — the four steps.**

| # | Step | Mechanism |
|---|---|---|
| 1 | **Harvest** action names | Re-point the existing `:78-102` join: instead of setting `has_action = 1` and breaking at the first annotation (`:89`/`:91`/`:101`), keep looping and extract `name[[:space:]]*=[[:space:]]*"([^"]*)"` from **every** `@NiagaraAction` buffer. Handles the single-line form (`:435` `@NiagaraAction(name = "tick", flags = Flags.HIDDEN)`) and the multi-line form (`:439-444` `ackAlarm`) with the same paren counter |
| 2 | **Resolve** action → handler | NRE convention: action `x` ⇒ method `"do" toupper(substr(x,1,1)) substr(x,2)`. `tick→doTick`, `powerOnExpired→doPowerOnExpired`, `ackAlarm→doAckAlarm`. **This closes B831-G1**: the annotation names `ackAlarm`, the body lives in `doAckAlarm`, and the mapping is the only thing that binds them |
| 3 | **Scope filter** | Only method bodies whose `meth_name` is in the resolved `do<Action>` set are scanned. `execute()`, `changed()` and every slotomatic-**generated setter** fall out **by construction**, because none is named `do<Action>` |
| 4 | **Write detection** | Inside a scoped body, slot `X` is written when the body contains `setX(`, `getX().setValue(`, or `.set(<x>,` (the property-constant form). `X` exempt ⟺ some scoped body writes `X` |

**Step 3 is load-bearing, not an optimisation.** Slotomatic emits `public void setFaultReset(BStatusBoolean v) { set(faultReset, v, null); }`
into the generated region of every adapter. A rule that scanned *any* method would find that generated setter and
exempt **every** complex OPERATOR slot in the kit — a strictly worse false negative than the one S22 is fixing. The
second trap is real code: `setFaultReset(new BStatusBoolean(false))` at **`BCompressorControl.java:2025`**, inside the
`execute()`/`changed()` one-shot auto-clear. B831 §831.2 flags it explicitly; the scope filter is what makes
`faultReset` correctly WARN rather than wrongly exempt.
`[ev: client BCompressorControl.java:2025 @ ff1b659]` `[ev: B831 §831.2, §B831-G1]`

**D2c — per-slot exemption + the EW10 contract change.** `_check_prop` (`:146-186`) is unchanged except its last
predicate: `if (has_action) return;` (`:180`) becomes `if (pname in exempt_slot) return;`. The WARN row text
(`:182-185`) and the exit map (0 / 1 under `--strict` / 3) are **byte-identical**. **CONTRACT CHANGE — EW10 CompPan-rt
0 → 1.**

**D2d — the exact-count real-tree pin, by NAME.** Verified by reading the ff1b659 worktree: the only `BStatus*` +
`Flags.OPERATOR` property in CompPan-rt is **`faultReset`**, whose `@NiagaraProperty(` opens at
**`BCompressorControl.java:381`** (`name` `:382`, `type = "BStatusBoolean"` `:383`, `flags = Flags.SUMMARY |
Flags.OPERATOR` `:385`). Its sibling at `:375-377` is `BStatusBoolean` but `TRANSIENT|SUMMARY|READONLY` (out of scope);
`rotationInterval`/`powerOnDelay` are `BRelTime` and `condenser1..3Mode` are `double` (plain, out of scope). **The pin
is `exactly 1 WARN, subject `faultReset` asserted by NAME**, plus an absence pin that no other CompPan-rt slot flips.
**Never pin the line**: the lint cites the annotation OPEN (`:381`), while `:1612` is the slotomatic-generated
`Property faultReset` the lint never emits — a line pin invites the wrong one. The three actions live at
`:435`/`:437`/`:439-444` and the class opens at `:445`. `[ev: client BCompressorControl.java:375-385,:435-445 @ ff1b659]`

**D2e — B831-G1 guard pin.** A `do`-body that writes a **different** slot must not exempt `faultReset`
(`qa/c10-ext-writable-per-slot` `954ebd7` carries `EW-s22-neg2`, the `doAckAlarm` shape). `[INFER — RED unread]`

### D3 — S23 `lint-silent-protection.sh`: a class-keyed alarm-surface index + the adapter→pure follow

**D3a — the boundary is architectural, not a token gap.** The main awk runs **per file** (`:475-481`), and its surface
criterion (1) is a **file-level** flag: section B (`:219-225`) sets `file_has_alarm` when the file contains
`BAlarmSourceExt` or `BAlarmRecord`, and `:424-425` consumes it. The CP-1 trip is at
**`CompressorControl.java:294`** (`if (suctionValid && c.suctionLowLimit > 0d && suction < c.suctionLowLimit) target =
Math.min(target, onCount - 1)` — read at ff1b659) in the **pure** class; the C9 PR9 surface is in the **adapter**
(`implements BIAlarmSource` `:447`, `new AlarmSupport(` `:1882`, `newOffnormalAlarm(` `:2093`). No token added to a
file-scoped flag can cross that boundary. `[ev: client CompressorControl.java:294, BCompressorControl.java:447 @ ff1b659]`
`[ev: kit lint-silent-protection.sh:219-225,:424-425,:475-481]` `[ev: B831 §831.3]`

**D3b — Chosen: a dir-wide Pass 0b producing `ALARM_CLASSES`, keyed by class name.** It runs beside the existing Pass 0
(`:97-99`) over the same `find … -prune … *.java` list, and emits the `class <Name>` of every file that carries a
surface:

| Pattern | Token set (ALL must hold) | Anchor at ff1b659 |
|---|---|---|
| **A** (declarative, existing) | `BAlarmSourceExt` **or** `BAlarmRecord` | `BEvaporatorUnit.java:193` child `BBooleanPoint freezeAlarmPt` + ext |
| **B** (programmatic, NEW) | `implements` … `BIAlarmSource` **AND** (`newOffnormalAlarm` **OR** `new AlarmSupport(`) | `BCompressorControl.java:447`, `:2093`, `:1882` |

Pattern B's **AND** is deliberate: an `import javax.baja.alarm.BIAlarmSource` or a javadoc mention must not exempt a
trip. Comments are stripped before the token match (D6).

**D3c — the follow, and its stated limit.** In the main awk, `this_class` = the `class <Name>` declared in `FILE`. The
trip is surfaced when `this_class ∈ ALARM_CLASSES` (this **reproduces today's file-level behaviour exactly**, so
Pattern A cannot regress) **or** `"B" this_class ∈ ALARM_CLASSES` (the adapter→pure follow). One level, one direction,
one naming convention (`B<Pure>`), resolved **inside the scanned tree only** — consistent with the single
`<java-src-dir>` CLI. **Rejected**: parsing constructor/field references to find the adapter — it needs a type
resolver, and a partial one silently loses exemptions. **Rejected**: a whole-tree "any alarm anywhere" flag — it would
exempt every trip in every module. RK6's convention miss therefore **over-reports** (a WARN, never a FAIL), which is
the safe direction. Criteria (ii-B), (2) and (3) (`:427-457`), the private-field/effect-slot exemptions and the
one-WARN-per-site dedupe (`:483-493`) are **unchanged**. `[ev: proposal RK6]`

**D3d — the Pattern A regression guard is an ABSENCE pin.** ColdRoomPan-rt must still be **0** after S23 (CR-3 surfaced
via Pattern A since C9 PR8). A "0 → 0" pin is only evidence when the mutation table shows the pin can move: the named
mutation is *drop Pattern A from Pass 0b* ⇒ ColdRoomPan-rt returns to 1.

### D4 — S24 `run-pure-test.sh`: the fix is runner-side, ONE edit (D4b dropped — tasks read 896846176)

**D4a — Chosen: `cd "$rt"` in a subshell around the `java` call at `:62`.**
`( cd "$rt" && java -cp "$tmp:$JU:$HC" org.junit.runner.JUnitCore "$testfqcn" )`. Safe because every classpath element
is already absolute — `$tmp` from `mktemp -d` (`:53`), `$JU`/`$HC` from `find "$HOME/.gradle"` (`:37-38`). The subshell
keeps the parent's cwd and its `trap 'rm -rf "$tmp"' EXIT` (`:54`); under `set -euo pipefail` (`:22`) the subshell is
the last command, so JUnit's non-zero status still propagates as the exit-1 **bite** (`:19`).

**D4b — DROPPED (investigador1 tasks read 896846176, after validator 47453742b).** The absolutise edit
`rt=$(cd "$rt" && pwd)` after `:30` was first justified by a `-sourcepath` break that does not exist (`javac` at
`:58-60` runs OUTSIDE the D4a subshell; a relative `$rt` from `/tmp` reproduces `OK (37 tests)` today), then kept as
"defensive". Under D4a's structure it is provably inert: the only `cd` is the subshell's own, which starts at the
caller's cwd where a relative `$rt` already resolves. The RED `a792d7a` invokes the runner with an ABSOLUTE `$RT`
in both S24 tests, so no pin can flip on that edit — a mutation "revert the absolutise" cannot produce a RED, and a
change that no RED can bite is would-flip prose (SC-7). PR4 is therefore ONE edit (D4a), one OBSERVED mutation
(revert the subshell `cd` → S24-cwd FAIL). `[ev: 896846176 §PR4 finding; a792d7a tests/run-pure-test.bats]`

**D4c — why runner-side, not test-side.** The structural WiringTests read `Paths.get("src/…")` and
`../../build.gradle.kts` **relative to cwd** (companero `4d5e6092c`). Those files live in the **client** repo and on
**QA RED branches**: a kit worker may not edit them (K12 — write only in your own worktree; K13 — workers never edit a
QA RED). Runner-side is also the only fix that is inherited by REDs **not yet written**, which is the whole reason R4
lands before any further C10 structural RED. **Rejected**: a `@Rule`/`@BeforeClass` chdir in each test — N files, all
in trees this campaign must not touch. **Rejected**: passing `-Duser.dir` — the JVM ignores it for relative
`Paths.get`. OBSERVED flip = revert the `cd` ⇒ S24-cwd fails.
`[ev: kit run-pure-test.sh:19,:22,:30,:37-38,:48-60,:62]` `[ev: apply-package S24 4d5e6092c (relayed)]`

### D5 — S25 `lint-write-path.sh`: a PER-ROW STALE pass beside an untouched uncovered FAIL

**D5a — where the two sets meet.** The **matrix side** is `_matrix_slots` (`:161-173`): one awk over `$MATRIX`, first
cell of every `^\|` row, backtick-inner extractor (`:165-170`), filtered to `^[a-z][A-Za-z0-9]*$`, then `sort -u`. The
**source side** is `_AWK_SCANNER` (`:310-343`), run **per profile** (`:346-381`) and printing only `prop_op` names.
Coverage today asks *source ⊆ matrix*; STALE asks the **opposite direction**, and that changes two things:

1. **Scope is the MATRIX ROOT, not the module root and not the profile.** `$MATRIX` is resolved once by the walk-up
   documented at `:17-21` and shared by every profile (`:161`); the **matrix root** is the directory whose
   `docs/write-path-matrix.md` was selected. The covered set is harvested from **every Java source under that root**
   (all modules), `build/` and dot-dirs pruned (D9b), plus `--bog` extras. Anything narrower is wrong in a way that is
   easy to miss: with a per-profile or per-module set, running the lint with `ColdRoomPan` as the module root would
   report every real `CompPan` slot documented in the shared matrix as STALE. Consequence and gate: **the STALE count
   is invariant across module roots** — 5 at ff1b659 from at least two different module roots. The **uncovered FAIL
   direction stays per-module exactly as today** (`:346-381`); only the STALE direction is root-scoped.
2. **The right-hand set is broader than `_op_slots` in TWO directions.** It is
   `ALL @NiagaraProperty names (any flag) ∪ ALL @NiagaraAction names ∪ --bog extras`. A matrix row legitimately
   documents a non-OPERATOR slot, **and it legitimately documents an ACTION**: rows `:64` `intervalExpired` and `:65`
   `forceDefrost` are real `@NiagaraAction`s at `BDefrostController.java:148` and `:152` (read at ff1b659) and must
   stay covered. **Chosen: a second scanner mode `_AWK_SCANNER_ALL`** — the identical paren-balance parser, `prop_op`
   filter dropped (`:328-329`, `:338`), and the annotation trigger widened from `@NiagaraProperty` (`:315`) to
   `@Niagara(Property|Action)` with the same `name[[:space:]]*=[[:space:]]*"…"` extractor. **Rejected**: reusing
   `_op_slots` — every documented non-OPERATOR slot floods as STALE. **Rejected**: properties only — the two action
   rows would flood, and an "actions are exempt by prefix" heuristic is the same comment-satisfiable class as D5d.
   `[ev: client docs/write-path-matrix.md:64-65, BDefrostController.java:148,:152 @ ff1b659]`

**D5b — STALE is PER ROW, and the row grammar must therefore carry the LINE.** One STALE row per **matrix row** whose
backtick-inner name is outside the union and which does not itself carry `[concept]`. A marked row never exempts
another row of the same name: the `[concept]` filter runs **per row, BEFORE** any dedupe, and the emit loops **rows**,
not names. Verified at ff1b659, `docs/write-path-matrix.md` yields **5** stale rows over **3** names —
`hoaMode` at `:31`, `:32` and `:52`, `inhibit` at `:33`, `freezeEnabled` at `:36` (`:40`'s first cell is `setpoint`,
which is covered; `inhibit` is **not** bog-traced). A name-keyed emit would print 3 and mark 5 rows as handled.
**Chosen row grammar**:

```
STALE  lint-write-path  <matrix>:<line>  slot <name>: no source slot with that name
```

`STATUS check subject detail` — the C9 D2a order, because `report-module.sh` parses `st=$1; chk=$2` (`:99`, `:195`,
`:342-353`). Subject is `<file>:<line>` (whitespace-free, the `lint-delays.sh:18` convention) because three rows share
the name `hoaMode`. **This diverges from proposal §2.1 R5's `write-path STALE <slot> …`** on both column order and
subject; the divergence is deliberate and is carried to Open Questions for the spec to reconcile.

**D5c — `report-module.sh` needs no change, and that is by design.** Its per-lint arms match `FAIL*|WARN*` only
(`:302`, `:341`), so a `STALE ` row falls through and never touches the aggregate — the correct property for an
advisory class. Default exit stays 0, so the member row stays PASS.

**D5d — the `[concept]` token is LITERAL, with no implicit exemption.** A row is exempt only when it carries the
literal token `[concept]`; there is no parenthetical heuristic and no "Engine link" source-column rule. Both were
considered and **rejected as comment-satisfiable pins** (C9 lesson 21): a heuristic over free-text columns is
satisfiable by prose a reviewer never intended as a marker. Markdown comments (`<!-- … -->`) are stripped before
matching, symmetrically with D6.

**D5e — the exit contract, with the uncovered FAIL untouched.** `:374` (the FAIL row) and its `FAILED=1` are
**byte-identical**. `:383` `exit "$FAILED"` becomes:

```sh
[ "$FAILED" -eq 1 ] && exit 1          # uncovered — unchanged, with and without --strict
[ "$STRICT" -eq 1 ] && [ "$STALE" -eq 1 ] && exit 1
exit 0
```

Uncovered FAIL dominates; `--strict` promotes STALE only; exit **3** (usage / no src / missing matrix, `:40-43`,
`:150-151`) is preserved ⇒ verdict range `{0,1}` and fault range `{3}` stay disjoint (K20). Weakening the uncovered
FAIL to WARN-only was **REJECTED** by the lead: a worker who forgets `--strict` would ship an uncovered OPERATOR slot.
`[ev: proposal §2.1 R5]` `[ev: kit lint-write-path.sh:25,:161-173,:310-343,:374,:383]`

**D5f — PR5/PR6 ordering, and why PR5's pin survives PR6.** PR5 lands with the **pre-edit** real-tree count pinned at
the frozen client tip: `docs/write-path-matrix.md` @ **ff1b659** ⇒ **5 STALE rows** (exit 0; exit 1 under `--strict`).
PR6 then adds `[concept]` to those five rows and the same lint reports **0** — that flip is PR6's OBSERVED evidence.
PR5's pin is **not** invalidated, because the kit smoke reads the frozen `main-ff1b659` worktree, not client `main`.
Consequence: **PR6 now merges after PR5** (it needs the STALE rule to exist to observe its flip), amending proposal §4's
"PR6 never blocks a kit merge".

### D6 — comment stripping is a shared, upstream primitive (RK4 / C9 lesson 21)

Section D strips only `//` per line (`:262`), and Pass 0's annotation join strips nothing — so a `//`-commented
`name = "faultReset"` inside an annotation buffer, or a `/* */` block containing a brace, changes both the parse and the
match. **Chosen: one strip stage over `lines[]` before every pass in S21/S22/S23 and before the matrix match in S25** —
`//` to end of line, and a `/* … */` state machine spanning lines, blanking rather than deleting so **line numbers are
preserved** (every row's subject is a line number; deleting would shift every anchor). Every S21/S22/S23 fixture carries
a comment-only decoy that must NOT satisfy its pin. `[ev: retro campaign9-close lesson 21]`

### D7 — S26 client `.gitignore` + the keep-set proof

Two lines only: `build/tmp/` (trailing slash — directories, at any depth) and `*.class`. The module jars under
`build/` are tracked by repo convention and stay tracked: `.gitignore` has **no effect on already-tracked paths**, so
the keep-set holds by git semantics. The pin is a guard against a future `git rm --cached`, not against the ignore
itself: **`git ls-files` for every tracked `build/**/*.jar` identical before and after**. A broad `build/` ignore was
**rejected** (RK7 — it would hide a jar the moment anyone re-adds one). No `vendorVersion` change, no jar, no deploy.
PR6 also carries the five `[concept]` matrix edits (D5f).

### D8 — R7 close PR

`tests/c10-close.bats` gated on `C10_CLOSE=1`, from `qa/c10-close-checklist` `41bca42` (skeleton: 12/12 inert skip,
BASE `1fb63d6`, `C10_CLOSE_COMMIT` param). QA freezes the three `TODO(freeze)` pins **at close, against the merged
tip** — VERSION, tag and SC-13. `VERSION` `0.20.0` → **`0.21.0`** + a `## [0.21.0]` CHANGELOG section with a
`### References` block (CONTRIBUTING §4-5); `retros/INDEX.md` pending = **0** (one retro per kit-changing push, six of
them); `BUILD-STATE.md` envelope; `sweep-build-state.sh` + `sweep-fold-audit.sh --strict` green; `kit-links.bats` green
with the `--strict` row added for `lint-write-path.sh` (K19, both `BUILD-LOOP.md` and `skill/SKILL.md`); `shellcheck`
exit 0; **zero attribution trailers in the whole PR range** (K11). Tool-pins = the C9 set — C10 adds **no** tool file.
PR7 opens only after PR1-PR5 merge; if R5 rolls to C11 (RK9) PR7 still ships, depending on PR1-PR4 only.

---

## Data Flow — where each rule now decides

```
  S21  file ─┬─ brace walk (ONE pass, depth counter)
             ├─→ methods[start,end,name]   (net-brace-open, depth>=2 guard)   ── D1b
             └─→ fields{}                  (decl at depth==1)                 ── D1c
                        └── FAIL ⟺ (X∈fields) ∧ (X=true ∈ m) ∧ (Clock.schedule ∈ m)

  S22  file ─┬─ Pass0 paren-join → actions{x}  ─→ do<X>()  ─┐                 ── D2b
             ├─ section D (+depth guard) → methods[]        ├─→ exempt_slot{} ── D2b/3
             └─ Pass2 @NiagaraProperty → complex+OPERATOR ──┴─→ WARN unless exempt

  S23  tree ─→ Pass 0b ─→ ALARM_CLASSES{BEvaporatorUnit, BCompressorControl, …}
       file(CompressorControl) ─→ trip@:294 ─→ surfaced ⟸ "B"+class ∈ ALARM_CLASSES
                                                          (adapter→pure follow) ── D3c

  S25  matrix rows ──(strip <!-- -->; skip [concept])────────────┐
       MATRIX ROOT/**/*.java ─→ @NiagaraProperty ∪ @NiagaraAction ├─→ row ∉ union ⇒ STALE
       --bog extras ──────────────────────────────────────────────┘   (per ROW, carries :line)
       (uncovered FAIL direction stays PER MODULE, unchanged)
```

---

## File Changes

| File | Action | Description |
|---|---|---|
| `build-n4-module-kit/toolbelt/lint-timers.sh` | Modify | D1 — replace Pass 1 (`:143-186`) with section-D parser + depth guard + class-scope FIELD pass; `:190-208` and row `:212` untouched |
| `build-n4-module-kit/toolbelt/lint-ext-writable-shape.sh` | Modify | D2 — new action-aware pass; `:78-102` join re-pointed to harvest names; `:180` becomes a per-slot lookup; row `:182-185` untouched |
| `build-n4-module-kit/toolbelt/lint-silent-protection.sh` | Modify | D3 — new Pass 0b `ALARM_CLASSES`; `:219-225` + `:424-425` become class-keyed with the `B<Pure>` follow; `:427-493` untouched |
| `build-n4-module-kit/toolbelt/run-pure-test.sh` | Modify | D4 — `rt=$(cd "$rt" && pwd)` after `:30`; subshell `cd` around `:62` |
| `build-n4-module-kit/toolbelt/lint-write-path.sh` | Modify | D5 — `--strict` flag, `_AWK_SCANNER_ALL` (properties **and** actions, any flag), per-row STALE pass, exit block at `:383`; `:374` byte-identical |
| `tests/{lint-timers,ext-writable-shape,lint-silent-protection,lint-write-path}.bats` + fixtures | Modify | From the QA RED branches; comment-decoy fixture per rule (D6) |
| `tests/c10-close.bats` | New | D8 close gate |
| `build-n4-module-kit/{BUILD-LOOP.md, skill/SKILL.md}` | Modify | K19 routing; the `--strict` row for `lint-write-path.sh` — **fragment merge** |
| `build-n4-module-kit/{retros/INDEX.md, BUILD-STATE.md, VERSION, CHANGELOG.md}` | Modify | Six retros; `0.20.0 → 0.21.0` — **fragment merge** |
| client `.gitignore` | Modify | D7 — `build/tmp/`, `*.class` |
| client `docs/write-path-matrix.md` | Modify | D5f — `[concept]` on rows `:31`, `:32`, `:33`, `:36`, `:52` |

---

## CLI Contracts

| Script | Usage | Exits |
|---|---|---|
| `lint-timers.sh` | unchanged — `<java-src-dir>` | `0` · `1` any FAIL · `3` usage (unchanged) |
| `lint-ext-writable-shape.sh` | unchanged — `[--strict] <java-src-dir>` | `0` · `1` WARN under `--strict` · `3` (unchanged) |
| `lint-silent-protection.sh` | unchanged — `[--strict] <java-src-dir>` | `0` · `1` WARN under `--strict` · `3` (unchanged) |
| `lint-write-path.sh` | **`<module-root> [--bog <f>] [--matrix <p>] [--strict]`** | `0` covered (STALE rows may print) · `1` any uncovered **or** STALE under `--strict` · `3` usage/env/missing-matrix |
| `run-pure-test.sh` | unchanged — `<module-rt-dir> <pkg.TestClass>` | `0` · `1` test failed · `2` usage · `3` env (unchanged) |

---

## PR Matrix, gates, and the OBSERVED-flip requirement

Every gate row below is a **merge precondition**. A gate is satisfied only by a verbatim RED-then-GREEN transcript; a
"would flip" prose claim is not evidence (SC-7). A smoke that cannot run is a **BLOCKER**, never an advisory.

| PR | R | Branch | RED (tip) | Gate (all) | OBSERVED flip (named mutation) |
|---|---|---|---|---|---|
| 1 | R1 | `feat/c10-lint-timers-scope` | `qa/c10-lint-timers-fp` `52ebd11` | S21-neg clean · **S21-pos still FAIL** · ColdRoomPan-rt @ ff1b659 exit **0** with `anyNoHardware` **ABSENT** · CompPan-rt + DashboardPan rt/ux verdicts unchanged · comment decoy · `kit-links.bats` · `shellcheck` 0 · issue **#89 closed by the PR** | drop the `depth>=2` guard ⇒ the class body is a method again and `anyNoHardware` re-FAILs; drop the FIELD pass ⇒ S21-neg re-FAILs; widen to any method ⇒ **S21-pos** must still FAIL |
| 2 | R2 | `feat/c10-ext-writable-per-slot` | `qa/c10-ext-writable-per-slot` `954ebd7` (re-read at apply) | EW-s22-pos clean · EW-s22-neg WARN · EW1-EW9 unchanged · **CompPan-rt exactly 1 WARN, subject `faultReset` by NAME** + absence pin (no other CompPan-rt slot flips) · DashboardPan-rt 1 (`BRoomPanel.setpoint`), ColdRoomPan-rt 0, DashboardPan-ux 0 · B831-G1 pin · comment decoy | drop the `do<Action>` scope filter ⇒ the generated `setFaultReset` setter and `:2025` exempt `faultReset` ⇒ EW10 collapses to 0; drop the name→`doX` mapping ⇒ EW-s22-pos WARNs |
| 3 | R3 | `feat/c10-silent-protection-pattern-b` | `qa/c10-silent-protection-surfaces` `f981754` | S23-pos clean · S23-neg WARN · SP1-SP8 unchanged · CompPan-rt **1 → 0** with `CompressorControl.java:294` **ABSENT** · **ColdRoomPan-rt 0 (Pattern A absence pin)** · DashboardPan 0 · comment decoy | drop the `B<Pure>` follow ⇒ `:294` WARNs again; drop Pattern A from Pass 0b ⇒ ColdRoomPan-rt returns to 1; relax the Pattern-B **AND** to OR ⇒ S23-neg (import only) stops WARNing |
| 4 | R4 | `feat/c10-cwd-independent-reds` | `qa/c10-structural-cwd` `a792d7a` | S24-cwd FAIL→pass · S24-cwd-regression stays pass · **same verdict from three cwds** (kit root, profile dir, `/tmp`) · no existing test's verdict changes · no client test file touched | revert the subshell `cd` ⇒ S24-cwd fails; revert the `rt` normalisation ⇒ S24-cwd-regression fails from a relative `.` |
| 5 | R5 | `feat/c10-write-path-stale` | **QA authoring** (`lint-write-path.bats`) | WP-stale-neg exit 0 with the row printed · WP-stale-strict exit 1 · WP-stale-regression no row · **uncovered FAIL exit 1 byte-identical with AND without `--strict`** · exit 3 preserved · real-tree matrix @ ff1b659 = **5 STALE rows** over 3 names (`:31`,`:32`,`:52` hoaMode · `:33` inhibit · `:36` freezeEnabled), and **5 from two different module roots** (root-invariance, D5a) · action rows `:64`/`:65` not STALE · `kit-links.bats` + K19 rows | key the emit by NAME instead of by ROW ⇒ 5 becomes 3; use `_op_slots` instead of `_AWK_SCANNER_ALL` ⇒ documented non-OPERATOR slots flood as STALE; drop the `@NiagaraAction` harvest ⇒ matrix rows `:64`/`:65` flood as STALE; scope the covered set to the module root instead of the matrix root ⇒ the count stops being root-invariant; accept an implicit exemption ⇒ WP-stale-neg's prose decoy passes |
| 6 | R6 | client `chore/c10-gitignore-build-caches` | none (chore) | after a client build `git status --porcelain` clean of `build/tmp` + `*.class` · **`git ls-files` for every tracked `build/**/*.jar` identical before/after** · no `vendorVersion` change · K12 | with PR5's lint: matrix STALE **5 → 0** after the `[concept]` edits |
| 7 | R7 | `chore/c10-close` | `qa/c10-close-checklist` `41bca42` | `c10-close.bats` green under `C10_CLOSE=1` with `C10_CLOSE_COMMIT` · BASE `1fb63d6` · sweeps green · INDEX pending 0 · VERSION **0.21.0** + CHANGELOG · tool-pins = C9 set · `shellcheck` 0 · **0 trailers in the range** | none — the gate is the test |

**Merge order.** PR1-PR4 are independent off kit `main`; PR5 is independent but **precedes PR6**; PR7 opens after
PR1-PR5 merge. Each branch rebases onto `main` **before** the QA ping, and the lead verifies `git log -1` equals the
blessed tip **before** settling the ledger (C8 lesson 10).

**Always-conflict files — FRAGMENT merge, never overwrite.** `BUILD-LOOP.md` routing line · `skill/SKILL.md` toolbelt
line · `retros/INDEX.md` row · `BUILD-STATE.md` envelope. Rule: **append, keep both rows, dedupe by script name**. A
second PR that rewrites the line silently reverts the first. `[ev: retro campaign8-close lesson 2]`

## Worktree / branch map (K12 — a worker writes ONLY inside its own worktree)

| Slice | Worktree | Branch |
|---|---|---|
| S21 | `niagara-tools-worktrees/c10-timers` | `feat/c10-lint-timers-scope` |
| S22 | `niagara-tools-worktrees/c10-extwr` | `feat/c10-ext-writable-per-slot` |
| S23 | `niagara-tools-worktrees/c10-silent` | `feat/c10-silent-protection-pattern-b` |
| S24 | `niagara-tools-worktrees/c10-cwd` | `feat/c10-cwd-independent-reds` |
| S25 | `niagara-tools-worktrees/c10-stale` | `feat/c10-write-path-stale` |
| S26 | `Cliente/Leon-Guanjuato-worktrees/c10-gitignore` | `chore/c10-gitignore-build-caches` |
| close | `niagara-tools-worktrees/c10-close` | `chore/c10-close` |

**Read-only trees, never written by any slice**: `Cliente/Leon-Guanjuato-worktrees/main-ff1b659` (the ONLY smoke tree).
**Poisoned tree, never read as a contract**: `niagara-tools-worktrees/c10-lints` (holds C9 pins — see the header).

---

## Testing Strategy

| Layer | What | Approach |
|---|---|---|
| Unit (bats) | Each narrowed rule, pos + neg + regression | The QA RED branch is the contract (K13); fixtures are inline heredocs into `$BATS_TEST_TMPDIR`, each with a comment-only decoy |
| Real-tree smoke | Four module roots @ ff1b659 | Exact **count + subject + absence** per root; subjects by NAME where the line is generated (D2d) |
| Mutation | Every rule change | The OBSERVED-flip column above; RED-then-GREEN transcript recorded in the retro |
| Structural (S24) | Same verdict from 3 cwds | `run-pure-test.sh` from kit root, profile dir and `/tmp` |
| Routing | K19/K20/D9b/K14 | `kit-links.bats`, `shellcheck 0.10.0`, exit-code table |

## Threat Matrix

| Boundary | Applicability | Design response | RED |
|---|---|---|---|
| Untrusted content decode | **Applicable** — all four lints read customer Java and one reads a customer markdown matrix | `LC_ALL=C`, `grep`/`awk` only, no `eval`, no interpolation of file content into a command; comment strip preserves line numbers; malformed input ⇒ exit 3, never a crash | a malformed-source fixture per touched lint |
| Subprocess / external tool | **Applicable** — `run-pure-test.sh` spawns `javac`/`java`; `lint-write-path --bog` spawns `python3` | The `cd` is a **subshell** (parent cwd and `trap … EXIT` intact); classpath elements are absolute; `command -v python3 \|\| exit 3` unchanged | S24-cwd + S24-cwd-regression |
| Filesystem write | **Applicable** — `run-pure-test.sh` compiles into `mktemp -d` | Nothing is written under `$rt`; the `cd` changes cwd only, never `-d` (`:60`) | "no new file under the module tree" pin |
| Input validation / silent coercion | **Applicable** — the `[concept]` marker and every regex pin | Literal token only, no heuristic; comments stripped before matching (D5d/D6) | WP-stale-neg prose decoy; comment decoys |
| Executable-file classification | **N/A** — no new artifact; the five touched scripts keep their existing mode | — | — |
| Git / VCS automation | **N/A** — every toolbelt script is VCS-free; `kit-links.bats` L2 fails the suite if any names `git` | — | L2 (existing) |
| Routing / auth boundary | **N/A** — W1 touches no servlet, no tunnel endpoint, no station | — | — |

## Migration / Rollout

**No migration.** No slot, no schema, no jar, no station write, no deploy in any C10 slice — `schema-risk.sh` has no
jurisdiction on the whole wave. Rollback per proposal §8: PR1/PR3/PR4/PR5 revert cleanly (issue #89 re-opens for PR1);
**PR2 is a contract revert and must revert the RED's EW10 re-pin in the same commit**, otherwise the suite is red
against a green tree; PR6 reverts two `.gitignore` lines and five markdown tokens; PR7 returns `VERSION` to `0.20.0`
and `retros/INDEX.md` rows to `pending` (retro files are never deleted).

---

## Risks

| # | Risk | Mitigation |
|---|---|---|
| DR1 | **The section-D parser is now load-bearing for three lints.** A defect in it becomes a three-lint defect | Port it **with** the D1b depth guard, in one shape, in PR1 first; PR2's copy is reviewed against PR1's. Do not "improve" it in two PRs at once — that is the always-conflict class applied to logic |
| DR2 | **Stale-checkout / stale-fixture reads** (C9 lesson 1), now with a pre-staged trap on disk | `c10-lints` is named in the header as poisoned; every cite names worktree + commit; smoke tree is `main-ff1b659` only |
| DR3 | **S22 contract drift** — a worker reads the C9 RED (`3726722`, EW10 = 0) | Re-read `qa/c10-ext-writable-per-slot` `954ebd7` at apply; workers never edit a QA RED (K13); EW10 pinned as exact count 1 + subject by NAME + absence |
| DR4 | **S21 over-narrows** and silences a real `startingUp` defect (proposal RK5) | S21-pos is the regression guard and must **stay FAIL**; the OBSERVED flip is recorded on S21-pos, not only S21-neg |
| DR5 | **S23 `B<Pure>` convention miss** (proposal RK6) | Stated limitation in the rule header; the fallback is a WARN, never a FAIL — a miss over-reports. Pattern A pinned unchanged by the ColdRoomPan-rt absence pin |
| DR6 | **S25 row grammar diverges from the proposal text** (order + `:line` subject) | Justified in D5b (report-module parser + three rows share `hoaMode`); carried to Open Questions for the spec to reconcile before PR5 opens |
| DR7 | **PR6 now depends on PR5**, against proposal §4 | Recorded in D5f; PR6's flip evidence needs PR5's rule. PR6 still never blocks a *kit* merge — only its own |
| DR8 | **S25 RED not yet authored** (proposal RK9) | RED-first is a hard precondition. If unblessed at close, R5 rolls to C11 and PR7 ships on PR1-PR4 |
| DR9 | **Comment-satisfiable pins** (C9 lesson 21) | D6 strip stage + a comment-only decoy in every S21/S22/S23 fixture and in WP-stale-neg |
| DR10 | **Always-conflict files across six parallel PRs** | Fragment merge (append, keep both rows, dedupe by script name); rebase before the QA ping |
| DR11 | **All RED pin texts are `[INFER]`** — no shell in this executor | Where this design and a landed RED disagree, **the RED wins**; the fresh-context checklist below is the reconciliation step |

---

## Fresh-context validator checklist

**Rule 0 — read `Cliente/Leon-Guanjuato-worktrees/main-ff1b659`, never `Cliente/Leon-Guanjuato`; and never treat
`niagara-tools-worktrees/c10-lints` as a RED.**

1. `git show origin/qa/c10-lint-timers-fp:build-n4-module-kit/tests/lint-timers.bats` — confirm S21-pos is the class
   FIELD + same-method shape and that it is asserted to **stay FAIL**.
2. `git show origin/qa/c10-ext-writable-per-slot:build-n4-module-kit/tests/ext-writable-shape.bats` — EW10 must read
   CompPan-rt **1**, subject `faultReset` **by name**, plus EW-s22-neg2 (`doAckAlarm`).
3. `git show origin/qa/c10-silent-protection-surfaces:build-n4-module-kit/tests/lint-silent-protection.bats` —
   SP-smoke CompPan-rt **0** and ColdRoomPan-rt **0** (Pattern A absence pin).
4. `git diff --name-only origin/main...origin/qa/c10-structural-cwd` — confirm no client test file is in the set.
5. In `main-ff1b659`: `BDefrostController.java:713/:718/:726` and `:808/:810/:850`;
   `BCompressorControl.java:381-385` (faultReset annotation), `:435/:437/:439-444` (actions), `:445-448` (class +
   `implements BIAlarmSource`), `:1882`, `:2025`, `:2093`; `CompressorControl.java:294`; `BEvaporatorUnit.java:193`.
6. `awk 'NR==31||NR==32||NR==33||NR==36||NR==40||NR==52||NR==64||NR==65' docs/write-path-matrix.md` in
   `main-ff1b659` — five stale rows over three names; `:40`'s first cell is `setpoint` (covered); `:64`/`:65` are
   ACTION rows and must **not** be STALE.
7. `grep -n 'FAIL\*|WARN\*' build-n4-module-kit/toolbelt/report-module.sh` — confirm a `STALE ` row cannot reach the
   aggregate (D5c).
8. Cross-check every proposal ID (R1-R7, SC-1..SC-12) against a D-number; a requirement with no design home is a gap.
   Confirm no line proposes a station write or a live jar deploy.

## Open Questions

- [x] **S25 row grammar — RESOLVED (coordinator 2026-09-07)**: the design's form
      `STALE  lint-write-path  <matrix>:<line>  slot <name>: no source slot with that name` is adopted; it matches the
      existing FAIL row shape at `lint-write-path.sh:374` (`FAIL  lint-write-path  <subject>  slot X: no matrix row`)
      and carries the line (three rows share `hoaMode`). Proposal R5 and spec R-S25.4 amended to it. QA's RED
      `a56a72e` asserts substrings (`STALE`, the slot name) and `grep -c '^STALE'` counts, so both forms satisfy it;
      the RED header comment (`:160`) is aligned pins-only.
- [ ] **S25 real-tree count** — **5** rows / 3 names is read at ff1b659 and assumes `inhibit` is not `--bog` traced
      (coordinator-verified read-only on the PANCCADIA bog). The apply worker re-measures with `--bog` before pinning.
- [ ] **S25 RED** — not yet authored (DR8). PR5 does not open until QA blesses the STALE pins.
- [x] **B831-G1** (`doAckAlarm` indirection) — RESOLVED in D2b step 2: action `x` ⇒ `do` + `cap1(x)`, and the write is
      sought only inside that body.
- [x] **S22 reuse-vs-new** — RESOLVED: a NEW pass inside `lint-ext-writable-shape.sh` reusing two idioms; explicitly
      **not** a reuse of `lint-silent-protection.sh`, which never reads `@NiagaraAction`.
- [ ] **Section-D depth guard, retroactive** — the `brace_depth >= 2` guard (D1b) also fixes a latent
      class-body-as-method defect in `lint-silent-protection.sh` itself. S23 inherits it; whether the C9 SP pins move as
      a result is an OBSERVED question for PR3, not a prose claim.
- [ ] **All QA RED pin texts are `[INFER]`** — no shell in this executor. The RED wins on any disagreement.

## Key Learnings

1. The `lint-timers` companion-flag false positive is not a missing field check but a candidate-detection failure —
   `@NiagaraProperty(` matches the signature regex, so the forward brace-walk swallows the entire class body.
2. Promoting `lint-silent-protection.sh`'s section-D method-boundary parser to a shared primitive fixes S21 and S22 at
   once, but it needs a `brace_depth >= 2` guard or an annotation `defaultValue` can name a class body as a method.
3. S22's `do<Action>` scope filter is load-bearing, not an optimisation: without it the slotomatic-generated
   `setFaultReset` setter would exempt every complex OPERATOR slot in the kit.
4. S25's STALE class must be per matrix ROW rather than per slot name, because `hoaMode` appears in three rows at
   ff1b659 and a name-keyed emit would report 3 where 5 rows need marking.
5. A pre-staged stale worktree on disk (`niagara-tools-worktrees/c10-lints`, holding the C9 EW10 and SP pins) is a
   higher-probability version of the stale-checkout defect than a stale remote, because it needs no network to read.
