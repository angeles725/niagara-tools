<!-- review-status: folded -->
# Retro: Campaign 6 Close (PR6) — tracked launcher + openspec durability + v0.16.0

**Date**: 2026-09-05
**Context**: niagara-tools kit · Campaign 6 PR6 (feat/c6-close) · SDD change build-n4-module-campaign6
**Module**: kit (self-section — no module build; this is a kit-infra PR)
**Gate exit**: (a) new retro (PR6 is a pure infrastructure close; no module lesson to promote)

---

## What was done

- Tracked the canonical Claude skill launcher (`build-n4-module-kit/skill/SKILL.md`, v0.4) with three targeted edits: dropped the non-actionable `state` column from the decision table, added an explicit wb-builder warning pointing to corpus B751–B753/B762/B780 since the type guide is still thin despite being exemplar-backed, and aligned §Execution step 1 with BUILD-LOOP §0's orient-then-corpus-nav order.
- Implemented `scripts/install-skill.sh` (D5 Option A): sha256 comparison, `--home`/`--dry-run`/`--force` flags, exits 0/1/2/3, VCS-free, `HOME=/nonexistent` identical. TDD cycle: IS1–IS4 RED → GREEN → named mutation (drop last line → cmp fails) → revert.
- Committed `openspec/` for the first time (hybrid-store durability): `niagara-tools-slotomatic-integration` moved to `archive/` with a supersession note (100% done outside the openspec flow; 0/40 tasks applied through sdd-apply/sdd-verify). `.gentle-ai-instance` added to `.gitignore` (SDD runtime state, not repo content).
- Bumped `ci.yml` setup-java v4 → v5 (deprecation notice observed in PR2 CI run 33956501710).
- Labeled the parse-mode `pct=` line as `(type-set)` in `slot-coverage.sh` to prevent B788's lesson from being forgotten: DashboardPan-rt measures 100% type-set coverage but only ~25% per-slot coverage.
- Corrected the BUILD-STATE.md open_issue: "CompPan-rt/module.lexicon is empty (T8)" was FALSE — CompPan-rt has 57 populated keys. Replaced with real B788 findings: ColdRoomPan-rt partial (32 keys, camelCase gaps), DashboardPan-rt per-slot ~25%, DashboardPan-wb palette empty scaffold.
- Released v0.16.0 (MINOR: new `scripts/install-skill.sh` surface, three new toolbelt scripts, new `verify-module.sh coverage` subcommand; CONTRIBUTING §4 new-surface = MINOR).

---

## Lessons

### L1 — Track the kit's own entry point or drift is structurally guaranteed

The skill launcher lived only at `~/.claude/skills/build-n4-module/SKILL.md` — a machine-local path that no git operation could restore. Once the launcher is the kit's entry point, it must be in version control like everything else the kit depends on. The D5 Option A decision (copy + sha comparison, not symlink) keeps the harness working without the repo while making divergence loud.

### L2 — openspec artifacts are durability liabilities when untracked

The `openspec/` directory accumulated five change directories (one archived, four live) that lived only as untracked files. Any `git clean` or fresh clone would silently lose them. Hybrid-store durability (track + engram) closes that gap. The `.gentle-ai-instance` lesson: runtime state looks like a file and must be `.gitignore`'d before `git add`.

### L3 — The T8 ledger item was wrong; B788 is the correction

The open_issue "CompPan-rt/module.lexicon is empty (T8)" was copied from the original campaign planning before the actual CompPan-rt lexicon was inspected. Corpus B788 (own-modules-vs-exemplars) measured all three modules with real data. A ledger item that is never verified against the real artifact becomes a silent drift vector — the correct fix is to run the actual measurement and record what the tool finds.

### L4 — Type-set coverage vs per-slot coverage is a non-obvious distinction

`slot-coverage.sh` parse mode returns "100.0%" when every declared type has at least one lexicon key. DashboardPan-rt achieves that while leaving ~75% of its slots without a lexicon entry. The `(type-set)` label on the parse-mode `pct=` line is the minimal intervention: it does not change the metric, it prevents the reader from assigning the wrong meaning.

### L5 — Deprecation notices from CI runs are low-cost, high-value signals

The setup-java v4 deprecation notice appeared in PR2 CI run 33956501710 and was deferred to the close PR (one-line fix). Capturing it immediately in the tasks backlog (T6.9) meant it did not get lost across the six-PR chain. CI deprecation notices deserve a task entry, not a "fix it someday" comment.

---

## Deltas proposed (propose-never-apply)

*(No kit-file deltas: this retro documents infra lessons, not module-authoring rules.)*

---

## Guards (all green before this retro was filed)

- `bats tests/*.bats`: 141 pass (≥140), ≤15 s
- `sweep-build-state.sh`: exit 0; pending = 6 (PR1/PR3/PR4/PR5a/PR5b/PR6 campaign retros; original 8 = 0)
- `rg "rt only" build-n4-module-kit/build-verify.md`: 0 matches
- `shellcheck 0.10.0`: scripts/*.sh build-n4-module-kit/toolbelt/*.sh tests/*.bats tests/helpers/*.bash — exit 0
- `scripts/install-skill.sh --dry-run` against real home: see PR6 description for dry-run output
