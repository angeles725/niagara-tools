[kit] ColdRoomPan BEvaporatorUnit: stranded inDefrost leaves resistanceOut stuck ON (no recovery path)

**Retro**: retros/2026-09-07-coldroompan-indefrost-resistance-recovery.md
**Filed**: 2026-09-07 (local ticket — GitHub create skipped: label `kit` not found on angeles725/niagara-tools, and the script's hardcoded `campaign-9` label does not apply to this work)

**What happened**: A `BEvaporatorUnit` can be left with `inDefrost==true` stranded (config changed mid-defrost, or a per-evap `BDefrostController` created/replaced while a room-level cycle had the unit in defrost, so `exitDefrost()` never runs for it). Once stranded the unit is frozen in the defrost output pattern — `resistanceOut` ON, `valveOut`/`evapOut` OFF — with NO recovery path: every output writer returns early under `if (inDefrost) return`, `stopped()` clears `inDefrost` but does not rewrite `resistanceOut`, and the restart path (`computeEvapCall → applyRunCmd`) never writes `resistanceOut`. Disable/enable does not clear it. Found live on Cuarto1/EvaporatorUnit_1 (resistanceOut=true, both controllers defrostActive=false); cleared only by `forceDefrost`.

**Evidence**: BEvaporatorUnit.java:1224 (applyHoaOutputs `if(inDefrost)return`), :1125/:1174 (same guard), :1053-1056 (stopped clears inDefrost, not resistanceOut), :1288 (computeEvapCall→applyRunCmd only). Live oBIX PANCCADIA JACE 2026-09-07. engram #8443.

**Proposed fix (code)**
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | On `started()`/`atSteadyState()` (unit enable), when `Sys.atSteadyState()` and the unit is not in an owned defrost cycle, force `resistanceOut=false` before applying the cooling call — unconditional reset so a stranded `inDefrost` cannot leave the heater energized. | ColdRoomPan `BEvaporatorUnit.java` `started()`/`computeEvapCall` | `[ev: engram #8443]` |
| Δ2 | Add kit lint `lint-recovery-path`: FAIL when a protection/heat output's only safe/off write lives inside methods gated by a transient mode flag, with no unconditional reset on `started()`/enable. | `toolbelt/` + `BUILD-LOOP.md` § step 5 | `[ev: BEvaporatorUnit.java:1224]` |

Labels (intended): `kit`, `from-run`
