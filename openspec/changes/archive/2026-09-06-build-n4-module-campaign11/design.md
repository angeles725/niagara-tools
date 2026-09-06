# Design: build-n4-module-campaign11

**Phase**: design · **Source**: kit `v0.21.0` (tag `dab0807`, C10 archive `154803c`) · **Target**: kit `v0.22.0`
**Inputs**: `proposal.md` (R1-R5, §4 PR chain, §7 RK1-RK11, §10 SC-1..SC-14) · `explore.md` (§1.1 constraints 1-6) ·
campaign-10 `design.md` (shape) · research **B832** `593019540` · apply-packages `4ef4f864c` / `8ad4bb36e` /
T2 re-cut `0ad09c658` (tip `4ef9726e7`) · investigador1 second reads `6f0069155` + `b12c98693` ·
coordinator T1-contract + T4-grammar amendments (2026-09-07, two messages) · QA T4 prototype `d6b840d` (relayed).
**Topic key**: `sdd/build-n4-module-campaign11/design`

**Read-tip discipline (K13/K21/C9 lesson 1).** Client cites are read from
**`/home/cristian/modulos_niagara_n4/Cliente/Leon-Guanjuato-worktrees/main-ff1b659`** and `…/main-00e7118`. The live
checkout `Cliente/Leon-Guanjuato` (`4f5f1c7`, Cristian's uncommitted files) is **never** read by this design and, after
PR3, is never a smoke target. `[ev: memory client-reads-use-a109249-worktree]`

**Verification status of this design — read this before trusting a number.** This executor has **no shell tool**; the
C10 design executor had none either and that is how the S24 rationale went wrong. Every kit line anchor below was
**opened and read** in the `niagara-tools` working tree at the C11 base and is marked `[CERT-read]`. The tree SHA
`dab0807`, the QA RED tips (`d88af78`, `ed2088f`, `54078f6`, `ff1b659`), the QA prototype `d6b840d` and the
research-repo SHAs are **relayed, not verified** — `[INFER]`. **Where this design and a landed RED disagree, the
executable RED wins.**

**Three claims in the upstream inputs are wrong and this design corrects them.**
(a) The proposal calls `lint-timers.sh:188-202` "the NET parser". Read: the parser block is **`:188-235`**; `:202` is
only the NET **gate line**. (b) The proposal lists `module-timer-lint` exit codes as "0 / 1 any FAIL / 3 usage". Read:
`lint-timers.sh:44` documents **`0 no FAIL · 1 any FAIL · 2 usage · 3 env`**, and `:58`/`:63` implement exactly that —
`lint-timers` is the one lint whose usage code is **2**, and T1 must not touch it. (c) The coordinator's "~20 toolbelt
scripts, only 2 declare a NAMED MUTATION" is directionally right and materially sharper than stated: the toolbelt has
**27** scripts, **3** `NAMED MUTATION` prose lines live in **2** files (`slot-coverage.sh:352`, `verify-module.sh:384`
and `:405`), and **zero of the 9 `toolbelt/lint-*.sh` declare one**. `[CERT-read]`

---

## Technical Approach

Four kit slices plus a close, one PR each, RED-first, ordered **T1 → T3 → T2 → T4 → close**. T1 is the only structural
move: it **extracts** the already-correct parser instead of writing a new one, and the extraction target is chosen by
behaviour, not by age — `lint-ext-writable-shape.sh` is the copy that already reads PEAK depth (`:139`, `:142`, `:147`)
and therefore the only copy whose verdict is right. T3, T2 and T4 are each a single-seam addition inside an existing
contract. Scripts stay `set -u`, `printf` rows, VCS-free (`kit-links.bats` L2), dot-dir pruned (D9b), `shellcheck
0.10.0` clean, exit codes disjoint (K20). `[ev: B832 593019540]` `[CERT-read: lint-ext-writable-shape.sh:139-147]`

---

## Architecture Decisions

### D1 — T1: one awk **function library** in a shell variable, three different consumption seams

**D1a — the three copies are not three styles of the same code; they are three *invocation mechanisms*, and that is
what dictates the fragment's form.** Read at the base:

| lint | how awk is invoked | parser block | depth rule | what it produces |
|---|---|---|---|---|
| `lint-timers.sh` | inline single-quoted program, `_cf=$(awk ' … ' "$f")` `:141` | `:188-235` | **NET** gate `:202` | `meth_start[]`/`meth_end[]` arrays (`:233`), no names |
| `lint-silent-protection.sh` | heredoc → temp file, `cat > "$_TMP/main.awk" << 'AWKEOF'` `:205`…`:524`, run `awk … -f "$_TMP/main.awk"` `:533-538` | `:302-373` | **NET** gate `:326` | `meth_start[]`/`meth_end[]`/`meth_name[]` (`:366-369`) |
| `lint-ext-writable-shape.sh` | inline single-quoted program, `out=$(awk -v FILE="$f" ' … ')` `:61` | `:136-186` | **PEAK** gate `:147`, `m_dep = max_d` `:176` | **no array** — inline `_scan_writes(body)` at close (`:179-184`), filtered by `mname in do_methods` (`:175`) |

`[CERT-read]` A quoted heredoc (`<< 'AWKEOF'`) cannot interpolate a shell variable, and `awk` accepts *either* an
inline program *or* one-or-more `-f` files, never both. So no single mechanism reaches all three call sites.

**D1b — Chosen: `toolbelt/lib/method-boundary.sh` defines ONE shell variable `MB_AWK` whose value is awk *function
definitions only* (no `BEGIN`, no `END`, no pattern rules).** A function-only text is legal in both mechanisms:
inline consumers write `awk -v FILE="$f" "$MB_AWK"' { lines[NR]=$0 } END { … } ' "$f"` (adjacent-quote concatenation;
the value's own `$` is not re-expanded, so no escaping is needed), and `lint-silent-protection.sh` writes it once with
`printf '%s\n' "$MB_AWK" > "$_TMP/method-boundary.awk"` and adds a **second `-f`** at `:538`. Both read the same source
of truth. The `_AWK_SCANNER='…'` awk-in-shell-var idiom is already in the kit at `lint-write-path.sh:348`, and the
multi-`-f` idiom is already in the kit at `lint-wb-threading.sh:81`/`:193`/`:220-221`. **Rejected**: converting
`main.awk`'s heredoc to unquoted — the awk body is dense with `$0`, `$2`, `\/` and would need whole-file escaping, a
silent-corruption class far worse than the bug being fixed. **Rejected**: shelling out to another lint — K20 exit
codes collide and each lint stops being independently runnable (the same rejection C10 §D2a already made).
`[CERT-read: lint-write-path.sh:348; lint-wb-threading.sh:81,:193,:220-221; lint-silent-protection.sh:205,:524,:533-538]`

**D1c — how the fragment is located. Chosen: `${BASH_SOURCE[0]%/*}`, not `$KIT`.** Line added to each of the three
lints immediately after `set -u`:

```bash
# shellcheck source=lib/method-boundary.sh
. "$(cd "${BASH_SOURCE[0]%/*}" && pwd)/lib/method-boundary.sh"
```

The kit has two location idioms: `$KIT` env → `BASH_SOURCE` parent → cwd (`kit-ticket.sh:25-32`,
`new-retro.sh:28-38`) and `BASH_SOURCE`-only (`report-module.sh:20`, `scaffold-module.sh:24`). **Chosen the second.**
`$KIT` is a *writable-target* seam — `retro-loop.bats:26-27` deliberately runs those two scripts as `env -C "$TK"
KIT="$TK"` to redirect their writes into a temp kit. A lint that resolved its own parser through `$KIT` would load a
**different** parser than the script the user invoked whenever `$KIT` is set, which is exactly the fixture-green /
real-red class C8 lesson 1 names. `BASH_SOURCE` binds the fragment to the script that is actually running.
**Rejected**: `. "$KIT/toolbelt/lib/method-boundary.sh"` as the proposal sketched it — it is unset in every current
lint bats (`lint-delays.bats:21`, `rc-scan.bats:23`, `lint-silent-protection.bats:27` all *derive* `KIT` locally and
never export it), so the lints would break outside bats. `[CERT-read]`

**D1d — the awk interface (ONE contract, resolving investigador1 `b12c98693` finding 1).** The union of what the three
lints need is the array form; `lint-ext-writable-shape` adapts to it.

```awk
# mb_strip(src, n, dst)        — blank // and /* */ content, block state carried across lines,
#                                line numbers preserved. dst[1..n] out. (C10 design §D6 primitive.)
# mb_parse(src, n, m_start, m_end, m_name)  -> returns cnt (number of methods)
#   INPUT   src[1..n]  ALREADY comment-stripped lines (caller runs mb_strip first), n = line count
#   OUTPUT  m_start[0..cnt-1] first line of the method (the line carrying the opening brace)
#           m_end  [0..cnt-1] line on which the method's brace closes (== m_start for a one-liner)
#           m_name [0..cnt-1] resolved identifier
#   All awk locals declared as trailing params. No globals read or written. No I/O.
```

Zero-indexed because both existing array consumers are zero-indexed (`lint-timers.sh:233`,
`lint-silent-protection.sh:366-369`). `m_name` is filled for **all three** consumers even though `lint-timers` ignores
it — one contract, not two. `[CERT-read]`

**D1e — what each lint keeps LOCAL (nothing shared beyond the boundary).**

| lint | deleted | kept, unchanged |
|---|---|---|
| `lint-timers.sh` | `:188-235` (48 lines) | Phase 1 class-FIELD pass `:163-186`; Phase 3 same-method companion pairing `:237-254` (reads `meth_start`/`meth_end` — already the fragment's output shape); Pass 2 `stopped()`/`started()` clear check `:257+`; FAIL rows; **exit map 0/1/2 usage/3 env** `:44`,`:58`,`:63` |
| `lint-silent-protection.sh` | `:302-373` (72 lines) **including the `:303-307` NET rationale comment** | Pass 0/0b/1 surface + alarm-class + `SURF_WRITE_FIELDS` harvest `:60-204`; section E trip/surface passes `:375-523` (reads `meth_name`); per-trip dedupe `:541-549`; exits `0/1 --strict/3` `:19` |
| `lint-ext-writable-shape.sh` | `:137-178` boundary core (42 lines) | Pass 1 `@NiagaraAction` paren-balanced harvest `:100-126`; `do_methods` set; `_scan_writes()`; Pass 3 `@NiagaraProperty` `:188+`; exits `0/1 --strict/3` `:27` |

**D1f — the ext-writable adaptation, line-itemized (coordinator finding 1).** It is the only consumer that must change
*shape*, and it **shrinks**:

| Current | Becomes |
|---|---|
| `:137` `brace_depth = 0; in_m = 0; m_start = 0; m_dep = 0` | `_n = mb_parse(slines, NR, ms, me, mn)` |
| `:138-174` per-line loop + Case A + Case B (37 lines) | *deleted* |
| `:175-177` `if (mname != "" && mname in do_methods) { in_m=1; … }` | filter moves into the new loop: `if (!(mn[k] in do_methods)) continue` |
| `:179-184` close test + `body` build + `_scan_writes(body)` (6 lines) | `for (k=0;k<_n;k++) { if (!(mn[k] in do_methods)) continue; body=""; for (bi=ms[k];bi<=me[k];bi++) body = body " " slines[bi]; _scan_writes(body) }` (7 lines) |

Net **−42 / +9**. The `do_methods` filter moving from *open-time* to *iterate-time* is behaviour-neutral: the old code
only ever built `body` for methods already in `do_methods`, and the new code only ever calls `_scan_writes` for the
same set. `[CERT-read: lint-ext-writable-shape.sh:136-186]`

**D1g — PEAK depth semantics, stated once so the replacement comment can be written from it.** Per line `i`: `old_d` =
depth **before**, `brace_depth` = depth **after** (NET), `max_d` = the maximum depth reached **during** the line
(PEAK), incremented at each `{` (`:142`).

- **Open** ⟺ `max_d > old_d && max_d >= 2`, then `m_dep = max_d`.
- **Close** ⟺ `brace_depth < m_dep`, tested **after** the open test, **in the same iteration**.
- Multi-line method: `max_d = brace_depth = old_d+1` → open at `i`, close at some `j > i`. Identical to NET.
- **One-liner** `void arm(){ flag=true; Clock.schedule(…); }`: `max_d = old_d+1 ≥ 2` → open fires; `brace_depth =
  old_d < m_dep` → close fires **on the same line** → method `= [i, i]`. NET never opens it at all (`brace_depth >
  old_d` is false), which is the false negative.

**This is the exact and complete answer to `:303-307`.** That comment fears "the method-open event being attached to a
post-close depth that spans until the class closes". Under PEAK that span is impossible **by construction**: the close
test runs in the same iteration as the open, so a one-liner is bounded to one line before the loop advances. The
`>= 2` guard independently prevents the class body itself from ever opening a method. `lint-ext-writable-shape` has
shipped this ordering since C9 (`:147` then `:179`). **The comment is replaced, not deleted** — a wrong rationale left
in the tree is what makes the next author restore NET depth (RK5b). Replacement text is D1g's three bullets.
`[CERT-read: lint-silent-protection.sh:303-307,:326; lint-ext-writable-shape.sh:147,:179]`

**D1h — the five invariants, with the OBSERVED mutation and the *named fixture* each one owes (K24(7)).** A mutation
without a named fixture is not a pin; a fixture that does not flip is not evidence.

| # | Invariant | Where it lives in `mb_parse` | OBSERVED mutation | Fixture it must flip |
|---|---|---|---|---|
| **I1** | `max_d >= 2` class-body/FIELD guard | open gate | drop `&& max_d >= 2` | **S21-misparse** (`tests/lint-timers.bats:436`) FALSE-FAILs — the `@NiagaraProperty(defaultValue="new BAlarmRecord()")` shape names a method again |
| **I2** | Case-B backward scan stops at `@`, `;$`, `{`; ≤20 lines back | Case B | drop the `substr(lk_t,1,1)=="@"` break | **S21-misparse** FALSE-FAILs (`BAlarmRecord` resolved as `mname`) |
| **I3** | Case-A keyword exclusion `if\|for\|while\|switch\|catch\|try\|else\|do\|new` | Case A | delete the exclusion regex | **G-samemethod** (`qa/c11-golden-parser` `ed2088f`) — an `if (…) {` is named as a method and splits the body |
| **I4** | **PEAK** open / **NET** close, same iteration | open + close | restore `brace_depth > old_d` (NET) | **C11-tl-oneliner** (`qa/c11-parser-oneliner` `d88af78`) returns exit 0 (FAIL vanishes) **and** **C11-sp-oneliner** returns 0 WARN |
| **I5** | one-line `get\|set\|is` accessor skip (B832-G1) | post-name filter | delete the skip | **C11-g1-setter** FALSE-WARNs |

**I5's exact predicate is narrower than "accessor".** Skip ⟺ `(max_d > old_d && brace_depth <= old_d)` — i.e. the
method opens **and closes** on this line — **AND** `mname ~ /^(get|set|is)[A-Z_]/`. A multi-line `isDirty()` that
schedules is untouched; only the one-liner form is skipped. This is the whole of RK3's mitigation and it is why the
skip cannot be written as a bare name test. `[ev: B832-G1]` `[ev: QA d88af78]`

**B832-G2 (`/* */` strip) gets a header NOTE and NO fixture.** `lint-silent-protection` today strips `//` only
(`:314`, `:341`); `mb_strip` strips `//` **and** `/* */` (the `lint-timers.sh:145-161` / `lint-ext-writable-shape.sh`
D6 form). QA could not build a biting fixture, and a vacuous pin is worse than none. It is pinned instead by the 3×3
baselines below: any delta there **is** the G2 evidence, and a delta blocks the merge. `[ev: B832-G2]` `[CERT-read]`

**D1i — the golden-set runner. ONE bats file, seven cases, each asserted against all three lints in the same test.**
`tests/golden-parser.bats` (cherry-picked from `qa/c11-golden-parser` `ed2088f`, never re-authored — K13). Shape: one
fixture tree per case under `tests/fixtures/golden-parser/<case>/`, and each `@test` runs `lint-timers.sh`,
`lint-silent-protection.sh` and `lint-ext-writable-shape.sh` over that same tree and asserts the exact triple of
verdicts. Running all three inside one test is the point: a per-lint file could go green on two lints and red on the
third across two merges, which is precisely the inconsistency T1 exists to remove.

| Case | What it pins | Expected triple (timers / silent / ext-writable) |
|---|---|---|
| G-multiline | ordinary multi-line method | unchanged verdicts, identical to `dab0807` |
| G-samemethod | flag + schedule in the same method, `if` blocks inside | FAIL / — / — |
| G-adapter | Pattern-B `BIAlarmSource` adapter body | — / clean / — |
| G-oneliner-timers | one-liner `arm()` FIELD flag + `Clock.schedule` | **FAIL** (was exit 0) / — / — |
| G-oneliner-silent | one-liner `step()` with a detected trip | — / **WARN** (was 0) / — |
| G-oneliner-extwritable | one-liner `doX()` writing a slot | — / — / clean (already correct at `dab0807`; regression pin) |
| G-accessor | one-liner `setInhibited(…)` guarded write | 0 / **0 WARN** / 0 |

**D1j — the 3×3 real-tree baselines, and why identity is the expected result.** Before the cut, capture verbatim (row
text **and** exit code) for `lint-timers.sh`, `lint-silent-protection.sh`, `lint-ext-writable-shape.sh` × `CompPan-rt`,
`ColdRoomPan-rt`, `DashboardPan-rt` under `main-ff1b659`; after the cut, re-capture and `diff`. **Byte-identical is the
EXPECTED and REQUIRED result**, because investigador1's second read found **0 one-liner methods carrying a schedule,
alarm or trip write in the 42 `.java` at `ff1b659`**. Any delta is a defect (most likely the G2 strip), not a win, and
blocks the merge. The nine captures are attached to the PR body verbatim. `[ev: investigador1 6f0069155]` `[INFER]`

**D1k — size. The `~500-600` estimate is a floor, not a ceiling; the honest band is 600-700.**

| Part | Δ lines |
|---|---|
| NEW `toolbelt/lib/method-boundary.sh` (`mb_strip` + `mb_parse` + header) | +150 |
| `lint-timers.sh` −48 / +9 | 57 |
| `lint-silent-protection.sh` −72 / +14 (incl. the replaced rationale) | 86 |
| `lint-ext-writable-shape.sh` −42 / +9 + source line + comment | 60 |
| golden + one-liner fixtures & bats (QA cherry-picks `ed2088f`, `d88af78`) | ~300 |
| **Total changed** | **≈ 663** |

Authored (non-fixture) work is **≈ 353**; per `sdd-phase-common.md` §E generated goldens are excluded from authored
risk but included in snapshot identity, so both numbers go in the PR body. **The `size:exception` is re-scoped from
"~500-600" to a ceiling of 700 changed lines.** If the measured diff exceeds 700, the lead **re-requests the exception
with the measured number** — the parser is never split, and the fixtures are never moved out of PR1 (moving them
breaks RED-first: the flip must be proven in the diff that causes it).

---

### D2 — T3: `DRIFT` is the `[concept]` branch **inverted**, not a new pass

**D2a — insertion point.** The S25 row pass reads `:421-464`. Today `:441` is
`case "$_row" in *'[concept]'*) continue ;; esac` — an **unconditional** skip that runs *before* the slot-name
extraction (`:443-448`) and before the covered-set test (`:450`). Chosen: **capture the marker instead of acting on
it**, then let the row fall through the existing extraction so a `[concept]` row is subject to the same
`^[a-z][A-Za-z0-9]*$` slot-name filter as every other row (a prose `[concept]` cell must stay silent).
`[CERT-read: lint-write-path.sh:432-455]`

```
:441  case "$_row" in *'[concept]'*) continue ;; esac
  →   _is_concept=0
      case "$_row" in *'[concept]'*) _is_concept=1 ;; esac
:450  case "$_covered_flat" in *" $_name "*) continue ;; esac
  →   case "$_covered_flat" in *" $_name "*)
              if [ "$_is_concept" -eq 1 ]; then
                  printf 'DRIFT  lint-write-path  %s:%d  slot %s: concept marker but a source slot exists\n' \
                      "$MATRIX" "$_ln" "$_name"
                  DRIFT=1
              fi
              continue ;;
      esac
      [ "$_is_concept" -eq 1 ] && continue          # true concept row: name absent from source -> silent
```

**D2b — the full truth table** (STALE and FAIL columns are byte-identical to `v0.21.0`):

| row | slot in covered set | verdict |
|---|---|---|
| `[concept]`-marked | **yes** | **DRIFT** (new) |
| `[concept]`-marked | no | silent (a true concept row) |
| plain | yes | silent (unchanged) |
| plain | no | STALE (unchanged, `:452-454`) |

**D2c — the decoy is protected by an existing line, and that is exactly why it needs a pin.** `:439` strips
`<!--…-->` from `$_mline` **before** the marker test, so a `[concept]` inside an HTML comment yields `_is_concept=0`
and can never produce DRIFT. The pin exists because that strip is one `sed` away from being moved; the existing
`WP-stale-concept-decoy` (`tests/lint-write-path.bats:237`) proves the same strip for STALE. `[CERT-read]`

**D2d — covered set: reuse, do not re-harvest.** `_covered_flat` (`:432`) is built from `_covered_names` (`:155-164`),
the matrix-root-wide `@NiagaraProperty ∪ @NiagaraAction` harvest with the multi-line `name = "X"` field-line match at
`:162` (K24(3), K24(5)) and D9b dot-dir + `build/` prune at `:156`, unioned with `$_bog_extra`. DRIFT reads the same
variable in the same loop — there is no second harvest and therefore no second count to disagree (RK8).

**D2e — exit expression.** `:462-464` becomes:

```bash
[ "$FAILED" -eq 1 ] && exit 1                                             # unchanged, wins first
[ "$STRICT" -eq 1 ] && { [ "$STALE" -eq 1 ] || [ "$DRIFT" -eq 1 ]; } && exit 1
exit 0
```

FAIL wins, then `--strict && (STALE || DRIFT)`. Exit 3 (usage / env / missing matrix) is unreachable from here — it is
emitted upstream at `:137` and documented at `:25` — so the range stays `{0,1} ∪ {3}` (K20). **No new flag**: DRIFT
rides the existing `--strict`, as the proposal requires. `[CERT-read]`

**D2f — OBSERVED mutation.** Delete the `if [ "$_is_concept" -eq 1 ]` wrapper so every covered row emits DRIFT → the
true-concept negative case flips to a false DRIFT. Named fixture: the T3 RED's negative case on
`qa/c11-concept-drift`. Real-tree pin: client `00e7118`, **0 DRIFT** (the five PR6 `[concept]` rows have no source
slot). `[INFER]`

---

### D3 — T2: one library, one default, `load`-based, and one assertion that stops pinning a bug

**D3a — the lib.** NEW `tests/lib/client-root.bash`:

```bash
# tests/lib/client-root.bash — the ONE blessed client read tree for every bats suite (C11 T2).
# Env override wins: `:=` only assigns when the caller left it unset/empty.
: "${CLIENT_READ_ROOT:=/home/cristian/modulos_niagara_n4/Cliente/Leon-Guanjuato-worktrees/main-ff1b659}"
: "${C9_CLIENT_ROOT:=$CLIENT_READ_ROOT}"
: "${C9_CLIENT_REPO:=$CLIENT_READ_ROOT}"
: "${C8_CLIENT_REPO:=$CLIENT_READ_ROOT}"
export CLIENT_READ_ROOT C9_CLIENT_ROOT C9_CLIENT_REPO C8_CLIENT_REPO
```

`:=` gives "env override wins" with **no branch** — the three override pins SC-6 demands each exercise one variable
against a temp dir. `[ev: 0ad09c658]`

**D3b — the sourcing line, one per bats, at file scope (not inside `setup()`).** Chosen: **bats `load`**, the kit's
existing test-library idiom (`preflight.bats:24` `load helpers/n4-fixtures`, `build-sh.bats:5`):

```bash
load lib/client-root
```

`load` resolves relative to `$BATS_TEST_DIRNAME` and appends `.bash`, so it is position-independent and matches the
existing `KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/…"` anchoring already used by all ten suites (`lint-delays.bats:21`,
`rc-scan.bats:23`, `lint-silent-protection.bats:27`, `c8-close.bats:18-19`, `kit-links.bats:7`). File scope, not
`setup()`, so the values exist for `@test` bodies that do not go through `setup`. **Rejected**: a bare
`. "$BATS_TEST_DIRNAME/lib/client-root.bash"` — identical behaviour, but `load` is what the kit already does and
`kit-links`-style conventions are cheaper to keep than to fork. `[CERT-read]`

**D3c — the ten conversions (all ten verified at the base).**

| # | site | today | becomes |
|---|---|---|---|
| 1 | `ext-writable-shape.bats:26` | `ROOT="${C9_CLIENT_ROOT:-/home/…/main-ff1b659}"` | `ROOT="$C9_CLIENT_ROOT"` |
| 2 | `demand-in-scope.bats:27` | same shape | `ROOT="$C9_CLIENT_ROOT"` |
| 3 | `lint-silent-protection.bats:30` | same shape | `ROOT="$C9_CLIENT_ROOT"` |
| 4 | `lint-timers.bats:418` | same shape | `ROOT="$C9_CLIENT_ROOT"` |
| 5 | `lint-write-path.bats:338` | same shape | `ROOT="$C9_CLIENT_ROOT"` |
| 6 | `c9-close.bats:108` | `R="${C9_CLIENT_REPO:-…main-ff1b659}"` | `R="$C9_CLIENT_REPO"` |
| 7 | `c10-close.bats:90` | same shape | `R="$C9_CLIENT_REPO"` |
| 8 | `c8-close.bats:107` | `R="${C8_CLIENT_REPO:-/home/…/Cliente/Leon-Guanjuato}"` ← **live checkout** | `R="$C8_CLIENT_REPO"` (default retargets to `main-ff1b659`) |
| 9 | `lint-delays.bats:53` | `CRP="$HOME/…/Leon-Guanjuato/Paccadia/ColdRoomPan/ColdRoomPan-rt/src"` ← **no override at all** | `CRP="$C9_CLIENT_ROOT/Paccadia/ColdRoomPan/ColdRoomPan-rt/src"` |
| 10 | `rc-scan.bats:75` | `UX="$HOME/…/Leon-Guanjuato/Dashboard/DashboardPan/DashboardPan-ux"` ← **no override at all** | `UX="$C9_CLIENT_ROOT/Dashboard/DashboardPan/DashboardPan-ux"` |

`[CERT-read: grep at the C11 base, 10 sites, 10 files]`

**D3d — LD5: the assertion flip, and the commit-body evidence it owes.** `tests/lint-delays.bats:52-58` today:

```bash
@test "LD5: real smoke — the current ColdRoomPan-rt src FAILs naming BDefrostController (SKIP if not present)" {
  CRP="$HOME/…/Leon-Guanjuato/Paccadia/ColdRoomPan/ColdRoomPan-rt/src"
  [ -d "$CRP" ] || skip "…"
  run "$LD" "$CRP"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]] && [[ "$output" == *"BDefrostController"* ]]
}
```

`:56-57` become `[ "$status" -eq 0 ]` and `[[ "$output" != *"FAIL"* ]]`, and the title is re-cut to name the **clean
state of the blessed tree**, not a defect. **The commit body must carry, verbatim:** (i) `lint-delays.sh` output +
exit on `4f5f1c7` (exit 1, FAIL `BDefrostController`), (ii) the same on `main-ff1b659` (exit 0, no FAIL), (iii) the
sentence that makes it a rule and not a preference — *the delay-floor rule is pinned by the synthetic fixtures `LD1`
(`:28`), `LD3` (`:40`), `LD6` (`:64`) and `LD11-misguard` (`:117`), which must stay green in the same run; LD5 is a
tree-state smoke, and a tree-state smoke asserts the tree's current state*. No new fixture is owed.
`[CERT-read: lint-delays.bats:28,:40,:52-58,:64,:117]`

**One consequence the inputs did not name.** `LD10` (`lint-delays.bats:86`) is already a *fixed-tree PASS* pin, gated
on `C8_CRP_FIXED` against `c66e412`. After the retarget LD5 and LD10 assert the same class on different trees.
Decision: **keep both, unmerged** — LD10 stays an explicitly env-gated historical-tree pin; LD5 becomes the
unconditional blessed-tree pin. Collapsing them would delete the only assertion that the *historical* fixed tree is
still clean. `[CERT-read: lint-delays.bats:86]`

**D3e — c8-close SC1-smoke and RC8.** `c8-close.bats:105-114` already accepts `[ "$status" -eq 1 ] || [ "$status" -eq
0 ]` and echoes the exit — it never pinned the defect, so **only its root changes** (site 8); its assertion is
untouched. `rc-scan.bats:74-80` **RC8** asserts `status 1` + `FAIL` + `host` on the DashboardPan-ux `:701` host
literal, which is 1 FAIL on **both** trees: **assertion unchanged**, only the root moves (site 10). RC8 is therefore
the control that proves the retarget did not silently change what the suite measures.
`[CERT-read: c8-close.bats:105-114; rc-scan.bats:74-80]`

**D3f — RK5 audit obligation, discharged in this PR.** While converting the ten sites the worker classifies **every**
real-tree smoke's assertion: a smoke asserting a FAIL must either have its rule carried by a synthetic fixture (LD5 →
`LD1`/`LD3`/`LD6`) or carry a named ticket. The audited list goes in the PR body. Recorded as a K-lesson candidate at
close.

---

### D4 — T4: the grammar is **created** in PR4, because the kit has none — and the RED must prove a positive MATCH

**D4a — the measured starting state (this is the finding, and it is not vacuous).** `[CERT-read]`

- `# Mutation:` in `build-n4-module-kit/toolbelt/**`: **0 occurrences**.
- `Mutation:` in the kit: `retros/*.md` only (e.g. `2026-09-05-campaign8-report-integration.md:46,:61,:74`) — the
  wrong file, and free prose.
- Free-prose `NAMED MUTATION` inside toolbelt scripts: **3 lines in 2 files** — `slot-coverage.sh:352` (→ `WB-LEX1`),
  `verify-module.sh:384` (→ `WB-SCAFFOLD1`), `:405` (→ `WB-DEP1`). **Neither file is a `lint-*.sh`.**
- Free-prose `NAMED MUTATION` in `tests/*.bats`: ~25 files, naming test IDs inside sentences
  (`lint-delays.bats:17-18`, `lint-timers.bats:21`, `lint-silent-protection.bats:23`, …).
- `toolbelt/lint-*.sh` at the base: **9** — `lint-delays`, `lint-demand-scope`, `lint-ext-writable-shape`,
  `lint-servlet`, `lint-silent-protection`, `lint-structure`, `lint-timers`, `lint-wb-threading`, `lint-write-path`.
  **Zero declare a mutation.**

So a blanket "no named mutation → WARN" over all 27 toolbelt scripts reports ~24 WARN, and a grammar that only checks
*declared* mutations reports **0 WARN over 0 declarations** — the vacuous result RK7 predicts. Both are useless.

**D4b — Chosen scope: `toolbelt/lint-*.sh` only. Non-lint toolbelt scripts are OUT of T4's scan, stated as a rule.**
K24(7) is a contract about **guards**: a script with a verdict to mutate owes a pin. `build.sh`, `scaffold-module.sh`,
`report-module.sh`, `station-snapshot.sh` and the rest emit artifacts, not verdicts. **Rejected**: scanning all 27 —
it manufactures ~18 WARNs that no one can honestly close, and a lint whose default output is noise gets muted.
**Rejected**: an allowlist of exempt scripts — an allowlist is a lie generator; a scope rule is checkable. The 3
legacy free-prose lines in `slot-coverage.sh` / `verify-module.sh` are out of scope **by this rule** and are recorded
as a **C12 seed**, not silently absorbed.

**D4c — the grammar. QA freezes the RED on exactly this.** One line per named mutation, in the script's comment
header (lines 1-60), ASCII separator:

```
# Mutation: <fixture-id> -- <what it flips>
```

Extraction ERE (`LC_ALL=C` safe — **no em dash**; several lints set `LC_ALL=C`, e.g. `lint-ext-writable-shape.sh:32`):

```
declaration :   ^# Mutation: [A-Za-z][A-Za-z0-9_-]* -- .+$
capture id  :   sed -E 's/^# Mutation: ([A-Za-z][A-Za-z0-9_-]*) -- .*/\1/'
fixture set :   grep -hoE '^@test "[^":]+:' tests/*.bats | sed -E 's/^@test "([^":]+):.*/\1/' | sort -u
```

The fixture id is the leading token of a bats test name **before the first colon** — the convention every suite
already follows (`TL1:` `lint-timers.bats:63`, `S21-misparse:` `:436`, `SP8:` `lint-silent-protection.bats:162`,
`EW-s22-nondo:` `ext-writable-shape.bats:245`, `WP-stale-perrow:` `lint-write-path.bats:250`, `DS2:`
`demand-in-scope.bats:51`, `LD1:` `lint-delays.bats:28`). Ids with spaces (e.g. `TC-A companion`) are unreferenceable
by construction and that is intentional. `[CERT-read]`

**D4d — rows and exits.**

```
MATCH  lint-guard-pins  <script>:<line>  mutation <id> -> tests/<file>.bats:<line>
WARN   lint-guard-pins  <script>:<line>  mutation <id>: no @test with that id in tests/*.bats
WARN   lint-guard-pins  <script>  no named mutation (K24(7): add '# Mutation: <fixture-id> -- <what it flips>')
```

Exit **0** default even with WARN · **1** any WARN under `--strict` · **3** usage / no `tests/` dir / no
`toolbelt/lint-*.sh` found (K20 disjoint, and never a silent 0 — the `lint-ext-writable-shape.sh:49-55` EW11
precedent). D9b: both scans prune dot-dirs. VCS-free (`kit-links.bats` L2).

**D4e — the PR4 retrofit, one line per lint, every fixture id verified to exist at the base.**

| lint | `# Mutation:` line(s) added | fixture verified |
|---|---|---|
| `lint-timers.sh` | `S21-misparse`, `S21-neg` | `lint-timers.bats:436`, `:367` |
| `lint-silent-protection.sh` | `S23-pos`, `S23-and` | `lint-silent-protection.bats:188`, `:270` |
| `lint-ext-writable-shape.sh` | `EW-s22-neg2`, `EW-s22-nondo` | `ext-writable-shape.bats:227`, `:245` |
| `lint-write-path.sh` | `WP-stale-perrow`, `WP-stale-concept-decoy` | `lint-write-path.bats:250`, `:237` |
| `lint-demand-scope.sh` | `DS2` | `demand-in-scope.bats:51` |
| `lint-delays.sh` | `LD1` | `lint-delays.bats:28` |
| `lint-servlet.sh` | `LSV1` | `lint-servlet.bats:25` |
| `lint-structure.sh` | `LS7` | `lint-structure.bats:44` |
| `lint-wb-threading.sh` | `WBT1c` | `lint-wb-threading.bats:32` |
| `lint-guard-pins.sh` (new) | `GP-pos` | its own RED |

`[CERT-read — all ten fixture ids opened and read]`

**D4f — the real-kit smoke asserts a MATCH COUNT, never "0 found" (coordinator finding 2).** After PR1-PR3 + the D4e
retrofit, running `lint-guard-pins.sh` over the kit must report **0 WARN and exactly 10 lint scripts each carrying
≥1 MATCH** (14 `# Mutation:` lines total). The smoke asserts, in this order: (i) `lint-*.sh` count = **10**, (ii) the
number of distinct scripts appearing in `MATCH` rows = **10**, (iii) one exact row, e.g.
`MATCH  lint-guard-pins  …/lint-timers.sh:<n>  mutation S21-misparse -> tests/lint-timers.bats:436`, (iv) WARN count
= **0**. Assertion (iii) is the anti-vacuity clause: a parser that finds nothing cannot satisfy it.

**D4g — RED cases.** `GP-pos`: a fixture lint header naming a non-existent id → exactly **1 WARN**, exit **0**;
`--strict` → **1**. `GP-nomut`: a fixture lint with no `# Mutation:` line → **1 WARN** (`no named mutation`).
`GP-neg`: every named mutation resolves → **0 WARN**, ≥1 MATCH. `GP-usage`: no argument → **3**. `GP-prune`: a lint
copy under `.deploy-baseline/` is not scanned (D9b). `GP-decoy`: `# Mutation:` inside a *bats* comment or a
`# # Mutation:` double-comment is not a declaration. `GP-real`: D4f.

**D4h — K19 routing.** `lib/method-boundary.sh` and `lint-guard-pins.sh` are added to **both** `BUILD-LOOP.md` and
`skill/SKILL.md` (`kit-links.bats` L5/L7 fail otherwise). `lib/method-boundary.sh` is a **sourced fragment, not a
routed command** — it is listed under the toolbelt table with an explicit `(library — sourced, not invoked)` note so
L4's file-exists row passes without implying a CLI. `[CERT-read: kit-links.bats:52,:68,:83,:97]`

**D4i — revised size.** Script ~140 + bats ~90 + fixtures ~40 + retrofit 14 lines + K19 rows 4 + BUILD-STATE/retro
~15 ≈ **300** (proposal said ~220). Still under 400; stated so the PR body carries the real number (C8 lesson:
real counts over estimates).

---

### D5 — T5: close

`VERSION` lives at the **repo root**, not in the kit (`c10-close.bats:39` reads `"$REPO/VERSION"`); it reads `0.21.0`
at the base → **`0.22.0`** (MINOR: two new capabilities, two additive widenings, no CLI removal). `[CERT-read]`

`tests/c11-close.bats`, mirroring `c10-close.bats:16-45`, freezes:

| pin | value |
|---|---|
| `BASE` | `dab0807` (the C10 close commit / tag `v0.21.0`) — the no-trailer sweep range base, mirroring `c10-close.bats:18` (`BASE="1fb63d6"` = the C9 close) |
| `VERSION_TARGET` | `${C11_VERSION:-0.22.0}`, `TAG="v0.22.0"` |
| `_close()` guard | `[ -n "${C11_CLOSE:-}" ] \|\| skip` |
| tool-pins | the C9/C10 set **+ `toolbelt/lib/method-boundary.sh` + `toolbelt/lint-guard-pins.sh`** (C11 is the first campaign since C9 to add tool FILES) |
| sweeps | `sweep-fold-audit.sh --strict` exit 0 **and** `sweep-build-state.sh` exit 0 (`c10-close.bats:24-30`) |
| `CLOSE-index-pending` | `retros/INDEX.md` `\| pending \|` rows = **0** (`:32-35`) |
| `CLOSE-changelog` | a `## [v0.22.0]` section naming T1-T4 (`:42-45`) |
| client versions | carry over 2.2.0 / 2.1.0 / 2.2.0 — **C11 ships no jar**, so no `vendorVersion` bump |
| `TODO(freeze)` | QA freezes the remaining pins at close on `qa/c11-close-checklist` |

**Retros**: one per kit-changing push (close-gate exit (a) NEW RETRO), folded to `retros/INDEX.md` pending = **0**.
Expected C11 retros: `campaign11-shared-method-boundary`, `campaign11-concept-row-drift`, `campaign11-client-root`,
`campaign11-guard-pins`, `campaign11-close-process-meta-lessons`. C12 seeds recorded at close: the 3 legacy
`NAMED MUTATION` prose lines in non-lint toolbelt scripts (D4b), the RK5 smoke-assertion-class rule (D3f), and
B832-G2's missing biting fixture.

---

## Data Flow — where each rule decides after C11

```
                      toolbelt/lib/method-boundary.sh   ($MB_AWK: mb_strip, mb_parse)
                                     │
        ┌────────────────────────────┼────────────────────────────┐
        │  "$MB_AWK"'…'              │ printf > $_TMP + 2nd -f    │  "$MB_AWK"'…'
        ▼                            ▼                            ▼
  lint-timers.sh              lint-silent-protection.sh    lint-ext-writable-shape.sh
  FIELD pass :163-186         Pass 0/0b/1 :60-204          @NiagaraAction harvest :100-126
        │                            │                            │
   m_start/m_end               m_start/m_end/m_name          m_start/m_end/m_name
        ▼                            ▼                            ▼
  Phase 3 pairing :237-254    section E trips :375-523      iterate ∩ do_methods → _scan_writes
        │                            │                            │
   FAIL companion-flag          WARN silent trip             WARN ext-writable-shape
   exit 0/1/2/3                 exit 0/1(--strict)/3         exit 0/1(--strict)/3

  lint-write-path.sh   covered set :155-164 ─┬─ plain+uncovered → STALE :452
                       row pass :434-455     ├─ [concept]+covered → DRIFT   (new)
                                             └─ uncovered OPERATOR → FAIL   (untouched)
                       exit :462-464   FAIL wins → --strict && (STALE||DRIFT) → 0

  lint-guard-pins.sh   toolbelt/lint-*.sh headers ──┐
                                                    ├─→ MATCH / WARN, exit 0 / 1(--strict) / 3
                       tests/*.bats @test ids ──────┘

  tests/lib/client-root.bash ── CLIENT_READ_ROOT=main-ff1b659 ──→ 10 bats sites (env override wins)
```

---

## File Changes

| File | Action | Description |
|---|---|---|
| `build-n4-module-kit/toolbelt/lib/method-boundary.sh` | **Create** | T1 — `$MB_AWK`: `mb_strip` + `mb_parse`; the **first** `toolbelt/lib/` entry |
| `build-n4-module-kit/toolbelt/lint-timers.sh` | Modify | T1 — delete `:188-235`, source + `"$MB_AWK"` at `:141`; exit map `0/1/2/3` untouched |
| `build-n4-module-kit/toolbelt/lint-silent-protection.sh` | Modify | T1 — delete `:302-373` incl. the `:303-307` NET comment; write `$MB_AWK` to `$_TMP`, second `-f` at `:538`; inherits `/* */` strip |
| `build-n4-module-kit/toolbelt/lint-ext-writable-shape.sh` | Modify | T1 — delete `:137-178`, iterate the array (D1f); gains the accessor skip |
| `build-n4-module-kit/toolbelt/lint-write-path.sh` | Modify | T3 — `:441` marker capture, `:450` DRIFT branch, `:462-464` exit |
| `build-n4-module-kit/toolbelt/lint-guard-pins.sh` | **Create** | T4 — header-mutation → fixture meta-check |
| `build-n4-module-kit/toolbelt/lint-{delays,demand-scope,servlet,structure,wb-threading}.sh` | Modify | T4 — one `# Mutation:` header line each (D4e) |
| `tests/lib/client-root.bash` | **Create** | T2 — one default, four exported names, env override wins |
| `tests/{ext-writable-shape,demand-in-scope,lint-silent-protection,lint-timers,lint-write-path,c9-close,c10-close,c8-close,lint-delays,rc-scan}.bats` | Modify | T2 — 10 sites (D3c); LD5 assertion flip (D3d) |
| `tests/{parser-oneliner,golden-parser,concept-drift,lint-guard-pins}.bats` + `tests/fixtures/**` | **Create** | RED suites (cherry-picked, never re-authored — K13) |
| `tests/c11-close.bats` | **Create** | T5 close gate, `C11_CLOSE=1` |
| `build-n4-module-kit/{BUILD-LOOP.md, skill/SKILL.md}` | Modify | K19 routing (D4h) — fragment-merge |
| `build-n4-module-kit/METHODOLOGY.md` | Modify | K24 fold: the `# Mutation:` grammar becomes doctrine |
| `build-n4-module-kit/{retros/INDEX.md, BUILD-STATE.md}` | Modify | one retro per push — fragment-merge |
| `VERSION`, `CHANGELOG.md` | Modify | `0.21.0` → `0.22.0` (repo root, `c10-close.bats:39`) |

---

## PR matrix, gates, and the OBSERVED-flip requirement

Every rule-changing PR records the flip **verbatim (RED output, then GREEN output)** and **names the fixture**
(K24(7)). A "would flip" claim is not evidence and does not pass the gate. QA confirms each flip independently.

| # | Branch | RED (cherry-pick, never merge — ff-only kit, K13) | Est. Δ | Gate |
|---|---|---|---|---|
| **PR1** | `feat/c11-shared-method-boundary` | `qa/c11-parser-oneliner` `d88af78` + `qa/c11-golden-parser` `ed2088f` | **~663, `size:exception` ceiling 700** | 7 golden cases green on all three lints in one test each (D1i); I1-I5 OBSERVED flips naming S21-misparse / G-samemethod / C11-tl-oneliner / C11-sp-oneliner / C11-g1-setter (D1h); 3×3 baselines byte-identical (D1j); `:303-307` **replaced** with D1g; three existing lint suites unchanged; `kit-links.bats`; `shellcheck` 0; 0 trailers |
| **PR2** | `feat/c11-concept-row-drift` | `qa/c11-concept-drift` (unblessed — PR does not open until blessed) | ~120 | truth table D2b exact; 1 DRIFT exit 0; `--strict` → 1; true concept silent; HTML-comment decoy → no DRIFT; STALE rows unchanged; uncovered FAIL exit 1 **identical with and without `--strict`**; exit 3 preserved; real tree `00e7118` → 0 DRIFT; OBSERVED flip D2f |
| **PR3** | `feat/c11-client-root-lib` | `qa/c11-client-root` `54078f6` | ~150 | 10 → 0 absolute literals outside the lib; full suite green with all four vars **unset**; one override pin per variable; **LD5 flips to clean exit 0** with the two-tree evidence in the commit body (D3d); `LD1`/`LD3`/`LD6`/`LD11-misguard` stay green; SC1-smoke root-only; **RC8 stays 1 FAIL**; RK5 audit list in the PR body; no toolbelt script touched |
| **PR4** | `feat/c11-lint-guard-pins` | `qa/c11-guard-pins` (unblessed) | ~300 | GP-pos/nomut/neg/usage/prune/decoy (D4g); **GP-real asserts MATCH count = 10 + one exact MATCH row + WARN = 0** (D4f); D4e retrofit complete with every id verified; scope rule D4b stated in the header; D9b; K19 rows in both files; `kit-links.bats`; `shellcheck` 0 |
| **PR5** | `chore/c11-close` | `qa/c11-close-checklist` (skeleton) | ~180 | `C11_CLOSE=1` green; BASE `dab0807`; tool-pins include the two new files; both sweeps 0; INDEX pending 0; `VERSION` `0.22.0` + CHANGELOG; `shellcheck` 0; **no attribution trailer in the whole PR range** (K11) |

**Order** T1 → T3 → T2 → T4 → close: T1 is the keystone; T3 touches a toolbelt script and T2 touches only `tests/`, so
ordered they cannot conflict; T4 **measures the headers the first three PRs leave behind** and therefore must be last
of the code slices. PR5 opens only after PR1-PR4 merge. Each branch rebases onto `main` before the QA ping; the lead
verifies `git log -1` equals the blessed tip before settling the ledger.

---

## Worktree / branch map (K12 — a worker writes ONLY inside its own worktree)

| Worktree | Branch | Owns |
|---|---|---|
| `niagara-tools-worktrees/c11-parser` | `feat/c11-shared-method-boundary` | `toolbelt/lib/`, the three lints, golden + one-liner fixtures |
| `niagara-tools-worktrees/c11-drift` | `feat/c11-concept-row-drift` | `lint-write-path.sh`, `tests/concept-drift.bats` |
| `niagara-tools-worktrees/c11-clientroot` | `feat/c11-client-root-lib` | `tests/lib/`, the 10 bats sites |
| `niagara-tools-worktrees/c11-guardpins` | `feat/c11-lint-guard-pins` | `lint-guard-pins.sh`, the D4e retrofit, `tests/lint-guard-pins.bats` |
| `niagara-tools-worktrees/c11-close` | `chore/c11-close` | `tests/c11-close.bats`, `VERSION`, `CHANGELOG.md`, retros |

`tasks.md` is ticked **inside the worker's own worktree**, never in the main checkout. A worker that reads another
session's worktree ships whatever that session happened to have checked out (the C10 design's named trap) — **the RED
is the tip on `origin`**: `git show origin/qa/c11-<x>:tests/<file>`. Nobody reads or deletes another worktree.

**Always-conflict files — fragment-merge, never overwrite.** `BUILD-LOOP.md`, `skill/SKILL.md`, `retros/INDEX.md`,
`BUILD-STATE.md`: **append**, keep both rows, dedupe by script name, resolve by union. A rebase can silently drop a
`BUILD-STATE` line — the pre-push hook catches it, and the sweep is re-run after every rebase.

---

## Testing Strategy

| Layer | What | Approach |
|---|---|---|
| Unit (golden) | the three lints agree on one parser | `tests/golden-parser.bats`, 7 cases, **all three lints per test** (D1i) |
| Unit (invariant) | I1-I5 each bite | one OBSERVED mutation per invariant, each naming its fixture (D1h) |
| Unit (verdict) | DRIFT truth table, exit expression | `tests/concept-drift.bats` — pos / true-concept / decoy / STALE-unchanged / FAIL-unchanged / exit 3 |
| Unit (meta) | T4 grammar | `tests/lint-guard-pins.bats` GP-pos/nomut/neg/usage/prune/decoy |
| Integration (real tree) | no verdict moved | 3 lints × 3 modules on `main-ff1b659`, byte-identical before/after (D1j) |
| Integration (real tree) | T3 / T4 real state | 0 DRIFT at client `00e7118`; **MATCH count = 10, WARN = 0** over the kit (D4f) |
| Regression | the suite does not depend on a machine | full `bats tests/` green with `CLIENT_READ_ROOT`, `C9_CLIENT_ROOT`, `C9_CLIENT_REPO`, `C8_CLIENT_REPO` **all unset** |
| Close | campaign gate | `C11_CLOSE=1 bats tests/c11-close.bats` |

Every new fixture strips `//` and `/* */` before any identifier or WARN-string assertion and carries a **comment-only
decoy** that must NOT satisfy the pin (SC-10). A smoke that cannot run is a **BLOCKER**, never a silent pass.

---

## Threat Matrix

The design adds shell scripts and subprocess invocations but no routing, VCS/PR automation, executable-file
classification, or network boundary.

| Row | Applicability | Expected behaviour / RED |
|---|---|---|
| Shell command construction | **Applicable** — `mb_parse` output feeds `printf` rows; T4 builds paths from `find` results | All variables quoted; no `eval`; no `$(…)` over untrusted content. `shellcheck 0.10.0` clean is a merge gate on every slice |
| Subprocess / `awk -f` | **Applicable** — the fragment materializes to `$_TMP/method-boundary.awk` | `_TMP=$(mktemp -d)` + `trap 'rm -rf "$_TMP"' EXIT` already exists (`lint-silent-protection.sh:28-29`); the fragment file inherits that trap. No new temp path is introduced anywhere else |
| Path traversal / hostile filenames | **Applicable** — T4 and all lints scan discovered files | `find … -print` + `while IFS= read -r`, dot-dirs pruned (D9b). Pinned by `GP-prune` and the existing `TC-D`/`EW8`/`SP7`/`WP-prune` prune fixtures |
| Executable-file classification | N/A | No slice marks, executes, or classifies produced files |
| VCS / PR automation | N/A | Toolbelt is VCS-free by design; `kit-links.bats` L2 enforces it |
| Process integration / network | N/A | No station write, no oBIX call, no jar deploy in any C11 slice |
| Untrusted input | N/A | Inputs are the repo's own Java/bats/markdown; there is no external submitter |

---

## Migration / Rollout

No data migration, no feature flag. **`DRIFT` and the T4 WARNs are advisory by default (exit 0)**, so neither can break
an existing CI invocation; both promote only under the caller's explicit `--strict`. The T1 cut changes **no real-tree
verdict** on the blessed tree (D1j) and therefore needs no coordination with the client repo or the station. Rollback
per proposal §8; PR1's revert must include the RED cherry-picks in the same commit, otherwise the suite is red against
a green tree.

---

## Risks (design-level, beyond proposal §7)

| # | Risk | Mitigation |
|---|---|---|
| DR1 | **PR1 measures above the 700 ceiling.** The estimate is 663 with a ±10% band | The lead re-requests the exception with the **measured** number before opening the PR. The parser is never split and the fixtures never leave PR1 — both break the properties the exception was granted for |
| DR2 | **`"$MB_AWK"'…'` concatenation is unfamiliar** and a worker "tidies" it into `$(cat …)` or an unquoted heredoc | The fragment header states the two legal consumption forms verbatim, `shellcheck` directives are attached at each call site, and the golden set fails loudly if the program text is malformed |
| DR3 | **Multi-`-f` ordering.** `-f method-boundary.awk -f main.awk` must put the library **first** | The function-only library has no rules, so order is semantically irrelevant to awk — but the PR pins the order in the code and the golden set covers it |
| DR4 | **`mb_strip` changes silent-protection counts** (B832-G2, `//`-only today at `:314`/`:341`) | The 3×3 baselines are the pin. **A delta is a BLOCKER, not a note.** No vacuous fixture is invented |
| DR5 | **The accessor skip swallows a real one-line `isDirty()` that schedules** | I5's predicate is `(one-liner) ∧ (get\|set\|is prefix)`, not the prefix alone; C11-g1-setter's OBSERVED flip is mandatory |
| DR6 | **T4's retrofit descriptions become prose nobody checks** | The lint checks the **id**, not the description; the description is documentation. GP-real's exact-row assertion keeps the id honest |
| DR7 | **Two unblessed REDs** (`qa/c11-concept-drift`, `qa/c11-guard-pins`) | RED-first is a hard precondition: PR2 and PR4 do not open until blessed. An unblessed slice rolls to C12 **without holding the `0.22.0` bump** |
| DR8 | **LD5 / LD10 duplication** after the retarget | Decided in D3e: both kept, LD10 stays env-gated on `C8_CRP_FIXED`; the reason is written into the bats header so the next author does not "clean up" the historical pin |
| DR9 | **`c8-close.bats:107` default silently retargets** an eight-campaign-old close gate to a newer tree | The change is explicit in the PR body, `C8_CLIENT_REPO` still overrides, and SC1-smoke's assertion (`status 1 || 0`) was already tree-agnostic — verified at `c8-close.bats:105-114` |

---

## Open Questions

- [ ] **(blocking PR2/PR4 only)** `qa/c11-concept-drift` and `qa/c11-guard-pins` tips are not yet blessed. QA freezes
      `qa/c11-guard-pins` against **D4c's exact ERE**; if QA's prototype `d6b840d` diverges from that grammar, the
      **RED wins** and this design section is amended, not the RED.
- [ ] **(non-blocking)** Whether `# Mutation:` should also become mandatory for `verify-module.sh` / `slot-coverage.sh`
      (the 3 legacy prose lines). Recorded as a **C12 seed**, deliberately out of T4's scope (D4b).
- [ ] **(non-blocking)** B832-G2 still has no biting fixture. Carried as a header NOTE + the 3×3 baselines; a
      fixture is owed to C12 if any real-tree delta ever appears.

---

## Key Learnings

1. Three copies of one parser were not three styles but three awk invocation mechanisms, and the mechanism dictated
   that the shared fragment be function-only text in a shell variable.
2. PEAK depth fixes the one-liner false negative only because the close test runs in the same loop iteration as the
   open test, which is exactly the runaway span the old NET comment feared.
3. The kit has zero `# Mutation:` header lines today, so a guard-pin lint that only checks declared mutations would
   report a vacuous zero and prove nothing.
4. A real-tree smoke must pin the tree's current clean state; LD5 asserted a ColdRoomPan defect and went green for the
   wrong reason once that defect was fixed.
5. `$KIT` is a writable-target seam in this kit, so a lint that resolved its own parser through it would load a
   different parser than the script the user invoked.
