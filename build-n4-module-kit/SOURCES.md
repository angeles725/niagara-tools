# Sources — where the knowledge lives (consult FIRST)

The 90% rule: before investigating anything, search the corpus. Most of module-building is already documented.

## Tools
- **corpus-nav** — our own blocks: `python3 /home/cristian/niagara-research/tools/corpus-nav.py find|grep|show|connections|by-marker "<topic>"`.
- **dashboard-preview** — local preview before compiling: `python3 /home/cristian/niagara-research/tools/dashboard-preview.py --rc <mod>/src/rc --prefix /<mount>` (open `/hmi`).
- **niagara-help** — official Tridium docs: `python3 /home/cristian/niagara-research/niagara-help/tools/niagara_help.py find|class ...`.
- **module-navigator** — code queries over `organized/`.

## Distilled guides (niagara-research/docs)
`docs/` = `/home/cristian/niagara-research/docs/` — narrative reference; the kit points, never copies.
- `docs/module-best-practices.md` — rt/ux/wb do & don't, §2 has the X-Requested-With rule + the CSRF-guard↔header pairing.
- `docs/module-dev-workflow.md` — toolchain, codegen round-trip, dev loop, testing.
- `docs/how-to-create-coldroom-module.md` — end-to-end module creation walk-through.

## Key corpus blocks (via corpus-nav show <N>)
- B636 module-anatomy (the reference skeleton + chihuahua deviations); B705–B715 best-practices + dev-workflow.
- B6/B8 alarm architecture (BAlarmSourceExt → AlarmClass → alarm DB); B166 chihuahua alarm read/ack.
- B179–B213 PX subsystem; B163–B177 chihuahua ux/servlet; B724 WEB-HMI10 panel.

## Exemplar source (READ, do not re-derive)
- Dashboard: `/home/cristian/modulos_niagara_n4/Cliente/Honeywell/MX60/chihuahua/chihuahua/` (chihuahua-rt/ux/wb).
- Control: `/home/cristian/modulos_niagara_n4/Cliente/Leon-Guanjuato/Paccadia/ColdRoomPan/` (WSL primary — holds ColdRoomPan-rt + gradlew at the Paccadia level; the same tree under the `/mnt/c` Windows mount is a fallback only).
- Dashboard (this project's clean build): `/home/cristian/modulos_niagara_n4/Cliente/Leon-Guanjuato/Dashboard/`.
- Original Tridium javadoc: `organized/docSource/docSource-doc/extracted/`.

## Memory
Engram topic `research/niagara/coldroom-ux/progress` holds this project's build log + every hard lesson.

## Research-tooling caveats
- **S1 / Tool-zero is not absence:** a `corpus-nav` / `mem_search` returning zero results is NOT proof that something is undocumented — run a control query or fall back to `rg` / `mem_context` / the source before concluding "not found." [ev: retro rt-authoring-campaign Δ4]
- **S2 / No `[CERT]` from a mangled decompile:** never mark a claim `[CERT]` when derived from an `ln`/`n`-corrupted decompiled body; prefer the vineflower/procyon tree, else mark `[INFER]` or decline. [ev: retro rt-authoring-campaign Δ3]
