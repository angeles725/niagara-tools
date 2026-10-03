# ODD feature — Deferred lint candidates and the polish close-retro fold

**Status**: IN PROGRESS · **Created**: 2026-10-02 · **Owner**: ODD orchestrator · **Repo**: angeles725/niagara-tools (main, PUBLIC)

## Objective
Fold the 5 deltas of the polish close retro (`retros/2026-10-02-polish-2026-10-02-close.md`), close the one real gap
on #226 (a `find` error silently skips files in six lints), and implement the lint candidates that earlier folds
recorded as DEFERRED in `BUILD-STATE.md` (`lint-size`, `lint-license-isoperational-gate`, `lint-set-null-ord`) plus the
project-level `split-package-check` of #142. Close with release v0.31.0.

## Problem / why
- The polish close retro is `pending`: its lessons (anti-cascade policy, structural parser rules, two shell traps, the
  `candidate_context_unavailable` recovery) live only in the retro until they reach doctrine.
- Six lints read `find … | sort` through a process substitution, so a sub-directory that `find` cannot enter is
  skipped and the lint reports the module clean. That is a fail-open, already true before the polish feature.
- Three lint candidates were evaluated and recorded but never built, each with a corpus-backed failure mode
  (an oversized class, a licensed component acting while unlicensed, a null ORD written into a slot). #142 has the
  same status for the split-package shadowing gotcha.

## Authorization & delivery
- Operator pre-authorized on 2026-10-02: ODD + RDD (consent granted), commit + push + PR + merge + issue
  comments/close + tags, chained and automatic.
- Delivery: one PR per task, branched from the current `origin/main`, merged after CI green (+ RDD when due) before
  the next starts. The ~400-line budget is advisory.
- Repository is PUBLIC: new text uses generic names (no customer, site, host, IP or user names).
- ANTI-CASCADE POLICY (carried over from `odd/tasks/polish-2026-10-02.md`, now doctrine in `ORCHESTRATION.md` §7):
  only a review finding that shows a fail-open (a check that can report PASS on a real defect) or a regression
  introduced by this feature becomes a sub-task. Every other advisory is posted on #226 and listed under
  "Parked advisories" below; no new sub-task is created for it.

## Scope
In: the toolbelt scripts, lib helpers, bats, fixtures, docs, ledgers and the release named by the tasks below.
Out: wiring `split-package-check` into a client repository's CI (that repository is outside this kit); doctrine not
traceable to the close retro, #226 or #142.

## Tasks
- [x] D1 · fold the 5 deltas of `retros/2026-10-02-polish-2026-10-02-close.md` — Δ1 anti-cascade policy up front
      (`ORCHESTRATION.md` §7), Δ2 structural rules for heuristic parsers (`METHODOLOGY.md` § Conformance rules), Δ3
      bash 5.2 `patsub_replacement` (`CONTRIBUTING.md` §2 + `ORCHESTRATION.md` §3.c, mechanical guard
      `tests/shell-hygiene.bats` SH1), Δ4 bats `! cmd` (`CONTRIBUTING.md` §2 + `ORCHESTRATION.md` §3.c; gated by
      shellcheck SC2314 in CI), Δ5 `candidate_context_unavailable` recovery (`ORCHESTRATION.md` §4) — and the #226 lead
      item: the six T3/T4/T5a lints fail closed (exit 3) when `find` cannot enter a sub-directory, through a shared
      `toolbelt/lib/scan-files.sh`. Retro flipped `folded`. Route: inline (the parent is the bounded writer for this
      feature; one lib helper + six mechanical edits + docs).
- [ ] D2 · `lint-size.sh` — advisory WARN-only class/method size smell (BUILD-STATE DEFERRED spec).
- [ ] D3 · `lint-license-isoperational-gate.sh` — a licensed class acting in `changed()`/timer/servlet-write callbacks
      without an `isOperational()`/`isFault()` gate.
- [ ] D4 · `lint-set-null-ord.sh` — a `setXxxOrd(getSlotPathOrd())` / ORD `set(...)` with no null guard.
- [ ] D5 · #142 `split-package-check.sh` — project-level check: one Java package declared in two modules.
- [ ] D6 · close — release v0.31.0 (VERSION, CHANGELOG, tag on the confirmed merge SHA), skill reinstall + drift
      check, feature retro, this doc COMPLETE.

## Acceptance
- Every task lands as its own merged PR with RED → GREEN evidence where a runnable deterministic test exists, an
  observed mutation per new guard (Mutation IDs equal bats test IDs), and its ledger entry.
- Each new lint grounds its detection rule in cited evidence (corpus block or decompiled source file:line), ships
  positive and negative fixtures, a guard-pin, a regenerated `toolbelt/INDEX.md` and, when it runs per module, its
  `report-module.sh` wiring.
- #226 lead item and #142 closed or commented with a mapping to the PR that resolved them.

## Checks (per task)
`bats tests/*.bats` serially (no GNU parallel) · `python3 -m unittest discover -s mcp-n4-kit/tests` ·
shellcheck per `.github/workflows/ci.yml` · `sweep-build-state.sh` · `sweep-build-state.sh --age --today <date>` ·
`sweep-fold-audit.sh --strict` · `sweep-fold-audit.sh --strict --deltas-since 2026-09-24` ·
`lint-guard-pins.sh --strict .` · `gen-lint-index.sh --check`.

## Progress / evidence
- 2026-10-02 D1 (branch `feat/dl-d1-fold-and-find-fail-closed`, from `origin/main` 255f2d9). Route: inline — the parent
  session is this feature's bounded writer; one lib helper, six mechanical lint edits, docs.
  Evidence:
  - RED: CHW-finderr, PHW-finderr, SSL-finderr, WEO-finderr, ICO-finderr, SPR-finderr not ok before the change (a
    `chmod 000` sub-directory: `find: … Permission denied` on stderr, exit 0, its files skipped).
  - GREEN: the six lint suites all ok (69 tests) after `toolbelt/lib/scan-files.sh` (find status kept, list sorted to a
    file) replaced the process-substitution walk; exit-contract cells now say "or a sub-directory find cannot enter".
  - Observed mutations (restored byte-identical, `cmp`): `scan_files … || true` in each lint → its `<ID>-finderr` not ok
    (CHW, PHW, SSL, WEO, ICO, SPR); SH1-neg (checker skips the replacement part) → SH1-neg not ok.
  - Δ3 guard: `tests/shell-hygiene.bats` SH1/SH1-neg/SH1-pos with `tests/helpers/patsub-check.py` (structural: pattern
    ends at the first unquoted depth-0 `/`, replacement at the matching `}`); the kit scripts are clean today. Δ4 is
    already a mechanical gate: shellcheck SC2314 (style) fires on a bats `! cmd` line and CI runs every severity.
  - `bats tests/*.bats` (serial): 975 ok / 0 not ok (67 env skips), count 975, exit 0. mcp-n4-kit unittest: 484 OK.
  - shellcheck 0.11.0: only the pre-existing SC2329 info in `lint-config-sanity.sh`.
  - `sweep-build-state.sh` exit 0; `--age --today 2026-10-02` exit 0; `sweep-fold-audit.sh --strict` 196/196 cited;
    `--deltas-since 2026-09-24` 145 checked, 0 not cited; `lint-guard-pins.sh --strict .` exit 0;
    `gen-lint-index.sh --check` fresh (regenerated for the six exit cells).
  - Observation (not a sub-task, anti-cascade): 12 other lints still walk with `find … 2>/dev/null`; same class, not
    introduced by this feature, outside the #226 item — recorded on #226.

## Parked advisories
Non-blocking review advisories parked under the anti-cascade policy (posted on #226, no sub-task).

## Delivery record
| Task | PR | Merge SHA | Review |
|---|---|---|---|

## Next step
D1 review + merge, then D2.
