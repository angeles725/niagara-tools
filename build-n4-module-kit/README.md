# build-n4-module-kit

The kit behind the `build-n4-module` skill (research-sdd pattern: the skill is a thin launcher in `~/.claude/skills/build-n4-module/`; this repo is the real content, resolved at runtime).

Building a Niagara N4 module — control logic (rt), a browser dashboard for an HMI (facade + servlet + SPA), a Workbench view (wb), or a mix — with an enforced checklist, the correct Java-8 + slotomatic build, and a verify gate.

## Layout
- `METHODOLOGY.md` — the rules + the common per-layer checklist (the verify gate).
- `BUILD-LOOP.md` — the operational cycle the launcher runs.
- `build-verify.md` — the Java-8 + slotomatic build command and how to verify.
- `types/` — per-type guides: `dashboard.md` (mature), `logic.md` (growing), `wb-widgets.md` (seed).
- `toolbelt/build.sh` — recommended WSL build (clean → slotomatic → jar, Java 8, then runs the gate). `toolbelt/verify-module.sh` — THE verify gate (major 52, NIAGARA4.SF, module.xml types resolve). `toolbelt/mirror-niagara-home.sh` — safe mirror for building against a live install. `toolbelt/stored-repack.sh` — STORED repackage for the Workbench re-sign path.
- `SOURCES.md` — where the knowledge lives (corpus-nav, docs, exemplars, tools).
- `retros/` — proposed kit deltas from real builds (propose-never-apply). This is how the kit grows.

## Status
v0.2, GROWING. The dashboard and build/verify paths are proven end-to-end (DashboardPan + ColdRoomPan, 2026-08/09) and gated by toolbelt/verify-module.sh. The wb path is still a stub to feed from real builds via retros/.

## Resolve
Default path: `/home/cristian/modulos_niagara_n4/niagara-tools/build-n4-module-kit`. Override with `$BUILD_N4_KIT`.
