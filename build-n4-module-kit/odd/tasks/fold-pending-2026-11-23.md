# Fold every pending build-n4-module retro (2026-11-23)

Locator: `build-n4-module-kit/odd/tasks/fold-pending-2026-11-23.md` · base origin/main 59931cec · one worktree and
branch per batch under `niagara-tools-worktrees/fold-<batch>` (`fold/2611-<batch>`). Engram mirror
`odd/fold-pending-2026-11-23/tasks`.

## Specs
- S1. The user's request: "Vamos a trabajar en las retros/deltas de /build-n4-module ... Trabaja de forma modular,
  optimizacion, tambien checa que esten todo trasladado a GO, tambien necesito que trabajes con la arquitectura ...
  Si ves cualquier mejora, implementala." Every pending retro delta is folded: landed with an exact
  `[ev: retro <stem> Δk]` tag, implemented then tagged, or parked out of repo with `[deferred: retro <stem> Δk]`
  and a reason.
- S2. Each retro whose deltas are all accounted for flips to `<!-- review-status: folded -->`; CI's
  `sweep-fold-audit.sh --strict --exact-stems --deltas-since 2026-09-24` stays at 0 uncited.
- S3. Code deltas follow the kit tooling doctrine (BUILD-LOOP § 5): new tools/checks in Go under `go/`, bash only as a
  launcher, a test per rule, CHANGELOG fragment or `Changelog: none (<reason>)`, a `Retro:` trailer.
- S4. Go-port audit: every toolbelt script that parses structured input is a launcher over `go/`; the remaining bash
  logic is listed and ported or owed as a `Go lint` ledger item.
- S5. Public repo: no client names in tracked files.
- S6. After build-n4-module, the same for mcp-n4 (its pending retros and improvements).

## Tasks
| ID | S# | Scope | Route | Commit / PR |
|---|---|---|---|---|
| T1 | S1 S2 | flip the 29 retros already fully cited | inline, `fold-flip` | |
| T2 | S1-S3 | go-* port retros (7 retros, 33 uncited Δ) | worker, `fold-go-ports` | |
| T3 | S1-S3 | doc-coverage-audit (24 Δ) | worker, `fold-doc-audit` | |
| T4 | S1-S3 | gate-followups-and-template-lints, ntpl-*, lint-dashboard-write-save (30 Δ) | worker, `fold-gate-lints` | |
| T5 | S1-S3 | build-kit-bench-session, kit-p1-security-guards, p1-build-deploy-safety, delivery-guards, small kit retros (~42 Δ) | worker, `fold-guards-misc` | |
| T6 | S1-S3 S6 | mcpn4-* retros in this kit (15 Δ) | worker, `fold-mcpn4` | |
| T7 | S4 | Go-port audit of `toolbelt/` and `scripts/` | explore then worker | |
| T8 | S6 | mcp-n4 pending retros and improvements | after T1-T7 | |

## Log
- L1 (user, verbatim): "Vamos a trabajar en las retros/deltas de /build-n4-module, vas a trabajar con ODD+RDD(GRANTED)+
  SI GENTLE-AI PIDE REVISION, DALE PERMISO + ENGRAM. Ahora tambien tienes permitido realizar commit + push + pr + pr
  viewv + issues + merge. Tu toma las decisiones de acuerdo con tus recomendacions, asi que no me preguntes a mi, tu
  toma las deciones, y con eso ve implementando, mejorando, etc. Y trabaja con todo lo que se pueda hasta completar
  todo, y las implementaciones o mejoras que veas que se puedan desarrollar de mejor forma, dale con lo tuyo. Trabaja
  de forma modular, optimizacion, tambien checa que esten todo trasladado a GO, tambien necesito que trabajes con la
  arquitectura, tienes permitido todo. Si ves cualquier mejora, implementala. Dale. Si llegas a terminar con todas las
  mejoras con /build-n4-module, ve con mcp=n4, y haz lo mismo. TIenes permitido trabajar con worktrees, etc."
- L2: baseline at 59931cec: 62 pending retros, 274 deltas; 142 deltas had no exact Δ tag; 29 retros were fully cited
  and only needed the marker flip (T1, strict audit 441 folded / 0 uncited after the flip).
