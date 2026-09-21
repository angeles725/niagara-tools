<!-- review-status: folded -->
# 2026-09-20 · kit · lint-wb-file-chooser (L5 — lint-candidate impl, auto-wired)

**Session**: implement-lint-candidates campaign, L5. Partial promotion of the `lint-wb-file-chooser` CANDIDATE from the folded retro apillm-headless-servlet-rt-4.14-deltas Δ8 (types/wb-widgets.md station-space-picker rule).

**Delta count**: 0 (implementation retro)

## What was done
New `toolbelt/lint-wb-file-chooser.sh` (WARN, `*-wb` src): flags `BWbFieldEditor.dialog(this, ..., BOrd.NULL)` (file-space chooser) in a file with no `BComponentChooser` / `targetType` — a station-component picker should be used instead. `# Mutation: WFC2` (MATCH); tests/lint-wb-file-chooser.bats (7). AUTO-WIRED into report-module.sh §5.22 (`*-wb` gate) + RM22, so build.sh runs it.

## Self-verify
shellcheck 0; lint bats 7/7; report-module.bats 24/24 (RM22); WFC2 MATCH; full suite 0 not-ok; kit-links 10/10; sweep 0. [ev: corpus build-n4-module/automation-audit]

---
**Status**: FOLDED.
