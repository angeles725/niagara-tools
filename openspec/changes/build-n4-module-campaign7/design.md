# Design: build-n4-module-campaign7

**Phase**: design · **Source**: `v0.17.0` (main `c136e3b`) · **Target**: `v0.18.0`
**Inputs**: `proposal.md`, `explore.md`, `spec.md`, `report-module-contract.md`, B794/B795/B796/B797/B798/B799.
**QA REDs bound**: `qa/c7-scaffold` `54636ca` · `qa/c7-plano` `c49504f` · `qa/c7-schema-risk` `6d27ff0` ·
`qa/c7-report-module` pending on `report-module-contract.md`.
**Precedence rule used throughout**: where `spec.md` and a landed QA RED disagree on a surface detail,
the executable RED wins (RED-first is the campaign's contract), and the divergence is listed in Open
Questions for a one-line spec correction. Three such divergences exist: D4a, D6a, D7a.

## Technical Approach

Eight chained PRs, fixed merge order, one destination-file group each. Three doc PRs (fold, tool
routing, exemplar) land first because they are rollback-cheap and clear the `--age` debt; four code
PRs add toolbelt scripts that follow the campaign-5/6 script shape verbatim — header usage block,
`set -u` (never `set -e`), `row()` printf, typed exits, VCS-free (`kit-links.bats` L2),
`shellcheck 0.10.0` clean, no `$HOME`. New behaviour is table-driven or fixture-driven so the
*data* (B795 CSV, MinimalPan tree) is the contract and the script stays thin.

## Architecture Decisions

### D1 — scaffold emission: fixture-driven copy+rename, not template heredocs

| Option | Tradeoff | Decision |
|---|---|---|
| Template strings (B794 prototype, 12 heredocs) | Skeleton drifts from any built module; ~500 lines of quoted Java inside bash; TC3 has nothing to diff against | Rejected |
| **Bundled fixture + copy/rename/substitute** | Fixture is the single source of truth, reviewable as Java/Kotlin, and TC3 byte-equality becomes a real oracle; script drops to ~150 lines | **Chosen** |

Rationale: the B794 round-trip proved *a tree*, not *a generator*. Shipping the proven tree and
substituting into it keeps the proof attached to the artifact. Skeleton path is resolved as
`"${BASH_SOURCE[0]%/*}/../fixtures/MinimalPan"` — never `$HOME` (K8, TC-K8).

### D2 — emitted project root is `<out-dir>/<ModuleName>`

The prototype emitted *into* `out-dir`. QA RED TC3 diffs `$OUT/MinimalPan` against
`fixtures/MinimalPan`, so `out-dir` is a container and the module project root is created inside it.
Existing destination → exit 3 (never overwrite). Consequence for QA: `qa/c7-scaffold` TC4's
`lint-timers` path must be rebased to `$OUT/MinimalPan/MinimalPan/MinimalPan-rt/src` (the proven
Niagara layout keeps the `<Module>/<Module>-rt/` nesting that `findProjects()` scans).

### D3 — the fixture is the PRE-slotomatic source tree

The built MinimalPan carries a slotomatic AUTO region whose header embeds a name-derived hash
(`$com.angeles.MinimalPan.BMinimalPan(3439825203)1.0$`). Copying that region into a *renamed* module
emits a wrong hash. Alternatives: keep the region (rejected — provably wrong after substitution) vs
strip it (chosen — B793 correction C3 says scaffolds emit pre-slotomatic state; slotomatic
regenerates it at build). TC3 therefore pins the pre-slotomatic tree; TC4 (local) proves the
regeneration still builds gate-green. Two further fixture sanitations: `gradle.properties` loses the
operator's `niagara_home=C:\Honeywell\...` / `niagara_user_home=...` (machine coupling, K8) and keeps
only the commented placeholder; `module.lexicon` display values are normalised to English (artifact
language contract; lexicon values are display-only and build-neutral).

### D4 — schema-risk: name-keyed two-snapshot slot diff, fail-safe subtype resolution

Bound to B799 rules. Snapshot dir = `{module-include.xml, <pkg-path>/*.java}` (the B799 fixture
shape). Pipeline:

    before/ ──┐                                     ┌─ per-slot change_kind ─┐
              ├─ normalise ─→ records ─→ pair by ───┤                        ├─ CSV lookup ─→ verdict
    after/  ──┘   (awk)     type|kind|name|         └─ type-level (include)  ┘        worst cell
                            type=|idx|flags|default|facets

- Annotation blocks are joined until parens balance, so one-line (fixture) and multi-line (real
  ColdRoomPan) `@NiagaraProperty/@NiagaraAction/@NiagaraTopic` parse identically.
- Slots key by **name** (B754 r2: `.bog` binds `byName`, index-free). Index is recorded only to
  detect `reorder_slot`.
- **Rename heuristic**: within one type, exactly one removal and exactly one addition of the same
  slot kind and the same `type=` → `rename_slot`. Ambiguous (≥2×≥2, or types differ) → not paired.
- **Documented limits** (stated in the script header and `--help`):
  - **L1** multiple simultaneous renames in one type are not paired; they surface as separate
    `remove_slot_unknown` + `add_slot` rows. Verdict is unchanged (both LOSSY-or-worse).
  - **L2** the parser cannot prove a `B*`/`module:Type` is BSimple vs BComponent, so
    `remove_slot_complex` (the only SAFE removal) and `retype_complex` (LOSSY) are **unreachable by
    design**: every unresolved removal is `remove_slot_unknown` (LOSSY) and every unresolved retype
    is `retype_unknown` (OUTAGE). Never downgrade on uncertainty (B795 §795.2).
  - **L3** anything unparseable (unbalanced annotation, `@NiagaraType` class with no readable body,
    slot-kind swap property↔action as in the `unknown_kind` fixture) emits `UNKNOWN` → OUTAGE. No
    silent skips.
- Only B795 §795.4 vocabulary is ever emitted; module verdict = worst cell (OUTAGE > LOSSY > SAFE).

**D4a — output format binds to `qa/c7-schema-risk` `6d27ff0`, not to `spec.md`.** Three formats are in
circulation: the RED asserts the substring `verdict=<V>`, `spec.md` §PR5 says `VERDICT: <V>`, and the
B799 `expected.txt` files open with `verdict: <V>`. The script emits **`verdict=<V>`** (the only
executable pin). Consequence: `expected.txt` is used as an *oracle* for the verdict token and the
change_kind set, never as a byte-golden — its `b795_row:`/`note:` lines are B799 human commentary and
pinning that prose would add brittleness without bite. Row shape also follows the RED header:
`<verdict>  <change_kind>  <slot>: <detail (B795 row)>`. SR5 additionally pins the literal label
`rename` and SR6 the literal `UNKNOWN`, so those two tokens are contract, not cosmetics.

**Exit codes** are `spec.md`'s five-value scheme (0 SAFE · 1 LOSSY · 2 OUTAGE · 3 usage · 4 env),
which the RED's SR8 (`missing args → 3`) confirms. Usage/env sit **above** the verdict range on
purpose: a caller testing `exit 2` for OUTAGE in a deploy script must never receive a bad invocation
masquerading as OUTAGE. This is a deliberate, documented deviation from the kit's usual
`2=usage / 3=env` convention and is restated in the script header.

### D5 — the CSV is embedded verbatim and mechanically guarded

`CSV_TABLE=$(cat <<'CSV' … CSV)` quoted-heredoc, byte-identical to B795 §795.4. A bats case
(`SR-CSV`) diffs the extracted block against `tests/fixtures/schema-risk/b795-795.4.csv`, so
"verbatim" is enforced by a test, not by review. Lookup is `awk -F,` on `$1==kind`; a miss falls back
to the `UNKNOWN` row. Alternative (a sourced `.csv` data file) rejected: the script must run copied
onto a station with no sibling files.

### D6 — `--plano`: an explicit-file mode on verify-module.sh, integer cross-multiplication only

Bound to B797 §797.2 and `qa/c7-plano` PL1-PL4. Dispatched **before** the flag loop (same pattern as
the existing `coverage` subcommand) so `--plano` never mixes with the jar loop.

| Symbol | Source | Extraction |
|---|---|---|
| `Rc` | `const IMG_W = a, IMG_H = b;` | `grep -oE` |
| `Rv` | zones `<svg … viewBox="0 0 vbW vbH">` | first viewBox; ≥2 distinct → FAIL (ambiguous) |
| `Ri` | `#planoImg` `data:image/png;base64,…` | first 64 b64 chars → `base64 -d` → `od -An -tu1 -N24`; verify the 8-byte PNG signature + `IHDR`, width = bytes 17-20 BE, height = 21-24 BE |
| `A` | every `aspect-ratio:` value | `n/m` or decimal (`d.dd` → `ddd/100`); `auto` exempt; anything else → FAIL, never silently exempt |

PASS iff `Rc == Rv == Ri` **and** every numeric `r == Rc`, compared as `w1*h2 == w2*h1`. Never floats
(shell has none; `awk` floats would make PL2's 1247/771-vs-1248/891 a tolerance argument). FAIL row
names the disagreeing value with its `grep -n` line. New dependency `base64` (coreutils) is declared
in the header and guarded with `command -v base64 || exit 3`; alternative pure-awk base64 decoding
rejected as ~30 fragile lines for a binary present everywhere the kit already runs.

**D6a — operand is a path, accepting both the RED's HTML form and the spec's jar form.**
`qa/c7-plano` pins `--plano <index.html>`; `spec.md` §PR6 says `--plano <jar>`;
`report-module-contract.md` invokes it when an `index.html` exists under `<artifact>/src/rc/`. The
mode therefore accepts **`--plano <index.html|jar>`**: an `.html` operand is read directly (the RED
path), a `.jar` operand has its `rc/**/index.html` extracted with `unzip -p` to a temp file and the
identical check runs on it. One code path, both contracts, no ambiguity.

### D7 — report aggregation: per-artifact loop, severity map, exit keyed on FAIL rows + member exits

Bound to `report-module-contract.md` (B798 baseline, `niagara-research d267d4e5a` — **B798 has
landed, so PR8 is no longer research-gated**; only R7's PR6 clause survives). `report-module.sh
<module-root> [--target-version v]` discovers every profile artifact `<MOD>-{rt,ux,wb,…}` under the
root and, per artifact, invokes `verify-module.sh --src` (SKIP with no built jar), `slot-coverage.sh`
parse, dup-lexicon-keys, `lint-timers.sh <artifact>/src`, and `--plano` only when
`<artifact>/src/rc/index.html` exists. It invents **no new checks**.

Severity map (from the contract): verify-module FAIL → FAIL · lint-timers FAIL → FAIL · dup-keys > 0
→ FAIL · `--plano` mismatch → FAIL · slot-coverage < 100% → WARN · empty palette → WARN (SKIP when
the artifact is scaffold-only / has no jar).

**D7a — the exit is keyed on the aggregated FAIL count, with member exit codes as a second, ORed
signal.** The contract's expected ColdRoomPan output is the authority and it *requires* row-level
aggregation: `slot-coverage.sh` exits 0 while printing a WARN, so an exit-code-only design cannot
produce `7 PASS · 1 FAIL · 1 WARN`. Row parsing is therefore the primary signal; a member exiting 2
or 3 additionally emits an `ERROR` row and forces exit 3, so a member that dies without printing rows
can never be read as clean. Alternative (exit codes only, as first drafted) rejected for the WARN gap
above; alternative (rows only) rejected because a crashed member prints nothing. `spec.md`'s
`report-module.sh <jar> [--src <module-dir>]` signature is superseded by the contract's
`<module-root>` form, which is what the pending `qa/c7-report-module` will pin.

### D8 — `types/logic.md` split boundary is line 91 (`## Author-side SPIs`), not 80

Explore said 80-136; the real audience boundary is line 91. Lines 80-90
(`## kitControl patterns`) are control authoring proven from exemplars and **stay** in `logic.md`.
Lines 91-136 (Author-side SPIs, point extension, containers, grouping, query surface, templates,
background jobs, watchdogs, action protection, minimal module) move to `types/logic-authoring.md`.
Each file cites the other so `kit-links.bats` L1 keeps both reachable.

### D9 — TC4 SKIPs; a SKIP is never a PASS

`[ -n "${NIAGARA_HOME:-}" ] || skip "…"`. CI has no JDK 8 + `niagara_home`, so TC4 SKIPs there and QA
runs it locally before bless with the exact command output pasted into the PR body. The
`run-pure-test.bats` CI guard (assert **zero** SKIPs) is deliberately **not** applied to
`scaffold-module.bats`.

### D10 — the PR2 routing guard is automated in PR7

PR2's binding file set is `BUILD-LOOP.md` + launcher `SKILL.md` only, so its guard is a manual grep
recorded in the PR body. The durable automated guards land in PR7, which already owns
`tests/kit-links.bats`: **L4** every `toolbelt/*.sh` is named in `BUILD-LOOP.md`; **L5** the same for
the launcher `SKILL.md` (skipped when not installed, mirroring L3); **L6** both `types/logic.md` and
`types/logic-authoring.md` exist and cite each other. Tradeoff: five PRs of unautomated coverage —
raised in Open Questions.

## CLI Contracts

| Script | Usage | Exits |
|---|---|---|
| `scaffold-module.sh` | `<ModuleName> <out-dir> [--vendor v] [--target-version x.y] [--plugin-version v]` | `0` ok · `2` usage / name not an uppercase-initial alphanumeric identifier · `3` env (fixture missing, out-dir not creatable, `<out-dir>/<ModuleName>` exists) |
| `schema-risk.sh` | `<before-dir> <after-dir>` | `0` SAFE · `1` LOSSY · `2` OUTAGE · `3` usage · `4` env (unreadable snapshot / missing tool) |
| `verify-module.sh --plano` | `--plano <index.html\|jar>` | `0` PASS · `1` FAIL · `2` usage · `3` env (`base64`/`od`/`unzip` missing, file unreadable) |
| `report-module.sh` | `<module-root> [--target-version x.y]` | `0` CLEAN (zero FAIL) · `1` any FAIL · `3` env (no JDK 8 / not a `niagara_home` / member env fault) |

Row formats (kit `row()` convention; the schema/report shapes are QA-pinned, not free choice):

    scaffold : (silent; one summary line "scaffold-module: emitted <Mod> -> <path>")
    schema   : SAFE|LOSSY|OUTAGE  <change_kind>  <Type>.<slot>: <detail (B795 evidence + note)>
               verdict=<WORST>
    plano    : PASS|FAIL  plano  <path>  <detail incl. the disagreeing value + its line>
    report   : <artifact>  PASS|FAIL|WARN|SKIP  <check>  <detail>
               report-module: <N> artifacts · <p> PASS · <f> FAIL · <w> WARN · <s> SKIP  ->  CLEAN|ISSUES

## Fixtures Layout

    build-n4-module-kit/fixtures/MinimalPan/          ← PR4, its own commit
      gradlew · gradlew.bat · gradle/wrapper/{gradle-wrapper.jar,.properties}   [copied verbatim]
      settings.gradle.kts · build.gradle.kts · gradle.properties                [templated]
      MinimalPan/MinimalPan-rt/MinimalPan-rt.gradle.kts                         [templated + renamed]
      MinimalPan/MinimalPan-rt/{module-include.xml,module.lexicon,
                                module.palette,module-permissions.xml}          [templated]
      MinimalPan/MinimalPan-rt/src/com/angeles/MinimalPan/{BMinimalPan,MinimalPanLogic}.java
      MinimalPan/MinimalPan-rt/srcTest/test/com/angeles/MinimalPan/MinimalPanLogicTest.java
    EXCLUDED: build/**, .gradle/**, any *.jar except gradle-wrapper.jar

Substitution table (ordered; templated files only, `gradle-wrapper.jar`/`gradlew*` are byte-copies):

| Token | Replacement | Applies to |
|---|---|---|
| `MinimalPan` | `<ModuleName>` | content + path segments + file names |
| `minimalPan` | lowerCamel `<moduleName>` | content (0 sites today; retained for var-name convention) |
| `Angeles` | `--vendor` | content (copyright, `defaultVendor`) |
| `angeles` | lowercased vendor | content + `src/com/<vendor>/` path |
| `mp=` / `mp:` in `module.palette` | derived symbol (uppercase initials, lowercased) | anchored to the palette `m=`/`t=` attributes only |
| `7.6.17` | `--plugin-version` | `settings.gradle.kts` |
| `4.14` | `--target-version` | `settings.gradle.kts` comment, `gradle.properties` |

Residue guard: emitting a differently-named module must leave **zero** `MinimalPan|Angeles|angeles`
occurrences in the tree (test TC5).

    tests/fixtures/schema-risk/{add_slot,remove_slot,retype_simple,reorder,rename_slot,
                               unknown_kind,mixed}/{before,after,expected.txt}   ← QA copies B799
    tests/fixtures/schema-risk/b795-795.4.csv                                    ← CSV verbatim oracle
    tests/fixtures/plano/ok/index.html                                           ← inline 2×3 base64 PNG

## Test / Mutation Matrix

| PR | Case | Assertion | Named mutation |
|---|---|---|---|
| 4 | TC1 | no args → 2 + `usage` on stderr | — |
| 4 | TC2 | `1bad` → 2 | drop the name validation → TC2 exits 0 |
| 4 | TC3 | emitted tree `diff -r` byte-equals the fixture | drop `<MOD>-rt.gradle.kts`; emit `module.xml` instead of `module-include.xml` |
| 4 | TC-K8 | identical under `HOME=/nonexistent` | reintroduce `MINIMOD_ROOT="$HOME/…"` → TC-K8 exits 3 |
| 4 | TC5 (new) | `ScaffoldPan --vendor Acme` leaves no `MinimalPan\|angeles` residue | drop the vendor substitution |
| 4 | TC6 (new) | existing `<out>/<Mod>` → 3, destination untouched | — |
| 4 | TC4 | local round-trip build → gate ALL PASS → lint-timers PASS; SKIP without `NIAGARA_HOME` | drop the `stopped()` cancel → lint-timers FAIL |
| 5 | SR1-SR7 (`6d27ff0` verbatim) | add 0/`verdict=SAFE` · remove 1/LOSSY · retype_simple 2/OUTAGE · reorder 0/SAFE · rename 1/LOSSY **and label `rename`** · unknown_kind 2/OUTAGE **naming `UNKNOWN`** · mixed 2/OUTAGE | worst-cell → first-cell (`mixed` reads SAFE) · drop the UNKNOWN fail-safe · retype detection off · rename detection off (caught only by SR5's label pin) |
| 5 | SR8 | no args → exit 3 (usage, above the verdict range) | map usage to 2 → SR8 reads as OUTAGE |
| 5 | SR9 (new) | unreadable `module-include.xml` → exit 4 | — |
| 5 | SR-CSV (new) | embedded heredoc byte-equals `tests/fixtures/schema-risk/b795-795.4.csv` | edit one CSV cell → SR-CSV fails |
| 6 | PL1-PL4 (`c49504f` verbatim) | 2×3 PNG; PL2 stale 1247/771 named in the FAIL; PL3 `auto` exempt; PL4 viewBox `4 3` ≠ image | count-only / intra-`aspect-ratio` comparison → PL2 + PL4 lose their FAIL |
| 6 | PL5 (new) | unparseable `aspect-ratio: var(--x)` → FAIL, not silently exempt | treat unparseable as exempt → PL5 passes |
| 6 | PL6 (new) | `.jar` operand: `rc/**/index.html` extracted, same verdict as the HTML form | — |
| 7 | L4/L5/L6 | routing + split link integrity | break one moved link → kit-links FAIL |
| 8 | RM1 | clean rt-only fixture tree → exit 0 + `report-module: CLEAN` summary | aggregation drops sub-tool rows → summary still CLEAN but member FAIL lost (see RM2) |
| 8 | RM2 | leak fixture (clean + a `BLeak` timer-leak class) → the lint-timers FAIL row surfaces, exit 1 | aggregation drops sub-tool FAILs → RM2 exits 0 |
| 8 | RM3 | rt-only tree → NO `--plano` row (gated on `<mod>-ux/src/rc/index.html`), anchored on the tool having run (exit 0 + summary) | always-run plano → a plano row appears on an rt-only tree |
| 8 | (apply extras beyond the RED) | slot-coverage <100 % stays WARN and does not flip the exit (mutation: promote WARN→FAIL); a member exiting 3 emits an `ERROR` row and the report exits 3 (mutation: swallow member env faults) | the ColdRoomPan-rt B798 report (7 PASS · 1 FAIL · 1 WARN → ISSUES, exit 1) is LOCAL bless evidence, not a CI pin |

## File Changes

| PR | Files | Action | Est. authored |
|---|---|---|---|
| 1 | `build-n4-module-kit/METHODOLOGY.md` (K11-K14 after K10), `BUILD-LOOP.md` §7, `CONTRIBUTING.md` (SDD ledger), `retros/INDEX.md` + 9 markers, kit `BUILD-STATE.md` | Modify | ~40 |
| 2 | `BUILD-LOOP.md` §0.b/§5/§7, `build-n4-module-kit/skill/SKILL.md` (tracked launcher; §References + step 5; orchestrator re-runs `scripts/install-skill.sh --force` after merge), `BUILD-STATE.md` | Modify | ~20 |
| 3 | `types/dashboard.md` (B796 `-ux` exemplar, 4/5 gates, gate 4 REQUIRED-but-absent → #49), `BUILD-STATE.md` | Modify | ~25 |
| 4 | `fixtures/MinimalPan/**` (commit 1), `toolbelt/scaffold-module.sh` (commit 2), `tests/scaffold-module.bats` (commit 3), `BUILD-STATE.md` | Create | ~700-880 `size:exception` |
| 5 | `toolbelt/schema-risk.sh`, `tests/schema-risk.bats`, `tests/fixtures/schema-risk/**`, `ci.yml`, `BUILD-STATE.md` | Create/Modify | ~200 |
| 6 | `toolbelt/verify-module.sh` (+`--plano`), `tests/plano-check.bats`, `tests/fixtures/plano/**`, `ci.yml`, `BUILD-STATE.md` | Modify/Create | ~140 |
| 7 | `types/logic.md` (91-136 removed), `types/logic-authoring.md` (new), launcher `SKILL.md` decision table, `BUILD-LOOP.md` §2, `tests/kit-links.bats` (L4/L5/L6), `BUILD-STATE.md` | Modify/Create | ~70 |
| 8 | `toolbelt/report-module.sh`, `tests/report-module.bats`, `tests/fixtures/report/**`, `ci.yml`, `BUILD-STATE.md` | Create/Modify | ~50 |
| last | `VERSION`, `CHANGELOG.md` | Modify | ~15 |

Every kit-changing PR also adds its retro + `retros/INDEX.md` row + kit `BUILD-STATE.md`
self-envelope in the same push range (campaign-6 close lesson 1).

## CI Changes (`.github/workflows/ci.yml`)

Appended after the existing `lint-timers` step, each mirroring its shape (no `|| true`):

| PR | Step | Command | Expect |
|---|---|---|---|
| 5 | schema-risk over fixtures | loop the 7 pairs; assert the mapped exit (0/1/2) and that `verdict=<V>` matches the `expected.txt` verdict token | exit 0 |
| 6 | plano | `verify-module.sh --plano tests/fixtures/plano/ok/index.html` | exit 0 |
| 8 | report-module (non-strict: members invoked without `--strict`) | `report-module.sh tests/fixtures/report/<mod>` | exit 0 |

`shellcheck` and `bats tests/*.bats` already glob the new files — no step edits needed there.

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| Documentation-like paths (executable emitted files, binary fixture) | **Applicable** — scaffold copies `gradlew` (+`chmod +x`) and `gradle-wrapper.jar`; substitution must never rewrite binary bytes | Explicit copied-vs-templated file classes (D3 table); `sed` runs only on the templated set | TC3 byte-equality covers both classes (a sed'd jar breaks the diff) |
| Untrusted content decode | **Applicable** — `--plano` decodes a base64 data URI from an arbitrary HTML file, and a `.jar` operand is `unzip -p`-extracted | Bounded read: only the first 64 b64 chars are decoded, `od -N24`, PNG signature verified before any width/height use; `unzip -p` streams a single fixed path (`rc/**/index.html`) to a temp file, never extracts a tree; malformed → FAIL/3, never a crash | PL5 (unparseable value), PL6 (jar operand) + a malformed-signature case |
| Git repository selection | **N/A** — every toolbelt script is VCS-free by design; `kit-links.bats` L2 fails the suite if any script names `git` | — | L2 (existing) |
| Commit state | **N/A** — no script reads or writes an index | — | — |
| Push state | **N/A** — no script pushes | — | — |
| PR commands | **N/A** — chain mechanics are orchestrator-owned; no script composes a PR command | — | — |

## Migration / Rollout

- **Chain**: PR1 targets `main`; every later PR branches only after its predecessor merges, and is
  rebased until the child diff shows only its own work unit (R8).
- **Version**: each kit-changing PR appends its line under a `## [Unreleased]` heading in
  `CHANGELOG.md`; the **last landed PR of the chain** renames it to `## [v0.18.0] - YYYY-MM-DD`,
  adds `### References` (SDD slug + engram IDs) and sets `VERSION` to `0.18.0`, per CONTRIBUTING §5.
  This decouples the bump from PR6 slipping (R7, now narrowed: B798 has landed, so PR8 is
  research-unblocked and only PR6 remains B797-gated) — no reordering is needed if the chain closes
  early.
- **Rollback**: PR5/PR6/PR8 and PR4 are `git revert` (additive new paths; `verify-module.sh` loses
  only `--plano`). PR1/PR2/PR3/PR7 are doc-only reverts; retro files are never deleted
  (propose-never-apply) and INDEX rows return to `pending`. The launcher `SKILL.md` is **outside
  git** — PR2 and PR7 must record its full before/after text in the PR body, and the orchestrator
  re-runs `scripts/install-skill.sh` after each of those merges.

## Lane Map

| Lane | Owns |
|---|---|
| investigador (orchestrator) | SDD chain, merge order, `install-skill.sh` re-run after PR2/PR7, live-doc updates |
| `sdd-apply` | One writer per PR, disjoint file sets, worktree-only writes |
| QA | RED branches `qa/c7-scaffold` `54636ca`, `qa/c7-plano` `c49504f`, `qa/c7-schema-risk` `6d27ff0` (B799 fixtures already copied into `tests/fixtures/schema-risk/`), `qa/c7-report-module` pending; runs TC4 locally before bless |
| investigador1 | Fidelity reads of PR1/PR3/PR7 |
| companero | B797 (`--plano`, PR6's only remaining gate); B798/B799 landed; second read of the scaffold and plano fixtures |

## Open Questions

- D2 (TC4 lint-timers path): CLOSED — QA rebased qa/c7-scaffold to 54636ca (ROOT=$OUT/MinimalPan, RT=$ROOT/MinimalPan/MinimalPan-rt, lint-timers on $RT/src).

- [ ] **D4a spec correction**: `spec.md` §PR5 scenarios say `VERDICT: <V>`; `qa/c7-schema-risk`
      `6d27ff0` asserts `verdict=<V>`. Design binds to the RED. Confirm the one-line spec edit.
- [ ] **D6a spec correction**: `spec.md` §PR6 says `--plano <jar>`; `qa/c7-plano` `c49504f` pins
      `--plano <index.html>`. Design accepts both operands. Confirm.
- [ ] **D7a spec correction**: `spec.md` §PR8 says `report-module.sh <jar> [--src <module-dir>]`;
      `report-module-contract.md` says `<module-root> [--target-version]` with a `WARN` severity and
      exits `0/1/3`. Design binds to the contract. Confirm before `qa/c7-report-module` is authored.
- [ ] D10: accept the five-PR gap where PR2's routing coverage is guarded only by a PR-body grep, or
      relax PR2's binding file set to include `tests/kit-links.bats`? **Recommendation**: relax it —
      the guard is ~10 lines and belongs with the change it protects.
- [ ] D2: `qa/c7-scaffold` TC4's `lint-timers` path needs rebasing to the nested layout. QA rebases,
      or apply lands the corrected path with the RED cited?
- [ ] D3: normalising the fixture `module.lexicon` values to English changes bytes from companero's
      built tree — confirm with companero before PR4's fixture commit.
- [ ] L2 escape hatch (an operator-supplied simple/complex type list for `schema-risk.sh`) is
      deferred to campaign 8; confirm nobody expects `retype_complex` to be reachable in v0.18.0.
