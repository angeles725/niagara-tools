<!-- review-status: pending -->

# Campaign 8 PR7 — doctrine fold (corpus B800/B801/B802/B803/B806/B812/B815)

**Date:** 2026-09-05
**Module:** kit (self-section)
**PR:** docs/c8-doctrine-fold

---

## What happened

Campaign 8 PR7 folded doctrine from corpus B800/B801/B802/B803/B806/B812/B815 and their retros into the kit core. This is a PROMOTION: the corpus evidence produced 10 proposed kit deltas (R7.1–R7.10) across 7 target files, each pasted verbatim from the apply-ready READY blocks in the doctrine draft (`0019c58e4`). No code was written; this is a doc-only PR (zero bats by rule).

The fold covered: 8-layer timer defense-in-depth index (D1), non-positive delay floor clause (D2), inter-module comms doctrine (D3), critical-write step-up auth section (D4/D5), cert-chain trust caveat folded as code in `verify-module.sh` (D8), schema-risk mandatory pre-deploy gate in `BUILD-LOOP.md` §6 (D7), triage-console attribution contract in `METHODOLOGY.md` §Conformance (from campaign8-triage-console retro), and the Excavador Técnico working profile + K21 + station load budget in `METHODOLOGY.md` + `skill/SKILL.md`.

---

## Evidence: grep counts + anchors touched

**Corpus token counts in kit after fold:**

| Corpus | `[ev: corpus B…]` hits in `*.md` |
|--------|-----------------------------------|
| B800   | 8 |
| B801   | 10 |
| B802   | 2 |
| B803   | 9 |
| B806   | 4 |
| B812   | 3 |
| B815   | 3 |

**Anchors touched (file:line):**

| Task | File | Line | Anchor description |
|------|------|------|--------------------|
| 7.2 | `types/logic.md` | 14 | 8-layer timer defense-in-depth index bullet (`[ev: corpus B729/B775/B801/B812]`) |
| 7.2 | `types/logic.md` | 15 | folded-as-code pointer for `lint-delays.sh` (`[ev: corpus B801]`) |
| 7.3 | `types/logic-authoring.md` | 12 | `## Inter-module communication [ev: corpus B802]` subsection |
| 7.4 | `types/dashboard.md` | 29 | DWS1 gate (2) CsrfUtil double-submit clause (`[ev: corpus B803]`) |
| 7.4 | `types/dashboard.md` | 54 | `## Critical-write step-up auth [ev: corpus B803]` (6 bullets, lines 55–60) |
| 7.5 | `METHODOLOGY.md` | 97 | triage-console attribution contract ADD line (`[ev: corpus B800]`, `[ev: retro campaign8-triage-console]`) |
| 7.5 | `METHODOLOGY.md` | 105 | D2 non-positive delay floor clause appended to §Conformance first bullet (`[ev: corpus B801]`) |
| 7.6 | `toolbelt/verify-module.sh` | 13–16 | cert-chain trust caveat comment block (`[ev: corpus B800 §800.8]`) |
| 7.7 | `BUILD-LOOP.md` | 52 | MANDATORY schema-risk gate as first bullet of §6 Deploy (`[ev: corpus B800 §800.8]`, `[ev: corpus B795]`) |
| 7.8 | `METHODOLOGY.md` | 3 | working-profile blockquote pointer (`[ev: corpus B801]`, `[ev: corpus B815]`) |
| 7.8 | `METHODOLOGY.md` | 85 | K21 rule (`[ev: corpus B801 §801.4]`, `[ev: corpus B815 §815.10]`) |
| 7.8 | `METHODOLOGY.md` | 113 | `## Station load budget [ev: corpus B806]` section |
| 7.8 | `skill/SKILL.md` | 12 | `## Working profile — Excavador Técnico` section (4 roles + 3 cases) |

**Guard results (task 7.10):**
- `bats tests/kit-links.bats` → 6/6 OK
- `sweep-fold-audit.sh --strict` → exit 0, 56 folded, 56 cited, 0 uncited
- `sweep-build-state.sh` → exit 0
- `bats tests/` → 234 tests OK

---

## Proposed kit deltas

| # | Delta | Status |
|---|-------|--------|
| Δ1 | 8-layer timer SM index bullet in `types/logic.md` §Safety | applied (7.2) |
| Δ2 | folded-as-code pointer for `lint-delays.sh` in `types/logic.md` | applied (7.2) |
| Δ3 | `## Inter-module communication` subsection in `types/logic-authoring.md` | applied (7.3) |
| Δ4 | CsrfUtil double-submit clause to DWS1 gate 2 in `types/dashboard.md` | applied (7.4) |
| Δ5 | `## Critical-write step-up auth` section in `types/dashboard.md` | applied (7.4) |
| Δ6 | triage-console attribution contract ADD line in `METHODOLOGY.md` §Conformance | applied (7.5) |
| Δ7 | D2 non-positive delay floor clause in `METHODOLOGY.md` §Conformance | applied (7.5) |
| Δ8 | cert-chain trust caveat in `toolbelt/verify-module.sh` | applied (7.6) |
| Δ9 | MANDATORY schema-risk gate in `BUILD-LOOP.md` §6 | applied (7.7) |
| Δ10 | Excavador Técnico working profile + K21 + station load budget in `METHODOLOGY.md` + `skill/SKILL.md` | applied (7.8) |

All deltas applied; none proposed-but-not-applied.

---

## Lessons (≤ 5)

1. **Placement beats task label:** draft said "§5" for the schema-risk gate but the READY block's own prose references "the §5 schema-risk verdict" — the block clearly belongs in §6 (Deploy). Trust the READY block content over the task's prose label when they conflict.

2. **K6 prevents double-fold:** confirmed `#49` hint (6 hits) and `Excavador Técnico` (0 hits) before folding; the grep-before-fold discipline blocked two potential duplicate insertions.

3. **sweep-fold-audit only audits `[ev: retro <token>]` in `*.md`:** comments in `.sh` files carry `[ev: corpus B…]` tokens that are legitimately NOT audited — the audit scope is doc-layer retro citations only; the shell script comment is a distinct "folded as code" mechanism.

4. **K21 cite-into-moving-tree rule:** the decompiled line number issue (B801 `Clock.java` differs between the Linux-snap build and the Windows `organized/` build) required a new K-rule rather than an inline note — it is a CLASS of citing mistake that recurs whenever the corpus has multiple builds of the same source.

5. **Station load budget belongs in METHODOLOGY, not BUILD-LOOP:** the B806 budget table is authoring-time doctrine (choose components to stay under the budget) rather than a deploy step — routing it to the new `## Station load budget` section of METHODOLOGY is more discoverable than burying it in §6.
