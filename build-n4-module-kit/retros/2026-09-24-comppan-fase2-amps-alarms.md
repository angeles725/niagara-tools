<!-- review-status: pending -->
# 2026-09-24 · module · comppan-fase2-amps-alarms

**Session**: PANCCADIA León field support. Feature doc
`Cliente/panccadia-leon/odd/tasks/comppan-fase2-amps-alarms.md`, base `eeab639`, HEAD
`34d734f`. Three user requests: (1) surface the existing Fase 2 suction-pressure control
on the dashboard as editable setpoint/diff-up/diff-down (CompPan-rt 2.3.0 unchanged,
DashboardPan facade+UI new), (2) commanded-ON-but-no-amps shown as a fault, (3) a new
over-amperage alarm (alarm only, never sheds). CompPan-rt 2.3.0 -> 2.4.0 (over-amps
alarm, prove-run defaults tightened 5 A/5 s -> 2 A/10 s, run-hours count only while
amps prove running). DashboardPan 2.5.0 -> 2.6.2 (Condensadoras Fase 2 config tab,
active-phase display, global alarm banner on every view, fail-closed "Fase --"/"SIN
DATOS" states). Three native RDD reviews approved (`review-9d18090bab819ee6`,
`review-67cb15e06826ff31`, `review-b2d8bff1a9a4bca7`), schema-risk SAFE x4, jars
packaged into `PANCCADIA-modulos-2026-09-24-v3/`; nothing deployed to the JACE.

**Delta count**: 8 (2 candidates from the session brief verified already covered by
current kit behavior and dropped — see the end of this section)

## What happened
Building and reviewing this feature exercised `verify-module.sh --src`, `build.sh`'s
deployed-baseline drift gate, the module-root resolution in `build.sh`, and an ad-hoc
HMI no-scroll/banner-visibility sweep across two writer sessions (T2 then T2c) with no
shared kit script, and a shared ODD feature doc edited by more than one delegated writer
in the same checkout. Verifying each observed friction point against the CURRENT kit
source (this session, 2026-09-24) confirmed six real, still-open gaps and two doctrine
gaps, and separately confirmed two candidates from the original session brief are
already handled by existing kit code — dropped rather than proposed twice.

## Evidence
- Feature doc (full task-by-task evidence, all commands run for real):
  `[ev: Cliente/panccadia-leon/odd/tasks/comppan-fase2-amps-alarms.md]`
- Commits: T1 `53de4fb`/`a4b8209`, T2 `ba39417`, T1b `904469d`, T2b `3c94e01`, T2c/fix
  `8ac7c97`, T3 doc `34d734f`.
- Native RDD reviews: `review-9d18090bab819ee6`, `review-67cb15e06826ff31`,
  `review-b2d8bff1a9a4bca7` (all approved).
- Kit-side verification of each candidate delta against the live kit source (this
  session, 2026-09-24): `toolbelt/verify-module.sh` (`check_facet_presence`),
  `toolbelt/lint-write-path.sh` + `toolbelt/report-module.sh` §11, `toolbelt/build.sh`
  (module-root resolution B7, deployed-baseline drift gate), `types/dashboard.md`,
  `types/logic.md`, `types/logic-authoring.md`, `METHODOLOGY.md` (Multi-session
  coordination, Kit maintenance sections), `ORCHESTRATION.md` — cited per delta below.
- Direct read of the live module source confirming Δ1 and Δ7:
  `Compresores/CompPan/CompPan-rt/src/com/angeles/CompPan/BCompressorControl.java:130-140`
  (facets) and `:120-146` (multi-line source comment above `@NiagaraProperty`).

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | `verify-module.sh --src`'s `facets-req` check only recognizes the presence of the literal parameter key `facets =` and the literal substring `UNITS` on the annotation/newProperty line; it does not recognize `BFacets.makeNumeric(unit, precision, min, max)`, which conveys a real engineering unit without the token `UNITS`. Confirmed live: `overAmpsLimit`'s annotation carries `facets = @Facet("BFacets.makeNumeric(BUnit.getUnit(\"ampere\"), ...)")`  — a real, working unit+precision+min facet — yet the setpoint-like-name branch (`nm ~ /Limit\|.../ && !(s ~ /UNITS/)`) still WARNs "missing UNITS" on every `makeNumeric` slot, alongside `suctionSetpoint`/`dischargeHighLimit` etc. that already had the same pre-existing pattern. Fix: extend the awk detector's UNITS/PRECISION test to also match `makeNumeric(` (which always supplies a unit as its first argument) so a slot using that helper is never misclassified as missing a unit. | `toolbelt/verify-module.sh` § `check_facet_presence` (the `has_fc`/UNITS regex around line ~297-307) | `[ev: Cliente/panccadia-leon/odd/tasks/comppan-fase2-amps-alarms.md T1 evidence; BCompressorControl.java:130-139]` |
| Δ2 | `build.sh`'s module-root resolution (B7) only walks UP from the given root to find `./gradlew` (handles "module dir passed one level too low"); it has no handling or hint for the opposite mistake — passing the REPO root, one or more levels ABOVE the actual gradle root (this client's layout nests `Compresores/`, `Dashboard/`, `Paccadia/` as siblings, each its own gradle root, under one repo). A writer who first passes the repo root gets a correct-but-generic failure ("no executable ./gradlew in $ROOT or any ancestor") with no hint that the fix is to descend into a child directory, not an ancestor. Fix: when the ancestor walk fails, do one shallow (depth <= 3) search under the ORIGINAL root for any `gradlew` and list the candidates found in the error message. | `toolbelt/build.sh` § module-root resolution (B7, the `GRADLE_ROOT` walk-up block, lines ~57-63) | `[ev: Cliente/panccadia-leon/odd/tasks/comppan-fase2-amps-alarms.md T1 evidence "module-root argument ambiguity"]` |
| Δ3 | **No kit tool for the HMI no-scroll + global-alarm-visibility sweep** that every Honeywell 10" kiosk dashboard feature now requires (first needed by the 2026-09-23 defrost/HMI-reload retro's Δ1 kiosk-recovery doctrine; used for real by T2 and T2c of this feature). Each writer re-implemented an ad-hoc puppeteer script from scratch: T2's view enumeration found 33 navigable views, T2c's independently re-derived enumeration (by walking the live DOM instead of reusing T2's script, which was not preserved across sessions) found 26 — a real drift in WHAT was swept, not in the app's navigation, purely because the sweep script itself was throwaway. Propose a toolbelt script (e.g. `toolbelt/hmi-sweep.js` invoked by a thin `toolbelt/hmi-sweep.sh` wrapper) taking a preview URL, viewport (default 1280x800), one or more scenario query strings, and an app-agnostic view-enumeration convention (walk `.nav-item`/registered route list + declared sub-tab selectors) so re-runs are comparable; emit a per-view PASS/FAIL table for (a) document-level scroll (`scrollHeight<=clientHeight`, `scrollWidth<=clientWidth`), (b) named inner-scroller exceptions, and (c) a target element visible+un-occluded via `document.elementFromPoint` at its center. | `toolbelt/hmi-sweep.sh` / `toolbelt/hmi-sweep.js` (new) + `types/dashboard.md` § kiosk-recovery checklist (cross-reference) | `[ev: Cliente/panccadia-leon/odd/tasks/comppan-fase2-amps-alarms.md T2 "33/33 views", T2c "26 views ... enumeration drift because the script was not preserved"]` |
| Δ4 | **New doctrine — a dashboard preview mock's `jsonUrl` must forward the page's own query string, or a scenario query parameter is silently ignored.** `index.html`'s `N4.jsonUrl` was a fixed `/dashboardpan/api/equipment` string that never carried through the page's `?worst=1`/`?degraded=1` scenario selector, so a plain `page.goto(url + '?worst=1')` (the obvious way to drive a scenario in an automated sweep) silently exercised the DEFAULT mock instead — found and fixed ad hoc in T2c by intercepting only the `/api/equipment` request and appending the scenario query server-side-visibly. This is a recipe every scenario-driven preview mock in this kit's dashboard pattern will hit again. Add a named checklist bullet: a preview server (or its SPA) must either read the scenario selector from its OWN `window.location.search` when building `jsonUrl`, or the sweep/test harness must intercept the API request and append the query explicitly — never assume `?scenario=X` on the page URL reaches the mock automatically. | `types/dashboard.md` § kiosk-recovery / preview-mock checklist (new bullet, adjacent to the existing kiosk-recovery bullet) | `[ev: Cliente/panccadia-leon/odd/tasks/comppan-fase2-amps-alarms.md T2b "preview-server.py: ?worst=1/?degraded=1 now parsed with urllib.parse.parse_qs", T2c "index.html's N4.jsonUrl is a fixed ... string that never forwards the page query string"]` |
| Δ5 | **New doctrine — a dashboard UI reading a status-carrying JSON point must gate on BOTH status and type before trusting the value; an unlinked/absent point must render an explicit unknown state, never fall open to a default meaning.** Confirmed real defect (fixed in T2b/T2c, `3c94e01`/`8ac7c97`): the initial `eqFlag()`-equivalent helper defaulted `st = p.st \|\| "ok"`, so a MISSING `st` field (not merely a degraded one) silently read as "ok" and a facade `BStatusBoolean` default `false` for the phase flag meant an unlinked/uncommissioned point silently displayed as "Fase 2 active" and an unlinked alarm point displayed as "no alarm" — both fail-OPEN. The corrected contract: a boolean point is trusted only when `p.st === "ok" && typeof p.v === "boolean"`; otherwise render an explicit unknown/degraded state (amber "SIN DATOS" chip, "Fase --"), additive to (never replacing) a real alarm from a sibling flag. A facade's own default value must equal the SAFE/fallback state (here: `pressureFallback` default flipped `false` -> `true`, Fase 1, the conservative fallback), not an arbitrary type default. This generalizes the RBAC/kiosk-recovery fail-closed doctrine already in `types/dashboard.md` from write-path/session-recovery to READ-path status display. | `types/dashboard.md` § (new bullet, adjacent to the existing RBAC fail-closed and kiosk-recovery bullets) | `[ev: Cliente/panccadia-leon/odd/tasks/comppan-fase2-amps-alarms.md T2b "pressureFallback default flipped false -> true", T2c "eqFlag() previously did const st = p.st \|\| \"ok\" ... tightened to ... p.st === \"ok\" AND typeof p.v === \"boolean\""]` |
| Δ6 | **New doctrine — changing what makes a persisted wear counter accrue (e.g. run hours) is a rotation-liveness change, not just a metering change, and needs an explicit rotation-consequence check.** T1b changed `CompressorControl`'s hour-integration guard from `if (cmd[k])` (commanded) to `if (running[k])` (amps-proven running) — correct per the user's requirement, but it means a compressor that never successfully starts now accrues ZERO run hours and is therefore PINNED first in a least-hours rotation indefinitely, silently, unless an operator manually intervenes or an opt-in auto-off (`autoOffOnProofFault`) is enabled. This consequence was caught only by the writer's own commissioning-note discipline, not by any kit check. Add a checklist/advisory item: any change to the predicate that drives a wear-based rotation/staging metric must be paired with either a rotation-consequence test (a no-start unit stays first/last as designed) or an explicit escape hatch, and the deploy checklist must name it. | `types/logic.md` § "Safety fail-modes & timers" (new bullet, adjacent to the existing HOA-OFF-dominance and cycle-anchor bullets) | `[ev: Cliente/panccadia-leon/odd/tasks/comppan-fase2-amps-alarms.md T1b "Mutation-proof", "Commissioning note: a no-start compressor accrues no hours and stays first in rotation"]` |
| Δ7 | **slotomatic's generated Javadoc for a `@NiagaraProperty`/`newProperty` field takes only the LAST line of a preceding multi-line `//` source comment block, discarding every earlier line — undocumented anywhere in the kit.** Confirmed live and reproduced by direct read: the 3-line source comment above `overAmpsLimit` (`// T7 (over-amperage alarm...) ... // compressors. One shared limit for all 3 // compressors. 0 (default) = disabled, same convention as suctionLowLimit/dischargeHighLimit.`) produced a generated Javadoc body of ONLY the final fragment ("compressors. 0 (default) = disabled, same convention as suctionLowLimit/dischargeHighLimit."), a mid-sentence, near-meaningless doc line — found twice in this feature (T2b fixed it for 4 slots, T2c found one more the same session, `pressureFallback`). This is a structural slotomatic behavior, not a one-off authoring mistake, and is not mentioned in `types/logic-authoring.md`'s `@NiagaraProperty` guidance or anywhere else searched in the kit. Authoring rule: write the LAST line of any multi-line source comment directly above a `@NiagaraProperty`/`newProperty` declaration as a complete, standalone sentence, since that is the only line slotomatic will keep. | `types/logic-authoring.md` § `@NiagaraProperty` authoring guidance (new bullet) | `[ev: Cliente/panccadia-leon/odd/tasks/comppan-fase2-amps-alarms.md T2b item 1, T2c item 3; BCompressorControl.java:128-138 direct read this session]` |
| Δ8 | **Extend the existing kit-file-only fragment-merge/APPEND convention to a shared ODD feature doc edited by concurrent delegated writers in the same checkout.** `METHODOLOGY.md`'s "Fragment-merge for the four always-conflict kit files" rule (resolve by APPEND, keep both, never overwrite) already solves this exact class of problem for `BUILD-LOOP.md`/`SKILL.md`/`retros/INDEX.md`/`BUILD-STATE.md` during a multi-worker KIT campaign, and "Multi-session coordination" covers refusing to touch another session's DIRTY tree — but neither covers the case this feature hit: two delegated writers in the SAME checkout both append their own T-section to one shared `odd/tasks/*.md` feature doc (exactly the pattern ORCHESTRATION's own delegation model produces), producing a doc hunk conflict with `git add -p` unavailable to resolve it surgically. Propose cross-referencing the existing APPEND-merge convention from ORCHESTRATION.md's delegation-trigger section for feature docs specifically, or naming one doc-owner per checkout who applies all writers' T-section text serially. | `ORCHESTRATION.md` § 3 Delegation triggers (new cross-reference to `METHODOLOGY.md`'s fragment-merge rule) | `[ev: Cliente/panccadia-leon/odd/tasks/comppan-fase2-amps-alarms.md session note "doc hunk conflict ... git add -p unavailable"]` |

**Two candidates from the original session brief verified already covered — dropped, not
proposed as new deltas:**
- *`lint-write-path.sh` exits 3 (ERROR) when no `docs/write-path-matrix.md` exists* — real
  behavior of the standalone script, but already handled cleanly by the recommended path:
  `report-module.sh` §11 (auto-chained by `build.sh`) catches exactly that exit-3/no-matrix
  case and emits a clean `SKIP lint-write-path "no write-path-matrix.md"` row instead of
  propagating the noisy ERROR. This feature's own T3 evidence confirms it: "lint-write-path
  stays SKIP on both modules." The noisy standalone exit-3 is intentional, documented K22
  behavior for direct/scripted invocation ("a root with no sources or no matrix is an ERROR
  exit 3, never a silent 0") — not a gap. `[ev: toolbelt/report-module.sh §11 lines
  ~1201-1207; METHODOLOGY.md K22; feature doc T3 "lint-write-path stays SKIP on both
  modules"]`
- *`build.sh`'s deployed-baseline drift gate forces a version bump against a WSL
  build-only/overlay `niagara_home`* — the gate already ships the documented escape hatch
  for exactly this case: `--no-drift-check` ("skip the deployed-baseline drift gate ...
  unless this niagara_home is not the deploy target"). The three bumps this session
  (2.6.0->2.6.1->2.6.2) were the gate working as designed against a real, if local, build
  target, and match the system-wide "bump on every shipped-bytes change" doctrine the
  2026-09-23 retro's Δ2 already mandated — not a bug. No delta proposed. `[ev: toolbelt/
  build.sh lines 13, 248; retro panccadia-defrost-sequencing-hmi-reload-deltas Δ2]`

## Lessons
- A field workaround for a lint's false positive (hand-editing a facet call, re-deriving
  a sweep script from scratch each session) is a signal the lint or tool itself has a
  gap — fix the tool, not just the call site (Δ1, Δ3).
- A generated status/JSON UI point must be trusted only on an explicit `st==="ok" &&
  typeof v==="boolean"` check; a facade's own default must equal the safe fallback state,
  not an arbitrary type default — extends existing fail-closed doctrine from write-path
  RBAC to read-path display (Δ5).
- Changing the predicate behind a wear-based rotation metric is a liveness change, not
  just a metering change — pair it with a rotation-consequence check or a named escape
  hatch (Δ6).
- Before proposing a kit delta, verify it against CURRENT kit source, not just the
  session's own friction log: two of this session's ten candidate frictions were already
  solved by existing kit code (`report-module.sh`'s SKIP translation, `--no-drift-check`)
  and only looked like gaps because the standalone/harder path was used instead of the
  documented recommended one.
- An undocumented generated-artifact quirk (slotomatic keeping only the last comment
  line) that recurs across sessions belongs in the authoring doc even when it never
  causes a build failure — it silently degrades documentation quality every time (Δ7).

---
**Status**: PENDING — INDEX row appended: `| 2026-09-24-comppan-fase2-amps-alarms.md | module | 2026-09-24 | pending | 8 |`
