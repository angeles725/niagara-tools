<!-- review-status: folded -->
<!-- Marker lifecycle: 'folded' = promoted into the kit core in this same PR. -->
# Kit Retro — kit · 2026-09-11 · Module worktree location guard (orient-guard.sh)

> **Kit close retro (§7).** Implements the guard proposed in the companion research retro
> (`niagara-research/retros/2026-09-11-module-worktree-location-retro.md`).
> All five proposed deltas D1–D5 are folded into the kit in this PR.
> Citation: `[ev: retro module-worktree-location-retro]`.

## What happened

Campaign 9 (2026-09-06/07) ran build sessions from
`/home/cristian/niagara-panccadia-leon` — an unauthorized worktree — instead of the
canonical `Cliente/` tree. The rule existed only as an engram memory key, invisible
to the fold pipeline and to any session's orient step. The research retro captures
the incident; this kit retro closes the guard implementation.

## Evidence

- Design confirmed sole legal prefix: `/home/cristian/modulos_niagara_n4/Cliente/`.
- `niagara-research-worktrees/` trees are explicitly NOT legal for module builds.
- The test seam `BUILD_N4_LEGAL_ROOT_PREFIX` + `BUILD_N4_CLIENTE_OVERRIDE=1` allow
  bats-testable coverage without machine coupling.
- 11/11 orient-guard.bats GREEN; 9/9 kit-links.bats GREEN (including new L9).

## Proposed kit deltas (fold list)

All five deltas D1–D5 from the research retro are now folded:

| # | Delta | Status | File |
|---|-------|--------|------|
| Δ1 | `toolbelt/orient-guard.sh` — PASS/FAIL/WARN guard; `realpath` + `cd -P` canonicalization; trailing-slash compare; fail-closed on non-`1` override; test seam WARN-loud | FOLDED | `toolbelt/orient-guard.sh` (new) |
| Δ2 | `BUILD-LOOP.md` §0.a: first orient bullet invokes `orient-guard.sh <module-root>` before reading BUILD-STATE | FOLDED | `BUILD-LOOP.md` §0.a |
| Δ3 | `skill/SKILL.md` step 1: prepend orient-guard call; `[ev: retro module-worktree-location-retro]` citation | FOLDED | `skill/SKILL.md` step 1 |
| Δ4 | `tests/orient-guard.bats` — 11 tests covering PASS/FAIL/WARN/non-1/symlink/relative/trailing-slash/ClienteX/usage | FOLDED | `tests/orient-guard.bats` (new) |
| Δ5 | `tests/kit-links.bats` L9 — asserts `orient-guard.sh` in BOTH routing docs | FOLDED | `tests/kit-links.bats` L9 |

## Lessons

1. A guard that exists only in engram memory is invisible to the fold pipeline and
   to future sessions. A formal retro + kit implementation is the only durable form.
2. `BUILD_N4_CLIENTE_OVERRIDE=1` is the sole sanctioned escape hatch; any non-`1`
   value (including `0` or `yes`) is treated as unset — fail-closed by design.
3. Canonicalizing with `realpath` before prefix compare prevents bypass via symlinks,
   relative paths, or trailing-slash variants; appending `/` prevents the `ClienteX/`
   sibling false-positive.
4. The test seam (`BUILD_N4_LEGAL_ROOT_PREFIX`) must ALWAYS be loud (emit WARN) —
   otherwise bats tests silently change production behavior.
5. K19 routing requires explicit citation in BOTH `BUILD-LOOP.md` §0.a AND
   `skill/SKILL.md` step 1; the L9 bats pin makes the obligation machine-verifiable.

## Deviations from design

None. All five deltas folded as specified. L9 added after L8, before L6 (existing order preserved).
