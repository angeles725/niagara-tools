<!-- review-status: folded -->

# Retro — Campaign 6 PR8: Conformance Lints + Retro-Debt Aging

**Date**: 2026-09-05  **Author**: sdd-apply (Claude Sonnet 4.6)  **Gate exit**: (a) carry retro  
**SDD change**: build-n4-module-campaign6  **PR**: feat/c6-conformance-lints

---

## What shipped

| Tool | Surface | Tests |
|---|---|---|
| `toolbelt/lint-timers.sh` | `lint-timers.sh <java-root>` — timer-ticket + discarded-ticket | TL1–TL4 (GREEN + named mutations) |
| `toolbelt/sweep-build-state.sh --age` | E5 retro-debt aging via `--today <YYYY-MM-DD> [--max-age N]` | D1–D6 (GREEN + named mutations) |
| `toolbelt/preflight.sh` | WSL jdk8 fallback: `bin/java -version` when `release` file absent | PF1–PF5 (GREEN + mutation) |

**T8.3 disposition**: No-op. V17 in `tests/verify-module.bats` already guards the typeless-module case (empty palette, zero `<type>` entries → no WARN). V13/V14 cover the failing path. No code added.

**T8.5 disposition**: Verified. `METHODOLOGY.md` line 77 explicitly names `Clock.Ticket`/`stopped()`-cancel as a lintable hard-fail, while action-without-OPERATOR, order-sensitive container, and poll-that-should-subscribe are marked "HUMAN-REVIEW checklist item — never a hard fail." No regression introduced in PR7.

---

## Real-module lint output (read-only, for grounding)

### ColdRoomPan-rt (Paccadia, Leon-Guanajuato)

```
PASS  timer-ticket  .../BDefrostController.java: timer cancelled in stopped()
FAIL  timer-ticket  .../BEvaporatorUnit.java: schedules a Clock ticket but stopped() does not cancel it
```

**Finding**: `BEvaporatorUnit` owns 4 `Clock.Ticket` fields (`startDelayTicket`, `stopDelayTicket`, `defrostEntryTicket`, `powerOnTicket`) and calls `Clock.schedule(` 4+ times. It has a `cancelTicket()` helper called on re-arm paths but has NO `stopped()` override — the tickets leak on station stop. `BDefrostController` has `stopped()` → `cancelAll()` and correctly passes.

### CompPan-rt (Compresores, Leon-Guanajuato)

```
PASS  timer-ticket  .../BCompressorControl.java: timer cancelled in stopped()
```

**Finding**: `BCompressorControl` has `stopped()` at ~:1799 that calls `cancelTick()` and `powerOnTicket.cancel()`. Conformant.

---

## TDD cycle summary

| Check | RED source | GREEN | Named mutation | Flipped test |
|---|---|---|---|---|
| `timer-ticket` | TL1 (b8e0b9a, lint-timers.bats) | `lint-timers.sh` AWK stopped() scan | Drop AWK block → `found_cancel=1` always | TL1 at bats line 65 |
| `discarded-ticket` | TL4 (added) | `while`-loop discarded check | Remove check block | TL4 at bats line 107 |
| `--age` retro-debt | D1–D6 (ea66684, retro-debt.bats) | `sweep-build-state.sh --age` mode | `>=` instead of `>` | D4; pending-only guard removed → D2+D6 |
| `preflight` jdk8 | PF5 (added) | WSL bin/java fallback in preflight.sh | Remove fallback block | PF5 |

---

## Key lessons

1. **AWK brace counting for method-body extraction is robust** for single-line and multi-line stopped() bodies when cancel is checked before the closing-brace branch exits.
2. **Discarded-ticket fixture must have a conformant stopped()** so only the discarded-ticket check triggers the FAIL, keeping the mutation proof clean.
3. **WSL openjdk-8 false-negative**: Debian/Ubuntu packages omit the `release` file; `bin/java -version` to stderr is the correct fallback (companero B792-B793).
4. **The `--age` mode slots cleanly** before the existing 3-arg guard in `sweep-build-state.sh` without touching existing behavior.
5. **Real-module grounding confirms the lint logic**: BEvaporatorUnit is a real WSL defect (4 tickets, no stopped()); BDefrostController and BCompressorControl correctly pass.
