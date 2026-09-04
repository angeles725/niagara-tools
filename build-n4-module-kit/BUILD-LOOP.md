# BUILD-LOOP — the operational cycle for one N4 module

The contract the launcher runs. Follow it in order; the gates are not optional.

## 0. Orient (before touching anything)
- `corpus-nav find "<topic>"` — read the matching blocks. 90% is already documented; don't re-derive.
- Read the EXEMPLAR source for the type (SOURCES.md), verbatim — not from memory.
- Pick the module type (SKILL.md decision table) → load `types/<type>.md`.

### 0.a Orient from BUILD-STATE (before touching anything)
- Read `BUILD-STATE.md` for the module you are about to build. From its `build-state.v1` envelope + prose, tell the operator in ONE line:
  `<module> · built <last_build>/gate <verify_gate>/deployed <deployed> · next: <last_session tail> · open_issues=N · retro_pending=Y/N`.
- If `retro_pending: true`, the previous session left an OWED retro — writing it is the FIRST task unless the operator redirects.
- If the module has no section, this is a first build — say so, and add its section at close.
- **Meta-work exemption:** auditing the kit / tooling / a retro, or a single one-off question, does NOT gate on orient — say so and proceed (mirrors the research-sdd PASO 0 exemption).

### 0.b Preflight (before the first build)
- **JDK 8** present (`ls /usr/lib/jvm`).
- **niagara_home chosen** = the LOWEST target version you must support, AND its pinned gradle plugin version (from `settings.gradle.kts`) is present in `<niagara_home>/etc/m2` — each install ships only one (build-verify.md).
- **Station live?** A running station LOCKS its `modules/<mod>.jar` — build against a mirror (`toolbelt/mirror-niagara-home.sh`), never stop a live supervisor.

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
- Three roles (build-verify.md §Doctrine): `toolbelt/build.sh` is the recommended WSL build (clean + slotomatic for every profile with sources + jar, then it calls the gate); `scripts/ng-deploy.sh --strict-slotomatic` is the station deploy wrapper (backup→build→copy→verify; strict aborts if annotations changed without slotomatic, and its slotomatic guard is rt-only); `toolbelt/verify-module.sh` is THE gate, run on the built jars.
- Confirm: bytecode major **52**, jars **signed**, no raw-double facet. (build-verify.md)
- A `gradle :jar` with the default JDK is NOT a build.

## 5. Verify gate (before "done")
- Run `METHODOLOGY.md` (common) + the `types/<type>.md` checklist against the built module. Every item pass, or fix it.
- The automated half of the gate is `toolbelt/verify-module.sh <jars…>` (bytecode 52, signature, type resolution; `--target-version` / `--stored` / `--src` opt-in). A jar that has not passed it does not go to a station.

## 6. Deploy (station) — operator
- Sign (auto), stop station, replace jars in `<niagara_home>/modules/`, start. Place components at their fixed ORDs; link points. Open the URL.

## 7. Retro + close (HARD close gate — not optional)
- **Update `BUILD-STATE.md`** for the module: refresh the `build-state.v1` envelope (`last_build`, `verify_gate`, `deployed`, `bytecode_major`, `signed`, `last_commit`, `last_session`, `open_issues`), set `retro_required` honestly, and set `retro_pending`.
- A session that changed KIT files is NOT "done" until ONE of:
  - (a) it wrote a retro at `retros/<date>-<module>.md` (line 1 `<!-- review-status: pending -->`, lessons as PROPOSED kit deltas — propose-never-apply), recorded it in the retro index, and set `retro_pending: false` in `BUILD-STATE.md`; OR
  - (b) it declared the change TRIVIAL: `Retro: none (trivial: <reason>)` in the commit trailer AND `retro_required: false` in the envelope.
- The **Output Contract MUST print** `retro: <path> (N deltas, review-status: pending)` — or `retro: none (trivial: <reason>)` — as an explicit line. A written-but-invisible retro reads as a missing one.
- Do NOT silently rewrite METHODOLOGY — propose; a human folds it in. This is how the kit matures from seed to solid.
- (The MACHINE enforcement of this gate lands in build-n4-retro-gate / PR2; this section is the wording + the `retro_pending` field it enforces.)
