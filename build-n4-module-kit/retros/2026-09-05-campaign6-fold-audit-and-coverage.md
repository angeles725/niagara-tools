<!-- review-status: pending -->
<!-- kit-retro -->
# Campaign 6 PR5a: fold-citation audit + coverage% model

Date: 2026-09-05 · Module: kit · SDD change: build-n4-module-campaign6 PR5a (feat/c6-tools-audit)

Two deliverables: `sweep-fold-audit.sh` (fold-citation audit, design D3) and the `coverage`
subcommand of `verify-module.sh` (gate coverage model, design D2). Delivered together as the
PR5a slice (B2 + MM1) of Campaign 6. Gate: exit (a) = feature PR carries its own retro.

## What was implemented

### verify-module.sh coverage subcommand (T5a.3/T5a.4)

Added before the flag parser so `coverage` is never parsed as a jar path. Contract:

    verify-module.sh coverage <npass> <nfail> <nwarn> <nskip>

- applicable = P + F + W (SKIP excluded — structurally not-applicable)
- covered = P (only clean passes count)
- result = `t=(1000*P + A/2)/A`, print `$((t/10)).$((t%10))` — integer-tenths, never awk/printf
- N/A sentinel when applicable == 0 (0P/0F/0W/5S → N/A, never "100.0")
- exit 2 + usage line on bad argc or non-integer arg

QA RED d7e52a8 cherry-picked first (8 pins MM1–MM8 RED for right reason: exit 2 "not a jar").
Implementation GREEN 8/8.

Named mutation (MM3): change `echo "N/A"` → `echo "100.0"` → MM3 pin flips
(`[ "$output" = "N/A" ]` fails, expected `100.0` instead).

### sweep-fold-audit.sh (T5a.1/T5a.2)

New tool: audit every `folded` row in `retros/INDEX.md` for a matching `[ev: retro <token>]`
citation in the core kit corpus (`*.md` in kit root, excluding `retros/` and `INDEX.md`).

Key design decisions from §D3:
- **Stem**: strip `YYYY-MM-DD-` prefix and `.md` suffix from the filename.
- **Abbreviated citations**: the live convention is abbreviated, not the full stem. Token `5rooms`
  cites `2026-09-01-dashboardpan-5rooms.md`. Exact-stem grep would produce ~30 false WARNs.
- **Hyphen-segment alignment**: `case "-$stem-" in *"-$T-"*)` — `ender-doors` does NOT credit
  `detail-render-doors` (segments: `detail`, `render`, `doors`; `ender` is mid-segment in `render`).
- **6-char token floor**: removes template placeholders (`[ev: retro X]` in BUILD-STATE.md:157).
- **retros/ exclusion**: self-citations in retros/ must not mask genuinely uncited entries.

Six bats tests written RED-first (F1–F6), implementation GREEN 6/6.

Named mutations (each flips one test):
- **F3**: drop `! -path "*/retros/*"` from `find` → `self-ref-only` found in corpus → WARN disappears → F3 fails
- **F4**: exact-stem match (`[ "$T" = "$stem" ]`) → `5rooms` ≠ `dashboardpan-5rooms` → WARN added → F4 fails
- **F6**: plain substring (`*"${T}"*`) → `ender-doors` substring of `detail-render-doors` → WARN removed → F6 fails

## Real-tree fold-audit output (non-strict, run on build-n4-module-kit)

Run: `bash build-n4-module-kit/toolbelt/sweep-fold-audit.sh build-n4-module-kit/retros/INDEX.md build-n4-module-kit`

```
fold-audit: NOTE ambiguous citation token credits 2026-09-02-comppan-fase3-floating-suction.md: comppan-fase3 comppan-fase3-floating-suction
fold-audit: NOTE ambiguous citation token credits 2026-09-02-module-palette-and-build-target.md: module-palette module-palette-and-build-target
fold-audit: NOTE ambiguous citation token credits 2026-09-03-coldroompan-dashboardpan-freeze-stat-leds.md: coldroompan-dashboardpan-freeze-stat-leds freeze-stat-leds
fold-audit: NOTE ambiguous citation token credits 2026-09-03-process-timers-and-defrost-audit.md: process-timers process-timers-and-defrost-audit
fold-audit: WARN 2026-09-04-junit-standalone-cached-jar-locations-for-wsl-pure-tests.md folded with no [ev: retro …] citation
fold-audit: WARN 2026-09-04-kit-continuity-and-retro-gate-campaign.md folded with no [ev: retro …] citation
fold-audit: 38 folded, 36 cited, 2 uncited
```

Exit 0 (non-strict). The 2 WARNs are follow-up work for PR7. The 4 NOTEs are ambiguous
abbreviated tokens that credit two stems each — both credits are correct, no action needed.

**Key lesson**: The abbreviated-citation convention (e.g. `5rooms` for `dashboardpan-5rooms`)
is established practice in the kit corpus. An exact-stem grep would reject all legitimate
abbreviated citations. The hyphen-segment alignment makes abbreviated tokens work correctly:
`-dashboardpan-5rooms-` contains `-5rooms-` as a full hyphen-segment.

## Gates

| Check | Result |
|-------|--------|
| `bats tests/*.bats` | 124/124 pass |
| `shellcheck 0.10.0` | exit 0 |
| `HOME=/nonexistent bats tests/sweep-fold-audit.bats tests/verify-coverage.bats` | 14/14 pass |
| `sweep-build-state.sh` real tree | exit 0 |
| `bats tests/kit-links.bats` | 3/3 pass |
| Real-tree fold-audit | exit 0 (2 WARNs, expected — PR7 follow-up) |
