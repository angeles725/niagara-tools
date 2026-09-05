# campaign6 — math-model spec E5: retro-debt aging (implementation contract)

**Author**: investigador1 (Opus 4.8). **Status**: SPEC (uncommitted, referenced by path — like MM1/MM2/MM3). Feeds QA
(RED pre-stage) + the SDD tools-lane. I do NOT implement in niagara-tools; this is the contract the writer builds to
and QA pins. Mirrors research-sdd's `sweep-retros.sh` "ESCALATED (aged 31d)" so BOTH kits speak the same language.

---

## E5 — retro-debt aging

**Why**: the build kit's retro pipeline (`retros/INDEX.md` + `review-status: pending`) has no aging signal — a pending
retro can sit un-folded indefinitely with no visible debt. research-sdd already ESCALATES aged retros; this gives the
build kit the same, deterministically.

**Pure function** (isolated from I/O — the caller reads the files, this scores):
```
age_days(retro_date: date, today: date) -> int
    = (today - retro_date).days          # calendar days, today >= retro_date

retro_debt(rows: list[{status, date}], today: date, max_age: int = 30)
    -> { per_row: [{date, age_days, escalated}], summary }
  pending = [r for r in rows if r.status == "pending"]       # ONLY pending rows age
  for each pending r:
      a = age_days(r.date, today)
      escalated = a > max_age                                 # STRICTLY greater than the threshold
  summary.total_pending   = len(pending)
  summary.escalated_count = count(escalated)
  summary.oldest_age      = max(age_days) over pending   if pending else "N/A"
  # NON-pending rows (folded/dismissed) never age and are excluded from every count.
```
- **Input parsing (the caller's half, tested with fixtures)**: `rows` come from `retros/INDEX.md` (the `review-status`
  column per row); each retro's DATE is the `YYYY-MM-DD` prefix of its FILENAME (e.g.
  `2026-08-30-…-retro.md` → `2026-08-30`), NOT any date inside the body. `today` is INJECTED (an `--today YYYY-MM-DD`
  arg or a `SWEEP_TODAY` env) — NEVER `date +%s`/wall clock in tests; production may default to the wall clock, but
  the pinnable function takes `today` as a parameter.
- **0 pending → `oldest_age` = "N/A"** (distinct from `0`); `escalated_count` = 0, `total_pending` = 0.
- **`escalated` is STRICTLY `age > max_age`** (an age exactly == max_age is NOT yet escalated).

**Pin vectors** (QA; `today = 2026-09-05`, `max_age = 30`):
| rows (status, filename-date) | total_pending | escalated_count | oldest_age |
|---|---|---|---|
| [] | 0 | 0 | "N/A" |
| [(folded, 2026-01-01)] | 0 | 0 | "N/A" |  ← non-pending never ages
| [(pending, 2026-09-05)] | 1 | 0 | 0 |       ← same-day = age 0
| [(pending, 2026-08-06)] | 1 | 0 | 30 |      ← exactly 30d = NOT escalated (strict >)
| [(pending, 2026-08-05)] | 1 | 1 | 31 |      ← 31d = ESCALATED
| [(pending, 2026-08-05),(folded, 2020-01-01),(pending, 2026-09-01)] | 2 | 1 | 31 |

**Mutations that MUST flip a pinned value:**
- threshold off-by-one (`age >= max_age` instead of `age > max_age`): the `2026-08-06` row (age 30) → escalated_count
  1 ≠ 0.
- date parsed from the BODY / a different field instead of the filename prefix: any row's age moves off its pinned value.
- escalation IGNORES pending state (ages folded/dismissed rows too): the `[(folded, 2026-01-01)]` row → total_pending
  1 ≠ 0 and oldest_age a big number ≠ "N/A".
- 0-pending → `0` instead of `"N/A"`: flips row 1/2.
- `today` read from the wall clock instead of the argument: makes every vector non-deterministic (the test itself
  becomes the guard — a wall-clock read fails the fixed-`today` pins on any day but 2026-09-05).

---

## Integration
- Destination: an `--age` mode of `toolbelt/sweep-build-state.sh` (it already parses `retros/INDEX.md`), OR a small
  `toolbelt/sweep-retro-debt.sh`. Keep the scoring an EXTRACTABLE pure function (`retro_debt(rows, today, max_age)`)
  separate from the INDEX.md parse + the file-date read, so QA pins exact values on fixed `today`.
- CI: NON-STRICT WARN (like the research-sdd sweep) — report escalated pending retros, do not FAIL the build; an
  `--strict`/`--max-age` flag can tighten it per the operator's call. Output line mirrors research-sdd:
  `PENDING  <retro>  · age: <N>d  · ESCALATED (aged <N>d)` past the threshold.
- This is E5 (the "T6.10 correction" companion is separate); it makes the build kit's retro debt visible the same way
  research-sdd's `sweep-retros.sh` does — one shared language across both kits.
