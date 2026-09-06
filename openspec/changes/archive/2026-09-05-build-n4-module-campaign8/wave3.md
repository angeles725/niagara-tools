# Wave-3 addendum: build-n4-module-campaign8

**Addendum to**: `spec.md` / `design.md` / `tasks.md` (do not edit those — other workers append to them;
the lead adds pointers at CLOSE). This file is self-contained; later PRs cite it as `wave3.md §…`.
**PRs**: PR16–PR20 + CLOSE addendum · **Branch**: `docs/c8-wave3` on main `d936317`
**Row format**: `PASS|FAIL|WARN|SKIP  <check>  <file>:<line>|<path>  <detail>` (D1 grammar)
**Exits** (new scripts): `0` no FAIL · `1` any FAIL · `3` env/usage (K20 disjoint)
**Sources**: orchestration.md · station-logic.md · wb.md · bog.md · B816 §816.6 · B817 §817.9 ·
B815 §815.12 · B812 · retro `2026-09-05-…-retro-automation-and-campaign8-backlog-retro.md` ·
BUILD-LOOP §7 · METHODOLOGY §Kit maintenance

---

## Spec

### PR16 — feat/c8-retro-loop

**Branch**: `feat/c8-retro-loop` | **QA RED**: `qa/c8-retro-loop` (RL1–RL6)

| ID | Requirement |
|----|-------------|
| R16.1 | `toolbelt/new-retro.sh <module-or-kit> [--ticket]` MUST create `retros/<date>-<slug>.md` from a template (line 1 `<!-- review-status: pending -->`, sections: what happened / evidence / proposed kit deltas / lessons), append exactly one row to `retros/INDEX.md`, and set `retro_pending: true` in `BUILD-STATE.md` — all three in one invocation; re-run MUST be idempotent (no duplicate INDEX row). |
| R16.2 | The stub MUST carry the slug as H1 title and today's ISO date (`date +%Y-%m-%d`). |
| R16.3 | `toolbelt/kit-ticket.sh "<title>" --from <retro-file>` MUST open a GitHub issue in the kit repo (`gh issue create`) with the retro slug as body link plus labels `kit,from-run`; when `gh` is absent or unauthenticated it MUST emit `SKIP  kit-ticket  gh unavailable: <reason>` and exit 0 — NEVER fails a run. |
| R16.4 | `skill/SKILL.md` MUST gain a close-of-run step: "every run ends with `new-retro.sh <kit>`; a defect or kit delta → `kit-ticket.sh`". |
| R16.5 | `BUILD-LOOP.md` §7 MUST be created: "Retro gate — every push range that changes kit code MUST include the retro stub + INDEX row + BUILD-STATE envelope in the same range (CD1); `new-retro.sh` performs all three atomically." |
| R16.6 | Named mutation: omit INDEX duplicate guard → RL2 (idempotent re-run) reveals a second row on the second call — the duplicate detection is the test. |

**RL1-TEMPLATE** GIVEN `new-retro.sh coldroompan`; WHEN it runs; THEN `retros/<date>-coldroompan.md` created, line 1 `<!-- review-status: pending -->`, 4 sections present.

**RL2-IDEMPOTENT** GIVEN `new-retro.sh coldroompan` called twice; WHEN second call runs; THEN `retros/INDEX.md` has exactly one new row.

**RL3-ENVELOPE** GIVEN `new-retro.sh coldroompan`; WHEN it runs; THEN `BUILD-STATE.md` contains `retro_pending: true`.

**RL4-TICKET-SKIP** GIVEN `gh` not in PATH; WHEN `kit-ticket.sh "My fix" --from retros/x.md` runs; THEN exits 0, SKIP row emitted.

---

### PR17 — docs/c8-orchestration

**Branch**: `docs/c8-orchestration` | **Type**: doc-only (CD2 — zero new bats tests for new script)

| ID | Requirement |
|----|-------------|
| R17.1 | `build-n4-module-kit/ORCHESTRATION.md` MUST be created with 8 sections: 1 Roles · 2 Model table (sonnet mechanical / opus schema-new-slot / haiku archive) · 3 Delegation triggers · 4 Escalation gate (multi-PR / new module / S1–S2 schema outage → full gentle SDD via `gentle-sdd-ff`) · 5 Artifact store (ledger, typed gate) · 6 Evidence discipline (RED-first, fold-audit, real-module smoke) · 7 Retro/Ticket loop · 8 Recovery. |
| R17.2 | `skill/SKILL.md` MUST gain steps 1b (Explore shard audit-first >3 files, sonnet), 1c (opus design shard for schema/new slot/facade), 5b (peer QA session before every code PR — RED by branch name K13). |
| R17.3 | Adopt-list from orchestration.md shard MUST appear in ORCHESTRATION.md §5/§8: ledger per work unit, typed phase gate, Judgment-Day for high-risk PRs. |
| R17.4 | `kit-links.bats` L7 extension: every script named in ORCHESTRATION.md exists at `toolbelt/<script>.sh` (resolves once wave-3 merges). |
| R17.5 | `sweep-fold-audit.sh --strict` exits 0; no K19 routing line for a script not yet merged (CD5 ordering). |

---

### PR18 — feat/c8-structure

**Branch**: `feat/c8-structure` | **QA RED**: `qa/c8-structure` (LS1–LS11)

| ID | Requirement |
|----|-------------|
| R18.1 | `types/structure.md` MUST be created: B817 module-structure layout, naming conventions, package map, `module-include.xml` vs hand-authored `META-INF/module.xml`, good-module checklist (L1–L11 summary). |
| R18.2 | `toolbelt/lint-structure.sh <module-root>` MUST implement L1 package naming · L2 one @NiagaraType per file · L3 pure-model package with tests and no baja imports · L4 lexicon non-empty per artifact · L5 palette non-empty for rt · L6 module-include.xml present and consistent · L7 3-part dependency floors · L9 no empty skeleton artifact · L10 no absolute host paths in tracked gradle.properties · L11 mixed pure+Baja srcTest declares both `:test-wb` and junit. L8 (signed-jar) stays in `verify-module.sh`. Row: `FAIL|WARN  lint-structure  <path>  L<n>: <reason>`; exits 0/1/3. |
| R18.3 | Real shapes proven: chihuahua-rt fires L4 (empty lexicon); DashboardPan-wb fires L9 (empty scaffold); client `gradle.properties` fires L10 (absolute Honeywell path); ColdRoomPan/CompPan fire L11 (mixed srcTest without both `:test-wb` and junit decl) — all four as sanitised fixtures (CD4). |
| R18.4 | `toolbelt/scaffold-module.sh` MUST emit a structure that passes L1–L11 (bats pin: `lint-structure.sh` on scaffold output exits 0). |
| R18.5 | Named mutation: remove one dependency's 3-part floor → L7 FAIL on that entry. |

**LS4** GIVEN chihuahua-rt module root (empty `module.lexicon`); WHEN `lint-structure.sh` runs; THEN FAIL `L4: lexicon empty`, exit 1.

**LS9** GIVEN DashboardPan-wb root (0 classes, empty palette); WHEN `lint-structure.sh` runs; THEN FAIL `L9: empty skeleton artifact`, exit 1.

**LS10** GIVEN client `gradle.properties` with `niagara_home=C:\Honeywell\…`; WHEN `lint-structure.sh` runs; THEN FAIL `L10: absolute host path`.

**LS11** GIVEN ColdRoomPan-rt with srcTest mixing BTest and JUnit without both `:test-wb` and `junit` gradle declarations; WHEN `lint-structure.sh` runs; THEN FAIL `L11`.

---

### PR19 — feat/c8-write-path

**Branch**: `feat/c8-write-path` | **QA RED**: `qa/c8-write-path` (WP1–WP6)

| ID | Requirement |
|----|-------------|
| R19.1 | `toolbelt/lint-write-path.sh <module-root> [--bog <config.bog>]` MUST check that every `@NiagaraProperty` with `Flags.OPERATOR` has a row in `docs/write-path-matrix.md` (cols: slot name · writer · timing · test name that exists in `src/test`); a missing row MUST FAIL naming the slot. |
| R19.2 | With `--bog <config.bog>`, every slot that a dashboard/RoomPanel component links INTO an own-module component (sourceSlotName in links targeting own-module prefixes) MUST also have a matrix row; missing MUST FAIL. |
| R19.3 | A mention in a comment line only does not satisfy the matrix row requirement. |
| R19.4 | `types/logic.md` and `types/logic-authoring.md` MUST gain doctrine lines from B816 §816.6: LINK_TARGET writes are ephemeral (overwritten next propagation); callbacks interleave; matrix template; every new line `[ev: corpus B816]`. |
| R19.5 | `bog-audit.sh` CHECK12 (dashboard/servlet-written slot that is a LINK_TARGET → WARN `[ev: corpus B816]`) lands here if PR10 is not yet merged; if PR10 has merged, a bats pin asserts CHECK12 already present (idempotent guard, no re-add). |
| R19.6 | Named mutation: remove one matrix row for a covered OPERATOR slot → that slot now appears in FAIL output. |

**WP1-COVERED** GIVEN client `docs/write-path-matrix.md` with 13 rows covering 13 OPERATOR slots; WHEN `lint-write-path.sh` runs on those slots; THEN exits 0.

**WP2-UNCOVERED** GIVEN same module with 22 dashboard-writable slots, 13 in matrix; WHEN `lint-write-path.sh` runs; THEN FAIL for the 9 uncovered slots, exits 1.

**WP3-BOG** GIVEN `--bog panccadia/config.bog`; WHEN lint runs; THEN FAIL for any bog-linked dashboard slot without a matrix row.

---

### PR20 — feat/c8-station-logic

**Branch**: `feat/c8-station-logic` | **QA RED**: `qa/c8-station-logic` (SL13–SL19)

| ID | Requirement |
|----|-------------|
| R20.1 | Extend `toolbelt/bog-audit.sh` with CHECK13–CHECK19 (rows: `<CHECK_ID>  PASS|FAIL|WARN  <component-path>  <detail>`, D10 grammar). |
| R20.2 | **CHECK13 relay-double-source** FAIL: two distinct source components link to the same output relay slot on a single device point. |
| R20.3 | **CHECK14 own-output-unlinked** WARN: a module output slot with `Flags.OPERATOR` has no outgoing relay link; suppressed for defrost-specific outputs when `hasDefrost=false` on the parent. |
| R20.4 | **CHECK15 sensor-crossed-by-name** WARN: a slot name containing `C{n}` sources a component whose name suffix carries a different unit index `E{m}` (C-room / E-unit label mismatch). |
| R20.5 | **CHECK16 hasDefrost↔DefrostController sibling** FAIL: a component has `hasDefrost=true` but no `DefrostController` sibling under the same parent path, or vice versa. |
| R20.6 | **CHECK17 roomN-index-mismatch** FAIL: a ColdRoom component's HOA/state/freeze link names carry a numeric suffix inconsistent with the room's own suffix (e.g. ColdRoom_1 whose links address `evap3*`). |
| R20.7 | **CHECK18 dashboard tile-number consistency** FAIL: for a given evaporator unit index, the HOA slot number, state slot number, and freeze slot number do not all agree (the real Cuarto 1 crossing — EvapUnit_1 HOA links `evap3*` while freeze links `evap1*`). |
| R20.8 | **CHECK19 link-direction** WARN: a link from a control component targets a config/panel slot (reverse direction — expected: config panel→control for setpoints, control→panel for state). |
| R20.9 | Named mutations per check: CHECK13 add second source → FAIL; CHECK14 remove relay link → WARN; CHECK16 set `hasDefrost=true` without sibling → FAIL; CHECK18 swap freeze links to mismatch tile → FAIL. |

**SL-PANCCADIA** GIVEN PANCCADIA `config.bog` (ColdRoom_1 units 1 and 3 with swapped HOA/freeze sets; ColdRoom_5/EvaporatorUnit2 evapOut unlinked); WHEN `bog-audit.sh` runs with CHECK13–CHECK19; THEN CHECK18 FAIL on ColdRoom_1 EvaporatorUnit_1 and EvaporatorUnit_3; CHECK14 WARN on ColdRoom_5/EvaporatorUnit2; all other new checks clean; exits 1.

**SL-SYNTHETIC** GIVEN per-check synthetic fixtures (one relay double-sourced CHECK13, one unlinked output CHECK14, one room-index mismatch CHECK17); WHEN `bog-audit.sh` runs; THEN FAIL CHECK13 and CHECK17; WARN CHECK14; exits 1.

---

## Design Decisions (D13–D17)

### D13 — new-retro.sh: atomic triple write + idempotent INDEX guard

**Chosen**: `new-retro.sh` performs three writes in one call (stub, INDEX row, BUILD-STATE flag); idempotency
is enforced by grepping the INDEX for date+slug before appending — a partial second call is a no-op on an
existing row. `kit-ticket.sh` is a separate script: not all retros become tickets, and coupling ticket
creation to stub creation would fail runs on unauthenticated CI boxes. The SKIP exit (R16.3) mirrors the
K3 principle. Rejected: baking `gh` inside `new-retro.sh`.

### D14 — ORCHESTRATION.md: doc-only, L7 resolved after wave-3 merges

`ORCHESTRATION.md` is a doc artifact only; no new toolbelt script ships with PR17. `kit-links.bats` L7
asserts that every script *named* in ORCHESTRATION.md exists — this bats extension is the only test
addition and is structural (analogous to PR13's `kit-links.bats` L4/L5 additions). SKILL.md steps 1b/1c/5b
are doc-additive and renumber no existing step. Rejected: merging ORCHESTRATION.md into METHODOLOGY.md
— orchestration is a session-level contract distinct from per-module build methodology.

### D15 — lint-structure.sh: standalone, source-tree-only; L8 stays in verify-module.sh

`lint-structure.sh` iterates all profiles under `<module-root>` (finds every `module-include.xml`);
source-tree-only, no jar needed. L8 (signed-jar check) needs the built jar and belongs in
`verify-module.sh`. `scaffold-module.sh` emitting a passing structure is the self-hosting proof (the
scaffold fixture IS the test). Rejected: extending `verify-module.sh --src` — the check family (source
layout) is orthogonal to facet/coverage checks.

### D16 — lint-write-path.sh: matrix-row presence only; CHECK12 is bog-audit's concern

`lint-write-path.sh` checks row presence only (slot name → row exists); it does not validate test
existence at lint time (a missing test is caught by `bats`). `--bog` adds link-traced dashboard slots to
the required set via bog parsing (same python3 stub as CHECK7 — imported as a subprocess helper, not
re-implemented). CHECK12 (`bog-audit.sh`, LINK_TARGET WARN) is complementary: it reports a write-hazard,
not a missing matrix row — no overlap. Doctrine lines in `types/logic.md` + `types/logic-authoring.md`
carry `[ev: corpus B816]`; `lint-write-path.sh` routing tagged `[ev: retro c8-write-path]` (CD5).

### D17 — CHECK13–CHECK19 appended to bog-audit.sh python3 block; no new parser state

The handle-graph and link-graph built by CHECK7–CHECK11 are already in memory. CHECK13–CHECK19 are
post-processing passes over those structures: CHECK14 reads the component's own `hasDefrost` slot value
(already in the parsed component dict); CHECK15/CHECK17/CHECK18 are name-pattern matches over the link
set; CHECK16 reuses the component-index from CHECK9 (sibling lookup). No new bog grammar needed.
Rejected: a separate `station-logic-check.sh` — it would re-parse a 68k-line bog and duplicate the
python3 invocation that already costs ~0.1 s.

---

## Tasks

### PR16 — feat/c8-retro-loop (~120 authored, ledger 260)

**RED**: `qa/c8-retro-loop` (RL1–RL6) — re-read tip at apply | **Gate**: R16.1–R16.6, CD1/CD5/CD6

- [x] 16.1 Create `qa/c8-retro-loop`; write failing bats: RL1 template shape, RL2 idempotent INDEX, RL3 envelope flag, RL4 gh absent → SKIP + exit 0; record tip SHA.
- [x] 16.2 Write `toolbelt/new-retro.sh`: template emit; idempotent INDEX `grep` guard; BUILD-STATE `retro_pending: true` sed-in-place; `set -u`; VCS-free; `shellcheck 0.10.0` clean; exits 0/3.
- [x] 16.3 Write `toolbelt/kit-ticket.sh "<title>" --from <retro-file>`: `command -v gh || { printf …SKIP…; exit 0; }`; `gh issue create --repo <kit-remote> … --label kit,from-run`; exits 0/3.
- [x] 16.4 Write `tests/retro-loop.bats` (RL1–RL6 verbatim from RED).
- [x] 16.5 Add K19 routing: `BUILD-LOOP.md` §7 retro gate (R16.5) + `skill/SKILL.md` close-of-run step (R16.4); `new-retro.sh [ev: retro c8-retro-loop]`; `kit-ticket.sh [ev: retro c8-retro-loop]` (CD5).
- [x] 16.6 **Named mutation**: omit INDEX duplicate guard → second call produces duplicate row (R16.6). Record in PR body.
- [x] 16.7 Real smoke: `new-retro.sh build-n4-module-kit` → stub created + INDEX row appended + BUILD-STATE flipped; second call → no duplicate. Paste output.
- [x] 16.8 Guards: bats all green; `shellcheck` exit 0; `sweep-build-state.sh`; `sweep-fold-audit.sh --strict`; `kit-links.bats` L4/L5.
- [x] 16.9 Retro + INDEX row + `BUILD-STATE.md` envelope in same push range (CD1); update openspec `apply-progress.md`.
- [ ] **[lead]** RL1–RL6 green; merge ff-only; `install-skill.sh --force` (SKILL.md changed, D12); ledger settle `--max-changed-lines 260`.

---

### PR17 — docs/c8-orchestration (~70 authored, ledger 180, doc-only)

**RED**: none (CD2) | **Gate**: R17.1–R17.5

- [x] 17.1 grep-before-fold: `rg '8 sections|Delegation triggers|Escalation gate|Judgment-Day.*high-risk' build-n4-module-kit/` → confirm 0 hits (K6).
- [x] 17.2 Create `build-n4-module-kit/ORCHESTRATION.md` (8 sections per R17.1 and orchestration.md shard); model table; escalation gate (multi-PR / new module / S1–S2 → `gentle-sdd-ff`); adopt-list (ledger, typed gate, JD); keep-from-kit list (RED-first, real-module smokes, fold-audit).
- [x] 17.3 Extend `skill/SKILL.md`: step 1b (Explore shard >3 files, sonnet), step 1c (opus design shard for schema/new slot), step 5b (peer QA session — RED by branch name K13).
- [x] 17.4 Extend `tests/kit-links.bats` L7: assert every script named in ORCHESTRATION.md exists at `toolbelt/<script>.sh`.
- [x] 17.5 `sweep-fold-audit.sh --strict` exits 0; `kit-links.bats` (incl. L7) exits 0.
- [x] 17.6 Retro + INDEX row + `BUILD-STATE.md` envelope (CD1); update openspec `apply-progress.md`.
- [ ] **[lead]** Guards green; merge ff-only; `install-skill.sh --force` (SKILL.md changed); ledger settle `--max-changed-lines 180`.

---

### PR18 — feat/c8-structure (~130 authored, ledger 290)

**RED**: `qa/c8-structure` (LS1–LS11) — re-read tip at apply | **Gate**: R18.1–R18.5, CD4/CD5/CD6

- [x] 18.1 Create `qa/c8-structure`; write failing bats: LS4 (chihuahua L4), LS9 (DashboardPan-wb L9), LS10 (client gradle.properties L10), LS11 (ColdRoomPan/CompPan L11), LS-PASS (scaffold output → exit 0); record tip SHA.
- [x] 18.2 Create `types/structure.md` (B817 layout + naming + good-module checklist L1–L11 summary; `[ev: corpus B817]`).
- [x] 18.3 Write `toolbelt/lint-structure.sh <module-root>`: iterate profiles; implement L1–L7, L9–L11 per R18.2; `set -u`; VCS-free; `shellcheck 0.10.0` clean; exits 0/1/3.
- [x] 18.4 Copy sanitised fixtures `tests/fixtures/lint-structure/{chihuahua-rt,dashboardpan-wb,client-gradleprops,coldroompan-rt}/` (real shapes, CD4).
- [x] 18.5 Write `tests/lint-structure.bats` (LS1–LS11 verbatim from RED).
- [x] 18.6 Update `toolbelt/scaffold-module.sh` to emit L1–L11-passing structure; add scaffold-output bats pin.
- [x] 18.7 Add K19 routing: `BUILD-LOOP.md` + `skill/SKILL.md` `lint-structure.sh [ev: retro c8-structure]` (CD5).
- [x] 18.8 **Named mutation**: remove one dependency 3-part floor → L7 FAIL on that entry (R18.5). Record in PR body.
- [x] 18.9 Real smoke: `lint-structure.sh` on chihuahua (L4 fires) + DashboardPan-wb (L9 fires); `lint-structure.sh` on scaffold output → exits 0. Paste output.
- [x] 18.10 Guards: bats all green; `shellcheck` exit 0; `sweep-build-state.sh`; `sweep-fold-audit.sh --strict`; `kit-links.bats` L4/L5.
- [x] 18.11 Retro + INDEX row + `BUILD-STATE.md` envelope (CD1); update openspec `apply-progress.md`.
- [ ] **[lead]** LS1–LS11 green; merge ff-only; `install-skill.sh --force` (SKILL.md changed); ledger settle `--max-changed-lines 290`.

---

### PR19 — feat/c8-write-path (~115 authored, ledger 250)

**RED**: `qa/c8-write-path` (WP1–WP6) — re-read tip at apply | **Gate**: R19.1–R19.6, CD4/CD5/CD6

- [x] 19.1 Create `qa/c8-write-path`; write failing bats: WP1 (covered slot → exit 0), WP2 (9 uncovered → FAIL), WP3 (`--bog` adds bog-linked slots → FAIL if uncovered), WP4 (comment-only mention not a valid row), WP5 (named mutation: remove row → FAIL); record tip SHA.
- [x] 19.2 Write `toolbelt/lint-write-path.sh <module-root> [--bog <config.bog>] [--matrix <path>]`: scan `@NiagaraProperty` with `Flags.OPERATOR`; matrix resolution walks up to vcs root (never silent exit 0 when absent; ERROR + exit 3); `--matrix` override; `--bog` adds link-traced dashboard slots; `FAIL  lint-write-path  <module>  slot <name>: no matrix row`; `set -u`; VCS-free; `shellcheck 0.10.0` clean; exits 0/1/3. WP7/WP8 pins prove walk-up.
- [x] 19.3 Extend `types/logic.md` §write-path and `types/logic-authoring.md` §write-path-matrix: LINK_TARGET ephemeral writes, callback interleave, matrix template; every new line `[ev: corpus B816]`.
- [x] 19.4 CHECK12 guard: if PR10 not yet merged, add CHECK12 to `bog-audit.sh` here; if merged, add bats pin asserting CHECK12 present (idempotent guard — no re-add).
- [x] 19.5 **Named mutation**: remove one matrix row for a covered OPERATOR slot → that slot appears in FAIL output (R19.6). Record in PR body.
- [x] 19.6 Real smoke: `lint-write-path.sh` on client module (CompPan or ColdRoomPan) → FAIL for 9 uncovered slots, exits 1; covered-only fixture → exits 0. Paste output.
- [x] 19.7 Guards: bats all green; `shellcheck` exit 0; `sweep-build-state.sh`; `sweep-fold-audit.sh --strict`; `kit-links.bats` L4/L5.
- [x] 19.8 Retro + INDEX row + `BUILD-STATE.md` envelope (CD1); update openspec `apply-progress.md`.
- [ ] **[lead]** WP1–WP6 green; merge ff-only; `install-skill.sh --force` (SKILL.md changed); ledger settle `--max-changed-lines 250`.

---

### PR20 — feat/c8-station-logic (~145 authored, ledger 320)

**RED**: `qa/c8-station-logic` (SL13–SL19) — re-read tip at apply | **Gate**: R20.1–R20.9, SC16

- [x] 20.1 Create `qa/c8-station-logic`; write failing bats: SL13–SL19 (one per check + per-mutation); record tip SHA.
- [x] 20.2 Extend `toolbelt/bog-audit.sh` python3 block: append CHECK13–CHECK19 as post-processing passes over the existing handle-graph and link-graph (no new parser state, D17); CHECK14 reads `hasDefrost` from parsed component dict; CHECK16 reuses component-index from CHECK9.
- [x] 20.3 Write synthetic per-check fixtures `tests/fixtures/bog/station-logic-CHECK{13..19}.bog` + named-mutation variants; clean fixture exits 0. No customer config.bog committed (D11).
- [x] 20.4 Write `tests/bog-audit.bats` extension: SL13–SL19 pins verbatim from RED + per-check mutations (R20.9).
- [x] 20.5 **Named mutations**: CHECK13 add second source → FAIL; CHECK18 swap freeze links → FAIL; CHECK16 set `hasDefrost=true` without sibling → FAIL; CHECK14 remove relay link → WARN. Record in PR body.
- [x] 20.6 Real smoke: `bog-audit.sh` on PANCCADIA `config.bog` → CHECK18 FAIL ColdRoom_1 EvaporatorUnit_1 and EvaporatorUnit_3; CHECK14 WARN ColdRoom_5/EvaporatorUnit2 evapOut; all other new checks clean; exits 1. Paste output.
- [x] 20.7 Guards: bats all green; `shellcheck` exit 0 (bash driver unchanged, only python3 inner block extended); `sweep-build-state.sh`; `sweep-fold-audit.sh --strict`; `kit-links.bats` L4/L5.
- [x] 20.8 Retro + INDEX row + `BUILD-STATE.md` envelope (CD1); update openspec `apply-progress.md`.
- [ ] **[lead]** SL13–SL19 green; merge ff-only; ledger settle `--max-changed-lines 320`. No install-skill (SKILL.md unchanged).

---

## Wave-3 PR Matrix

| Unit | PR | Goal | Ledger | Focused test | Rollback |
|------|----|------|--------|-------------|---------|
| 16 | PR16 | `new-retro.sh` + `kit-ticket.sh` | 260 | `bats tests/retro-loop.bats` | new scripts; `git revert` |
| 17 | PR17 | `ORCHESTRATION.md` + SKILL.md steps 1b/1c/5b | 180 | `kit-links.bats` L7 + `sweep-fold-audit` | doc-only; `git revert` |
| 18 | PR18 | `lint-structure.sh` + `types/structure.md` | 290 | `bats tests/lint-structure.bats` | new script + doc; `git revert` |
| 19 | PR19 | `lint-write-path.sh` + write-path doctrine | 250 | `bats tests/lint-write-path.bats` | new script + doc; `git revert` |
| 20 | PR20 | `bog-audit.sh` CHECK13–CHECK19 | 320 | `bats tests/bog-audit.bats` | additive python3 block; `git revert` |

---

## Success Criteria (wave 3)

| ID | Assertion |
|----|-----------|
| SC12 | `new-retro.sh` creates stub + INDEX row + BUILD-STATE flag atomically; idempotent on re-run; `kit-ticket.sh` exits 0 with SKIP row when `gh` absent; RL1–RL6 green. |
| SC13 | `ORCHESTRATION.md` present with 8 sections; SKILL.md has steps 1b/1c/5b; `kit-links.bats` L7 green; `sweep-fold-audit.sh --strict` exits 0. |
| SC14 | `lint-structure.sh` fires L4 on chihuahua, L9 on DashboardPan-wb, L10 on client gradle.properties, L11 on ColdRoomPan/CompPan; scaffold output passes L1–L11; LS1–LS11 green. |
| SC15 | `lint-write-path.sh` exits 1 on client module with 9 uncovered OPERATOR slots; exits 0 on fully-covered fixture; `types/logic.md` and `types/logic-authoring.md` write-path lines carry `[ev: corpus B816]`; WP1–WP6 green. |
| SC16 | `bog-audit.sh` CHECK13–CHECK19 emit CHECK18 FAIL on ColdRoom_1 units 1 and 3 of PANCCADIA bog; CHECK14 WARN on ColdRoom_5/EvaporatorUnit2 evapOut; all other new checks clean; per-check synthetic fixtures + named mutations proven; SL13–SL19 green. |

---

## CLOSE addendum

**Retros pending**: each of PR16–PR20 writes its retro via `new-retro.sh` in the same push range (CD1) → `retros/INDEX.md` pending = 0 at CLOSE for all wave-3 entries. The existing C.2 gate (`grep -c '| pending |' retros/INDEX.md` == 0) applies unchanged.

**VERSION**: `0.19.0` covers all waves if PR16–PR20 merge before CLOSE (no separate version bump for wave 3). If wave 3 merges after a separate wave-1/2 CLOSE, bump to `0.20.0` in a follow-up PR (single CHANGELOG entry covering PR16–PR20 + a `## [v0.20.0]` header). CHANGELOG entries per PR under the current unreleased block.

**Campaign-9 seed** (create `niagara-research/campaign9-research-candidates.md` at CLOSE):
- *Latch/SR and heartbeat PoC (B812 + B815 §815.12)* — build half CERT-live (2026-09-05); run half open: invoke `test.exe` via WSL interop against a disposable Windows-side copy of the install (`C:\niagara-test-414`, never the live OptimizerSupervisor dir); RED→GREEN proof for B815-G1; blocks niagaraTest doctrine (PR15 R15.1/R15.2) moving from OPEN to CLOSED.
- *Write-path matrix completion* — 9 uncovered OPERATOR slots (W14–W22); pure-seam test cases per the lint-write-path RED; closes the client residue noted in B816 §816.6.
- *Structure lint false-positive tuning* — run `lint-structure.sh` on additional client repos with non-standard layouts; capture new L1–L11 edge cases as a follow-up MINOR PR.
