# Skill Registry — niagara-tools

Detected skills available for this project. Maintained by `sdd-init` (last refresh 2026-05-17).

## Project nature

Tooling and knowledge-base repo for Niagara N4 module development. Consumers live under
`../Cliente/**` (today: `../Cliente/Honeywell/MX60/chihuahua/`). The repo itself ships
shell scripts (`scripts/`), markdown docs (`docs/`), and agent instructions
(`README.md`, `CLAUDE.md`). It is NOT a Niagara module: no Gradle, no Java,
no jars.

## User Skills (~/.claude/skills/)

| Skill | Triggers | Relevance |
|-------|----------|-----------|
| `cognitive-doc-design` | "design doc", "README", "onboarding", "guide" | HIGH — Repo is largely docs; every guide here is review-facing |
| `work-unit-commits` | "split commits", "chained PRs" | HIGH — Bootstrap and follow-ups will mix scripts + docs; keep tests with code |
| `chained-pr` | "stacked PRs", "split PR", `>400 LOC` | MEDIUM — Only triggers if a single PR crosses the 400-line budget |
| `branch-pr` | "open PR", "create PR" | MEDIUM — Requires GitHub remote (not configured yet); skill enforces `type/desc` branches and `Closes #N` |
| `issue-creation` | "bug report", "feature request" | MEDIUM — Same as above; only when GitHub remote exists |
| `comment-writer` | PR/issue replies, reviews | OPTIONAL — Voseo rules + Rioplatense tone for async replies |
| `judgment-day` | "judgment day", "dual review", "juzgar" | OPTIONAL — Worth invoking on shell scripts that touch the live station deploy (high blast radius) |
| `skill-creator` | "new skill", "agent instruction" | LOW — Only if niagara-tools starts shipping its own project-level skills |
| `go-testing` | Go test patterns | NONE — No Go in this repo |
| `sdd-*` (init/explore/propose/spec/design/tasks/apply/verify/archive/onboard) | SDD workflow commands | HIGH — This is the active workflow; first change is `niagara-tools-bootstrap` |

## Project Conventions

| File | Purpose |
|------|---------|
| `~/.claude/CLAUDE.md` | Global instructions (Senior Architect persona, Strict TDD enabled, never amend commits, deploy = backup → build → copy → verify, use bat/rg/fd/sd/eza instead of cat/grep/find/sed/ls) |
| `README.md` | Top-level entry; planned MVP scope listed there |
| `CLAUDE.md` (planned) | Agent-facing companion: when to invoke ng-deploy.sh, where gotchas live, how knowledge-base is structured |
| `scripts/ng-deploy.sh` (planned) | Canonical WSL → Windows station deploy wrapper. MUST: backup to `_backups/`, build jars, copy to `modules/`, verify `<type>` count + cache buster |
| `docs/GOTCHAS.md` (planned) | Top-level cross-project gotchas (BQL, Slot-O-Matic regions, hot-reload, cache busters) |
| `docs/knowledge-base/*.md` (planned) | Per-topic deep-dives (e.g. `bql-gotchas.md`) |
| `.gitignore` | Excludes IDE artifacts, secrets (`.env*`, `*.key`, `*.pem`, `secrets/`), logs, PIDs |

## Consumers (NOT in this repo)

| Path | Purpose |
|------|---------|
| `../Cliente/Honeywell/MX60/chihuahua/` | Active Niagara N4.14 module (Honeywell BMS MX60). Has its own `openspec/`, `.atl/skill-registry.md`, `BUILD_WORKFLOW.md`, `FRONTEND_ARCHITECTURE.md`. niagara-tools should not duplicate chihuahua's per-module docs — only host cross-project knowledge |
| `../Cliente/**` | Future Niagara modules under the same Cliente umbrella |

## Reference Resources (NOT in this repo)

| Path | Purpose |
|------|---------|
| `/home/cristian/modules/BMS/SanLuis/` | Reference module: floor-grouped monitors pattern |
| `/home/cristian/modules/BMS/angeles/` | Reference module: multi-equipment hierarchy pattern |

## Compact Rules (auto-resolved for sub-agent injection)

### Repo layout (planned for bootstrap)
- Pure tooling repo: `scripts/` (bash), `docs/` (markdown, includes `knowledge-base/` subtree), `README.md`, `CLAUDE.md`, `.atl/`, `openspec/`-equivalent persistence handled via engram.
- No language toolchain: no `package.json`, no `go.mod`, no `pyproject.toml`, no Gradle. Do not introduce one unless a real need surfaces.
- Secrets: NEVER commit `.env*`, `*.key`, `*.pem`, or anything under `secrets/` — `.gitignore` already covers them.

### Shell scripts (`scripts/*.sh`)
- POSIX-leaning bash. Start every script with `#!/usr/bin/env bash` and `set -euo pipefail`.
- Make scripts executable (`chmod +x`) when committing.
- Quote every variable expansion (`"$var"`, `"${arr[@]}"`); never rely on word-splitting.
- Use `mktemp` for temp files/dirs; clean up via `trap '...' EXIT`.
- Logging: prefer `printf` over `echo` for portability; emit human-readable progress to stderr, machine output to stdout.
- Use `bat`/`rg`/`fd`/`sd`/`eza` only when documenting commands for humans — scripts should use POSIX `grep`/`sed`/`find` so they run on bare stations.

### `ng-deploy.sh` invariants (per global CLAUDE.md rule)
- Order: **backup → build → copy → verify**. Skipping backup is non-negotiable.
- Backup naming: `_backups/<module>-pre-$(date +%Y%m%d-%H%M%S).tar.gz`.
- Verify post-deploy: `unzip -p "$jar" META-INF/module.xml | rg -c "<type"` matches the expected count for that module; if `index.html` is part of the jar, the cache buster `?v=$BUILD_ID` matches the new `BUILD_ID`.
- Skip deploy only when the user says "no deploy" or the build is test-only.
- WSL → Windows path translation: hardcode `/mnt/c/...` paths in env vars or flags, never assume they exist — fail loudly with a clear message if the station path is missing.

### Documentation (`docs/`, `README.md`, `CLAUDE.md`)
- Apply `cognitive-doc-design`: outcome-oriented title → one-paragraph context → Quick path (numbered) → Details table → Checklist → Next step.
- Lead with the answer; progressive disclosure for edge cases.
- Knowledge-base entries: one topic per file, link bidirectionally from `docs/GOTCHAS.md` index.
- Never embed secrets or station IPs in docs — use placeholders like `<STATION_HOST>`.

### Git workflow
- Conventional commits only (`feat`, `fix`, `chore`, `docs`, `refactor`, `test`, `ci`, `build`, `revert`, `style`, `perf`).
- Never add `Co-Authored-By` or AI attribution (global CLAUDE.md rule).
- Branch naming (when a remote exists): `type/description` matching `^(feat|fix|chore|docs|style|refactor|perf|test|build|ci|revert)/[a-z0-9._-]+$`.
- Single trunk today (`main` tracks `origin/main` per `git status`). PR workflow only activates once a GitHub remote is wired.

### Testing (Strict TDD enabled — gap documented)
- Strict TDD marker is **enabled** session-wide, but the repo has NO test runner yet.
- Bootstrap MUST decide: either (a) introduce `bats-core` for `scripts/*.sh` and mark `strict_tdd: true` with a real runner, or (b) explicitly document `strict_tdd: false` with rationale (e.g. "scripts validated by smoke-deploy against chihuahua").
- Until that decision lands: every shell script in `scripts/` MUST be paired with a smoke-test checklist in `docs/` so the verification step is auditable.

### Persistencia SDD
- Modo activo: **engram**. Artefactos viven sólo en Engram bajo topic keys `sdd/{change-name}/{phase}`.
- NO se crea `openspec/` en este repo (chihuahua usa hybrid; niagara-tools usa engram).
- Project context: `sdd-init/niagara-tools`. Testing capabilities: `sdd/niagara-tools/testing-capabilities`. Skill registry: `skill-registry`.
