# Fold the three pending round-10 retros (2026-10-27)

Locator: `build-n4-module-kit/odd/tasks/fold-r10-retros-2026-10-27.md` · branch `docs/fold-r10-retros` from
origin/main 6c34e20c · worktree `niagara-tools-worktrees/fold-r10-retros`.

## Objective
Every delta of the three `review-status: pending` round-10 retros is either already landed (with evidence) or
folded into the kit, each retro is flipped to `folded`, and CI's strict fold audit
(`sweep-fold-audit.sh --strict --exact-stems --deltas-since 2026-09-24`) stays at 0 uncited deltas.

## Problem / why
The retros were written as propose-never-apply. Most of their deltas landed in the same PRs that wrote them, but
the fold audit credits a delta only by an exact `retro <stem> Δk` evidence tag in a `*.md` under the kit root
(`retros/` excluded, `ledger.d/` included). `CONTRIBUTING.md` is outside that corpus, several landed deltas carry
no Δ-numbered tag, and two deltas (a BUILD-LOOP gate line, a driver-fixture note) are not in the kit at all.

## Scope and constraints
- Allowed: the delta targets, their tests, the three retro files, this document. Not `ledger.d/`, not
  `types/dashboard.md`, not other retros, not the main checkout.
- `retros/INDEX.md`, `lints/INDEX.md` and `BUILD-STATE.md` are generated and NOT committed (`.githooks/pre-push`
  header; `BUILD-LOOP.md` § 5 pre-gate bullet): they are regenerated into a temp dir and swept, never committed.
- No behaviour change is folded (docs, an evidence tag, a fixture comment), so each commit carries
  `Changelog: none (<reason>)` and `Retro: none (trivial: <reason>)` trailers instead of a fragment and a new retro.
- Public repo: no client names. This document cites deltas in plain text, never as evidence tags, so it cannot
  credit the fold audit by itself.

## Per-delta classification
| Retro | Δ | Target | Classification | Evidence (before this branch) |
|---|---|---|---|---|
| r10-changelog-gate | Δ1 | `CONTRIBUTING.md` § 6.2 | landed; to-fold: kit-corpus citation | `CONTRIBUTING.md:291-312` (§ 6.2, tagged Δ1) and `:238`; but `CONTRIBUTING.md` is outside the fold-audit corpus (`sweep-fold-audit.sh:21`) and the ledger fragment tags the stem without Δ1 |
| r10-lint-test-types-in-prod | Δ1 | `types/moduleTest.md` § moduleTest-include.xml | landed; to-fold: Δ-numbered tag | `types/moduleTest.md:86-93` says run `git status` and revert new `<type>` lines; tag at `:93` has no Δ number |
| r10-lint-test-types-in-prod | Δ2 | `BUILD-LOOP.md` § 5 | to-fold | `BUILD-LOOP.md` § 5 (`:207-607`) never names `test-type-in-prod` or the verify-module `moduletest` FAIL row |
| r10-scaffold-station-test | Δ1 | `templates/station-test`, `scaffold-module.sh`, `types/moduleTest.md` § Checklist | already-landed | `templates/station-test/`; `toolbelt/scaffold-module.sh:13-14,185-198`; `types/moduleTest.md:204-213` (tagged Δ1); `tests/scaffold-module.bats` TC-ST1..TC-ST6 |
| r10-scaffold-station-test | Δ2 | `linktargetflags.go`; `lint-link-target-flags.sh` header | already-landed | commit 0cb4ac7c; `go/internal/lint/rules/linktargetflags/linktargetflags.go:142,187-188`; goldens `testdata/classdoc-*.txt`; `toolbelt/lint-link-target-flags.sh:11,17,47-48`; credited by `ledger.d/2026-10-25-r10-scaffold-station-test.md:5` |
| r10-scaffold-station-test | Δ3 | `go/internal/niagaratest/run.go` | landed (runner D6, #566 merge 1be6011a); to-fold: Δ-numbered tag | `go/internal/niagaratest/diagnostics.go:101-157` (tails + kept log); `types/moduleTest.md:306-318` tagged only to rnt-fail-diagnostics Δ1 |
| r10-scaffold-station-test | Δ4 | `fixtures/MinimalDriver`; `types/driver-authoring.md` § 12.1 | doc landed; to-fold: fixture note + tag | `types/driver-authoring.md:982-987` (Fox-off for loopback-only, tagged to r10-run-niagara-test-testng Δ4); `BMinimalDriverLoopbackTest.java:47-48` says Fox stays on a free port without saying it listens on every interface |
| r10-scaffold-station-test | Δ5 | `signingprofile.go`; `run-niagara-test.sh` header; `types/moduleTest.md` § Running from WSL | already-landed | `go/internal/niagaratest/signingprofile.go` + `_test.go`; `toolbelt/run-niagara-test.sh:72-74`; `types/moduleTest.md:252-260` (tagged Δ5) |
| r10-scaffold-station-test | Δ6 | `fixtures/MinimalDash` `-ux`; TC-DASH3 | landed; to-fold: kit-corpus citation | `BMinimalDashServlet.java:59-71` (doGet/doPost); `tests/scaffold-module.bats:66` TC-DASH3; no `*.md` in the corpus tags Δ6; `BUILD-LOOP.md:72` still describes the scaffold as MinimalPan-only |

No delta is rejected.

## Tasks
- [x] T1 · lint-test-types-in-prod: Δ1 tag in `types/moduleTest.md`; Δ2 gate line in `BUILD-LOOP.md` § 5; flip the
  retro to folded. Route: delegated writer (this worker). Checks: fold audit RED→GREEN; sweep; kit-links bats.
- [x] T2 · changelog-gate: Δ1 citation in `BUILD-LOOP.md` § 7 (kit-infrastructure close: the CHANGELOG entry);
  flip to folded. Route: delegated writer. Checks: fold audit RED→GREEN; sweep.
- [x] T3 · scaffold-station-test: Δ3 tag in `types/moduleTest.md` § Running from WSL; Δ4 fixture javadoc note +
  tag in `types/driver-authoring.md` § 12.1; Δ6 in the `BUILD-LOOP.md` § 1 skeleton bullet; flip to folded.
  Route: delegated writer. Checks: fold audit RED→GREEN; sweep; scaffold-module bats; lint-structure bats.

## Acceptance criteria
- Strict fold audit over a fresh `kit-state.sh retro-index`: 0 uncited rows, 0 uncited deltas.
- `sweep-build-state.sh` over fresh kit-state output: exit 0. `gen-retro-index.sh --check`, `gen-lint-index.sh
  --check`: exit 0.
- Bats covering the touched files green; `git status --short` clean after the last commit.

## Test-first policy
No behaviour change, so no unit RED applies. The deterministic check is the strict fold audit: flipping a retro
to folded before its citations exist must FAIL it (RED), and the citations make it pass (GREEN).

## Progress
(updated after each work unit)

- T1 (route: delegated writer, this worker). RED: with only the retro header flipped, the strict fold audit printed
  `r10-lint-test-types-in-prod.md Δ1 not cited`, `Δ2 not cited`, `deltas: 759 checked, 2 not cited`, exit 1.
  GREEN after the Δ1 tag (`types/moduleTest.md` plugin-version trap) and the Δ2 bullet (`BUILD-LOOP.md` § 5, before
  the aggregated punch-list): `0 not cited`, exit 0; sweep exit 0. Bats kit-links, kit-libs-signing-pins,
  kit-p1-control-pins, kit-k15-pins, kit-p1-web-pins: 25/25 ok (`bats -c` 25). Commit 3558a728 (+79/−2).
- T2 (route: delegated writer, this worker). RED: header flipped alone, the strict fold audit printed
  `r10-changelog-gate.md Δ1 not cited`, `folded with no [ev: retro …] citation`, `1 uncited`, exit 1 (the
  `CONTRIBUTING.md` § 6.2 tag is outside the kit corpus). GREEN after the `BUILD-LOOP.md` § 7 kit-infrastructure
  sub-bullet: `0 uncited`, `0 not cited`, exit 0; sweep exit 0. Bats changelog-gate + kit-links: 33/33 ok.
  Commit 159cebb5 (+13/−4).
- T3 (route: delegated writer, this worker). RED: header flipped alone, the strict fold audit printed
  `r10-scaffold-station-test.md Δ3 not cited`, `Δ4 not cited`, `Δ6 not cited`, `deltas: 766 checked, 3 not cited`,
  exit 1. GREEN after the Δ3 tag (`types/moduleTest.md` failed-run diagnostics), the Δ4 note + tag
  (`types/driver-authoring.md` § 12.1 and the `BMinimalDriverLoopbackTest` javadoc) and the Δ6 sentence
  (`BUILD-LOOP.md` § 1 skeleton bullet, which now also names the three `--type` fixtures): `0 not cited`, exit 0;
  sweep exit 0. Bats scaffold-module, kit-driver-pins, lint-structure, kit-links, kit-libs-signing-pins,
  kit-p1-control-pins, kit-k15-pins, kit-p1-web-pins: 89/89 ok (`bats -c` 89).

## Next step
Close: generator checks, worktree clean, hand back to the parent (push and PR are the user's decision).
