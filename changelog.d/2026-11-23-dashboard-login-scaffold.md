### Added — a dashboard module ships its own login (issue #847)

- `scaffold-module.sh --type dashboard` emits, in the `-rt` part, a `BLoginTemplate` subclass skeleton
  (`B<Mod>LoginTemplate`, `static INSTANCE` declared before `TYPE`), its asset prefix (`src/rc/<mod>-login/`, served
  through `resourceToOrd`, no remote URL), the pure classes behind it (`login/model/`: `LoginActivation`, `LoginView`,
  `LoginRedirect`, `LoginAssets`, `LoginPage`, `Html`) and a service `B<Mod>Service` whose `stationStarted` applies the
  activation rule behind a boolean `applyLogin` slot (default true): `WebService.loginTemplate` is set only when it is
  empty, the stock template or a retired type of the module, never over another custom template, and the station is
  saved after the change. Pure tests cover the rule, the screens, the redirect, the assets and the page, and
  `TypeInitOrderTest` fails when a class declares `INSTANCE` after `TYPE`. The page is a neutral card (no brand); the
  module map, `module-include.xml`, palette, lexicon and the `-rt` gradle file (`web-rt`, the asset copy) get their
  lines. The overlay is `templates/dashboard-login` (`go/internal/scaffoldmodule/login.go`).
- `types/dashboard.md` § The module ships its own login: the doctrine (the template in `-rt`, not a separate module),
  the origin URI, the assets, the fallback to Niagara's own template, SCRAM and the forced password change, the
  activation rule, the rollback (flag off and clear `loginTemplate`), and that removing the module while the slot is
  still set gives HTTP 500 on `/login`. A pointer is in `BUILD-LOOP.md` § 1.

### Fixed — the dashboard scaffold names its servlet after the module

- The emitted `-ux` servlet (`getServletName()`, `ModuleDispatch.REDIRECT_HOME`) and `config.js` `apiBase` kept the
  fixture's lower-case `minimaldash` for every module; they now carry the module's lower-cased name (the mount the
  login redirects to). A module scaffolded as `MinimalDash` is unchanged.

### Changed — tests

- `tests/scaffold-module-login.bats` (TC-LG1 to TC-LG11) pins the overlay; `TC-DASH1` allows the overlay's files and
  inserted lines only; `TC-REP1` lets a dashboard carry the `lint-private-api-import` rows the login's `com.tridium`
  imports cannot avoid, and nothing else.
