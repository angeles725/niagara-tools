<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-11-19 · kit · kit-gzip-bundle-writer-briefs

**Session**: kit work on deltas K2 and K3 of a client dashboard upgrade retro (a large SPA served from a `-ux` part through a site tunnel).
**Delta count**: 3

## What happened
The client dashboard's 832 KB `dist/app.js` was served plain through a slow tunnel and delayed the whole page; a deterministic gzip twin made it 241 KB. Nothing in the kit flagged the plain bundle. Separately, four parallel SPA writers each passed their own tests, and their merge showed an MM4 FAIL on 19 files (new files under no module-map row) that only `build.sh` found.

## Evidence
- `go/internal/reportmodule/special_bundle.go` (native step, no toolbelt script): RED was `r.bundleGzip undefined` in `special_bundle_test.go` (TestBundleGzip, TestBundleGzipNoDist, TestBundleGzipUnreadable); GREEN after the step and its `members.go` row. Mutations RM-bundle-gzip-twin (drop the twin test) and RM-bundle-gzip-limit (300 000 bytes) each flip one case. End to end: `tests/report-module.bats` RM-bundle-gzip.
- No golden case holds a `src/rc/dist/`, so the 123 golden cases are unchanged.
- Exemplar of the serving side (a client `-ux` part): a pure `GzipVariant.of(path, acceptEncoding)` helper, the static handler setting `Content-Encoding` and `Vary`, and the bundle step writing the twins with `gzipSync`.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-19. report-module `bundle-gzip`: WARN a `-ux` `src/rc/dist/*.js` over 300 KB with no `.gz` twin; PASS otherwise; no row without `dist/`; ERROR on an unreadable `dist/` | `go/internal/reportmodule/special_bundle.go`, `members.go` | `[ev: retro kit-gzip-bundle-writer-briefs Δ1]` |
| Δ2 | APPLIED 2026-11-19. Static serving doctrine: deterministic `.gz` twins in the bundle step, `Content-Encoding: gzip` + `Vary: Accept-Encoding`, the plain Content-Type kept | `types/dashboard.md` § Static serving, `types/frontend-standard.md` § 5 and § Enforcement | `[ev: retro kit-gzip-bundle-writer-briefs Δ2]` |
| Δ3 | APPLIED 2026-11-19. Parallel SPA writer briefs name the module-map row of each path; each writer runs `lint-module-map.sh` and the whole `npm test` before done; the parent runs the seam check after the merge | `ORCHESTRATION.md` §3.c, `BUILD-LOOP.md` § 2 | `[ev: retro kit-gzip-bundle-writer-briefs Δ3]` |

## Lessons
- A size problem the operator feels (a slow first load) can be a cheap static check on the shipped tree.
- Writers that each pass their own tests still collide on shared lists; the brief should prevent the overlap and the writer should run the shared checks.
