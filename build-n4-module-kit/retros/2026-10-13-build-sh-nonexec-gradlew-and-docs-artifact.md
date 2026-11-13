<!-- review-status: pending -->
# 2026-10-13 · kit · build-sh-nonexec-gradlew-and-docs-artifact

**Session**: panccadia-leon CompPan 2.8.1 kit verify gate (branch fix/comppan-hand-proof-count, commits e1ec65b + 7ca1b57)
**Delta count**: 2

## What happened
A previous writer reported that `toolbelt/build.sh` "hung with an empty log" on `Cliente/panccadia-leon/Compresores` and fell
back to a hand-run `sh gradlew ...`. Re-running build.sh under `timeout 600` did NOT reproduce a hang: with the tracked
`Compresores/gradlew` at mode 100644 (repo has `core.fileMode=false`), build.sh exits 10 in under a second with
"no executable ./gradlew in <root> or any ancestor" and the hint `chmod +x`. The kit offers no flag/env to run a
non-executable wrapper through `sh`, so the only way to run the chain without touching the client repo was a temporary
working-tree `chmod +x` (invisible to git under `core.fileMode=false`) restored to 644 afterwards. With that, the full chain ran
end-to-end (preflight PASS, gradle BUILD SUCCESSFUL in 8s, verify gate ALL PASS, report-module 27 PASS / 2 FAIL / 55 WARN / 22 SKIP).
The report also showed a second defect: report-module treats every immediate subdirectory of `<module-root>` as a profile
artifact, so `CompPan/docs/` became an "artifact" contributing 17 SKIP rows and inflating the artifact count to 2.

## Evidence
- build.sh with mode-644 gradlew: `rc=10`, single line "no executable ./gradlew in .../Compresores or any ancestor" — `[ev: code toolbelt/build.sh:88-100]`
- gradle invocation requires exec bit: `( cd "$GRADLE_ROOT" && ./gradlew ... )` — `[ev: code toolbelt/build.sh:240]`
- client repo: `git ls-files -s Compresores/gradlew` = 100644, `git config core.fileMode` = false — `[ev: commit 116c49a]`
- full chain after temporary chmod: rc=50 only from 2 pre-existing lint-structure FAILs (L10 gradle.properties abs path, L14 hardcoded gradlePluginVersion) — `[ev: commit 7ca1b57]`
- artifact discovery `find "$MODULE_ROOT" -maxdepth 1 -mindepth 1 -type d` (no `<MOD>-<profile>` filter) → `docs  SKIP ...` x17 — `[ev: code toolbelt/report-module.sh:194-198]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | When `gradlew` exists but is not executable, run it as `sh ./gradlew` (or behind an explicit `--sh-gradlew` / `$BUILD_N4_SH_GRADLEW=1`) with a WARN naming the mode, instead of exit 10; keep the walk-up keyed on existence, not `-x`, so a 644 wrapper at the group root is not skipped in favour of an ancestor | `toolbelt/build.sh` § B7 gradlew walk-up | `[ev: code toolbelt/build.sh:88-100]` |
| Δ2 | Restrict report-module artifact discovery to `<MOD>-{rt,ux,wb,se,doc}`-shaped dirs (or dirs with a gradle file), so `docs/` and similar are not reported as artifacts | `toolbelt/report-module.sh` § Discover profile artifacts | `[ev: code toolbelt/report-module.sh:194-198]` |

## Lessons
- A "hang with an empty log" must be re-run under `timeout` before it is recorded: here it was a fast, clearly-worded exit 10, not a hang.
- Client repos with `core.fileMode=false` routinely carry a 100644 `gradlew`; any kit gate keyed on `-x` will reject them.
- An artifact list derived from "every subdirectory" drifts as soon as a module grows a docs/ or notes/ folder; key discovery on the profile naming contract.
