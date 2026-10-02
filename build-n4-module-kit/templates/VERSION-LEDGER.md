# VERSION-LEDGER — <client>

Template from the build-n4-module kit (`templates/VERSION-LEDGER.md`). Copy it to the CLIENT repository as
`odd/VERSION-LEDGER.md` the first time a module of that client is built, then keep it there. One ledger per
client, covering every module of that client. `[ev: retro panccadia-version-defect-ledger Δ1]`

**When to write:** at every build or deploy hand-off — the same moment the module's deployed-baseline fields in the
kit `BUILD-STATE.md` are updated — never reconstructed later from `git log`. A feature doc does not close until its
version rows are here (`BUILD-LOOP.md` §7).

**When to read:** before the first write of any new session on this client (`BUILD-LOOP.md` §0). Compare the
`Class` column with the classes the new work is likely to hit (see the table under "Root-cause classes").

## Versions

One row per built or shipped version per module, newest last. `Status` is one of `built`, `packaged`,
`deployed`, `superseded`, `do-not-install`. `Defect found` / `Root cause` / `Class` stay `—` until a defect is
traced to this version. `Prevention` names the retro and delta that would have stopped it, or `none yet`.

| Module | Version | Date | Status | What changed | Defect found | Root cause | Class | Resolved by | Prevention (retro + Δ) |
|---|---|---|---|---|---|---|---|---|---|
| <Module> | <x.y.z> | <YYYY-MM-DD> | built | <one line> | — | — | — | — | — |

## Root-cause classes

Use these class numbers in the `Class` column; add a class only when no row fits.

| Class | Description | Typical new work that should check it |
|---|---|---|
| 1 | Framework rule unknown — a stock Niagara/baja mechanism not known in advance | a new facet, unit, flag or link target |
| 2 | Requirement or behavior assumed, not asked with the field | a new automatic action or mode |
| 3 | Persistence, restart or release-point gate design gap | a new persisted slot or restart path |
| 4 | Rotation-liveness or wear-metric consequence | a change to a rotation or hours-counting predicate |
| 5 | Fail-open UX or status-trust gap | a new indicator, status read or default |
| 6 | Version or deploy discipline — built is not deployed, rollback assumptions | any deploy |
| 7 | Skipped design step for a new control state machine | a new state machine |
| 8 | Test hygiene — a test that is vacuous under a disabled flag | a new feature flag |

**Repeated class = escalation** (the rule lives in `BUILD-LOOP.md` §7; this is a pointer, not a second copy). When a
class already has a live hit in this ledger and its prevention delta has not shipped yet, a second hit on this
client makes that delta PRIORITIZED: apply it in the current work or open it as the next work unit, ahead of its
normal promotion queue. Citing it again as evidence is not enough.
`[ev: retro panccadia-version-defect-ledger Δ3]`

## Open items

Aggregates, across features, what each feature doc tracks only for itself — so it does not age out when a doc is
archived. Cross-reference the source table; do not restate it. `[ev: retro panccadia-version-defect-ledger Δ4]`

- Values owed by the field: `<module-root>/docs/values-owed.md` (`BUILD-LOOP.md` §6.b item 9).
- Manual station links still to create: `<feature doc>` — `<link>`.
- Behavior assumptions still open: `<feature doc>` § Assumptions still open (`BUILD-LOOP.md` §7).
