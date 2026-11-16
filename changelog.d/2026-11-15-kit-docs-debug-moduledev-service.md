### Added — debugging, moduledev, theme, service, removal and licensing guides

- `types/diagnostics.md` § 4: taking and reading a station thread dump (Application Director, the Provisioning
  station proxy, the console verb), attaching a remote Java debugger, and the launcher and console flags an author
  uses, each with what is safe on a production station.
- `types/wb-widgets.md`: moduledev mode (edit and reload without rebuilding the jar) and webdev mode (unminified
  files), Hx view authoring, custom-painted Workbench widgets and Workbench tools; `types/dashboard.md` points to
  the two modes.
- `types/theme.md` § 8-10: the official theme authoring flow, NSS, icon, font and sprite rules, and how a custom
  widget becomes themable.
- `types/service.md`, a new card for station services: registration by placement, lookup, the callback order on
  start, drop, disable, removal and station stop, licence behaviour and tests; `skill/SKILL.md` routes to it.
- `types/distribution.md` § 10a: the module removal checklist, with a row in `types/platform.md` § 2.
- `types/security.md` § 5.2-5.3: how a feature reaches a licence, the validation chain and how to test the
  unlicensed path.
- `corpus-index.md`: 22 rows for the blocks these sections cite; the title states the range the index reaches.
