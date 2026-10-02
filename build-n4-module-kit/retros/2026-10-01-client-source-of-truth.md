<!-- review-status: folded -->
# 2026-10-01 · kit · client-source-of-truth

**Session**: PANCCADIA gap review — sessions built or read from the wrong checkout; installed skill drifts from the kit copy
**Delta count**: 3

## What happened
Over three weeks the "current" PANCCADIA source moved between a Windows Downloads checkout, a stale local client
checkout, a read-only worktree, and finally `Cliente/panccadia-leon`. Each move was recorded only in project memory, so
sessions re-discovered which tree was authoritative (one design drifted because of a stale checkout). The kit already
has optional deployed-baseline fields in BUILD-STATE, and the pending 2026-09-26 ledger retro covers the version-defect
side; neither makes the source of truth a required, checked field. Separately, the installed copy of the skill lacks the
`build.sh` auto-chain wording present in the kit's `skill/SKILL.md`, so an agent following the installed skill gets older
instructions.

## Evidence
- Authoritative source moved three times, each only in memory: Downloads checkout on 2026-09-23, stale local `Leon-Guanjuato` at 4f5f1c7 with worktree `main-a109249` as the read tree, then `Cliente/panccadia-leon` from 2026-09-27 `[ev: memory panccadia-deploy-2026-09-23-defrost-hmi.md, client-reads-use-a109249-worktree.md, panccadia-source-location-2026-09-27.md]`
- Not every client tree is a git repo: `find Cliente -maxdepth 3 -name .git` lists `Leon-Guanjuato`, `panccadia-leon` and worktrees; a deeper `find` (maxdepth 5) under `Honeywell`, `Juarez`, `LLM` finds only `Honeywell/MX60/chihuahua/.git`, so `Juarez/Umbrella` and `LLM/Apillm` have no repo `[ev: find Cliente 2026-10-01]`
- BUILD-STATE documents optional deployed-baseline fields only (`deployed_jar_sha256`, `deployed_build_millis`, `deployed_source_commit`, `deployed_baseline_date`) `BUILD-STATE.md:23`; no `source_of_truth` field exists `[ev: BUILD-STATE.md:23]`
- `toolbelt/preflight.sh` contains no git-repository check (`grep -n git` returns no check) `[ev: toolbelt/preflight.sh]`
- Installed skill differs from the kit copy: `diff skill/SKILL.md ~/.claude/skills/build-n4-module/SKILL.md` differs at lines 60-61 (kit: build.sh runs the full automatic chain; installed: "Build with build.sh; verify major-52 + signed") `[ev: skill/SKILL.md:60-61]`
- Sibling pending retro on version/source ambiguity, not duplicated here: `retros/2026-09-26-panccadia-version-defect-ledger.md` `[ev: retros/2026-09-26-panccadia-version-defect-ledger.md]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | `preflight.sh`: WARN when the module root is not inside a git work tree, FAIL when it is a detached or stale checkout behind its declared `source_of_truth` branch (compare with `git rev-parse` only; no network). | `toolbelt/preflight.sh` header | `[ev: find Cliente 2026-10-01]` |
| Δ2 | Add a required (DECLARED) per-module field `source_of_truth: <path>@<branch>@<commit>` to the envelope and update it whenever the authoritative tree moves; `unknown` stays honest. | `BUILD-STATE.md` § `How to read this file` (DECLARED field list) | `[ev: memory panccadia-source-location-2026-09-27.md]` |
| Δ3 | Close gate / sweep fails when the installed skill differs from `skill/SKILL.md`; remedy is `scripts/install-skill.sh --force`. | `BUILD-LOOP.md` § `7. Retro + close (HARD close gate — not optional)` + `toolbelt/sweep-build-state.sh` | `[ev: skill/SKILL.md:60-61]` |

## Lessons
- "Where is the source" is a fact of the project, so it belongs in the project file, not in a session's memory.
- A checkout without git cannot answer "which commit is on the station".
- Installed copies of the kit rot silently; compare them mechanically.
