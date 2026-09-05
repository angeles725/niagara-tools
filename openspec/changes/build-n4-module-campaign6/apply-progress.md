# Apply Progress — build-n4-module-campaign6

**Status**: all PR1–PR8 tasks complete  
**Last updated**: 2026-09-05 · Campaign 6 PR8

---

## PR8 — feat/c6-conformance-lints (T8.1–T8.6)

### Completed tasks

- [x] T8.1 `toolbelt/lint-timers.sh` — timer-ticket check (AWK stopped() brace-counted scan)
- [x] T8.2 `toolbelt/lint-timers.sh` — discarded-ticket check (line-by-line Clock.schedule grep)
- [x] T8.3 No-op — V17 in `tests/verify-module.bats` already guards the typeless-empty-palette case
- [x] T8.4 `toolbelt/sweep-build-state.sh --age` — E5 retro-debt aging mode
- [x] T8.5 Verified — METHODOLOGY.md advisory rules marked HUMAN-REVIEW, no hard-fail added
- [x] T8.6 PR8 retro + kit self-envelope (`build-n4-module-kit/retros/2026-09-05-campaign6-conformance-lints.md`)

### TDD cycle evidence

| Task | Tests | RED | GREEN | Mutation → flip |
|---|---|---|---|---|
| T8.1 timer-ticket | TL1-TL3 (qa/c6-timer-ticket-lint b8e0b9a) | Exit 127 (no script) | lint-timers.sh AWK | Drop AWK block → TL1 at bats:65 flips |
| T8.2 discarded-ticket | TL4 (added) | TL4 expects FAIL, no discarded check | while-loop discarded check | Drop check block → TL4 at bats:107 flips |
| T8.4 --age mode | D1-D6 (qa/c6-retro-debt-red ea66684) | Exit 3 (unknown --age flag) | sweep-build-state.sh --age | `>=` instead of `>` → D4 flips; pending guard removed → D2+D6 flip |
| T8.6a PF5 | PF5 (added) | PF5 expects PASS, no fallback | WSL bin/java fallback in preflight.sh | Remove fallback block → PF5 flips |

### Test count

- Baseline: 150 tests (141 pass, 9 RED for TL1-TL3 + D1-D6)
- After PR8: 152 tests (152 pass, 0 fail)

### Files created/modified

| File | Action | Notes |
|---|---|---|
| `build-n4-module-kit/toolbelt/lint-timers.sh` | Created | timer-ticket + discarded-ticket lint |
| `build-n4-module-kit/toolbelt/sweep-build-state.sh` | Modified | Added --age mode |
| `build-n4-module-kit/toolbelt/preflight.sh` | Modified | WSL jdk8 fallback (PF5) |
| `tests/lint-timers.bats` | Cherry-picked + TL4 | QA b8e0b9a + TL4 added |
| `tests/retro-debt.bats` | Cherry-picked | QA ea66684 |
| `tests/preflight.bats` | Modified | PF5 added |
| `tests/fixtures/lint-timers/` | Created | CI fixtures (NoTimerSample, ConformantSample) |
| `tests/fixtures/preflight/jvm-no-release/` | Created | PF5 fakebin fixture |
| `.github/workflows/ci.yml` | Modified | lint-timers + sweep-retro-debt steps |
| `build-n4-module-kit/retros/2026-09-05-campaign6-conformance-lints.md` | Created | PR8 retro |
| `build-n4-module-kit/retros/INDEX.md` | Modified | PR8 row added |
| `build-n4-module-kit/BUILD-STATE.md` | Modified | Kit self-envelope PR8 |
| `openspec/changes/build-n4-module-campaign6/tasks.md` | Modified | T8.* marked [x] |

### Rollback boundary

Revert `build-n4-module-kit/toolbelt/lint-timers.sh` (delete), revert `--age` block from `sweep-build-state.sh`, revert WSL fallback from `preflight.sh`, remove `tests/lint-timers.bats` + `tests/retro-debt.bats` + `tests/preflight.bats` (PF5), revert `tests/fixtures/lint-timers/` and `tests/fixtures/preflight/jvm-no-release/`, revert CI yml additions, delete PR8 retro + INDEX row, revert BUILD-STATE.md.

---

## Prior PRs (PR1–PR7): all complete

See engram observation #8186 for PR7 details. All 8 PRs of Campaign 6 are now complete.
