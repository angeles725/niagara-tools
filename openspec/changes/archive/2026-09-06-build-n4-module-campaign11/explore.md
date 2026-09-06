# Exploration — build-n4-module-campaign11

**Date**: 2026-09-07 | **Phase**: explore | **Status**: complete
**Engram**: `sdd/build-n4-module-campaign11/explore` (observation 8404)
**Kit**: v0.21.0 (tag dab0807; C10 archived at 154803c) | **Client modules** (client main 00e7118): CompPan-rt 2.2.0, ColdRoomPan-rt 2.1.0, DashboardPan rt/ux 2.2.0 — C10 added no client jar
**Tunnel**: PRs #1/#2/#3 (config login, change_log, audit mirror) blessed, NOT merged
**Station (PANCCADIA)**: still running ColdRoomPan-rt 2.0.3 / CompPan-rt 2.0.1 / DashboardPan 2.0 — four repo versions behind; harness session owed
**Read trees**: `Leon-Guanjuato-worktrees/main-ff1b659` (frozen, the bats default) and `main-00e7118` (post-PR6, STALE = 0); the live checkout `Cliente/Leon-Guanjuato` is at 4f5f1c7 with Cristian's uncommitted files and is NEVER read
**Source**: consolidates companero `2026-09-07-c11-explore-draft.md` (9d4024ad0 + e6666eeb9) + `2026-09-07-c11-t1-t4-apply-packages.md` (4ef4f864c, 8ad4bb36e, T2 re-cuts 17306e9b9 / 0ad09c658) + before-C11 checklist + investigador1 B832 `niagara-mental-model-bloque832.md` (593019540) + C11 draft second read (29d203e3d) + METHODOLOGY K24 at dab0807 + QA REDs `qa/c11-parser-oneliner` d88af78 and `qa/c11-client-root` 011d127 (widening in progress).

> Shape mirrors `openspec/changes/archive/2026-09-06-build-n4-module-campaign10/explore.md`. Every claim carries an `[ev:]` token. SHAs are research-repo (niagara-research) unless noted `[client]`, `[kit]` or `[tunnel]`.

---

## 1. Mandate

### 1.1 Confirmed product decisions (constraints — do NOT re-open)

1. **Keep improving.** C11 is authorized. It opens with the kit wave W1 = T1 (keystone shared parser) → T3 (concept-row-drift lint) → T2 (centralise client-tree defaults) → T4 (guard-pins meta-check), WSL-only, nothing from Cristian. `[ev: Cristian standing order 2026-09-06 "sigan trabajando … preparar el terreno"]`
2. **T1 is the keystone and ships as ONE PR.** The shared fragment replaces three copies at once; T3/T2/T4 follow. `[ev: 9d4024ad0 §ordering; B832 §shipping]`
3. **Deploy chain and the niagaraTest harness session are PREREQUISITES, not C11 work.** Kit lanes need neither. `[ev: before-c11-checklist 4ef4f864c]`
4. **Product seeds P1-P5 are CONDITIONAL (W2)** — gated on tunnel merge + deploy chain + harness session + Cristian's station decisions (P2 is a per-operator VIEW/RBAC question; attribution already shipped as R14). `[ev: 29d203e3d; c10 archive explore §4 W2]`
5. **Research SELECTED for T1 and DELIVERED as B832** (open gaps B832-G1 accessor-skip pin, B832-G2 silent-protection Case-B strips `//` only). T2/T3/T4 are kit-internal, no external research. `[ev: B832 593019540]`
6. **Process constraints from C10 carry over (K24 + retros):** pin-attribution rule (every OBSERVED mutation names the fixture it flips, QA confirms); the kit is ff-only → REDs are cherry-picked, never merged; fragment-merge of the four always-conflict files; K12/K13; executors read the origin tip, never a peer worktree; workers never write in the main checkout (tasks.md ticks only in their worktree); one `[ev:]` per paragraph. `[ev: METHODOLOGY.md K24 @ dab0807; retros/2026-09-06-campaign10-close-process-meta-lessons.md]`

---

## 2. Current State

| Axis | State at C11 open | Token |
|---|---|---|
| Kit | v0.21.0 (dab0807; archive 154803c) — C10 lints landed (S21-S25), 6 retros folded, issue #89 closed; c10-close 11/12 (CLOSE-harness-run pending) | `[kit]` `[ev: BUILD-STATE.md @ dab0807]` |
| T1 live FN | The three section-D parser copies disagree on one-liner methods: lint-timers (:188-202) and lint-silent-protection (:302-364) gate the method open on NET brace depth and miss `void arm(){ flag=true; Clock.schedule(...); }`; lint-ext-writable-shape (:132-176) uses PEAK max_d and catches it. Real trees do not flip today | `[kit]` `[ev: B832 593019540; QA d88af78]` |
| T2 hazard | 10 bats carry an absolute Leon-Guanjuato path: 5 × C9_CLIENT_ROOT (ext-writable-shape:26, demand-in-scope:27, lint-silent-protection:30, lint-timers:418, lint-write-path:338), 2 × C9_CLIENT_REPO (c9-close:108, c10-close:90), 3 reading the LIVE checkout (c8-close:107 C8_CLIENT_REPO, lint-delays:53, rc-scan:75 — override-less) — the live checkout is 4f5f1c7, a pre-C9 tree | `[kit]` `[ev: 0ad09c658; QA 011d127; lead check 2026-09-07]` |
| Client | 00e7118 = ff1b659 + PR6 (gitignore + five `[concept]` rows); versions unchanged | `[client]` |
| Tunnel / Station | unchanged since C10 (PRs unmerged; station 2.0.3/2.0.1/2.0; deploy runbook not executed) | `[tunnel]` |

---

## 3. Evidence Inventory

### 3.1 Research blocks

| Block | SHA | One-line finding | Token |
|---|---|---|---|
| B820–B831 | see C10 archive explore §3.1 | C10 research set, folded | `[CERT]` |
| **B832** | `593019540` | The three parser copies are NOT equivalent (NET vs PEAK depth; one-liner FN). Canonical = lint-ext-writable-shape :132-176. Invariants for the shared fragment: `brace_depth >= 2` guard; Case-B backward scan stops at any line starting with `@` (a single-line annotation with a constructor default never misparses, a multi-line one relies on the guard); keyword exclusion; PEAK depth + one-line getter/setter skip; `/* */` strip. Golden set: BMisparse multi-line, anyNoHardware same-method local, CP-1 adapter, one-liner, accessor. Gaps B832-G1/G2 | `[CERT]` |

### 3.2 QA RED branches (kit, origin) — the contract per K13

| Branch | Tip | Mapping | Status |
|---|---|---|---|
| `qa/c11-parser-oneliner` | `d88af78` | T1 — `tests/parser-oneliner.bats`: C11-tl-oneliner (FIELD flag + schedule in a one-liner `arm()` → must FAIL; dab0807 exit 0), C11-sp-oneliner (detected trip in a one-liner `step()` → must WARN; dab0807 0), C11-g1-setter (one-liner accessor `setInhibited` with a guarded write → 0 WARN; proven to flip a PEAK-only fix without the get/set/is skip). B832-G2 as a header NOTE, no vacuous pin | RED verified RED-for-the-right-reason on dab0807 |
| `qa/c11-client-root` | `011d127` → widening | T2 — C11-T2-lib-exists (`tests/lib/client-root.bash`), C11-T2-no-hardcode (no absolute Leon-Guanjuato literal in tests/*.bats outside the lib): 7 today, being widened to 10 (path pattern, not variable name); the three tail smokes re-measured on main-ff1b659 | RED; widened tip pending |
| `qa/c11-golden-parser` | being authored | T1 golden set across the three lints (five cases) | pending |
| `qa/c11-concept-drift` | being authored | T3 pos/neg/decoy + real-tree | pending |
| `qa/c11-guard-pins` | being authored | T4 pos/neg | pending |

### 3.3 Apply-packages (companero, cut against dab0807)

- **T1** (4ef4f864c): fragment `toolbelt/lib/method-boundary.sh` (awk-in-shell-var, the kit idiom) sourced by the three lints; canonical = ext-writable's peak-depth :132-176; replace lint-timers :188-202/:202 and lint-silent-protection :302-364/:326; adopt PEAK depth; add the get/set/is accessor skip (B832-G1) and the `/* */` strip in the Case-B backward scan (B832-G2); golden set must include the one-liner. `[CERT]`
- **T2** (0ad09c658): `tests/lib/client-root.bash` owns ONE default (`CLIENT_READ_ROOT` = main-ff1b659) exported as C9_CLIENT_ROOT, C9_CLIENT_REPO and C8_CLIENT_REPO; env override wins; 10 sites converted (lint-delays:53 and rc-scan:75 gain the override form they lack). Decision: single default, no post-PR6 second default; the three live-checkout reads are a rule violation, not a feature. `[CERT]`
- **T3** (4ef4f864c): concept-row-drift = the INVERSE of S25 STALE — a `[concept]`-marked matrix row whose backtick-inner slot name IS now in the covered set (all @NiagaraProperty ∪ @NiagaraAction ∪ --bog, matrix-root-wide) is a stale marker → advisory row (STATUS-first grammar, matrix line), default exit 0, `--strict` → 1, FAIL untouched; seam = the S25 row pass at lint-write-path.sh:422-458. A true concept row (name absent from source) stays silent. `[ev: 4ef4f864c §T3]`
- **T4** (8ad4bb36e): `toolbelt/lint-guard-pins.sh` — meta-check that every OBSERVED mutation named in a lint's header (the `# Mutation:` / runbook contract) maps to an existing bats fixture name; WARN-only default, `--strict` → 1, exit 3 usage; K19 routing. Exact header grammar to be fixed in design. `[ev: 8ad4bb36e §T4; K24(7)]`

---

## 4. Candidate Slices — Waves

### Wave 1: T1 → T3 → T2 → T4 (kit-only)

| Slice | Class | Repo | Seam | RED status | `[ev:]` |
|---|---|---|---|---|---|
| **T1** shared method-boundary parser (keystone, ONE PR touching three lints) | KIT | `niagara-tools` | NEW `toolbelt/lib/method-boundary.sh`; lint-timers :188-202, lint-silent-protection :302-364, lint-ext-writable-shape :132-176 source it; PEAK depth; invariants: depth guard, Case-B `@`-stop, keyword exclusion, accessor skip, `/* */` strip | `qa/c11-parser-oneliner` d88af78 + `qa/c11-golden-parser` (pending); all three lint suites + golden set green in one PR; real-tree baselines recorded before/after | `[ev: B832]` `[ev: 4ef4f864c §T1]` |
| **T3** concept-row-drift advisory | KIT | `niagara-tools` | `toolbelt/lint-write-path.sh` :422-458 row pass; inverse of STALE; grammar STATUS-first with matrix line; exit 0 / --strict 1 / FAIL untouched | `qa/c11-concept-drift` (pending); real-tree at 00e7118 expected 0 DRIFT (five concept rows have no source slot) | `[ev: 4ef4f864c §T3]` |
| **T2** centralise client-tree defaults | KIT/test | `niagara-tools` | NEW `tests/lib/client-root.bash`; 10 sites converted; single default main-ff1b659 | `qa/c11-client-root` 011d127 → widened (10 → 0) | `[ev: 0ad09c658]` `[ev: QA 011d127]` |
| **T4** guard-pins meta-check | KIT | `niagara-tools` | NEW `toolbelt/lint-guard-pins.sh`; scans lint headers for named mutations and tests/*.bats for fixture names; D9b prune; K19 routing | `qa/c11-guard-pins` (pending) | `[ev: 8ad4bb36e §T4]` |

### Wave 2: Product seeds (GATED — not committed scope)

| Slice | Gate | `[ev:]` |
|---|---|---|
| P1 viewer per-user re-auth + role list | Tunnel PRs #1-#3 merged + Supabase re-check | `[ev: c10 archive explore §4]` |
| P2 HMI per-operator screens (RBAC/VIEW) | Cristian confirms the need (attribution already = R14) + harness green | `[ev: 29d203e3d]` |
| P3 `airDefrost` flag | defrost trial green light | `[ev: c10 archive explore §4]` |
| P4 intercambiador Cuarto 3 | Cristian: on a Niagara output? | `[ev: c10 archive explore §4]` |
| P5 `coolOnSensorFault` link | Cristian approves | `[ev: c10 archive explore §4]` |

---

## 5. Risks

1. **T1 three-file cut in one PR.** A partial cut leaves the toolbelt inconsistent; the golden set must run against all three lints in the same PR, with real-tree baselines (CompPan-rt / ColdRoomPan-rt / DashboardPan-rt for the three lints) recorded at dab0807 and identical after the cut. `[ev: B832 §shipping]`
2. **Accessor skip is new.** A too-broad `get|set|is` match could skip a real method (e.g. `isDirty()` that schedules); C11-g1-setter is the guard and the OBSERVED flip is mandatory. `[ev: B832-G1; QA d88af78]`
3. **`/* */` strip may shift counts** in lint-silent-protection's Case-B scan; baselines before/after. QA could not build a biting fixture for B832-G2 — it stays a header note, not a vacuous pin. `[ev: B832-G2; QA d88af78]`
   **3b. lint-silent-protection.sh:303-307 documents its NET depth as intentional** ("correctly skips single-line methods … preventing the method-open event from being attached to a post-close depth"). After the PEAK cut that comment argues against its own code and its rationale is wrong — the runaway-span fear is prevented by the `brace_depth >= 2` guard + close detection, not by skipping one-liners. The T1 design must replace that comment when migrating to the fragment. On the current corpus (42 client .java at ff1b659) there are 0 one-liner methods with a schedule, alarm or trip write, so the cut changes no real-tree count — the one-liner behaviour is pinned only by the golden fixtures, as designed. `[ev: investigador1 6f0069155]`
4. **T2 retargets three smokes from the stale live checkout to ff1b659** — their expected numbers may change; the RED pins the ff1b659 numbers and records the delta. `[ev: lead check 4f5f1c7]`
5. **Always-conflict files** (BUILD-LOOP, SKILL, INDEX, BUILD-STATE): fragment-merge, never overwrite; the rebase can drop a BUILD-STATE line — the pre-push hook catches it. `[ev: K24; C10 PR2 merge]`
6. **Pin attribution** (K24(7)): every mutation in a lead gate names its fixture; QA confirms — C10 caught three unpinned guards this way. `[ev: K24]`
7. **P1-P5 scope creep** if a station gate resolves mid-W1 — queue for W2. `[ev: c10 archive explore §5]`

---

## 6. Open Questions — Requires-Execution Gates + Cristian Decisions

### 6.1 Requires-execution (WSL, during W1 apply)

| Gate | Question |
|---|---|
| T1 baselines | three lints × three modules at dab0807 before the cut == after the cut; golden set five cases green on all three lints; one-liner fixtures flip from RED |
| T1 accessor | dropping the accessor skip → C11-g1-setter FALSE-WARNs (OBSERVED) |
| T3 real tree | 0 DRIFT at 00e7118; a synthetic marked row with a present slot → 1 DRIFT; decoy in a comment → none |
| T2 | 10 → 0 offenders; suite green with env unset; the three retargeted smokes' ff1b659 numbers pinned |
| T4 | on dab0807 + T1..T3: every named mutation in the lint headers maps to a fixture (expected 0 WARN, else the list is the finding) |

### 6.2 Cristian's decisions (block W2, never W1)

| Decision | Unblocks |
|---|---|
| Tunnel PRs #1-#3 merge | P1/P2 |
| Deploy chain (2.0.7/2.0.3/2.1.1, then C9 jars) | all client jars |
| niagaraTest harness session (Windows; `qa/c9-harness-runsheet.md`) | C9/C10 CLOSE-harness-run; alarm REDs |
| P2: per-operator screens needed? (attribution is done) | P2 |
| Defrost trial green light | P3 |
| Intercambiador Cuarto 3 on a Niagara output? | P4 |
| coolOnSensorFault station link | P5 |

---

## 7. Research Gate Answer

**Research**: SELECTED for T1 — DELIVERED (B832). T2/T3/T4 kit-internal. Propose may start.

**Product decisions**: W1 CONFIRMED (T1 → T3 → T2 → T4). W2 gated.

**Next recommended**: `sdd-propose`

---

## Key Learnings

1. NET and PEAK brace depth differ only on same-line `{ … }` methods, invisible to every golden case until a one-liner is added.
2. A shared parser must be cut in all three lints in one PR, or the golden set exposes the inconsistency across lint boundaries.
3. T2 is 10 sites, not 5: two close bats carry a second variable and three tests still read the stale live checkout.
4. The accessor skip is a new invariant; its pin needs an OBSERVED flip, not a positive pass.
5. DRIFT (T3) is STALE's inverse and lives in the same lint under the same `--strict`.
