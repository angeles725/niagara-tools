<!-- review-status: folded -->
# 2026-10-01 · kit · operator-manual-lockstep

**Session**: PANCCADIA gap review — client-facing documentation goes stale when UI labels or behavior change
**Delta count**: 3

## What happened
The DashboardPan configuration manual (HTML + PDF) is a client deliverable tracked in the client repo. Two commits on
2026-09-30 only re-synced it to behavior that had already changed (a renamed cut-out label and a login flow step), i.e.
the manual lagged the code and was fixed after the fact. Another client repo carries a different manual set (md/tex/pdf)
and untracked `.bak` report copies. The kit has no rule that a manual is part of the deliverable, no structure or
version-stamp convention, and no PDF build recipe; the only PDF recipe known is in a project memory note.

## Evidence
- Manual fix commits after the behavior change: `9f48f0d` "rename cut-out label to Temp. de corte in configuration manual and PDF" and `3bc54cd` "manual says to tap the value again after the login", both 2026-09-30 `[ev: Cliente/panccadia-leon 9f48f0d, 3bc54cd]`
- The PDF is tracked binary next to its HTML source: `Dashboard/docs/manual-configuracion/PANCCADIA-Manual-Configuracion-DashboardPan-2026-09-30.pdf` + `manual-configuracion.html` (`git ls-files`); `9f48f0d` changed both (2 files, PDF binary delta) `[ev: Cliente/panccadia-leon 9f48f0d]`
- Second client repo uses a different manual layout: `docs/MANUAL-modulos-panccadia.{md,tex,pdf}` (md+tex tracked, pdf untracked) and `docs/reporte-panccadia-modulos.{html,pdf}.bak` on disk `[ev: Cliente/Leon-Guanjuato docs/]`
- PDF recipe lives only in memory: Windows Chrome headless, report HTML+PDF in `deliverables/` `[ev: memory harbor-greenmax-delivery-report.md]`
- Kit has no manual rule: `grep -niE 'manual|\.pdf' BUILD-LOOP.md types/distribution.md` returns only an unrelated "manual wiring" line (`types/distribution.md:270`) and the commissioning-verify "MANUAL footer" (`BUILD-LOOP.md:171`, a different meaning) `[ev: kit grep 2026-10-01]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Close-gate rule: any operator-visible label, default, unit, or behavior change updates the operator manual (source and rendered PDF) in the SAME commit; the commit trailer or retro states `manual: updated` / `manual: n/a (no operator-visible change)`. | `BUILD-LOOP.md` § `7. Retro + close (HARD close gate — not optional)` | `[ev: Cliente/panccadia-leon 9f48f0d, 3bc54cd]` |
| Δ2 | Document a client documentation deliverable: manual structure (install, configuration, operation, alarms, troubleshooting), a version stamp equal to the module `defaultModuleVersion`, one source format (HTML or md) plus a rendered PDF, the headless-Chrome PDF recipe, and "no `.bak` or dated duplicates in the repo". | `types/distribution.md` (new §) "Client documentation deliverable" | `[ev: memory harbor-greenmax-delivery-report.md]` |
| Δ3 | Advisory `toolbelt/lint-manual-labels.sh <manual.html|md> <rc-or-lexicon-dir>`: every quoted UI label in the manual must exist in the UI source or lexicon; WARN-only (labels may be paraphrased). [INFER] this would have caught the cut-out rename. | `toolbelt/` (new) + `BUILD-LOOP.md` § `5. Verify gate (before "done")` | `[ev: Cliente/panccadia-leon 9f48f0d]` |

## Lessons
- A manual is code that nobody compiles: it drifts unless the same commit touches it.
- Tracking the rendered PDF with its source makes drift visible in `git log`.
- Recipes known only to one session's memory are not kit knowledge.
- Backup copies in the repo are noise; git is the backup.
