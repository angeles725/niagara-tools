<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-11-16 · kit · t34-ntpl-advisories

**Session**: row T34 of the 2026-11-15 coordination-stretch retro: the native-review advisories of niagara-tools #716
(WARNING `go/internal/lint/rules/ntplmanifest/messages.go:105-110`, resilience; SUGGESTION `budget_test.go:25-46`,
reliability).
**Delta count**: 2

## What happened
The review record of #716 was not retrievable (no PR comments; the burned review artifacts are gone), so both
findings were re-derived from the code. The run-bounds comment promised that the run's reads end in a few seconds,
but the budget counted inflated bytes only: an empty or tiny entry draws next to nothing, yet each costs an open and
an XML parse. 100 templates of 10,000 four-byte Px entries took 12.4 s while drawing about 4 MiB of the 128 MiB
budget, and the 1,000 templates the count bound allows about two minutes. The run now also stops after 100,000
entries read (the same scratch run: 0.78 s), with its own row. The amplifying-jar test hard-coded the template it
stops at (T003) and the budget in bytes; both are now derived from `maxInflate` and `maxUsesRead`.

## Evidence
- RED before GREEN: TestRunStopsAtTheEntryReadBound (bound written as the literal 100000) exits 0 with no row on
  origin/main and passes after the fix. `[ev: go/internal/lint/rules/ntplmanifest/budget_reads_test.go]`
- With `maxInflate` set to 96 MiB the base TestAmplifyingJarSpendsOneRunBudget fails, the derived one passes.
  `[ev: go/internal/lint/rules/ntplmanifest/budget_test.go]`
- Mutation, not committed: replacing the read-count guard with `case false:` turns TestRunStopsAtTheEntryReadBound
  red. `[ev: go/internal/lint/rules/ntplmanifest/budget.go]`
- Observed, not changed: when the run stops inside an archive whose manifest is not valid, the manifest row is the
  archive's only row, so the stop is not named (as for the byte budget); the exit is 1 either way.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-16. The ntpl lint run stops after 100,000 entries read, with its own row, as well as after 128 MiB inflated | `go/internal/lint/rules/ntplmanifest/budget.go` | `[ev: go/internal/lint/rules/ntplmanifest/budget_reads_test.go]` |
| Δ2 | APPLIED 2026-11-16. The amplifying-jar test derives its stop from the two bounds | `go/internal/lint/rules/ntplmanifest/budget_test.go` | `[ev: go/internal/lint/rules/ntplmanifest/budget_test.go]` |
