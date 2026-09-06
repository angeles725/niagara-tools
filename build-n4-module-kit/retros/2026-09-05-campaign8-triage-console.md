<!-- review-status: folded -->
# Retro: campaign8-triage-console — triage-console.sh station log exception triage

**Session**: Campaign 8 PR2 · 2026-09-05
**Change**: toolbelt/triage-console.sh (new), tests/triage-console.bats (TR1-TR9),
BUILD-LOOP.md §6, skill/SKILL.md v0.8→0.9, openspec design D3 / B800

---

## Proposed deltas (propose-never-apply)

### D1 — Three attribution channels are a hard requirement; two is not enough
Real corpus proof (CERT-live PANCCADIA 2026-09-02/03): a station reload after the
BDefrostController "time <= 0" fix triggered slot-drift load failure (BRelTime→BComplex).
The failure produced ZERO own frames (C1 miss) and ZERO own logger lines (C2 miss).
Only channel 3 ([sys] fatal + [sys.xml] drift shape) surfaced it. The outage was invisible
without C3. The named mutation proof: drop C3 → `console-load-fatal-only.txt` exits 1→0.
[ev: corpus B800]

### D2 — Locale/encoding contract: LC_ALL=C + ASCII prefix for all level detection
Niagara stations running on Spanish-locale JACE/Atlas snap write GRAVE, ADVERTENCIA,
INFORMACI\xD3N (with mojibake byte 0xD3 for Ó). `iconv` / python re-encoding crashes on
these files; `LC_ALL=C grep -a` + awk byte-safe reads never crash. Level normalization
uses ASCII prefix match (INFORMACI covers INFORMACIÓN and its mojibake form). Month table
carries both EN and ES abbreviations (ene/abr/ago/set/dic differ from EN).
B800 §800.5 specifies byte reading as the contract. [ev: corpus B800]

### D3 — Digit normalization for grouping only; display shows first-occurrence message
The two BChiDashboardService "controlTick 1" and "controlTick 2" ADVERTENCIA lines in
chihuahua must group to count 2 (TR5). norm(msg) = s/[0-9]+/N/g collapses them.
Display uses the original message from the first occurrence; the normalized key is
for grouping only. Do NOT normalize the frame (it contains line numbers that are
diagnostic, and normalizing them would lose precision).

### D4 — Bare exception class lines (no colon) must be handled
`javax.baja.sys.NotRunningException` appears as a continuation line with NO colon in
PANCCADIA consoles (it throws with no message). The exception class parser must detect
the bare-class form (no space, ends in Exception/Error/Throwable, or starts with java./javax.)
in addition to the colon form. Without this fix, NotRunningException rows (6 total on
PANCCADIA) are missed — significant own-code defect evidence lost.

---

## Smoke results (CERT-live evidence)

### PANCCADIA (console_backup_260903_1704.txt — load-fail)
```
FAIL  triage-console  console_backup_260903_1704.txt  1x 17:04:32 03-Sep-26 -> 17:04:32 03-Sep-26  SEVERE ClassCastException: javax.baja.sys.BRelTime cannot be cast to javax.baja.sys.BComplex @ [sys]
FAIL  triage-console  console_backup_260903_1704.txt  1x 17:04:32 03-Sep-26 -> 17:04:32 03-Sep-26  WARNING Cannot set property RoomPanel.setpoint: ... @ [sys.xml]
FAIL  triage-console  console_backup_260903_1704.txt  1x 17:04:32 03-Sep-26 -> 17:04:32 03-Sep-26  WARNING Missing frozen property: differentialUp [944:35] @ [sys.xml]
(+3 more sys.xml drift rows)
```

### PANCCADIA (all consoles — own-code exceptions)
- `IllegalArgumentException: time <= 0 @ BDefrostController.armTrigger`: 5 total occurrences
  (1+1+2+1 across 4 files; first 16:20:10 02-Sep-26, last 18:58:19 03-Sep-26)
- `NotRunningException @ BEvaporatorUnit.applyRunCmd`: 6 total occurrences (1+1+1+1+1+1)

### HoneywellMX605132026
- `modifyThread` ADVERTENCIA rows: 9 total (3x in one file + 1x in 6 others)
- `cannot force-load ChiAlarmHelper` (WARNING [chihuahua]): 1 occurrence (C2 channel)

### REFLOW
- `modifyThread` rows: 12 total (3x + 9×1x)
- cert-chain `Could not validate certificate path … BChiDashboardService.class`: 7 rows
  → **caught by C2 (own logger channel)**: the [loader] tag is not in the framework denylist
  (sys/sys.xml/fox/box/driver/station/alarm/history/jetty). Class-name content in the message
  path (com/angeles/) is coincidental; C2 fires on the tag, not the message.

### D5 — Fixture-green, real-red: the smoke is the contract
TR1-TR11 were green but the tool failed on real PANCCADIA data in three ways:
(a) C2 used a denylist, so foreign loggers (bacnet.transport, authentication,
platDataRecovery.service) leaked through — the fixture never exercised a mixed console.
(b) Grouping was per-file; the same exception in five separate console files appeared
as five rows instead of one (count=N, first/last across all files).
(c) Timestamp had a leading `[` (parse-offset bug) so ts_key ordering was incorrect.

The real smoke is the non-negotiable contract: run triage-console.sh on at least two
real CERT-live stations before declaring GREEN. Fixture-only passes miss multi-file
behaviour, foreign-logger leakage, and locale quirks that only appear in real logs.
Add a TR for each class of real-data failure to lock in the fix. [ev: corpus B800]


### D5 addendum — B818 registry pattern robustness
`[sys.registry] Missing class` appears as both `Missing class for "Prefix:Type"` and
`Missing class "Prefix:Type"` (no "for"). The handler now strips the optional "for"
and optional leading quote before extracting the prefix. [ev: corpus B818]
