# Design: build-n4-module-campaign6

**Phase**: design · **Source**: `main 48f3736` v0.15.1 · **Target**: v0.16.0
**Inputs**: `proposal.md`, `explore.md`, `harvest.md` (authoritative on conflict), `math-models-mm2-mm3.md` (MM2 implementation contract), QA RED `qa/c6-marker-index-drift` `cb0dd7d`, QA RED `qa/c6-coverage-pct-red` `d7e52a8`
**Store**: hybrid — this file + engram `sdd/build-n4-module-campaign6/design`

---

## 1. Technical Approach

Every mechanism in this campaign is a **pure, path-argument shell predicate over kit text files**, tested with generated fixtures and a named mutation. No new runtime, no new dependency, no VCS inside `toolbelt/`. The campaign closes three enforcement gaps (marker↔INDEX, CI actually running P1-P6, fold-citation audit), folds the 8 pending retros, and adds three advisory scripts — delivered as 6 chained PRs with disjoint file sets and a fixed merge order, plus a PR7 slot for the research lane.

Two invariants bind every slice:

- **`toolbelt/*.sh` may never contain `git` followed by a space or EOL** — `kit-links.bats` L2 greps `(^|[^A-Za-z0-9_./-])git( |$)` across `toolbelt/*.sh` and asserts *no match*, comments included. New toolbelt scripts must say "version control" in prose.
- **`kit-links.bats` L1** resolves every bare `X.md` in kit docs against kit root / `types/` / `retros/` / repo root. Any new `.md` mention in a folded delta must resolve, or L1 goes red.

---

## 2. Architecture Decisions

### D1 — Marker parse: column-0 anchored, first 5 lines, first word (option 2-lite)

**Choice**: inside the existing INDEX loop of `sweep-build-state.sh`, read the retro's marker with

```bash
marker_of() {   # -> the word, or "" when absent
  sed -n '1,5p' "$1" \
    | grep -m1 -E '^<!--[[:space:]]*review-status:' \
    | sed -E 's/^<!--[[:space:]]*review-status:[[:space:]]*//' \
    | awk '{print $1}'
}
m=$(marker_of "$retrodir/$fname")
if [ -n "$m" ]; then
  case "$m" in pending|folded) ;; *) fail "retro marker out of domain: $fname has review-status '$m' (pending|folded)";; esac
  [ "$m" = "$rstatus" ] || fail "retro marker disagrees with INDEX row: $fname marker='$m' INDEX='$rstatus'"
fi
```

**Alternatives considered**: (a) exact-match the whole marker line — dies on the *three* real suffix shapes on disk (`folded v0.2 · 2026-09-01`, `fresh · 2026-09-02`, bare `pending`); (b) whole-file `grep review-status | head -1` — `retros/2026-09-04-kit-continuity-and-retro-gate-campaign.md:40` mentions `review-status` inside a table cell, so an unanchored parse reads prose as data; (c) require a marker on every retro — breaks the ~25 legacy folded files (M3 forbids it).
**Rationale**: first-word parsing is the only rule that sees all three real shapes; the `^<!--` column-0 anchor is the same doctrine S8 already pins for `build-state.v1`; a 5-line window bounds the read. Absence stays tolerated.
**Gap to close with QA**: the RED set has no *prose-mention* case. Recommend **M6**: a retro whose line 1 has no marker but whose line 40 contains `review-status: folded` in a table cell, INDEX `pending` → must exit 0. It bites the anchor+window independently of M1.

### D2 — Coverage% stays on `verify-module.sh coverage`, dispatched before the flag parser

**Choice**: accept QA's pinned surface. Intercept `coverage` as subcommand **before** the `while [ $# -gt 0 ]` loop (today `coverage` falls into `JARS` and dies at `not a jar`). Pure path, stdout, exit 0.
`applicable = P+F+W`, `covered = P`, SKIP excluded, `applicable == 0 → "N/A"`.
**Rounding**: integer tenths in bash, **never `awk`/`printf %.1f`** — `t=$(( (1000*P + A/2) / A ))`, print `$((t/10)).$((t%10))`. Checks: `3 1 0 2 → 750 → 75.0`; `1 2 0 0 → 1001/3=333 → 33.3`; `4 0 1 0 → 800 → 80.0`.
**Alternatives considered**: (a) a separate `toolbelt/gate-coverage.sh` — the counters (`NPASS/NFAIL/NSKIP/NWARN`, `row()` line 47) live in `verify-module.sh`; a second script would have to re-parse the gate's own report; (b) `awk 'printf "%.1f"'` — locale-dependent decimal separator (`33,3` under a comma locale) would silently break the exact-string pins, and the `command -v awk` guard runs *after* the subcommand dispatch.
**Rationale**: QA's surface is correct; only the implementation route needed pinning. **Do not** add the pct to the closing summary line in this campaign — `verify-module.bats:89,127` assert on that line; a consumer change needs its own RED.

### D3 — Fold-audit matches *abbreviated* citation tokens, hyphen-aligned

**Evidence**: the live citation convention is abbreviated, not the date-stripped stem — `[ev: retro 5rooms · …]` cites `2026-09-01-dashboardpan-5rooms.md`; `[ev: retro rt-hardening]` cites `2026-08-31-coldroompan-rt-hardening.md`; `[ev: retro corpus-index]` cites `…-corpus-index-rt-authoring-and-organization-blocks.md`. An exact-stem grep would report ~30 false "uncited" rows and be discarded as noise on day one.
**Choice**:
1. `stem` = filename minus `.md` minus a leading `^[0-9]{4}-[0-9]{2}-[0-9]{2}-`.
2. Harvest tokens from the core corpus: `grep -ohE '\[ev: retro [A-Za-z0-9][A-Za-z0-9._-]*' <corpus> | sed 's/^\[ev: retro //'`.
3. A `folded` row is **cited** iff some token `T` has `${#T} -ge 6` **and** `case "-$stem-" in *"-$T-"*)` — hyphen-segment aligned.
4. Corpus = `<kit>/*.md` + `<kit>/types/*.md`, **excluding `retros/**` and `INDEX.md`**.

**False-positive / false-negative guards**: `retros/` exclusion kills self-citation; the 6-char floor kills the `BUILD-STATE.md:157` template `[ev: retro X`; hyphen alignment kills mid-segment accidents (`ender-doors` must not credit `detail-render-doors`). A token aligning into ≥2 stems emits an advisory `NOTE ambiguous citation token` line and still credits both.
**Alternatives considered**: exact-stem match (rejected: rewrites ~40 live citations for no behavioral gain); a new `cite` column in `INDEX.md` (rejected: schema change to the file `sweep-build-state.sh` and 19 bats cases already pin).
**Rationale**: advisory WARN over silent blindness (kit rule K3), and the audit must be *credible* on the tree it ships against.

### D4 — CI gets junit by pinned direct download, not a gradle build

| Option | Cost / determinism | Verdict |
|---|---|---|
| Pre-fetch via a gradle build | needs a gradle project this repo does not have + a ~100 MB distribution download | Rejected |
| **`curl` the 2 jars into `~/.gradle/caches/…`, sha256-checked** | ~400 KB, no JVM build, `find $HOME/.gradle -name` finds them unchanged | **Chosen** |
| `JUNIT_JAR`/`HAMCREST_JAR` env override in `run-pure-test.sh` | changes a kit toolbelt contract → PR2 stops being a `ci.yml`+`tests/` PR and inherits the pre-push retro gate; also weakens P6 | Rejected (fallback only if Maven Central is unreachable) |

Pinned: `junit:junit:4.13.2` sha256 `8e495b634469d64fb8acfa3495a065cbacc8a0fff55ce1e31007be4c16dc57d3`, `org.hamcrest:hamcrest-core:1.3` sha256 `66fdef91e9739348df7a096aa384a5685f4e875584cce89386a7a47251c4d8e9`. The apply worker MUST verify each against Maven Central's `.sha256` sidecar on first run; **a mismatch is a hard stop**, never a silent re-pin. Target path `~/.gradle/caches/modules-2/files-2.1/junit/junit/4.13.2/`. Also add `actions/setup-java@v4` `temurin` `8` — `javac -source 8 -target 8` on the runner's default JDK is a deprecation-warning path at best.
**Skip becomes a failure in CI**: `tests/run-pure-test.bats` `setup()` keeps `skip` locally but, when `${CI:-}` is set, fails instead. A workflow step then runs `bats --formatter tap tests/run-pure-test.bats` and greps `# skip` to keep "zero SKIPs" visible in the log.

### D5 — Launcher `SKILL.md`: track a canonical copy in-repo (Option A)

**Choice**: add `build-n4-module-kit/skill/SKILL.md` (tracked source of truth) + `scripts/install-skill.sh` that **copies** it into `<home>/.claude/skills/build-n4-module/SKILL.md` with a sha comparison, and `tests/install-skill.bats` proving byte parity.
**Alternatives considered**: (b) keep it external and document the manual rollback — leaves `git revert` unable to restore PR6's edit, leaves a fresh machine with no launcher, and leaves the kit's own continuity doctrine unenforceable on its entry point; (c) symlink into the repo — a moved or absent clone silently breaks the launcher for every session.
**Rationale**: the launcher is the kit's entry point; an entry point outside version control is exactly the drift this campaign exists to remove. A copy (not a symlink) keeps the harness working without the repo; the sha check makes divergence loud. Prior art: the research-sdd installer (`~/investigacion/sdd-investigacion/research-sdd/install/`) — its `--home <dir>` and `--dry-run` flags are the testability idea adopted here; no code is copied.
**Contract**: `install-skill.sh [--home <dir>] [--dry-run] [--force]` · exit `0` installed or already current · `1` installed copy diverged and `--force` absent · `2` usage · `3` environment (target dir not creatable). `--home` defaults to `$HOME` but every test passes `--home "$BATS_TEST_TMPDIR/home"`, so **no test touches the real `$HOME`** and the suite is identical under `HOME=/nonexistent`.
**Scope note**: `kit-links.bats` L1 keeps its unconditional `SKILL.md` skip and L3 keeps its `skip` — teaching L1 to resolve `skill/$ref` is a 1-line follow-up deliberately deferred, because changing an existing test's resolver needs its own mutation proof.

### D6 — `slot-coverage.sh` binds to the MM2 `set_coverage` contract, split pure/parse

**Choice**: implement `math-models-mm2-mm3.md` §MM2 verbatim, exposed as a **pure subcommand QA can pin from the spec's vectors**, with the file-parsing half as a separate surface over the same function:

```
set_coverage(declared, required) -> (pct, missing, extra)
  present = required ∩ declared
  pct     = round(100*|present| / |required|, 1)   |required| > 0
          = "N/A"                                   |required| == 0   (never 100)
  missing = required − declared      # the coverage gap, the footgun
  extra   = declared − required      # dangling declarations: REPORTED, never scored
```

`required` = the authoritative exposed `@NiagaraType`/slot set (from `module-include.xml`); `declared` = what the artifact lists (lexicon `key=`, palette `<p t="mod:Type">`, `<type>`). The denominator is **`|required|`, never `|declared|`**, and `extra` never enters the numerator — those are two of the four mutations the spec requires to flip.
**Alternatives considered**: a single fused parse+score entry point (rejected — MM1's lesson is that a metric buried in a parse pipeline cannot be pinned to exact values); reusing `verify-module.sh coverage` (rejected — different denominator semantics: MM1 scores check outcomes, MM2 scores set membership; overloading one name would make both untestable).
**Rationale**: one shared set-math function serves all three documented silent-deploy footguns (empty palette B5, missing lexicon key T8, dangling `module-include` type B12) and QA can stage RED straight from the spec's pin table. Output is line-oriented and sorted so every field is an exact string pin.

**MM3 is out of campaign 6 core** (its two-snapshot slot-diff parser is the expensive half). Recorded in the PR7/follow-up slot as a candidate `toolbelt/schema-risk.sh` pre-deploy guard against the same spec's classification table and pin vectors. No design work now.

---

## 3. Data Flow

```
INDEX.md row (fname,status) ─┐
                             ├─→ sweep-build-state.sh ─→ exit 0|1  ─→ .githooks/pre-push, ci.yml
retros/<f>.md line 1 marker ─┘        (D1, PR1)

INDEX.md folded rows ─┐
                      ├─→ sweep-fold-audit.sh ─→ WARN lines, exit 0 (1 under --strict)
kit core *.md tokens ─┘        (D3, PR5)          ↑ corpus EXCLUDES retros/ + INDEX.md

module-include.xml ─┐                       verify-module.sh coverage P F W S ─→ pct|N/A
                    ├─→ slot-coverage.sh ─→ pct + WARN            (D2, PR5)
module.lexicon ─────┘        (PR5)
```

---

## 4. Per-PR File Sets, Sequencing, Ownership

Merge order is fixed: **PR1 → PR2 → PR3 → PR4 → PR5 → PR6 → (PR7)**. A child branches **only after its parent merges into `main`**, then ff-only. File sets are disjoint except `retros/INDEX.md`, which each promotion PR edits only in its own rows (sequential ff-only merge prevents conflicts); `METHODOLOGY.md` is touched only by PR3. QA RED branches are cited by tip at design time: the apply worker MUST re-read the branch tip at apply time, never hard-pin the hash. Coverage% rounding is bash integer-tenths per §D2 regardless of the QA commit message wording.

| PR | Branch | Files (exclusive) | Auth. lines | Retros flipped | Gate exit |
|---|---|---|---|---|---|
| PR1 | `feat/c6-marker-index` | `toolbelt/sweep-build-state.sh`, `tests/build-retro-sync.bats` (merge QA `cb0dd7d`), re-stamp `2026-09-02-comppan-fase1-staging.md` + `2026-09-02-dashboardpan-detail-render-doors.md` `fresh`→`folded` | ~60 | 0 | (a) new retro |
| PR2 | `feat/c6-ci-pure-test` | `.github/workflows/ci.yml`, `tests/run-pure-test.bats` | ~25 | 0 | n/a (no kit file) |
| PR3 | `feat/c6-doctrine` | `METHODOLOGY.md`, `CONTRIBUTING.md`, `retros/INDEX.md`, 7 retro markers | ~80 | kit-continuity, run-pure-test-set-e, gate-exit-taxonomy, campaign3, campaign4, campaign5, ci-server-side | (a) |
| PR4 | `feat/c6-types` | `types/logic.md`, `types/dashboard.md`, `build-verify.md`, `SOURCES.md`, `corpus-index.md`, `retros/INDEX.md` (1 row), freeze-stat marker | ~80 | coldroompan-dashboardpan-freeze-stat-leds | (a) |
| PR5a | `feat/c6-tools-audit` | `toolbelt/sweep-fold-audit.sh`, `verify-module.sh` (coverage), `tests/sweep-fold-audit.bats`, `tests/verify-coverage.bats` (QA `d7e52a8`) | ~150 | 0 | (a) |
| PR5b | `feat/c6-tools-env` | `toolbelt/preflight.sh`, `toolbelt/slot-coverage.sh`, `tests/preflight.bats`, `tests/slot-coverage.bats`, `--plano` (stretch) | ~160 | 0 | (a) |
| PR6 | `feat/c6-close` | `build-n4-module-kit/skill/SKILL.md`, `scripts/install-skill.sh`, `tests/install-skill.bats`, `openspec/**` (tracked + slotomatic → `archive/`), `VERSION`, `CHANGELOG.md` | ~130 | 0 | (a) |
| PR7 | `feat/c6-research-fold` | new research retros + their INDEX rows + the core file each cites; **candidate slot** for `toolbelt/schema-risk.sh` (MM3) | tbd | research-lane retros | (a) |

PR5 is **pre-split** into 5a/5b: the single-PR estimate (~290) plus QA's 8 coverage pins lands too close to 400 for one honest slice. `--plano` is the first thing cut if 5b crosses budget.

**Lane map**: `sdd-apply` workers write (one per PR, disjoint sets) · QA stages RED (`cb0dd7d`, `d7e52a8`, plus M6) and verifies each PR before its ff-only merge · the research lanes (investigador1 `module-ux-testing-and-write-surface`, companero `module-authoring-exemplars`) **never write to `niagara-tools`** — their output arrives as retro files consumed by PR7, which reuses PR3/PR4 mechanics verbatim (fold → cite `[ev:]` → flip row → stamp marker → sweep green).

**Coupling that must not be missed**: PR3/PR4 flip INDEX rows to `folded`, and 7 of those retros carry a `pending` marker on line 1. After PR1 the sweep reads markers, so **each flipped row's marker must be re-stamped in the same PR** or `sweep`/M5/CI go red. Freeze-stat has no marker (tolerated) but gets `<!-- review-status: folded -->` for consistency with the marker-is-the-promotable-unit rule.

**Grep-before-fold (R1, rule A8/K6)**: before writing any delta, `rg` the *whole kit* for the rule. A2 (`BDouble.make` + import) is already at `METHODOLOGY.md:9,11` — **flip the marker/row only, fold nothing**. `tasks.md` must carry a grep-before-fold column per delta row.

---

## 5. Script Contracts

| Script | Usage | Exits | Notes |
|---|---|---|---|
| `sweep-fold-audit.sh` | `[--strict] <INDEX.md> <kit-root>` | 0 clean or WARN-only · 1 uncited under `--strict` · 3 usage/env | corpus per D3; output `fold-audit: WARN <f> folded with no [ev: retro …] citation` + `fold-audit: N folded, M cited, K uncited` |
| `preflight.sh` | `[--jvm-dir <d>] <niagara_home> <gradle-root>` | 0 all executed checks pass (WARN/SKIP allowed) · 1 a FAIL · 2 usage · 3 env | rows `PASS\|FAIL\|WARN\|SKIP  <check>  <detail>` (verify-module style). Checks: JDK 8 under `--jvm-dir` (default `/usr/lib/jvm`) or `JAVA_HOME`, **never `$HOME`**; `settings.gradle.kts` plugin pin present in `<nh>/etc/m2`; target-jar lock (`lsof` when present, else `SKIP` — never a false PASS); Windows path form (`C:` or `\` → FAIL with the `/mnt/c/…` remedy) |
| `slot-coverage.sh` (pure, MM2) | `set-coverage <declared-csv> <required-csv>` (`""` = ∅) | 0 · 2 argc≠2, with `usage: slot-coverage.sh set-coverage …` on stderr | prints exactly three lines, sets deduped + lexicographically sorted, comma-joined, empty when ∅:<br>`pct=50.0` / `missing=C,D` / `extra=` — and `pct=N/A` when `\|required\|==0` |
| `slot-coverage.sh` (parse) | `[--strict] <module-include.xml> <module.lexicon>` | 0 (WARN allowed) · 1 uncovered under `--strict` · 2 usage · 3 env | two explicit files, zero layout guessing. `required` ← `<type name="X"`, `declared` ← lexicon `key=`; calls the same pure function. `\|required\|==0` → `N/A`; lexicon empty + `\|required\|≥1` → `pct=0.0` + WARN (CompPan T8) |
| `verify-module.sh coverage` | `coverage <npass> <nfail> <nwarn> <nskip>` | 0 · 2 argc≠4 or non-integer, with `usage: verify-module.sh coverage …` on stderr | D2 |
| `install-skill.sh` | `[--home <dir>] [--dry-run] [--force]` | 0 · 1 divergence · 2 usage · 3 env | D5 |

The integer-tenths rounding block is **duplicated verbatim** in `verify-module.sh` and `slot-coverage.sh` rather than introducing a `toolbelt/lib/` sourcing layer (no `$0`-relative source path exists in this toolbelt; both copies are pinned by their own tests).

---

## 6. Fixtures and Test / Mutation Matrix

Fixtures are **generated at test time** under `$BATS_TEST_TMPDIR` (existing convention, `tests/helpers/n4-fixtures.bash`); a new `tests/fixtures/` dir is created only for the `--plano` HTML sample. **No fixture ever lives inside a live module or a real retro.**

| ID | Test | Named mutation → expected flip |
|---|---|---|
| M1 | marker `folded` vs row `pending` → exit 1 | delete the two marker-check lines → M1 exits 0 |
| M4 | `fresh · DATE` vs row `folded` → exit 1 | exact-match the whole marker line → M4 exits 0 |
| M5 | real kit tree → exit 0 | skip the two `fresh`→`folded` re-stamps → M5 exits 1 |
| M6 *(new, QA)* | `review-status` in a line-40 table cell, no line-1 marker → exit 0 | drop the `^<!--` anchor / 5-line window → M6 exits 1 |
| P2 | failing pure test exits non-zero | break the runner's exit propagation → CI red |
| P1-P6 in CI | zero SKIPs | remove the junit pre-fetch step → CI red (fail, not skip) |
| F1/F2 | folded, uncited → WARN + exit 0; `--strict` → exit 1 | — |
| F3 | cited only inside its own retro file → still WARN | drop the `retros/` corpus exclusion → F3 loses its WARN |
| F4 | abbreviated token `5rooms` credits `…-dashboardpan-5rooms.md` | require exact stem → F4 gains a spurious WARN |
| F6 | token `ender-doors` must NOT credit `detail-render-doors` | replace aligned match with plain substring → F6 loses its WARN |
| MM1-MM8 | QA `d7e52a8` value + arity pins | SKIP into denominator / WARN into numerator / `covered=P+F` / truncation / `0/0→100` — each moves a pinned number |
| PF1 | `etc/m2` missing the pinned plugin → exit 1 | make the plugin check always PASS → PF1 exits 0 |
| PF2 | identical output under `HOME=/nonexistent` | resolve any path via `$HOME` → PF2 diverges |
| SC1-SC5 | MM2 pin vectors, exact strings: `{A,B,C,D}/{A,B,C,D}`→`100.0`,∅,∅ · `{A,B}/{A,B,C,D}`→`50.0`,`C,D`,∅ · `{A,B,C,D,X}/{A,B,C,D}`→`100.0`,∅,`X` · `∅/{A}`→`0.0`,`A`,∅ · `∅/∅`→`N/A`,∅,∅ | denominator `\|declared\|` → SC2 reads `100.0` · `extra` in the numerator → SC3 reads `125.0` · `0/0→100` → SC5 loses `N/A` · `missing = declared−required` → SC2/SC3 swap |
| SC6 | parse surface: empty lexicon + 3 exposed types → `pct=0.0` + WARN, exit 0 | return `N/A` (or 100) for an empty lexicon → SC6 flips |
| IS1 | `--home <tmp>` install is byte-identical to the tracked copy | make the installer transform the copy (drop the last line) → `cmp` fails |
| PL1 | two `aspect-ratio` declarations → FAIL *(stretch)* | count-only → PL1 passes |

Suite budget: baseline 104 tests / 8.6 s; +~30 tests, wall must stay ≤ ~15 s (R4). `shellcheck` 0.10.0 exit 0 on `scripts/*.sh build-n4-module-kit/toolbelt/*.sh tests/*.bats tests/helpers/*.bash`.

---

## 7. Threat Matrix

| Boundary | Applicability | Design response | RED test |
|---|---|---|---|
| Documentation-like paths (`.md` parsed as data) | **Applicable** — retro markers and `[ev:]` tokens are read out of prose files | column-0 `^<!--` anchor + 5-line window (D1); token floor + hyphen alignment + `retros/` exclusion (D3) | M4, **M6**, F3, F6 |
| Git repository selection | **N/A** — `toolbelt/*.sh` may not invoke version control (L2); every script takes explicit paths; `install-skill.sh` uses none | — | L2 (existing) |
| Commit state | **N/A** — nothing stages or commits | — | — |
| Push state | **N/A** — `.githooks/pre-push` is unchanged by this campaign | — | H1-H10 (existing) |
| PR commands | **Applicable** — chained `gh pr create --base <parent>`; risk is a polluted child diff (R8) | branch only after the parent merges; confirm `git diff --stat <parent>..HEAD` shows only the slice before opening | human/CI verified, no automated test |
| Subprocess + network (CI) | **Applicable** — `curl` of two jars | pinned URL + sha256, fail closed on mismatch, no `curl \| bash`, ~400 KB | CI step fails on checksum mismatch |
| Environment coupling | **Applicable** — `preflight.sh` probes the machine; `install-skill.sh` writes under a home dir | `--jvm-dir` / `--home` injection, `$HOME` never used for resolution | PF2, IS1 |

---

## 8. VERSION / CHANGELOG and Rollback

**VERSION**: unchanged through PR1-PR5b; PR6 sets `0.16.0` (MINOR — new `scripts/install-skill.sh` surface, three new toolbelt scripts, a new `verify-module.sh` subcommand; CONTRIBUTING §4 "new flag / new optional surface → MINOR", §5 release recipe).
**CHANGELOG**: each PR appends its entry under a single `## [Unreleased]` heading (created above the newest section if absent); PR6 renames it to `## [v0.16.0] - <date>` with the `### References` block (SDD slug + engram observation IDs) and tags `v0.16.0` after the merge. Because a child branches only after its parent merges, these appends never race.
**Also in PR3**: `CONTRIBUTING.md §8` still says "**No CI**" while `.github/workflows/ci.yml` has existed since campaign 5 — same doc-vs-code defect class as BV1; fix it in the doctrine PR.

| Slice | Rollback |
|---|---|
| PR7 | `git revert` — retro files survive (propose-never-apply); rows return to `pending` |
| PR6 | `git revert` restores `VERSION`/`CHANGELOG`/the archive move **and the tracked `skill/SKILL.md`**; re-run `scripts/install-skill.sh --force` to restore the deployed launcher (D5 removes the manual-only rollback of R6) |
| PR5b | `git revert` — purely additive scripts + tests; `verify-module.sh` loses `--plano` only |
| PR5a | `git revert` — `sweep-fold-audit.sh` disappears; `verify-module.sh` loses the `coverage` subcommand (no existing consumer) |
| PR4 / PR3 | `git revert` — doc-only; markers and INDEX rows return to `pending` |
| PR2 | `git revert` — CI returns to skipping P1-P6 |
| PR1 | `git revert` — sweep returns to marker-blind; the 2 re-stamped markers return to `fresh` |

No station, jar, or operator data is touched by any slice.

---

## 9. Open Questions

- [ ] **QA**: add **M6** (prose-mention marker) to `qa/c6-marker-index-drift` before PR1 applies, or accept it as a PR1-authored case?
- [ ] `--plano`: `types/dashboard.md:54` must be read to pin "the four plano values agree"; if it cannot be pinned exactly, ship only the exactly-one-`aspect-ratio` half and record the remainder as owed in the PR5b retro.
- [ ] `sweep-fold-audit.sh` on the real tree will WARN on the handful of folded retros that were never cited (e.g. `dashboardpan-ux-direct-build`). That output is the point, but confirm CI runs it **non-strict** so the list is visible without blocking.
