### Changed — kit doctrine from retro deltas (issues #865, #866, #867, #881, #896, #902, #909, #911, #914, #915, #919, #920, #921)

- #865: `ORCHESTRATION.md` §3.b item 7 and `types/dashboard.md` § Facade contract: a protection timer the HMI shows
  leaves with the protection that replaced it, so no unused slot stays on a cached page.
- #866: `types/logic.md` § Flag vs console alarm: each alarm flag the client wants in the alarm database needs a
  station-side alarm point; the module docs list the alarm-ready flags and give the mcp-n4 recipe.
- #867: `types/logic.md` § Safety fail-modes: check the start inrush before enabling the automatic over-current
  lockout; a 2 s delay is shorter than a typical compressor inrush.
- #881: `types/driver-authoring.md` § 11.2: Load from Library moves the FILT input to DI1. The Discover Networks filter,
  the object instance and the stock Modbus manager were already there.
- #896: `types/frontend-standard.md` § 7: boot order of a dashboard with a 3D scene (dashboard first, 3D in phases
  with a frame released before each, overlay on the 3D view only, phase timings in the console).
- #902: `types/logic-authoring.md` § BFacets key reference card: `maxOverrideDuration` is a facet; read `facets`, not a
  slot (BQL answers `NULL` for an unknown column).
- #909: `types/frontend-standard.md` § 5: a poll repaint restores the scroll position it cleared; the lint or webtest
  that flags a missing restore is still owed.
- #911: `types/logic-authoring.md` § Logging a point to history and `types/dashboard.md` § Charts on an HMI: capacity ×
  interval must cover the days a view reads, and the reply says when it does not.
- #914: `BUILD-LOOP.md` §5 kit tooling doctrine: a new FAIL lint ships with a sweep of the client modules it will fail
  (counts only in the public repository) and one ticket per module.
- #915: `ORCHESTRATION.md` §3.c: a writer brief for a registry-backed tool names the golden `tools/list` file and the
  200-line Go file limit.
- #919: `BUILD-LOOP.md` §0.c and `types/frontend-standard.md` § 5: the generated `dist/` bundle gets its own commit or
  stays out of the review candidate.
- #920: `types/logic-authoring.md` § Link lifecycle callbacks and `types/dashboard.md` § Write path: disabling a link
  does not release its target, so a module that switches a driver off that way also writes the target null.
- #921: new `templates/feature-doc/feature-doc.md` with `## Specs`, `## Behavior decisions` and `## Tasks` in that
  order, named in `BUILD-LOOP.md` §1.
