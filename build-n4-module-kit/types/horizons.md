# Horizons — N4 capabilities our modules do not use yet

A backlog of framework capabilities with a known value for cold-room, compressor and dashboard modules. Each row
says where the kit already teaches it; a row with no recipe is a `toolbelt/kit-ticket.sh` candidate, not a rule.
Pick from here when a module needs the capability; never add one just because it exists.

## 1 · Course capabilities not yet used `[ev: corpus B1242 §1242.2]` [ev: retro n4-course-parity-observability-horizons Δ6]

| Capability | Kit recipe today | Value for our modules |
|---|---|---|
| Spy pages (`spy(SpyWriter)`, call `super.spy`) | `types/observability.md` § 2 | field techs read defrost state and staging decisions at `spy:/` |
| `-doc` help module (`toc.xml`, `<module>-<TypeName>.html`) | `types/structure.md` (toc.xml structure) | F1 help on setpoints and HOA modes in Spanish |
| `BSimpleJob` with progress and a lexicon log | `types/logic-authoring.md` § Background jobs | commissioning jobs visible in the Jobs sidebar with cancel |
| History export to CSV | `types/logic-authoring.md` (`ITableToCsv` view) | daily temperature export per cold room |
| BQL from code with the caller's Context | `types/logic-authoring.md` § BQL from code | one query builds a dashboard room snapshot instead of N ORD reads |
| BQL against a remote station | `types/logic-authoring.md` § Cross-station ORD pipe composition; `[INFER]` course BQL Reference only | Supervisor fleet view across JACEs |
| NDriver | `types/driver-authoring.md` § 1.6 | a native driver for a non-BACnet controller instead of oBIX workarounds |
| Tags, implied tags from facets, tag dictionary, NEQL | `types/logic.md` (tag dictionary); `types/logic-authoring.md` § Grouping and relating declaration surfaces | dashboards and cloud select "all cold-room temps" by tag, not by path |
| `kitControl:BqlExprComponent` (ExprLogic / ExprMath presets) | none | station-side derived values (percent load, deltas) without a module release |
| `control:TimeTrigger` + a hidden link driving an action | none (`types/logic.md` timer rules) | operator-editable periodic tasks instead of a hard-coded `Clock.schedule` period |
| Custom field editors and facets | `types/wb-widgets.md` § Field editors | safer setpoint and ORD entry in Workbench |
| Resource Manager engine stats, lexicon diagnostics | `types/diagnostics.md` § 1; `types/lexicon.md` | faster triage of engine stalls and missing Spanish labels |

`BqlExprComponent` ships in 4.15.6.20 `kitControl-rt.jar`; `BTimeTrigger` is `javax.baja.control.trigger` in `control-rt`
(4.15.6.20 source).

## 2 · Small web and Workbench horizons `[ev: corpus B1315 §1315.1 ranks 5-7]` [ev: retro devguide415-horizons-authoring-security Δ10]

- **SPA test helper:** `nmodule/js/rc/jasmine/promiseUtils` (`doPromise`, `waitForTrue`) is a Jasmine spec helper
  ("utility functions for running async Jasmine specs"), never runtime code: require it only from `srcTest` specs
  (devguide `jsdoc/js-ux/module-nmodule_js_rc_jasmine_promiseUtils.txt:150,174,236`). Test harness: `types/dashboard.md` DUX-TEST1.
- **Operator-facing SPA errors:** `nmodule/js/rc/errors/LocalizableError` takes lexicon parameters (`module`, `lex`,
  `messageArgs`) or a `%lexicon(module:key)%` string (`…module-nmodule_js_rc_errors_LocalizableError.txt:265,300`);
  lexicon side: `types/lexicon.md` § 4.
- **CSRF header name:** use `csrfUtil.CSRF_TOKEN_HEADER_KEY`, not a literal (`…module-nmodule_js_rc_csrf_csrfUtil.txt:247,285`);
  filter rules: `types/security.md` § 3.
- **Login branding from the theme module:** add `baja:String` slots `logo`, `loginCss`, `favicon` on `WebService` with
  `module://<module>/…` ORDs. Since 4.10U3 `file:^` login resources are not permitted; since 4.10U4 `file:^^public/…`
  is (devguide `loginScreen.txt:20-33`). Theme modules: `types/theme.md`.
- **Popup editor** replacing the Property Sheet on a wiresheet click: `types/wb-widgets.md` § Field editors.
