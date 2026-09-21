<!-- review-status: folded -->
# 2026-09-20 · kit · wire lint-guard-pins into CI (kit self-check gap)

**Session**: automation-audit follow-up. The audit noted `lint-guard-pins.sh` (the kit's own meta-check: every `toolbelt/lint-*.sh` must carry a resolvable `# Mutation:` guard-pin) was invoked by nothing — a kit SELF-check with no gate. It belongs in CI, not the per-module report.

**Delta count**: 0 (implementation retro)

## What was done
Added a `lint-guard-pins --strict .` step to `.github/workflows/ci.yml` (after sweep-fold-audit). The tree is currently fully clean (0 WARN, strict exit 0 across all lints incl. the new L1/L2/L3/L5), so this gate now PREVENTS a future lint from landing without a guard-pin resolvable to a bats @test. This is a kit self-check (not a per-module lint), so it is NOT in report-module.sh — the correct home is CI.

## Self-verify
`lint-guard-pins.sh --strict .` exit 0 (0 WARN); sweep-build-state exit 0; sweep-fold-audit exit 0. [ev: corpus build-n4-module/automation-audit]

---
**Status**: FOLDED.
