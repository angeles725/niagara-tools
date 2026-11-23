### Added — mcp-n4: facets as key/value, flat scope and down split, create hint (issues #858, #892, #893, #905, #941)

- `n4_read_slots` adds `facets_parsed` to a `baja:Facets` slot: its parts as key to value (`precision`, `units`,
  `maxOverrideDuration`), so override limits can be audited per point (#893).
- `n4_histories_audit` has two optional arguments for `flat=true`, both off by default (the old reply is unchanged):
  - `flat_scope: "sensors"` keeps interval histories and COV histories of a temperature, humidity, pressure, flow, ppm
    or lux point; settings and enums that hold one value are left out (#905).
  - `flat_down: true` lists a flat history whose point is down, stale or in fault in `flat_down`, not in `flat` (#892).
  - Either one reads the control points' `status.bits` and `facets` once; if that read fails, `flat` keeps every
    history and a note says so.
- `n4_create_component` names the palette type for an unknown `alarm:...Ext` type: `alarm:AlarmSourceExt`, with its
  algorithm as a child (#858). METHODOLOGY section 3 has the rule.
- `n4call` refuses to execute a write whose dry-run reply does not show `dry_run: true` (#941).
- The mcp-n4 skill and METHODOLOGY section 4 compare `server.build` with niagara-tools `main` after a kit merge (#907).

### Fixed — retro-issues sends no search term to GitHub without a denylist (issue #943)

- A dry run with no client denylist on the machine used to search GitHub for similar issues with terms that had not
  been scrubbed. It now skips that search, matches against the labeled issues it already listed, and prints
  `note: dedup search skipped: no denylist on this machine, so no search term is sent to GitHub`.
