<!-- review-status: pending -->
# Retro: Campaign 8 — lint-write-path.sh OPERATOR-slot write-path matrix coverage (Campaign 8 PR19, B816)

**Date:** 2026-09-06
**Change:** build-n4-module-campaign8 PR19 — `feat/c8-write-path`
**Scope:** `build-n4-module-kit/toolbelt/lint-write-path.sh` (new); `types/logic.md` §Write-path & overlap; `types/logic-authoring.md` §Write-path test matrix; CHECK12 idempotent guard; K19 routing
**Lead:** Campaign 8 PR19 executor
**RED tip:** `411f3e1` (WP1, WP2, WP3, WP4, WP-prune, WP-usage)

---

## What happened

`lint-write-path.sh <module-root> [--bog <config.bog>]` checks that every
`@NiagaraProperty` with `Flags.OPERATOR` has a ROW in `docs/write-path-matrix.md`
(D16: row-presence only; CHECK12 in bog-audit.sh handles link-target WARN separately).

The paren-balance multi-line awk parser from `slot-coverage.sh per-slot` was reused
verbatim for the `@NiagaraProperty` + OPERATOR extraction. The matrix row parser
extracts the first pipe-delimited cell, handling backtick-wrapped slot names
(`` `freezeSetpoint` (sp) `` → `freezeSetpoint`) and filtering to Java identifiers only
(`^[a-z][A-Za-z0-9]*$`) — this gates out headers, separators, and descriptive prose rows.

`--bog` uses a line-based bog XML parser (same grammar as bog-audit.sh, no second parser,
D16) to find `<p t='b:Link'>` links where the source module matches Dashboard/RoomPanel
and the target module matches the own-module (derived from the -rt/-wb/-ux directory name).
The `targetSlotName` of those links is added to the required set.

Doctrine folded:
- `types/logic.md`: new §Write-path & overlap — `set()` calling-thread, LINK_TARGET
  advisory-flag trap, Transaction not atomic.
- `types/logic-authoring.md`: new §Write-path test matrix — 5-column template (slot ×
  writer × timing × invariant × test), coverage legend, cross-references to §Slot types
  (B823) for the overlap/ephemeral fact and to `lint-delays.sh` (no double-bite on ≤0).

CHECK12 idempotent guard: since PR10 merged with CHECK12, a bats pin asserts CHECK12
is present in bog-audit.sh — no re-add (R19.5).

K19 routing: `lint-write-path.sh` appended to BUILD-LOOP.md §5 pre-gate and
`skill/SKILL.md` step 5 + Toolbelt list. `[ev: retro campaign8-write-path]` in both.

Final bats: **292/292** green. shellcheck exit 0. All sweep guards exit 0. kit-links 7/7.

---

## Evidence

### Named mutations (on mktemp copies, observed output pasted verbatim)

**(a) Remove matrix row for a covered OPERATOR slot → that slot appears in FAIL output:**
```
# mutation: grep -v '| fanMode ' covered/docs/write-path-matrix.md > tmp
FAIL  lint-write-path  covered  slot fanMode: no matrix row
Exit: 1   ← was 0 before mutation
```

**(b) Comment-only mention does NOT count as a row → WP3/WP4 held (no flip):**
```
# uncovered fixture: <!-- hoaMode: TODO ... --> (HTML comment, no table row)
FAIL  lint-write-path  uncovered  slot hoaMode: no matrix row
Exit: 1   ← hoaMode still FAIL; comment mention rejected
```

**(c) Drop --bog link tracing → WP3-BOG scenario: bog-derived slots not added:**
Script has `--bog` branch; without it, only OPERATOR slots required. The 8 bog-added
slots (`duration`, `evapHighAlarmLimit`, `evapLowAlarmLimit`, `interval`,
`resistanceTempThreshold`, `staggerDelay`, `startDelay`, `terminateOnResistanceTemp`)
do not appear in FAIL when `--bog` is omitted.

**(d) Dot-dir prune: `staleKnob` under `src/.deploy-baseline/` never flagged (D9b):**
```
# uncovered fixture: src/.deploy-baseline/com/x/BStale.java has staleKnob OPERATOR
# Tool output: hoaMode FAIL; staleKnob absent from output
FAIL  lint-write-path  uncovered  slot hoaMode: no matrix row
# staleKnob: not present  ← dot-dir pruned
```

### Real smokes (client modules at deed38c; verbatim output)

**ColdRoomPan-rt — WITHOUT `--bog` (8 uncovered OPERATOR slots; design guessed 9):**
```
FAIL  lint-write-path  ColdRoomPan-rt  slot coolOnSensorFault: no matrix row
FAIL  lint-write-path  ColdRoomPan-rt  slot fanMode: no matrix row
FAIL  lint-write-path  ColdRoomPan-rt  slot fanRunMode: no matrix row
FAIL  lint-write-path  ColdRoomPan-rt  slot freezeDiffStop: no matrix row
FAIL  lint-write-path  ColdRoomPan-rt  slot freezeProtect: no matrix row
FAIL  lint-write-path  ColdRoomPan-rt  slot powerOnDelay: no matrix row
FAIL  lint-write-path  ColdRoomPan-rt  slot resistanceMode: no matrix row
FAIL  lint-write-path  ColdRoomPan-rt  slot valveMode: no matrix row
Exit: 1
```

**ColdRoomPan-rt — WITH `--bog /mnt/c/.../PANCCADIA/config.bog` (+8 bog-linked slots):**
```
FAIL  lint-write-path  ColdRoomPan-rt  slot coolOnSensorFault: no matrix row
FAIL  lint-write-path  ColdRoomPan-rt  slot duration: no matrix row
FAIL  lint-write-path  ColdRoomPan-rt  slot evapHighAlarmLimit: no matrix row
FAIL  lint-write-path  ColdRoomPan-rt  slot evapLowAlarmLimit: no matrix row
FAIL  lint-write-path  ColdRoomPan-rt  slot fanMode: no matrix row
FAIL  lint-write-path  ColdRoomPan-rt  slot fanRunMode: no matrix row
FAIL  lint-write-path  ColdRoomPan-rt  slot freezeDiffStop: no matrix row
FAIL  lint-write-path  ColdRoomPan-rt  slot freezeProtect: no matrix row
FAIL  lint-write-path  ColdRoomPan-rt  slot interval: no matrix row
FAIL  lint-write-path  ColdRoomPan-rt  slot powerOnDelay: no matrix row
FAIL  lint-write-path  ColdRoomPan-rt  slot resistanceMode: no matrix row
FAIL  lint-write-path  ColdRoomPan-rt  slot resistanceTempThreshold: no matrix row
FAIL  lint-write-path  ColdRoomPan-rt  slot staggerDelay: no matrix row
FAIL  lint-write-path  ColdRoomPan-rt  slot startDelay: no matrix row
FAIL  lint-write-path  ColdRoomPan-rt  slot terminateOnResistanceTemp: no matrix row
FAIL  lint-write-path  ColdRoomPan-rt  slot valveMode: no matrix row
Exit: 1
```

**CompPan-rt — WITHOUT `--bog` (16 uncovered OPERATOR slots; design guessed 9):**
```
FAIL  lint-write-path  CompPan-rt  slot condenser1Mode: no matrix row
FAIL  lint-write-path  CompPan-rt  slot condenser2Mode: no matrix row
FAIL  lint-write-path  CompPan-rt  slot condenser3Mode: no matrix row
FAIL  lint-write-path  CompPan-rt  slot dischargeHighLimit: no matrix row
FAIL  lint-write-path  CompPan-rt  slot faultReset: no matrix row
FAIL  lint-write-path  CompPan-rt  slot floatingSuction: no matrix row
FAIL  lint-write-path  CompPan-rt  slot minOn: no matrix row
FAIL  lint-write-path  CompPan-rt  slot powerOnDelay: no matrix row
FAIL  lint-write-path  CompPan-rt  slot runningAmpsThreshold: no matrix row
FAIL  lint-write-path  CompPan-rt  slot stageDelay: no matrix row
FAIL  lint-write-path  CompPan-rt  slot stageDownDelay: no matrix row
FAIL  lint-write-path  CompPan-rt  slot stageUpDelay: no matrix row
FAIL  lint-write-path  CompPan-rt  slot startProveDelay: no matrix row
FAIL  lint-write-path  CompPan-rt  slot suctionBand: no matrix row
FAIL  lint-write-path  CompPan-rt  slot suctionLowLimit: no matrix row
FAIL  lint-write-path  CompPan-rt  slot suctionMismatchTol: no matrix row
Exit: 1
```

**Covered-only fixture (WP1) → exit 0, no FAIL output.**

Note: design estimate was "9 uncovered" for both modules; actual was ColdRoomPan=8, CompPan=16.
The deed38c matrix is a repo-root shared matrix (not per-module); when placed at
`<module-root>/docs/write-path-matrix.md`, coverage differs. The campaign-9 seed
lists write-path matrix completion (W14–W22) as an open item.

---

## Proposed kit deltas

| # | File | Delta |
|---|------|-------|
| Δ1 | `toolbelt/lint-write-path.sh` | NEW — OPERATOR-slot matrix coverage lint |
| Δ2 | `types/logic.md` | NEW §Write-path & overlap [ev: corpus B816] |
| Δ3 | `types/logic-authoring.md` | NEW §Write-path test matrix [ev: corpus B816] |
| Δ4 | `tests/bog-audit.bats` | CHECK12-pin (idempotent guard) |
| Δ5 | `BUILD-LOOP.md` §5 + `skill/SKILL.md` | K19 routing lint-write-path.sh |

---

## Lessons

1. The Niagara bog uses `<p t='b:Link'>` wrapper elements with child `<p n='sourceOrd'>` slots — not a `<link>` element; a regex over `<link\b>` finds nothing and a line-based parser matching `t='b:Link'` is required.
2. Matrix cells with backtick-wrapped slot names (`` `slot` (desc) ``) need special extraction: strip leading backtick, then truncate at the second backtick; bare cells need to stop at the first space or punctuation.
3. The real uncovered counts (ColdRoomPan=8, CompPan=16) exceed the design estimate of 9 because the client's deed38c matrix covers fewer slots than the design assumed; future specs should reference a real count from the client repo rather than estimating.
4. Pushing every non-self-closing `<p>` to the stack (not just components) is required for correct `</p>` balance when the bog has complex nested structures; bog-audit.sh relies on this for its `in_link` tracking.
