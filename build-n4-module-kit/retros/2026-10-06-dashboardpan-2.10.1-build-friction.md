<!-- review-status: pending -->
# 2026-10-06 · kit · dashboardpan-2.10.1-build-friction

**Session**: PANCCADIA operator support. DashboardPan 2.10.1 (EXIT button, header stamp removed, auto-reload watchdog removed then restored) built in a fresh worktree `Cliente/panccadia-leon-worktrees/hmi-manual-reload-only`, branch `fix/hmi-manual-reload-only`.
**Delta count**: 7

## What happened
A one-file UI change (P0) took far longer to build than its 10-14 s of gradle time. The first `build.sh` run hung about 6 min with an EMPTY log until the user interrupted it. Root cause: in the new worktree `Dashboard/gradlew` was not executable. The client repo stores it as `100644` with `core.fileMode=false`, and the original checkout had a local `chmod +x`. I also passed a RELATIVE module root (`Dashboard`). The gradlew walk-up then looped `dirname "." == "."` forever and never printed or failed.

Once fixed, a full chain run took 224 s, of which gradle was 47 s. Most of the rest was preflight (~1 min) plus `report-module` (~2-3 min) linting a 2.5 MB `index.html` with inline data URIs. A fast loop (`--no-preflight --no-report`) took 12 s. The user complained twice ("antes tardabas menos"). The previous sessions had used `fast-build.sh` (gradle only), and nothing tells the operator which phase is slow or how long each one should take.

Smaller friction:
- `rc-scan` ord-literal flagged the word `station:` inside a JS COMMENT. This was a false positive; I fixed it by rewording the comment.
- `kit-ticket.sh` failed outright because `gh` is installed but the repo has no known GitHub host. The offline fallback only covers `gh` being absent.
- The module's `preview-server.py` takes a POSITIONAL port. I passed `--port 8770`, so it crashed. My readiness loop ran in the same backgrounded command and burned the 120 s tool timeout before I saw the traceback, so the 1280×800 preview was skipped.
- Jars were built twice before the scope settled (the watchdog removal was reverted by the user). Superseded jar hashes had to be called out by hand.

## Evidence
- `build.sh:86-88` walk-up loop `while [ -n "$GRADLE_ROOT" ] && [ "$GRADLE_ROOT" != "/" ] && [ ! -x "$GRADLE_ROOT/gradlew" ]` is never terminated for a relative path. The first log was 0 bytes after about 6 min. `[ev: build.sh:88]`
- `git ls-files -s Dashboard/gradlew` gives `100644`, the main checkout has `-rwxr-xr-x`, and the worktree has `-rw-r--r--`. `[ev: Cliente/panccadia-leon 17c80d3]`
- Full chain: `build rc=50 in 224s`, `BUILD SUCCESSFUL in 47s`. Fast chain: `rc=0 in 12s`, `BUILD SUCCESSFUL in 10s`, `verify-module 23 passed, 0 failed`. `[ev: cf8a8d3]` `[ev: ba301c9]`
- `rc-scan  index.html:2896  ord-literal` on the comment line `... (LogoutServlet de la station: ...`. Gone after rewording. `[ev: d1de5e5..ba301c9]`
- `kit-ticket: gh issue create failed`, preceded by "none of the git remotes ... point to a known GitHub host". `[ev: toolbelt/kit-ticket.sh:9]`
- `preview-server.py` line 33 `PORT = int(sys.argv[1])` raised `ValueError: invalid literal for int() with base 10: '--port'`. `[ev: DashboardPan-ux/preview-server.py:33]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Resolve ROOT with `cd "$ROOT" && pwd -P` before the walk-up, stop when `dirname` returns its own input, and FAIL FAST (exit 10) with "gradlew exists but is not executable: chmod +x <path>" when a non-exec `gradlew` is found on the way. Add a bats test for both cases. | `toolbelt/build.sh` (gradlew walk-up, B7 block) | `[ev: build.sh:88]` |
| Δ2 | Worktree recipe: after `git worktree add`, run `chmod +x <group>/gradlew`. Recommend `git update-index --chmod=+x` in client repos so the mode is versioned. Preflight checks `-x gradlew` under the gradle root. | `build-verify.md` § `Build location — ext4 vs 9p/drvfs` + `toolbelt/preflight.sh` | `[ev: Cliente/panccadia-leon 17c80d3]` |
| Δ3 | Print a per-phase timer line (`==> <phase> done in Ns`) and a final summary in build.sh, and document the two modes with expected times: full chain (~4 min on a 2.5 MB rc) for hand-off, fast loop `--no-preflight --no-report` (~15 s) for iteration. The skill states the expected duration before a long build. | `toolbelt/build.sh` + `BUILD-LOOP.md` § `4. Build — the ONLY valid build` | `[ev: cf8a8d3]` |
| Δ4 | Speed up `report-module` on big rc files: rc-scan skips the body of `data:` URIs (scan only up to the attribute start), and report-module runs lints in parallel (capped jobs). Measure before and after on DashboardPan-ux. | `toolbelt/rc-scan.sh` + `toolbelt/report-module.sh` | `[ev: cf8a8d3]` |
| Δ5 | rc-scan `ord-literal` ignores JS/CSS/HTML comments (`//…`, `/*…*/`, `<!--…-->`). Add a bats fixture where `station:` appears only in a comment (must pass) and one in a string (must FAIL). | `toolbelt/rc-scan.sh` § `ord-literal` | `[ev: d1de5e5..ba301c9]` |
| Δ6 | kit-ticket.sh falls back to the offline ticket file when `gh issue create` FAILS (no known host, auth, or network), not only when `gh` is absent, and says which path it took. | `toolbelt/kit-ticket.sh` (SKIP path) | `[ev: toolbelt/kit-ticket.sh:9]` |
| Δ7 | Preview step: start the server in its own background process and poll readiness with a bounded loop (≤10 s) that also checks the PID is alive and prints the server log on death. Module preview servers accept `--port N` as well as a positional port. | `BUILD-LOOP.md` § `3. Preview (UI types) — BEFORE compiling` + `types/dashboard.md` | `[ev: DashboardPan-ux/preview-server.py:33]` |

## Lessons
- A silent hang is worse than a failure: every loop over paths must have a fixed point that terminates (absolute path + "dirname unchanged" stop).
- In a fresh worktree, check executable bits before the first build; `core.fileMode=false` hides them.
- Tell the user which phase is slow and how long it should take before running the full chain; use the fast loop while iterating and the full chain once for hand-off.
- Lints that grep source text must ignore comments, or the fix becomes "reword the comment", which hides the defect.
- Rebuild only after the change scope is confirmed; when jars are superseded, list the stale hashes explicitly in the hand-off.

---
**Status**: PENDING — INDEX row appended: `| 2026-10-06-dashboardpan-2.10.1-build-friction.md | kit | 2026-10-06 | pending | 7 |`
