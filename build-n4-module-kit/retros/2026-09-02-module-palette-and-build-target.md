# Retro — empty module.palette passes the gate + per-module build target · 2026-09-02

Two cross-cutting gaps found while shipping CompPan and ColdRoomPan (session "Acompañante").
Both cost real time in the field. PROPOSED kit deltas (propose-never-apply).

## What this PROVED

1. **The verify gate does NOT check module.palette — a module can build, pass the gate, deploy,
   and open in Workbench with an EMPTY palette (nothing to drag onto the station).**
   CompPan shipped with the scaffold's default `module.palette` = just `<p m="b=baja" t="b:Folder">`
   with zero component entries. It built, passed the whole gate (bytecode 52, signed, types resolve,
   typecount, facets), was STORED and deployed — and the operator opened the palette in Workbench and
   saw nothing. The fix is one line per exposed component:
   `<p n="CompressorControl" m="CP=CompPan" t="CP:CompressorControl"/>` (name from module-include.xml).
   → **PROPOSED gate delta:** add a `palette` check to `verify-module.sh` — if `module.palette` exists
   and its folder has ZERO component entries while `module-include.xml` declares ≥1 type, WARN (or FAIL
   under `--strict`). Cheap: unzip module.palette, count `<p n=...>` children vs the type count.
   → **PROPOSED methodology delta:** the build checklist must include "populate module.palette with one
   entry per exposed `@NiagaraType` (name = module-include.xml `name`, alias for the module)". The
   palette is how components are placed in commissioning; an empty one reads as "the module is broken".
   NOTE: the module is NOT actually broken — the type resolves and can be instantiated other ways — but
   the drag-from-palette workflow that commissioning relies on is missing.

2. **In a multi-module client repo, each module can target a DIFFERENT niagara_home + plugin
   version. Do not assume one target for the whole repo.**
   - ColdRoomPan → `niagara_home=C:\PowerB\PowerB-4.15.3.28`, plugin 7.6.22.
   - CompPan → `niagara_home=C:\Honeywell\OptimizerSupervisor-N4.14.0.162`, plugin **7.6.17**
     (hardcoded in its own settings.gradle.kts).
   Building CompPan with ColdRoomPan's target (`--plugin-version 7.6.22` against PowerB 4.15.3) failed:
   `could not resolve plugin artifact com.tridium.niagara ... 7.6.17` — the plugin the module actually
   wants was not in the niagara_home passed. Each install ships exactly one niagara-module plugin; the
   module's `gradle.properties` (niagara_home) + `settings.gradle.kts` (plugin version) are the truth.
   → **PROPOSED build.sh delta:** when arg-3 niagara_home is omitted, read the module's own
   `gradle.properties` `niagara_home` and translate `C:\...` → `/mnt/c/...` as the default target,
   instead of failing or requiring the caller to know it. Log which target was chosen.
   → **PROPOSED methodology delta:** BUILD-LOOP §0.b preflight — "read THIS module's gradle.properties +
   settings.gradle.kts first; a sibling module in the same client repo may target a different Niagara
   version, so never carry over the last module's niagara_home/plugin."

## Cost / evidence

Both surfaced in production use, not in the gate: (1) the operator opened Workbench and reported "no
veo nada en mi module palette"; (2) the palette-fix rebuild failed on the wrong niagara_home before
succeeding against Honeywell 4.14. A `palette` gate check would have caught (1) at build time; reading
the module's own gradle.properties would have avoided (2).
