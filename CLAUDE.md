# CLAUDE.md — niagara-tools agent guide

Agent-facing companion. Read this before taking any deploy or build action.

## 1. When to invoke `ng-deploy.sh`

| Change type | Mode | Station restart? |
|---|---|---|
| Java in `*-rt/src/com/...` (BComponents) | C (or A if also touching ux) | Yes |
| Java in `*-ux/src/com/...` (servlets, handlers) | B (or A if also touching rt) | Yes |
| JS / CSS / HTML in `*-ux/src/rc/...` | B | No — browser hard-reload only |
| Both rt and ux | A | Yes |
| Slot / Property / Type / Action add or modify | A + run `:slotomatic` separately first | Yes |
| Doc-only / SDD artifacts | n/a — do NOT run ng-deploy.sh | n/a |

Invocation pattern (from consumer module root):
```bash
cd /path/to/chihuahua
/path/to/niagara-tools/scripts/ng-deploy.sh --mode A
```

## 2. Invariants (non-negotiable)

- **Order is non-negotiable**: backup → build → copy → verify. Never reorder.
- **Never skip backup**: `--no-backup` requires `--i-know-what-im-doing`. No exceptions.
  Skipping backup removes the only rollback path on a live station.
- **Never edit AUTO GENERATED CODE regions manually** — only slotomatic may write those.
  One exception: slot REMOVAL requires manual deletion of the AUTO region block in the
  same commit as the annotation removal (see `docs/knowledge-base/slotomatic.md`).
- **Never deploy without verify**: the type count check (`unzip -p ... | grep -c "<type"`)
  is what catches a stale or partially-compiled jar before it reaches the station.

## 3. Onboarding a new module

1. Copy `.env.local.example` to the consumer module root:
   ```bash
   cp /path/to/niagara-tools/.env.local.example /path/to/<module>/.env.local
   ```
2. Edit `.env.local` — uncomment and adjust all 6 required vars +
   `EXPECTED_RT_TYPES` / `EXPECTED_UX_TYPES` for the module.
3. Confirm the script is executable:
   ```bash
   chmod +x /path/to/niagara-tools/scripts/ng-deploy.sh
   ```
4. Run smoke checklist mode A: `tests/smoke-checklist.md §Mode A`.

## 4. Gitignore exception note

`.env.local.example` is intentionally tracked despite the `.env.*` wildcard in
`.gitignore`. The exception rule `!.env.local.example` immediately follows the
wildcard. Do NOT remove this exception — without it, `git add .env.local.example`
silently fails and the config schema disappears from the repo.

## 5. Test runner setup

Install bats-core and shellcheck before running unit tests:

```bash
# Debian / Ubuntu (WSL default)
sudo apt-get install -y bats shellcheck

# macOS
brew install bats-core shellcheck
```

Run unit tests:
```bash
bats tests/ng-deploy.bats
```

Lint the script:
```bash
shellcheck scripts/ng-deploy.sh
```

If `apt-get` requires `sudo` and is unavailable in the agent context, install via:
```bash
npm install -g bats          # bats-core (npm package)
brew install shellcheck      # shellcheck via linuxbrew
```

Fallback (Option F): if bats-core is uninstallable, set `strict_tdd: false` in the
sdd-init engram artifact, document the rationale in this section, and strengthen
`tests/smoke-checklist.md` as the sole integration gate. This fallback is pre-approved
per design decision #1813.

## 6. Engram topic_key conventions

| Scope | Pattern | Examples |
|---|---|---|
| In-flight SDD phases | `sdd/{change-name}/{phase}` | `sdd/niagara-tools-bootstrap/spec` |
| Cross-project Niagara knowledge | `niagara/{topic}` | `niagara/hot-reload-rules` |
| niagara-tools repo conventions | `niagara-tools/{topic}` | `niagara-tools/gitignore-env-local-example-exception` |

Phase values: `explore`, `proposal`, `spec`, `design`, `tasks`, `apply-progress`,
`verify-report`, `archive-report`, `state`.

## 7. Cross-project search hints

Knowledge from the chihuahua project is indexed in engram project `honeywell-mx60-chihuahua`.
Search it before starting any Niagara N4 SDD:

```bash
# BQL alarm gotchas
mem_search(query: "BQL ackState sourceState", project: "honeywell-mx60-chihuahua")

# WSL build overrides
mem_search(query: "WSL gradle niagara_home overrides", project: "honeywell-mx60-chihuahua")

# Hot-reload rules
mem_search(query: "hot-reload Java restart station", project: "honeywell-mx60-chihuahua")

# Slotomatic patterns
mem_search(query: "slotomatic coordinated edit slot removal", project: "honeywell-mx60-chihuahua")

# BAlarmService detached snapshot CRITICAL bug
mem_search(query: "BAlarmService ackAlarm detached snapshot", project: "honeywell-mx60-chihuahua")
```

Key observation IDs (honeywell-mx60-chihuahua project):
- `#1795` — session summary with BQL + hot-reload discoveries
- `#1788` — CRITICAL: BAlarmService.ackAlarm(rec) from cursor is a silent no-op
- `#1779` — hot-reload rule (Java = station restart; JS/CSS = browser reload)
- `#1567` — WSL gradle build invocation with -P overrides
- `#1403` — slot removal coordinated-edit pattern + slotomatic-in-WSL myth busted
- `#1204` — full build workflow recipe for chihuahua

## 8. Knowledge-base index

See [docs/GOTCHAS.md](docs/GOTCHAS.md) for:
- Cross-project anti-patterns table (symptom / fix / reference)
- Links to all KB topic files
