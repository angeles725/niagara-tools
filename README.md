# niagara-tools — cross-project tooling and gotchas for Niagara N4 modules

Reusable deploy wrapper and knowledge base for Niagara N4 module development
in a WSL + Windows station environment. Not a Niagara module — pure bash and docs.

## What this is

A tooling repo that lives next to your consumer modules (e.g. `chihuahua`).
It provides `scripts/ng-deploy.sh`, a single bash wrapper that automates the
backup → build → copy → verify cycle for any Niagara N4 module jar, and a
knowledge base of cross-project gotchas discovered across multiple N4 projects.

## Quick path

1. Clone this repo alongside your consumer module:
   ```bash
   git clone <repo-url> /home/you/modulos_niagara_n4/niagara-tools
   ```

2. Copy the config template into your consumer module root and fill in values:
   ```bash
   cp /path/to/niagara-tools/.env.local.example /path/to/chihuahua/.env.local
   # edit .env.local — uncomment and adjust all variables
   ```

3. Run a full deploy from your consumer module root:
   ```bash
   cd /path/to/chihuahua
   /path/to/niagara-tools/scripts/ng-deploy.sh --mode A
   ```

## Repo layout

```
niagara-tools/
├── scripts/ng-deploy.sh        # Deploy wrapper (backup → build → copy → verify)
├── .env.local.example          # Config schema with chihuahua reference values
├── tests/
│   ├── ng-deploy.bats          # bats-core unit tests for ng-deploy.sh
│   └── smoke-checklist.md      # Manual integration checklist (modes A, B, C)
└── docs/
    ├── GOTCHAS.md              # Cross-project anti-patterns index + KB topic links
    └── knowledge-base/
        ├── bql-gotchas.md      # BQL N4.14 confirmed bugs + persistent-ack pattern
        ├── wsl-build-gotchas.md # WSL build overrides, gradlew path, slotomatic myth
        ├── hot-reload-rules.md # Java needs station restart; JS/CSS = browser reload
        └── slotomatic.md       # When to run slotomatic, slot removal pattern, AUTO rules
```

## Smoke checklist

Before and after each real station deploy, run through:
[tests/smoke-checklist.md](tests/smoke-checklist.md)

## Next steps

- **Agents**: read [CLAUDE.md](CLAUDE.md) — decision rules, engram conventions,
  cross-project search hints, and invariants for automated sessions.
- **Gotchas and knowledge base**: read [docs/GOTCHAS.md](docs/GOTCHAS.md) —
  cross-project anti-patterns table and KB topic index.
