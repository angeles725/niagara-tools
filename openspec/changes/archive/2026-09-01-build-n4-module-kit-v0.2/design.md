# Design: build-n4-module-kit-v0.2

**Status**: design | **Version**: niagara-tools `0.3.0` → `0.4.0` (MINOR) · kit `v0.1` → `v0.2`
**Topic key**: `sdd/build-n4-module-kit-v0.2/design`
**Inputs**: proposal engram `#7938` + `openspec/changes/build-n4-module-kit-v0.2/proposal.md` · explore `#7937` · lessons inventory `#7935`
**Spec**: written in parallel (`spec.md`); this design does not depend on it and does not restate contracts it owns.

> This is the **HOW at architectural level**. Concrete ordered steps live in `tasks.md` (next phase).
> Size note: this design exceeds the generic 800-word phase budget on purpose. The repo convention
> (`openspec/changes/niagara-tools-slotomatic-integration/design.md`) is a long, table-dense design
> that `sdd-tasks` reads literally; six decision areas were mandated by the coordinator. Prose is
> kept minimal; almost everything is a table.

---

## 0. Technical approach

Three additive slices, fixed merge order, one owner each. Nothing in this change alters a runtime
contract that already exists: `scripts/ng-deploy.sh` is **not touched at all** (its flag/exit/env
surface and its committed file mode both stay exactly as they are), the kit keeps every existing file
and section, and every new toolbelt script is standalone and callable on its own.

```
PR1 docs-foldin (Investigador2)   PR2 toolbelt (QA)              PR3 release (Investigador1)
  retros committed verbatim         verify-module.sh  ← THE gate    retro markers + README v0.2
  41 lessons → 7 kit files          build.sh rewrite  ─┐            GOTCHAS + CHANGELOG + VERSION
  2 dangling-link renames           mirror + stored-repack │        launcher SKILL.md (outside git)
  See-also pointers                 5 bats suites + helpers │
        │                                 │  kit-links.bats ┘ asserts PR1's renames
        └──── merge ──────────────────────┴──── merge ─────────────────► main
```

The architectural spine is one rule: **the kit is a gate, not a narrative.** Every folded lesson
becomes a checkable rule with an evidence marker; every automatable rule that could be a script
becomes one; the narrative stays in `niagara-research/docs/` and is referenced, never copied.

**Doctrine — the three roles.** This exact wording is what `build-verify.md`, `BUILD-LOOP.md` and the
launcher `SKILL.md` must all say:

| Command | Role |
|---|---|
| `toolbelt/verify-module.sh` | **THE gate** — whoever built the jars. |
| `toolbelt/build.sh` | **The recommended WSL build** — it runs the gate. |
| `scripts/ng-deploy.sh` | **The station deploy wrapper** — backup → build → copy → verify. After an `ng-deploy` build the operator runs the gate on `build/libs`. |

**Hard rule:** *A jar that has not passed `toolbelt/verify-module.sh` does not go to a station.*

Wording only — this change does **not** wire the gate into `ng-deploy.sh` (frozen decision, proposal §12.3).
These three roles replace the old "primary vs fallback" framing that caused F1: the commands do three
different jobs; they are not two ranked ways of doing the same one.

---

## 1. Fold-in placement per target file (PR1)

### 1.0 Fold-in form — the four rules every folded line obeys

| Rule | Shape |
|---|---|
| Lead with the rule | The bullet opens with a bold imperative or a bold noun-rule, never with the story. |
| Evidence inline | The evidence marker closes the same bullet. No footnote blocks, no "as we learned when…". |
| One rule per bullet | If a lesson needs two rules, it becomes two bullets. |
| Pointer, never duplicate | Anything already narrated in `niagara-research/docs/` gets a `See also:` line, not a paraphrase. |

**Evidence marker grammar** (one of):

| Marker | Use for |
|---|---|
| `[ev: retro 5rooms #10]` · `[ev: retro rt-hardening #6]` · `[ev: retro hmi-touch-ux Δ9]` | Reproducible or kit-retro-sourced lessons. |
| `[ev: bitácora 5cuartos §7]` | Bitácora-sourced lessons (groups G, J). |
| `[CERT-live 2026-09-01 · bitácora 5cuartos §9]` | Station-only facts QA cannot reproduce in WSL (B6, G2, G3; A1 is [CERT]+[INFER], never smoke-tested live). **Preserved verbatim; never asserted by a test.** |

**See-also grammar** — each touched kit file carries at most one See-also line, and the *first* file
in reading order that uses it (`SOURCES.md`) carries the single definition line:

```
`docs/` = `/home/cristian/niagara-research/docs/` — narrative reference; the kit points, never copies.
```

Other files then write only `See also: docs/module-best-practices.md §2`.

Because a `docs/<f>.md` reference points **outside this repository** (into `niagara-research`), it is an
external pointer, not a repo-relative link: `tests/kit-links.bats` L1 explicitly **ignores every ref
beginning with `docs/`** (§3.3). That is why the `docs/` prefix is the mandated form — it is the marker
that makes an external pointer recognisable to both a reader and the link test.

### 1.1 `types/dashboard.md` — 22 items (C1–C13, G2–G5, G10, J1–J4)

Existing sections are kept verbatim and in place. New sections are inserted; the alarms section stays last.

| # | Section | State | Items | Notes |
|---|---|---|---|---|
| 1 | `# Type: dashboard (facade + servlet + SPA) — MATURE` | kept | — | title unchanged |
| 2 | `## rt — the facade (no logic)` | kept | — | unchanged |
| 3 | `## Facade contract & commissioning links` | **NEW** | J1, C8, G2, G10, J2 | link *direction* convention (telemetry control→facade, config facade→control); room-count is data-driven; facade stays a pure slot container — never `setOut` from `changed()`; the worked "add one operator config field" path (G10); what belongs on the HMI vs Workbench (J2) |
| 4 | `## ux — servlet + SPA` | kept | — | unchanged |
| 5 | `## Extending an existing dashboard` | **NEW** | C6, C7 | data-vs-code decision before any edit; module is the skeleton, the operator's standalone HTML is the data. Points at `METHODOLOGY.md` §editing technique for the *how* (D1/D2), does not restate it |
| 6 | `## Config panel UX on a fixed touch panel` | **NEW** | C1, C11, J3, J4, C10, G5 | sub-tab groups when fields overflow; `buildField()` row-builder from the start; hybrid type+stepper field; a dense secondary feature gets its own page; where writable ranges actually live; multi-viewer prefill-but-skip-dirty |
| 7 | `## Charts on an HMI` | **NEW** | C2, C3, C9 | measure-the-box viewBox; pointer-event crosshair (hover is dead on capacitive); explicit per-series palette |
| 8 | `## Plano overlay` | **NEW** | C4, C12, C13 | **exactly ONE** aspect-ratio declaration for the frame == `IMG_W/IMG_H`; fix = delete the stale value, never shadow it; `lbl:[x,y]` label override; `#plano` src may be inline base64 |
| 9 | `## Triage — a UI element is "missing"` | **NEW** | C5 | ordered a→b→c; do not touch code until the live-vs-source gap is ruled out |
| 10 | `## Deploy on a JACE` | **NEW** | G3, G4 | never a raw servlet path as User Home Page; RBAC is deploy-side config. Both `[CERT-live]` |
| 11 | `## HMI kiosk (WEB-HMI10/CF …)` | kept | — | unchanged; §6/§7 point back to it for the 44px/no-scroll budget |
| 12 | `## Real alarms (Phase B)` | kept | — | stays last |
| 13 | `See also:` line | **NEW** | — | `docs/module-best-practices.md §2` (X-Requested-With) |

Reader budget: 8 new sections × ≤7 bullets. §6 is the largest (6 items) and is the split candidate if
the commit outgrows ~200 lines.

### 1.2 `types/logic.md` — 9 items + 2 link renames

| # | Section | State | Items |
|---|---|---|---|
| 1 | title | **edited** | `— SEED (feed as built)` → `— GROWING (control core proven on ColdRoomPan, 2026-08-31)` |
| 2 | existing seed bullets (control engine / links / slots / no-servlet) | kept | — |
| 3 | `## Safety fail-modes & timers` | **NEW** | A1 (`HIDDEN BAbsTime` restart-persistent timer, re-arm `max(interval-elapsed,0)`), A3 (fail-mode is a `SUMMARY\|OPERATOR` slot defaulting to current behavior), **G8** (alarm-limit slots the control does not read are correct: alarms NOTIFY, never STOP) |
| 4 | `## Staging & interlocks` | **NEW** | G7 (FIFO `ArrayDeque<Integer>`, dedupe, `pollFirst` on terminate), G9 (filter by capability before granting the token), G6 (HOA = TRANSIENT double 0/1/2; priority defrost > HOA > auto) |
| 5 | `## Linking across custom modules` | **NEW** | G1 — a linked cross-module value is a plain `double`, **not** a `BFrozenEnum` (H4 correction) |
| 6 | `## Logging` | **NEW** | A2 — a plain non-`BObject` helper compiles and bundles; one `java.util.logging.Logger` per module |
| 7 | `## Regenerating slots` | **NEW** | A4 — hand-write annotation + AUTO stub, then run slotomatic; slotomatic is authoritative |
| 8 | verify line + TODO | **edited** | `checklist-common.md` → `METHODOLOGY.md`, `type-dashboard.md` → `types/dashboard.md`; TODO loses the items §3/§4 now close, keeps what remains |
| 9 | `See also:` | **NEW** | `docs/how-to-create-coldroom-module.md` |

> **Design addition**: the coordinator's subsection map did not place **G8**, but the proposal counts
> it (§5.1: A1–A4, G1, G6–G9). It is folded into §3 because it is a boundary rule about what may stop
> control. Called out here so it is not read as scope creep or as a dropped item.

### 1.3 `build-verify.md` — 8 items + corrections + doctrine

| # | Section | State | Items |
|---|---|---|---|
| 1 | intro (`gradle :jar` is not a build) | kept | — |
| 2 | `## Doctrine — which command does what` | **NEW** | The §0 three-role table verbatim: `verify-module.sh` = **THE gate** · `build.sh` = **the recommended WSL build**, which runs the gate (and slotomatic for every profile with sources) · `ng-deploy.sh` = **the station deploy wrapper**, after which the operator runs the gate on `build/libs`. Then the hard-rule sentence. No "primary/fallback" wording anywhere in the file |
| 3 | `## Primary: ng-deploy.sh` | kept | — |
| 4 | `## Fallback: raw gradle` | kept | — |
| 5 | `## Build target & plugin version` | **NEW** | B1 (build against the LOWEST target's `niagara_home`; `baja vendorVersion` stamp), B2 (**one plugin version per install** table + mandatory `-PniagaraPluginVersion` override), B3 (`gradle.properties` can lie), B4 (Windows path breaks the WSL m2 repo) |
| 6 | one-plugin-per-install table | **NEW** | `4.13.2 → 7.3.40` · `4.14 → 7.6.17` · `4.15.3 → 7.6.22`. Explicitly supersedes the retro's "common 7.6.1/7.6.3/7.6.5" claim |
| 7 | `## niagara_home on WSL` | **edited** | mirror paragraph collapses to one line pointing at §8 (no duplication) |
| 8 | `## Building against a running station: mirror` | **NEW** | B5 + `toolbelt/mirror-niagara-home.sh` usage + its two guards |
| 9 | `## Verify` | **edited** | opens with `toolbelt/verify-module.sh <jars…>`; the existing `unzip`/`od`/`grep` one-liners stay below as "what it checks under the hood / how to check by hand" |
| 10 | `## Signing per deploy target` | **NEW** | B6 `[CERT-live]` — supervisor accepts the gradle DEV cert; a JACE enforces the project CA |
| 11 | `## Workbench re-sign: STORED repackage` | **NEW** | B7 + H1 (deflater mismatch, **not** transient build state; clean+rebuild does not fix it) + H2 (STORED is a manual post-build step; `build/libs` stays deflated) + local `jarsigner` is a false negative + `toolbelt/stored-repack.sh` + `[CERT-live 2026-09-01]` |
| 12 | `## Unit tests in WSL` | **NEW** | B8 — extract a zero-Baja pure class; `javac -source 8 -target 8` + JUnit4; `srcTest/` is dead weight in WSL |
| 13 | `## Known gap — ng-deploy.sh runs slotomatic for -rt only` | **NEW** | H3 — documented, **not fixed**; use `build.sh` when editing `-ux` annotations |
| 14 | `## Deploy (station)` | **edited** | trimmed; signing detail now lives in §10/§11 |
| 15 | `See also:` | **NEW** | `docs/module-dev-workflow.md` |

### 1.4 `METHODOLOGY.md`, `BUILD-LOOP.md`, `SOURCES.md`, `types/wb-widgets.md`

| File | Change |
|---|---|
| `METHODOLOGY.md` | **NEW** `## Editing technique — asset-laden single-file artifacts` (D1 `sed -n 'A,Bp' \| cut -c1-160`, length-filtered search, `node --check` re-verify; D2 diff a working version to isolate the one stray value), placed after `## Domain correctness`. `## Build` checklist gains one line: `[ ] toolbelt/verify-module.sh passed on the built jars.` |
| `BUILD-LOOP.md` | Step 0 gains sub-step **`0.b Preflight`**: JDK 8 present → `niagara_home` chosen and its pinned plugin present in `etc/m2` → is the station live? (mirror if yes) → target jar not locked. Step 4's "Primary / Fallback" wording is replaced by the §0 three-role table. Step 5 names `verify-module.sh` as the automated half of the gate. |
| `SOURCES.md` | ColdRoomPan exemplar → `/home/cristian/modulos_niagara_n4/Cliente/Leon-Guanjuato/Paccadia/ColdRoomPan/` primary, `/mnt/c/...` noted as Windows-only fallback. Carries the single `docs/` definition line. |
| `types/wb-widgets.md` | Link renames only (`type-dashboard.md` → `types/dashboard.md`, `checklist-common` → `METHODOLOGY.md`) + one `See also:` line. **Stays SEED** (F3). |

---

## 2. Script design (PR2)

Shared conventions for all four scripts: `#!/usr/bin/env bash`, `set -euo pipefail`, English messages
only, POSIX/coreutils + `zip`/`unzip`/`od`/`find` only (no `rg`, no `jq`, no GNU-only flags), shellcheck
clean, and any `# shellcheck disable=` carries an adjacent `# why:` comment (existing `ng-deploy.sh` pattern).
**No script in this change invokes `git`** — see §5 threat matrix.

### 2.1 `toolbelt/verify-module.sh` — THE gate

```
Usage: verify-module.sh [--target-version X.Y] [--src <module-dir>] [--stored] <jar|dir>...
```

One function per check; every check runs (a failure never aborts the run), each prints one row, and a
results table closes the report.

| Check function | Default? | What it asserts | Regression it guards |
|---|---|---|---|
| `check_bytecode_major` | **on** | **every** `.class` in the jar has major 52 | a Java-11 class hiding behind a compliant first class (today's `build.sh` blind spot) |
| `check_signed` | **on** | `META-INF/NIAGARA4.SF` present | unsigned jar on a signing-enforcing station (B6) |
| `check_types_have_classes` | **on** | every `module.xml` `<type>` has its `.class` in the jar | station boot-loop `Type "…" not found` |
| `check_type_count` | `--src` | `<type>` count == the module-include count for that profile | a type added to the annotation but never regenerated |
| `check_raw_double_facets` | `--src` | `BFacets.make(BFacets.(MIN\|MAX), -?[0-9]` grep is empty | raw-double facet that does not compile / silently mis-facets |
| `check_baja_version` | `--target-version` | `META-INF/module.xml` `baja vendorVersion` **≤** the declared target | B1 — a 4.15-built jar rejected by a 4.14 station |
| `check_stored` | `--stored` | `unzip -v` reports zero `Defl:` entries | B7 — a deflated jar handed to Workbench re-sign |

**`--target-version` comparison operator**: `vendorVersion <= --target-version`, **not** equality. A jar
built against 4.14 is accepted by a 4.15 station; only the reverse is rejected (B1 is one-directional).
Equality would fail a legitimate lowest-common-denominator build — exactly the practice B1 prescribes.

**`--src <module-dir>` semantics** — `<module-dir>` is the directory that **holds the profile
directories**, not a `src/` tree. For each jar named `X-p.jar` the script derives:

| Derived path | Used by |
|---|---|
| `<module-dir>/X-p/module-include.xml` | `check_type_count` — **`module-include.xml` sits at the profile-directory root, not under `src/`** |
| `<module-dir>/X-p/src` | `check_raw_double_facets` — grep root |

`build.sh` therefore passes `--src "$ROOT/$MOD"` **once**, for every jar it just built, and never a
per-profile path. If a derived path is absent the check reports `SKIP`, never `FAIL` — a missing
`module-include.xml` is a different defect from a wrong count.

**Decision — conservative default set.** Only the three checks QA reproduced against real DashboardPan
jars run by default. `--stored`, `--target-version` and `--src` are opt-in.
*Alternatives rejected*: (a) all-on-by-default — a legitimate deflated `build/libs` jar would fail the gate,
so operators would learn to ignore it (proposal R9); (b) auto-detect `--src` from the jar path — silent
magic that changes the check set based on where the jar happens to sit.

| Exit | Meaning |
|---|---|
| 0 | every executed check passed |
| 1 | at least one check FAILED |
| 2 | usage error — **no arguments**, unknown flag, or an argument that is not a jar or a directory |
| 3 | environment error (`unzip` missing, jar unreadable) |

Report line format: `PASS|FAIL|SKIP  <check>  <jar>  <detail>` — two spaces between fields, **no colon**,
so the output stays greppable and column-scannable. `SKIP` covers an opt-in check that was not requested,
a check that does not apply to that jar, and a `--src`-derived path that does not exist. Non-`0` exits
print the table first.

**Control-flow note**: `set -e` must not abort mid-table. Every check is invoked as
`if check_x "$jar"; then :; else FAILED=1; fi` — never as a bare call.

### 2.2 `toolbelt/build.sh` — rewrite

```
Usage: build.sh [--profiles rt,ux,wb] [--target-version X.Y] [--plugin-version V] \
                <module-root> <MOD> [niagara_home]
```

**`niagara_home` resolution**: positional argument 3, else the `$niagara_home` environment variable.
There is **no `--niagara-home` flag**.
*Rationale*: the existing `build.sh` already takes `niagara_home` as positional 3 with a `$niagara_home`
fallback, and every operator invocation and bitácora note is written that way. Adding a flag would create
two spellings for one input and silently invalidate muscle memory during a rewrite that is supposed to
fix bugs, not move the controls. `--plugin-version` is a flag because it is new (B2's mandatory
`-PniagaraPluginVersion` override) and has no positional precedent.

**Profile selection predicate** — a profile `p` is built iff all three hold:

1. `-d "$ROOT/$MOD/$MOD-$p"`
2. it has a gradle file: `-f .../build.gradle` **or** `-f .../build.gradle.kts`
3. it has sources: `find "$ROOT/$MOD/$MOD-$p/src" -type f \( -name '*.java' -o -name '*.js' -o -name '*.html' \) -print -quit` is non-empty

`--profiles` overrides auto-detection entirely (no union, no filtering) — an explicit operator choice is
never silently amended.
*Rationale*: condition 3 is the whole fix for the confirmed `DashboardPan-wb` regression, where an empty
stub subproject with a gradle file added `-wb` tasks to every build. `-print -quit` keeps the predicate
O(1)-ish on a big tree.

**Target guard**: refuse a `niagara_home` that has no `etc/m2` directory → exit 10. That directory is the
discriminator B3/B4 identified (a path can look like an install and still have no plugin repo).

**Gate call**: after gradle succeeds, `build.sh` invokes its sibling — `"$(dirname "$0")/verify-module.sh"` —
forwarding `--target-version` when given and `--src "$ROOT/$MOD"` always (one `--src`, never per-profile;
§2.1). Resolving via `$(dirname "$0")` (not `$PWD`, not a `$KIT` env var) keeps the pair movable as one
directory.

| Exit | Meaning | Chosen to match |
|---|---|---|
| 0 | build + gate passed | — |
| 2 | usage / bad args (also the no-args case: print usage, exit 2) | `verify-module.sh` 2 |
| 10 | environment or path: no JDK 8, `niagara_home` missing or not a real install | `ng-deploy.sh` 10 = env/path |
| 30 | gradle failed | `ng-deploy.sh` 30 = build |
| 50 | the `verify-module.sh` gate failed | `ng-deploy.sh` 50 = verify failed |

**Exit-code alignment is literal, not approximate.** `build.sh` reuses `ng-deploy.sh`'s meanings for the
same stages so one exit vocabulary spans both scripts. **50, not 40**: `ng-deploy.sh` already assigns
40 = copy failed and 50 = verify failed, and the gate is a verify stage. `build.sh` does not propagate
`verify-module.sh`'s own exit code (1/2/3) — those describe the gate's internals; `build.sh` reports one
meaning, "the verify stage failed", as 50.

### 2.3 `toolbelt/mirror-niagara-home.sh` — promote + guard

```
Usage: mirror-niagara-home.sh <source-niagara-home> <mirror-dir> [exclude-jar]...
```

**Behavior** (the full contract, not only the guards):

| Step | Rule |
|---|---|
| 1. Guard G-A — source protection | `realpath` both sides; refuse if `mirror == source`, if `mirror` is inside `source/`, or if `source` is inside `mirror/` → exit **20** |
| 2. Guard G-B — wipe protection | if `mirror` already exists, refuse unless `mirror/.niagara-mirror` is present → exit **20** |
| 3. Link the install | symlink **every top-level entry of `source`** into `mirror`, **except `modules/`** (`bin/`, `etc/`, `lib/`, `jre/`, `defaults/`, … — whatever is actually there; the set is discovered, never hardcoded) |
| 4. Populate `modules/` | create a **real, writable** `mirror/modules/` directory, then symlink each `*.jar` from `source/modules/` into it |
| 5. Excludes | every `exclude-jar` argument is matched by **exact file name** (`ColdRoomPan-rt.jar`, not a prefix, glob or path) and is **not** linked — that is the jar the build will write |
| 6. Empty source `modules/` | tolerated: zero jars linked is a success, not an error (a fresh install or an already-cleaned mirror) |
| 7. Report | print the number of module jars linked and the number excluded |
| 8. Marker | write `mirror/.niagara-mirror` (source path + date) **on creation**, so step 2 permits a re-run |

Both guards run before any `rm` or `ln`. The tilde example in the bitácora header is rewritten to an
absolute path.
*Exact-name matching rationale*: a prefix or glob match would silently exclude `ColdRoomPan-ux.jar` when
the operator meant only `-rt`, and the build would then fail to resolve a dependency it expected to find.
*Alternative rejected*: a `--force` escape hatch. This script deletes and re-links directories; an escape
hatch would become the copy-pasted default and both guards would be decorative.

### 2.4 `toolbelt/stored-repack.sh` — the B7 recipe

```
Usage: stored-repack.sh <in.jar> <out.jar>
```

Two explicit positional paths — **no `--out-dir`, no batch mode, no default destination.** The operator
names the output every time.
*Rationale*: this script exists to feed a jar into a Workbench re-sign. An implicit output directory
invites "where did it go?" and a batch loop invites repacking a jar the operator did not mean to touch.
One in, one out, both named.

Ordering is the whole point (a JAR whose manifest is not the first entry is not a JAR):

1. `unzip -q "$IN" -d "$WORK"` (fresh `mktemp -d`, `trap`-cleaned)
2. `zip -0 -X -D "$OUT" META-INF/MANIFEST.MF` — **manifest is entry #1**
3. `zip -0 -X -D "$OUT" META-INF/*.SF` then `META-INF/*.RSA` — signature block next, `.SF` before `.RSA`
4. `zip -0 -X -D -r "$OUT" . -x 'META-INF/MANIFEST.MF' -x 'META-INF/*.SF' -x 'META-INF/*.RSA'` — everything else
5. self-check: `unzip -v "$OUT" | grep -c 'Defl:'` must be `0`, else exit 1

`-0` store, `-X` no extra fields, `-D` no directory entries.

| Exit | Meaning |
|---|---|
| 0 | repacked, zero `Defl:` entries |
| 1 | **`out.jar` already exists** (refused, never overwritten), input missing/unreadable, or the self-check found a deflated entry |
| 2 | usage — wrong argument count |

**Refusing to overwrite an existing `out.jar` is a hard rule, not a prompt.** The likely mistake is
repacking onto a jar the operator already signed in Workbench; that output cannot be regenerated from the
input. An unsigned input repacks fine (there is simply no `.SF`/`.RSA` to order in step 3). The gradle
artifact in `build/libs` is never mutated — H2: STORED is a manual post-build step for the Workbench
re-sign path only.

---

## 3. Test design (PR2)

### 3.1 Helper location

**Decision**: `tests/helpers/n4-fixtures.bash`, loaded with `load helpers/n4-fixtures` (bats resolves
`load` relative to the test file and appends `.bash`).
*Rationale*: four of the five new suites need generated jars; inlining would quadruple the fixture code and
let the copies drift. `tests/ng-deploy.bats` **does not** load the helpers and is not touched, so its
26 cases cannot regress through a shared file.
*Alternative rejected*: inline-per-file (matches today's single-file convention but duplicates ~60 lines ×4).

### 3.2 Helper API

| Helper | Contract |
|---|---|
| `make_class_file <path> <major>` | Writes a minimal fake class: `printf '\312\376\272\276\0\0\0\%o'` (magic `CAFEBABE`, minor `0000`, major at byte offset 6–7 — exactly what `od -An -t d1 -j6 -N2` reads), then pads. Octal escapes because `printf '\xNN'` is not POSIX. |
| `make_jar <jar> <dir>` | `(cd "$dir" && zip -q -r "$jar" .)` — deflated by default. |
| `make_stored_jar <jar> <dir>` | same with `-0 -X -D`, for the `--stored` PASS case. |
| `add_signature <dir>` | writes `META-INF/NIAGARA4.SF` with dummy digest lines. |
| `make_module_xml <dir> <vendorVersion> <type>...` | writes `META-INF/module.xml` with a `<baja vendorVersion="…">` element and one `<type name= class=>` per argument. |
| `make_module_include <profile-dir> <type>...` | writes `<profile-dir>/module-include.xml` with matching `<type>` entries — **at the profile-directory root, not under `src/`** (§2.1). |
| `make_niagara_home <dir> <plugin-version>` | creates `etc/m2/repository/com/tridium/niagara-module/<v>/` — the `build.sh` target guard fixture. |
| `make_fake_gradlew <bin-dir>` | fake `gradlew` on `PATH` appending `"$*"` to `$TMPDIR_T/gradlew.calls.log` and honoring `FAKE_GRADLEW_EXIT` — same shape as `tests/ng-deploy.bats setup()`. |

Every suite uses the existing lifecycle verbatim: `setup() { TMPDIR_T="$(mktemp -d)"; export TMPDIR_T; … }`
and `teardown() { rm -rf "$TMPDIR_T"; }`. Nothing is written outside `$TMPDIR_T`; no real jar, station,
gradle, or `niagara_home` is touched. Zero committed binaries — every fixture is generated.

### 3.3 Naming convention

`@test "<PREFIX><n>: <behavior> (<regression it guards>)"` — prefixes `V` verify-module, `B` build.sh,
`M` mirror, `S` stored-repack, `L` links. The parenthetical is mandatory: a test whose regression cannot
be named does not get written (operator rule, proposal §6).

**Five new bats files**, one per script plus the link suite:

| Suite | Cases | Regression source |
|---|---|---|
| `tests/verify-module.bats` | V1 forged major `65` on a **non-first** class · V2 missing `NIAGARA4.SF` · V3 `<type>` without a `.class` · V4 `<type>` count ≠ the profile's `module-include.xml` · V5 `baja` **>** `--target-version` fails, `<` and `==` pass · V6 `--stored` on a deflated jar · V7 raw-double facet grep non-empty · V8 all-pass exit 0 · V9 non-jar argument exits 2 · **V-usage** no arguments → exit 2 | proposal §6 + coordinator |
| `tests/build-sh.bats` | B1 stub `-wb` (gradle file, no sources) is skipped · B2 no args prints usage, exit 2 · B3 `niagara_home` without `etc/m2` exits 10 · B4 `--profiles` overrides detection · B5 `verify-module.sh` is invoked after gradle with a single `--src "$ROOT/$MOD"` · B6 gradle failure exits 30 · B7 gate failure exits **50** | proposal §6 + coordinator |
| `tests/mirror-niagara-home.bats` | M1 mirror == source refused (20) · M2 mirror inside source refused (20) · M3 existing mirror dir without `.niagara-mirror` refused (20) · M4 **happy path**: every top-level entry symlinked except `modules/`, `modules/` real and writable, jars linked, marker written · M5 **exclude-jar** by exact name is not linked while its sibling profile jar is · M6 **empty `source/modules/`** succeeds with a zero linked count | proposal §6 + coordinator |
| `tests/stored-repack.bats` | S1 output has zero `Defl:` entries and every entry is **byte-identical to the input** (`cmp` per extracted file) · S2 entry order is MANIFEST.MF #1, then `.SF`, then `.RSA` · S3 an **unsigned** jar repacks successfully · S4 refusals are non-zero: existing `out.jar` (1), missing input (1), wrong argument count (2) | coordinator |
| `tests/kit-links.bats` | L1 every **repo-relative** ref in kit `*.md` resolves — refs beginning `docs/` are external pointers into `/home/cristian/niagara-research/docs` and are **ignored**, as are anchors, absolute paths and URLs · L2 no toolbelt script invokes `git` | dangling `checklist-common.md` / `type-dashboard.md`; §5 threat matrix |
| `tests/ng-deploy.bats` | **untouched** — must stay 26/26 green | regression guard |

S1's `cmp` assertion is the load-bearing one: a repack that stores the entries but corrupts their bytes
would still report zero `Defl:` and still fail in Workbench, so "stored" alone is not the property under
test — "stored **and** unchanged" is.

---

## 4. Chain mechanics

### 4.1 Merge method — evidence-based decision

`.git/logs/HEAD` shows 5 commits and **zero merge entries**; the repo has never carried a merge commit.

**Decision: rebase onto `main`, then `git merge --ff-only` (GitHub: "Rebase and merge"). No squash, no merge commits.**
*Alternative rejected — squash*: it would collapse each slice into one commit, destroying the work-unit
split. PR1 specifically requires the verbatim retro import to stay a separable commit so the fold-in diff
and the audit-trail diff can be reviewed and reverted independently (proposal R1).
*Alternative rejected — merge commits*: no precedent in this history.

### 4.2 Commit plan (work-unit-commits: tests ship with the code they verify)

| PR | # | Conventional commit subject |
|---|---|---|
| **PR1** `feat/kit-v0.2-docs-foldin` | 1 | `chore(build-n4-module-kit): commit the 3 unmerged kit retros as fold-in baseline` *(done — `abebfee`; quoted here as the actual subject, not a paraphrase)* |
| | 2 | `docs(kit/dashboard): fold facade, extension and triage lessons into types/dashboard.md` |
| | 3 | `docs(kit/dashboard): fold touch-panel, chart, plano and JACE lessons` *(split only if commit 2 exceeds ~200 lines)* |
| | 4 | `docs(kit/logic): fold control, staging and cross-module linking lessons; fix dangling refs` |
| | 5 | `docs(kit/build-verify): fold build-target, mirror, signing and STORED lessons with corrections` |
| | 6 | `docs(kit): add preflight step, editing technique, ColdRoomPan path and See-also pointers` |
| **PR2** `feat/kit-v0.2-toolbelt` | 1 | `docs(contributing): add the bats-core install step` ← *moved here by resolution: documented before used* |
| | 2 | `feat(kit/toolbelt): add verify-module.sh gate with generated-jar bats fixtures` |
| | 3 | `feat(kit/toolbelt): rewrite build.sh with usage, source-based profiles and the gate call` |
| | 4 | `feat(kit/toolbelt): promote mirror-niagara-home.sh with destructive-path guards` |
| | 5 | `feat(kit/toolbelt): add stored-repack.sh for the Workbench re-sign path` |
| | 6 | `test(kit): assert every relative kit link resolves` |
| **PR3** `feat/kit-v0.2-release` | 1 | `docs(kit): mark the folded retros and raise the kit README to v0.2` |
| | 2 | `docs: add GOTCHAS entries for the STORED repack, the mirror and the verify gate` |
| | 3 | `chore(release): v0.4.0` |

No `Co-Authored-By`, no AI attribution (CONTRIBUTING §6).

**No exec-bit commit — the earlier "repo defect" reading was wrong.** The git index has carried
`100755` for `scripts/ng-deploy.sh` since v0.2.0; only `main`'s **working tree** had drifted to `100644`,
and the coordinator already restored it with `git checkout -- scripts/ng-deploy.sh`. A
`git update-index --chmod=+x` commit would therefore be a no-op against the index and would record a
fix for a defect that never existed in the repository. `scripts/` is untouched by this entire change;
the only thing that survives is a **verify-checklist assertion** (§9) that the index still reads
`100755`, which catches the drift if it recurs.

### 4.3 PR titles and required description content

| PR | Title |
|---|---|
| 1 | `docs(kit): fold 41 proven build lessons into build-n4-module-kit v0.2` |
| 2 | `feat(kit): add the verify-module gate, rewrite build.sh, and cover the toolbelt with bats` |
| 3 | `chore(release): build-n4-module-kit v0.2 / niagara-tools v0.4.0` |

Every PR description carries, in this order (cognitive-doc-design + chained-pr):

1. **What changed and why** — one paragraph.
2. **Chain context** with the dependency diagram, current slice marked `📍`:
   `PR1 docs-foldin → PR2 toolbelt → PR3 release`
3. **Review this first** — the one file that carries the decision.
4. **Out of scope** — copied from proposal §2.2 (no `ng-deploy.sh` behavior change; the `-ux` slotomatic
   gap is documented, not fixed; the live stale `.frame` in DashboardPan is a separate module fix).
5. **Verification** — exact commands and their exact results.
6. **Rollback boundary** — the files removable without touching unrelated work.

**PR3 additionally MUST contain the launcher before/after as a fenced diff.**
`~/.claude/skills/build-n4-module/SKILL.md` is outside every git repo: it appears in no diff and
`git revert` cannot roll it back. The two hunks are (a) the Hard-Rules build bullet, rewritten to the §0
three roles — `toolbelt/build.sh` is the recommended WSL build **and it runs the gate**,
`scripts/ng-deploy.sh --strict-slotomatic` is the station deploy wrapper, and
`toolbelt/verify-module.sh` is THE gate before anything reaches a station — and (b) the frontmatter
`version: "0.1"` → `"0.2"`. That resolves F1 without reintroducing a primary/fallback ranking. The same block is saved to engram
`sdd/build-n4-module-kit-v0.2/launcher-diff`. Rollback is a manual re-edit from that block.

### 4.4 Rebase procedure for PR2 after PR1 merges

```bash
# 1. land PR1 on main (linear)
git -C /home/cristian/modulos_niagara_n4/niagara-tools checkout main
git -C /home/cristian/modulos_niagara_n4/niagara-tools merge --ff-only feat/kit-v0.2-docs-foldin

# 2. rebase the toolbelt worktree onto the new main
cd /home/cristian/modulos_niagara_n4/niagara-tools-worktrees/toolbelt
git rebase main

# 3. prove the child diff is clean — only PR2 files may appear
git diff --stat main...HEAD

# 4. re-run the gate on the rebased head, then publish
bats tests/*.bats && shellcheck build-n4-module-kit/toolbelt/*.sh scripts/*.sh tests/*.bats tests/helpers/*.bash
git push --force-with-lease
```

`--force-with-lease`, never `--force`. PR3 repeats the same procedure against the post-PR2 `main`.

### 4.5 Gate ownership and the one ordering hazard

**QA runs the full gate on each branch head immediately before its merge** — `bats tests/*.bats`
(ng-deploy.bats still 26/26) and `shellcheck` over `build-n4-module-kit/toolbelt/*.sh`, `scripts/*.sh`,
`tests/*.bats`, `tests/helpers/*.bash`, all exit 0.

**Hazard**: `tests/kit-links.bats` is authored in PR2 but asserts the renamed link targets introduced in
PR1. It is therefore **red on the un-rebased PR2 branch by construction** and can only be green after
step 2 of §4.4. The gate contract is explicit: `kit-links.bats` must be green **at PR2 merge time**, on
the rebased head — not on the pre-rebase branch.

---

## 5. Threat matrix

Applicable: this change composes shell commands, spawns subprocesses, automates VCS/PR work across three
worktrees, and changes an executable-file classification (the `ng-deploy.sh` mode bit).

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| Documentation-like paths | **Applicable** — `kit-links.bats` walks `*.md` and `verify-module.sh` takes path arguments | Link checking only *resolves* refs; it never sources, executes, or renders a `.md`. `verify-module.sh` rejects any argument that is not a readable `.jar` or a directory of jars → exit 2. Anchors (`#…`), absolute paths, URLs and **external `docs/…` pointers** are skipped, not resolved. | `L1` (a `.md` ref to a missing file fails; an anchor, URL or `docs/` pointer does not); `V9` (a `.md` argument exits 2, never treated as a jar) |
| Git repository selection | **Applicable** — three worktrees share one object store | No script shipped in this change invokes `git`; every git command is operator-run from an explicit worktree root, and cross-tree commands use `git -C <abs-path>` (§4.4). | `L2` — no file under `build-n4-module-kit/toolbelt/` contains a `git ` invocation |
| Commit state | **Applicable** — three worktrees staging in sequence; a mode-vs-content distinction was mis-read once already | **No index-only mode change is made.** The index has held `100755` for `scripts/ng-deploy.sh` since v0.2.0; only `main`'s working tree drifted and was restored with `git checkout --`. Every commit stages explicit paths; **never `commit -a`**, so an unrelated worktree drift can never ride along in a slice. | Verify-phase assertion: `git ls-files -s scripts/ng-deploy.sh` starts with `100755` (drift detector, not a fix) |
| Push state | **Applicable** — `origin` = `github.com/angeles725/niagara-tools`; the three branches are first-push | First push is explicit: `git push -u origin <branch>`. Republished after a rebase only with `--force-with-lease`. Never `--force`, never a bare `git push` with an implicit refspec. | Verify-phase assertion (procedure, §4.4) — no automatable RED test |
| PR commands | **Applicable** — three chained PRs | `gh pr create --base main --head <branch> --title <t> --body-file <f>`; the base and head are always explicit, never inferred from the current branch, and PR3's body comes from a file so the launcher diff survives verbatim. | Verify-phase assertion: each PR's `--base` is `main` and `gh pr view --json additions,deletions` fits the 800-line budget |

---

## 6. Version, changelog and marker design (PR3)

| Artifact | Exact form |
|---|---|
| `VERSION` | `0.3.0` → `0.4.0` (MINOR — new toolbelt scripts, new tests, new docs; per CONTRIBUTING §4) |
| `CHANGELOG.md` | New `## [v0.4.0] - 2026-09-01` section inserted **above** `## [v0.3.0]`, Keep-a-Changelog with `### Added — build-n4-module-kit-v0.2`, `### Changed — …`, `### Fixed — …`, `### References`. **Written in English** (the v0.3.0 entry is Spanish; the artifact-language contract applies to new text, and existing entries are not rewritten). |
| `### References` block | `- SDD slug: build-n4-module-kit-v0.2` · `- Engram: explore #7937, lessons-inventory #7935, proposal #7938, design <this id>, spec <spec id>.` · `- Tag: v0.4.0.` — mirrors the v0.2.0/v0.3.0 shape exactly. |
| Not in the CHANGELOG | The launcher `SKILL.md` edit — it is outside the repo. It is recorded in the PR3 body and in engram only; a changelog line would imply the tag carries it. |
| Kit `README.md` | `## Status` line: `v0.1, SEEDED.` → `v0.2, GROWING. The dashboard and build/verify paths are proven end-to-end (DashboardPan + ColdRoomPan, 2026-08/09) and gated by toolbelt/verify-module.sh. The wb path is still a stub to feed from real builds via retros/.` The `## Layout` list gains `verify-module.sh`, `mirror-niagara-home.sh`, `stored-repack.sh`. |
| Retro markers | `<!-- review-status: folded v0.2 · 2026-09-01 -->` as **line 1**, then a blank line, then the existing `# Retro — …` heading. Body preserved byte-for-byte — the retros keep their superseded claims verbatim; the corrections live in the kit, not in the audit trail. **No retro file is deleted** (propose-never-apply). |
| Tag | `git tag v0.4.0` on the PR3 merge commit on `main` (CONTRIBUTING §5 step 5). |

---

## 7. File changes

| File | Action | Slice |
|---|---|---|
| `build-n4-module-kit/retros/*.md` (3) | Create (import) → Modify (marker line 1) | PR1 → PR3 |
| `build-n4-module-kit/types/dashboard.md` | Modify — 8 new sections | PR1 |
| `build-n4-module-kit/types/logic.md` | Modify — 5 new sections, 2 link renames, maturity label | PR1 |
| `build-n4-module-kit/types/wb-widgets.md` | Modify — 2 link renames + See-also | PR1 |
| `build-n4-module-kit/build-verify.md` | Modify — 8 new sections + doctrine | PR1 |
| `build-n4-module-kit/METHODOLOGY.md` | Modify — editing-technique section + gate line | PR1 |
| `build-n4-module-kit/BUILD-LOOP.md` | Modify — preflight sub-step 0.b | PR1 |
| `build-n4-module-kit/SOURCES.md` | Modify — ColdRoomPan path + `docs/` definition | PR1 |
| `build-n4-module-kit/toolbelt/verify-module.sh` | Create | PR2 |
| `build-n4-module-kit/toolbelt/build.sh` | Rewrite | PR2 |
| `build-n4-module-kit/toolbelt/mirror-niagara-home.sh` | Create (promote + guards) | PR2 |
| `build-n4-module-kit/toolbelt/stored-repack.sh` | Create | PR2 |
| `tests/helpers/n4-fixtures.bash` | Create | PR2 |
| `tests/verify-module.bats` | Create | PR2 |
| `tests/build-sh.bats` | Create | PR2 |
| `tests/mirror-niagara-home.bats` | Create | PR2 |
| `tests/stored-repack.bats` | Create | PR2 |
| `tests/kit-links.bats` | Create | PR2 |
| `tests/ng-deploy.bats` | **Untouched** | — |
| `scripts/ng-deploy.sh` | **Untouched** — index already `100755`; no content and no mode change | — |
| `CONTRIBUTING.md` | Modify — bats-core install step | PR2 |
| `build-n4-module-kit/README.md` | Modify — status + layout | PR3 |
| `docs/GOTCHAS.md` | Modify — 3 anti-pattern rows | PR3 |
| `CHANGELOG.md`, `VERSION` | Modify | PR3 |
| `~/.claude/skills/build-n4-module/SKILL.md` | Modify — **outside git**; before/after in the PR3 body + engram | PR3 |

---

## 8. Risks and mitigations

| # | Risk | Mitigation |
|---|---|---|
| R1 | PR1 exceeds the 800-line budget once the ~200 imported retro lines are counted | Import is its own first commit (imported, not authored); `types/dashboard.md` splits into commits 2 and 3 if the first exceeds ~200 lines |
| R2 | `bats` not installed when PR2's gate must run | The `CONTRIBUTING.md` install step is PR2 **commit 1** (documented before used, per resolution); QA installs `bats-core` before authoring |
| R3 | The launcher `SKILL.md` is outside git and not revertible | PR3 body carries the full before/after diff; engram `sdd/build-n4-module-kit-v0.2/launcher-diff` mirrors it; rollback is a manual re-edit |
| R4 | `mirror-niagara-home.sh` wipes a real install | Two independent guards (§2.3), each with its own bats case, written before the promotion |
| R5 | Station-only lessons cannot be reproduced in WSL | Folded with `[CERT-live]` + a bitácora `§` reference; never asserted by a test |
| R6 | A superseded claim gets folded next to its correction | Proposal §4's correction table is the fold-in checklist; the verify phase greps for the superseded strings (§9) |
| R7 | 41 items across 7 files produce an unreviewable diff | §1.0 form rules + one commit per target file + ≤7 bullets per subsection |
| R8 | PR2's diff shows PR1's commits | PR2 branches/rebases only after PR1 merges; `git diff --stat main...HEAD` must show only PR2 files (§4.4 step 3) |
| R9 | The gate rejects a jar a station would accept | Conservative default set (§2.1); `--stored`/`--target-version`/`--src` opt-in |
| **D1** | `kit-links.bats` is red on the un-rebased PR2 branch by construction | Stated as a contract, not a bug: green is required at PR2 **merge** time, on the rebased head (§4.5) |
| **D2** | The forged-class fixture is not portable if written with `printf '\xNN'` | Helpers use octal escapes (`\312\376\272\276`), which are POSIX `printf`; asserted indirectly by V1/V8 |
| **D3** | A working-tree mode drift on `scripts/ng-deploy.sh` is mistaken for a repo defect again and "fixed" with a pointless commit | The index has held `100755` since v0.2.0; the drift is a checkout artefact and is repaired with `git checkout -- scripts/ng-deploy.sh`, never with a commit. The `git ls-files -s` line in §9 is a **drift detector**, not a fix, and `scripts/` is out of this change's file list |
| **D4** | `--src` being opt-in means the type-count and facet checks do not run by default | `build.sh` always passes `--src`, so the recommended WSL path gets the full set; only a bare `verify-module.sh <jar>` runs the reduced set |

---

## 9. Verify-phase checklist (what `sdd-verify` must execute)

Superseded-string greps — all must return **zero matches**, with `retros/` excluded (the retros keep their
original wrong claims verbatim as the audit trail):

```bash
KIT=build-n4-module-kit
grep -rn "transient build state" "$KIT" --include='*.md' | grep -v "$KIT/retros/"
grep -rn "BFrozenEnum"           "$KIT" --include='*.md' | grep -v "$KIT/retros/"
grep -rnE "7\.6\.(1|3|5)\b"      "$KIT" --include='*.md' | grep -v "$KIT/retros/"
grep -rn "checklist-common.md"   "$KIT" --include='*.md'
grep -rn "type-dashboard.md"     "$KIT" --include='*.md'
grep -rn "/mnt/c/modulos_niagara_n4/Cliente/Leon-Guanjuato/Paccadia/ColdRoomPan" "$KIT/SOURCES.md"
grep -rniE "primary:|fallback" "$KIT/build-verify.md" "$KIT/BUILD-LOOP.md"   # F1 framing must be gone
```

- [ ] `bats tests/*.bats` exits 0 across all six suites; `tests/ng-deploy.bats` still reports 26 cases.
- [ ] `shellcheck build-n4-module-kit/toolbelt/*.sh scripts/*.sh tests/*.bats tests/helpers/*.bash` exits 0.
- [ ] `tests/kit-links.bats` green on the rebased PR2 head.
- [ ] `git ls-files -s scripts/ng-deploy.sh` starts with `100755` — **drift check only**; `scripts/` must
      appear in no commit of this change.
- [ ] Each of the 3 retros has `<!-- review-status: folded v0.2 · 2026-09-01 -->` as line 1 and none was deleted.
- [ ] `cat VERSION` = `0.4.0`; `CHANGELOG.md` has a `[v0.4.0]` section with a `### References` block.
- [ ] Kit `README.md` states `v0.2`.
- [ ] Launcher diff recorded: `~/.claude/skills/build-n4-module/SKILL.md` states the three roles
      (`build.sh` = recommended WSL build that runs the gate · `ng-deploy.sh` = station deploy wrapper ·
      `verify-module.sh` = THE gate) and `version: "0.2"`; the before/after block exists in the PR3 body
      **and** in engram. No "primary/fallback" wording survives in the launcher, `build-verify.md` or
      `BUILD-LOOP.md`.
- [ ] `verify-module.sh` run against the real DashboardPan `build/libs` reproduces the QA outcome
      (deflated jars pass by default, fail with `--stored`).
- [ ] `build.sh` on DashboardPan selects `-rt` and `-ux` and skips the `-wb` stub.
- [ ] Each PR merged with `--ff-only`; `git log --merges` on `main` stays empty.

---

## 10. Open questions

None. Proposal §12 froze the option, the slicing, the doctrine wording and Q1–Q6. The coordinator resolved
the `CONTRIBUTING.md` placement, the doctrine's three-role wording, the conservative default check set,
the `[CERT-live]` markers and the one-time catch-up, and — in the design-gate review — the `build.sh` and
`stored-repack.sh` CLIs, the 50 exit code, the `--src` derivation, the `<=` baja operator, the five-suite
test layout, the mirror contract, the `docs/` link-test exemption and the exec-bit non-defect.

**`G8`'s placement** in `types/logic.md` §Safety fail-modes is **confirmed** by the coordinator (the spec
is being corrected to match); it is no longer an open item. The only item decided here without an external
confirmation is the **`logic.md` maturity relabel** SEED → GROWING (§1.2 row 1) — cosmetic, and revertible
in one line if a reviewer disagrees.

## 11. Next

`sdd-tasks` — ordered TDD steps per slice, mapping every §1 subsection, every §2 script function, every
§3 test case and every §5 applicable threat-matrix row to a work unit with its own verification and
rollback boundary.
