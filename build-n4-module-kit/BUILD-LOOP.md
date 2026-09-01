# BUILD-LOOP — the operational cycle for one N4 module

The contract the launcher runs. Follow it in order; the gates are not optional.

## 0. Orient (before touching anything)
- `corpus-nav find "<topic>"` — read the matching blocks. 90% is already documented; don't re-derive.
- Read the EXEMPLAR source for the type (SOURCES.md), verbatim — not from memory.
- Pick the module type (SKILL.md decision table) → load `types/<type>.md`.

## 1. Design
- State the module's job, its profiles (rt/ux/wb), and the slot/endpoint contract in one paragraph.
- For a dashboard: the facade slots (display link-in + writable config), the servlet routes, the JSON `{v,st}` contract, the HMI resolution.

## 2. Build the layers
- Follow `types/<type>.md` + `METHODOLOGY.md`. Keep a facade pure; keep control logic in rt; keep UI in ux/wb.
- Apply the slot rules as you write each `@NiagaraProperty` (flags, facets/units, the annotation+generated+imports rule).

## 3. Preview (UI types) — BEFORE compiling
- `python3 /home/cristian/niagara-research/tools/dashboard-preview.py --rc <mod>/src/rc --prefix /<mount>`
- Open `http://localhost:<port>/hmi` (1280×800 frame). Iterate design here — refresh, no build.

## 4. Build — the ONLY valid build
- Primary: `niagara-tools/scripts/ng-deploy.sh --strict-slotomatic` (backup→build→slotomatic→deploy→verify; strict aborts if annotations changed without slotomatic). Fallback for quick WSL iteration: `toolbelt/build.sh`. See build-verify.md.
- Confirm: bytecode major **52**, jars **signed**, no raw-double facet. (build-verify.md)
- A `gradle :jar` with the default JDK is NOT a build.

## 5. Verify gate (before "done")
- Run `METHODOLOGY.md` (common) + the `types/<type>.md` checklist against the built module. Every item pass, or fix it.

## 6. Deploy (station) — operator
- Sign (auto), stop station, replace jars in `<niagara_home>/modules/`, start. Place components at their fixed ORDs; link points. Open the URL.

## 7. Retro (propose-never-apply)
- Append what this build PROVED (a new gotcha, a corrected fact, a type-guide gap you filled) to `retros/<date>-<module>.md` as PROPOSED kit deltas. Do NOT silently rewrite METHODOLOGY — propose; a human folds it in. This is how the kit matures from seed to solid.
