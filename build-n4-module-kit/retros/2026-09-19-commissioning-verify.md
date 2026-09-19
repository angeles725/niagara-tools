<!-- review-status: folded -->
# 2026-09-19 · kit · commissioning-verify

**Session**: 2026-09-19 kit-improvement campaign (branch odd/issue-114-commissioning) — close the kit's BUILD-LOOP §6.b gap (issue #114).
**Delta count**: 2

## What happened
BUILD-LOOP §6.b named the kit's #1 gap: no commissioning-verify tooling — the facade↔rt link /
config-sanity / parity / recovery checklist was operator-manual. Built commissioning-verify.sh as an
orchestrator over the existing, tested checks (not new analysis), plus a MANUAL footer for the live-only
steps it cannot statically verify.

## Evidence
- commissioning-verify.bats 5/5; kit-links 10/10 (L5 — script named in BUILD-LOOP §6.b); smoke on MinimalPan prints the punch-list; full suite 583 ok / 0 not-ok; shellcheck 0.10.0 exit 0. `[ev: tests/commissioning-verify.bats]`

## Proposed kit deltas (folded in this PR)
| Δ | Delta | Target | Token |
|---|---|---|---|
| Δ1 | new commissioning-verify.sh: aggregates lint-config-sanity + lint-status-parity + lint-recovery-path over src, bog-audit CHECK11/13-19/20 over --bog, wiring-map note, MANUAL live-only footer; one punch-list, exit 0/1/3 | `toolbelt/commissioning-verify.sh` + `tests/commissioning-verify.bats` | `[ev: retro live-commissioning-verification-gaps]` |
| Δ2 | BUILD-LOOP §6.b closed: the auditor now exists and is named (was "add obix-nav.py-style auditor") | `BUILD-LOOP.md` §6.b | `[ev: tests/commissioning-verify.bats]` |

## Lessons
- The §6.b gap closes by ORCHESTRATING the tested checks into one entry point + honestly rowing the live-only steps as MANUAL, not by pretending to statically verify live plant control.
- A new toolbelt/*.sh must be named in BUILD-LOOP.md (kit-links L5); commissioning-verify.sh is not a lint-*.sh so guard-pins does not require a Mutation header.

---
**Status**: FOLDED — commissioning-verify.sh closes BUILD-LOOP §6.b (2026-09-19), issue #114.
