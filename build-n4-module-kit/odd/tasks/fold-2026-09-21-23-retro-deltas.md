# ODD feature — Fold the 2026-09-21/23 pending retro deltas into the kit core

**Created**: 2026-09-24 · **Owner**: ODD orchestrator · **Repo**: angeles725/niagara-tools (main)

## Objective
Fold ALL remaining pending propose-never-apply deltas (5 retros, dated 2026-09-21 and 2026-09-23)
from the live-commissioning / Apillm / PANCCADIA sessions into the build-n4-module kit CORE, as a
chained-PR campaign — the same fold contract used by the prior 2026-09-20 apply-deltas campaign
(`odd/tasks/apply-build-n4-module-deltas.md`).

## Problem / why
Five retros sit in `build-n4-module-kit/retros/` with `review-status: pending`:
- `2026-09-21-secure-authoring-isoperational-gate.md` (1Δ)
- `2026-09-21-wb-mapping-ord-npe-and-wsl-windows-jdk.md` (2Δ)
- `2026-09-21-live-diagnosis-hardening-deltas.md` (6Δ)
- `2026-09-21-apillm-wb-subscription-refresh-and-points-deltas.md` (5Δ)
- `2026-09-23-panccadia-defrost-sequencing-hmi-reload-deltas.md` (8Δ)

Until folded, the kit guidance (`types/*.md`, lints) does not carry these lessons, so a builder
cannot benefit from them. The operator authorized folding all of them.

## Authorization & delivery
- Operator authorized: ODD + RDD chained, with commit + push + PR + PR-view + issues + merge, on
  2026-09-23.
- Delivery strategy: **feature-branch-chain**, one PR per retro (FULL promotion → clean INDEX
  flip), branching the next retro's branch from the previous one's commit; the parent merges to
  main. Chain IS the slicing; the ~400-line budget is advisory, not a cap.

## Fold contract (BUILD-LOOP §7 promotion, carried from the prior campaign)
Per retro folded:
1. Insert each Δ's guidance into its target kit file, carrying `[ev: retro <retro-stem> Δn]`
   citation.
2. Flip the retro's `retros/INDEX.md` row `pending → folded` ONLY when the retro's FULL content
   (docs + any script/lint ask) has landed (METHODOLOGY.md doc-vs-script rule, ~line 81). If the
   retro marks a lint as optional/deferred, record it as an explicit deferred item (BUILD-STATE.md
   kit open_issue or a forward reference) so the fold is honest.
3. Flip the retro file's own `<!-- review-status: pending -->` marker to `folded` in the same
   commit (matches what already-folded retro files use).
4. Update the kit `BUILD-STATE.md` self-envelope in the same commit range (envelope-pairing rule).
5. Commit trailer: `Retro: promotion (folds <ids> from <retro-stem>)`.
6. Gates that must pass before closing a task:
   - `bash toolbelt/sweep-build-state.sh BUILD-STATE.md retros retros/INDEX.md`
   - `bash toolbelt/sweep-fold-audit.sh --strict retros/INDEX.md .` (run from `build-n4-module-kit/`)
   - `shellcheck (pinned 0.10.0) scripts/*.sh build-n4-module-kit/toolbelt/*.sh tests/*.bats tests/helpers/*.bash`
   - `bats tests/*.bats` (full suite green)
   - any other CI step in `.github/workflows/ci.yml` relevant to the files touched (lint-guard-pins
     --strict when a new lint script is added).

## Constraints
- No AI attribution in commits (conventional commits). Promotion trailer required.
- Docs default to English, neutral/professional register.
- A retro's optional/deferred lint ask is implemented only when small and clearly specified
  (script + `# Mutation: <id> -- <desc>` guard-pin whose id matches a bats `@test` name, a bats
  file with fixtures that FAIL when the guard is removed, and the script named in BUILD-LOOP.md or
  skill/SKILL.md per kit-links). Otherwise it is recorded as deferred, not silently dropped.
- TDD mode: **off** — source: no project TDD configuration found for this worktree; functional
  checks are the shellcheck 0.10.0 + bats + sweep gates above (runner: bash/bats as pinned by CI).

## Task checklist (one PR per retro)
- [x] T1 · secure-authoring-isoperational-gate (1Δ) → `types/security.md` new §5.1 licensing
      ongoing-work gate. Route: delegated direct (single doc file, trivial size) — bundled with the
      feature-doc creation commit.
- [x] T2 · wb-mapping-ord-npe-and-wsl-windows-jdk (2Δ) → `types/issues-and-gotchas.md` (new §A/B
      gotcha + §D symptom entry) + `types/wb-widgets.md` cross-link. Optional `lint-set-null-ord`
      candidate — evaluated and recorded DEFERRED (not implemented this task; see progress log).
      Route: delegated direct writer (2 non-trivial doc files — writer trigger).
- [ ] T3 · live-diagnosis-hardening-deltas (6Δ: 3 lints `lint-changed-hot-write` /
      `lint-persist-hot-write` / `lint-session-store-lazy-evict` + 3 gotcha docs RUN8/PER8/UXS7).
      Route: delegated direct writer (script+bats+doc — writer trigger). NOT started by this writer.
- [ ] T4 · apillm-wb-subscription-refresh-and-points-deltas (5Δ incl.
      `lint-wb-external-ord-value`). Route: delegated direct writer. NOT started by this writer.
- [ ] T5a/T5b · panccadia-defrost-sequencing-hmi-reload-deltas (8Δ: lints `spa-poll-no-recovery`,
      `inert-coordination`; version-bump drift gate; deployed-baseline template; HOA contract;
      cycle anchors; 9p build-location gate; preflight lsof perf). Route: delegated direct writer,
      likely split T5a/T5b given size. NOT started by this writer.

## Acceptance criteria
- Every retro row in `INDEX.md` for the 5 target retros is `folded` (or explicitly left `pending`
  with a documented reason) when the campaign closes.
- Every folded Δ cited in a kit file (`sweep-fold-audit --strict` clean for the retros this writer
  touched).
- shellcheck + bats + both sweeps green after each task's commit.
- `BUILD-STATE.md` kit envelope updated in the same commit range as each fold.
- Deferred lint asks (if any) are recorded honestly, not silently dropped.

## Progress log
- 2026-09-24: Feature doc created. Bounded writer scope for this run: T1 + T2 only (T3-T5b left
  for a later task/writer).
- 2026-09-24: T1 FOLDED — `types/security.md` new §5.1 "Ongoing-work gate — serviceStarted() only
  guards STARTUP" (advisory license-fault rule; `isOperational()` gate on `changed()`/timers/
  servlet-write callbacks; deferred lint candidate `lint-license-isoperational-gate` recorded,
  NOT implemented — retro explicitly asks to check overlap with `lint-status-parity`/
  `lint-silent-protection` first, which is out of this task's bounded scope). INDEX row + retro
  marker flipped to folded. BUILD-STATE.md kit envelope updated. Commit `72fcd33` on branch
  `odd/fold-secure-authoring`. Gates: `sweep-build-state.sh` exit 0; `sweep-fold-audit.sh
  --strict` 167 folded/167 cited/0 uncited; shellcheck 0.10.0 exit 0; `bats tests/*.bats`
  628 ok / 0 not-ok.
- 2026-09-24: T2 FOLDED — `types/issues-and-gotchas.md` new §H3 "A `-wb` manager mapping
  `point.getSlotPathOrd()` into a `BOrd` `set()` can NPE on a null ORD" (Δ1, distinguished from the
  PD-FE1 picker gotcha) + new §D5 "WSL `gradlew` run directly can auto-detect a Windows JDK and die
  with `URISyntaxException`" (Δ2, points at `structure.md §L10`). `types/wb-widgets.md`
  field-editors § cross-linked to §H3. `lint-set-null-ord` optional candidate evaluated: DEFERRED,
  not implemented — recorded as a `BUILD-STATE.md` kit open_issue (a single-purpose static-source
  guard-body lint needs its own guard-pin/bats/report-module wiring pass; folding it in-line with a
  doc task risked a rushed, under-tested lint). INDEX row + retro marker flipped to folded.
  BUILD-STATE.md kit envelope updated (2 open_issue lines for both deferred lints from T1+T2).
  Branch `odd/fold-wb-mapping-ord-npe`, branched from the T1 commit `72fcd33`. Gates:
  `sweep-build-state.sh` exit 0; `sweep-fold-audit.sh --strict` 168 folded/168 cited/0 uncited;
  shellcheck 0.10.0 exit 0; `bats tests/*.bats` 628 ok / 0 not-ok. Commit sha recorded in this same
  commit's parent commit-message reference (see `git log` on this branch).
