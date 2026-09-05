<!-- review-status: folded -->
# Campaign 6 PR5b — preflight.sh + slot-coverage.sh: environment preflight and MM2 exposed-set coverage

**Date:** 2026-09-05
**Kit version at merge:** 0.15.1 (PR5b adds no version bump — tools only, no core-doc change)
**PR branch:** feat/c6-tools-env
**Stacked on:** PR5a (feat/c6-tools-audit) @ 4551dc9

---

## What shipped

Two new toolbelt scripts (TDD RED → GREEN → named mutation, revert → full suite green):

| Script | Subcommands / flags | Gate |
|---|---|---|
| `toolbelt/preflight.sh` | `[--jvm-dir <d>] <niagara_home> <gradle-root>` | 4 tests (PF1–PF4), 3 named mutations |
| `toolbelt/slot-coverage.sh` | `set-coverage <decl-csv> <req-csv>` · `[--strict] <xml> <lex>` | 8 tests (SC1–SC6, SC6-parse, dup-keys) |

**Bats suite at commit:** 136 tests, 136 green.
**shellcheck 0.10.0:** exit 0 on all toolbelt scripts and test files.
**HOME=/nonexistent guard:** 12/12 identical (both new scripts, full output compared).

---

## TDD RED → GREEN cycle

### preflight.sh (T5b.1 / T5b.2)

```
RED  tests/preflight.bats  (4 tests, all FAIL — script absent)
GREEN  toolbelt/preflight.sh  (checks: win-path, jdk8, plugin-pin, jar-lock)
```

Named mutations proved (each flipped one test; reverted for full GREEN):

| Mutation | Test flipped |
|---|---|
| PF1: plugin-pin check always returns PASS | PF1 exits 0 instead of 1 |
| PF2: embed `$HOME` value in PASS row output | PF2 output diverges under HOME=/nonexistent |
| PF4: lsof-absent branch emits PASS instead of SKIP | PF4 assertion `[[ $out != *PASS  jar-lock* ]]` flips |

### slot-coverage.sh (T5b.3 / T5b.4)

```
RED  tests/slot-coverage.bats  (cherry-pick QA SHA 5a7d90a = 08a8b93, 6 pins SC1–SC6)
GREEN  toolbelt/slot-coverage.sh  (pure set-coverage + parse subcommand + dup-keys)
```

SC1–SC6 from QA cover the pure MM2 function (denominator, extra, N/A sentinel, arity).
SC6-parse and dup-keys added in this PR for the parse subcommand.

Mutations already documented in the QA pin file (slot-coverage.bats header):
- denominator `|declared|` not `|required|` → SC2 flips
- extra folded into numerator → SC3 flips
- 0/0 → 100 instead of N/A → SC5 flips
- missing computed as `declared - required` (swapped) → SC2 and SC3 flip

---

## Real-module scan results (CompPan-rt and DashboardPan-rt)

Running `slot-coverage.sh <module-include.xml> <module.lexicon>` on the live modules:

| Module | pct | missing | extra (not scored) |
|---|---|---|---|
| CompPan-rt | 100.0 | (none) | 53 slot-level keys (expected — lexicon has fine-grained entries per slot, xml has type-level entries only) |
| DashboardPan-rt | 100.0 | (none) | (none) |

Both modules pass at pct=100.0. The CompPan extra count (53) is a known artifact: the lexicon
uses `Type.slot=Display Name` keys and the xml uses `<type name="TypeName">` — the type name
is extracted as the prefix before the first dot, so every fine-grained slot key collapses to
the same type name that is already in required; the remainder are genuinely extra names not in
the xml. This is the B12 "dangling lint" surface the tool operationalizes.

CompPan T8 footgun confirmed: the `comppan-t8` fixture (empty lexicon, 3 declared types in xml)
produces `pct=0.0` + `WARN empty lexicon` — the tool catches the silent-deploy risk.

---

## Lessons

**L1 — Lexicon format is `slot=DisplayName`, not `key=slot=DisplayName`.**
The spec described the key as "the left-hand side of `=`". The real module.lexicon files use
bare keys (`fan`) or dotted keys (`FanMode.fan`). The cut-based parser (`cut -d'=' -f1`) is correct.

**L2 — WARNs on stdout (not stderr) for machine readability in bats.**
The `$output` bats variable captures stdout. Putting WARNs on stderr makes them untestable
with standard bats assertions. Convention matches sweep-fold-audit.sh.

**L3 — HOME=/nonexistent mutation needed a PASS-row carrier, not a FAIL-row carrier.**
The JDK 8 fixture is found (PASS row), so the first mutation attempt (embedding `$HOME` in the
FAIL row's detail) never triggered because the FAIL row was not emitted. Fixed by embedding
`$HOME` in the PASS row instead.

**L4 — QA cherry-pick by direct SHA when branch is unavailable.**
`qa/c6-slot-coverage-red` was not fetchable by name. `5a7d90a` existed in the object store
(`git cat-file -t` confirmed `commit`). Cherry-pick by SHA is the correct recovery path.

**L5 — SC6 naming collision between QA pin and task description.**
QA's SC6 is the arity test; tasks.md called SC6 the "parse subcommand" test. Resolution:
kept QA SC6 as-is (arity), named the parse test `SC6-parse` as a new addition.

---

## T5b.5 (stretch) — deferred

`verify-module.sh --plano` and `tests/fixtures/plano-sample.html` were deferred. The authored
line count for PR5b was already at the design ceiling for a single PR. Deferred to a future
campaign PR (Campaign 6 PR8 candidate or a standalone PR after the main chain closes).

---

## Open issues carried forward

- `CompPan-rt/module.lexicon` is not empty in the real module — it has 53 slot-level keys,
  all collapsing to types that are already in required. The T8 footgun (raw camelCase in operator
  views) is a module-level fix, out of this campaign's kit scope.
- `verify-module.sh --plano` (stretch T5b.5): deferred.
- The 53 CompPan "extra" slot keys: not a kit defect — they are genuinely extra names below the
  type level that the xml does not declare. Reported by the tool as lint; not scored in pct.
