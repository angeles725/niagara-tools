# Changelog

All notable changes to **niagara-tools** (cross-project Niagara N4 tooling) are documented in this file.

Format based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Versioning follows [SemVer](https://semver.org/spec/v2.0.0.html) (currently `v0.x` — see CONTRIBUTING.md §Versioning policy for stability promises). Each entry references the SDD change slug and engram observation IDs for traceability.

---

## [Unreleased]

## [v0.34.0] - 2026-11-23

### Fixed — the build-module drift gate hashes entry names and sizes, and refuses an unreadable entry

- The deployed-baseline content hash now covers each non-META-INF entry's name and size as well as its bytes,
  so bytes moved between entries or a renamed entry at an unchanged vendorVersion are drift (exit 51), no longer
  a silent pass. An entry that fails its CRC or cannot be decompressed, or two entries under one name, make the gate
  "cannot verify" (exit 52);
  before, the bytes `unzip -p` did print were hashed and two equally damaged jars passed. The jar is read with
  Go's archive/zip, so the gate no longer runs `unzip`; the "cannot verify" reason now reads `cannot read the jar
  content (not a readable zip, no entry, a duplicate entry name, or an entry that fails its CRC)`. The hash is
  never stored: both sides are computed in the same run, so the new format needs no migration. Retro
  `2026-11-14-go-build-module-port` Δ5 (and the zip-listing part of Δ3).

### Fixed — build-module detects the scaffold's plugin-version default

- `build-module` now reads the `niagaraPluginVersion` default from the form the scaffold emits,
  `providers.gradleProperty("niagaraPluginVersion").getOrElse("X")` (and `gradleProperty(...)`): the old pattern
  stopped at the `)` after the property name, so no version was detected and gradle fell back to its own default.
  The detected value is the `getOrElse` argument, no longer the first number on the line (a name such as
  `niagaraPluginVersion10x` gave `10`). A one-digit default such as `getOrElse("7")` is not a
  version: the line does not match, so a settings file holding a NUL byte no longer prints a binary-file row for it. Retro `2026-11-14-go-build-module-port` Δ1.

### Changed — build-module names the module root it cannot enter

- A module root that exists but cannot be entered now ends with `build.sh: cannot enter module root: <path as typed>`
  (exit unchanged) instead of `build.sh: cannot resolve module root: ` with no path. Retro
  `2026-11-14-go-build-module-port` Δ2.

### Added — run-niagara-test proves the NRE wrote into the scratch home

- Every `run` now proves that the NRE wrote into the scratch `niagara_user_home` during this boot: some regular file under `nuh\` must be new or changed since just before test.exe started, with its mtime inside the boot window (±2 s skew), or the run FAILs. The PASS row names the files. This closes the `--reuse-home` gap where a kept home's `etc\` and `security\` already existed from the earlier boot. Checked live on fresh, keep-home and reuse-home boots.

### Removed — the Python mcp-n4 server

- `mcp-n4-kit/mcp_n4/` (the Python server, `python3 -m mcp_n4.server`) is removed; the Go binary
  `go/bin/mcp-n4` (`cd go && go build -o bin/ ./cmd/mcp-n4`), the default since v0.32.0, is the only
  mcp-n4 server. **Migration:** register `go/bin/mcp-n4` with the same flags in `.mcp.json`;
  `MCP_N4_IMPL=python` and `tools/live_smoke.py --impl python` no longer exist.
- `tools/new_retro.py` keeps its flags and drafts through the new `go/bin/retro-draft`
  (`cd go && go build -o bin/ ./cmd/mcp-n4/retro-draft`, or `MCP_N4_RETRO_DRAFT_BIN`).
- The Python-only tests and the Go-port harness that compared the two servers are gone; the Go
  differentials replay frozen goldens and no Go test needs Python
  (mcp-n4-kit/docs/go-port-plan.md section 23).

### Added — a changelog gate in pre-push and CI

- `.githooks/pre-push` and the new `changelog` CI job run `scripts/changelog-gate.sh`: a range that changes `go/`,
  `build-n4-module-kit/`, `tools/` or `scripts/` (except `go/bin/`, `ledger.d/`, `retros/` and `BUILD-STATE(.src).md`) fails
  unless it adds a `changelog.d/<YYYY-MM-DD>-<slug>.md` fragment, adds a line inside `## [Unreleased]`, or carries a
  `Changelog: none (<reason>)` commit trailer. A malformed fragment or trailer fails any range. See CONTRIBUTING.md § 6.2.

### Added — gen-changelog folds changelog.d/ at release time

- `scripts/gen-changelog.sh --write` appends every `changelog.d/<YYYY-MM-DD>-<slug>.md` fragment, in date+slug
  order, to the end of `CHANGELOG.md`'s `## [Unreleased]` section and deletes the folded fragments; it writes
  nothing when a fragment is malformed. The release runs it first (CONTRIBUTING.md § 5 step 2).
- CI runs `scripts/gen-changelog.sh --check`, so a fragment the fold would refuse fails before it merges.

### Changed — run-niagara-test names the plugin module-include trap in the gradle-root FAIL row

- `run-niagara-test run`: when the build changed a `*/module-include.xml` in the gradle root, the
  `gradle root changed by the build` FAIL row now names the known cause (niagara gradle plugin 7.3.40 and 7.6.22,
  observed, append the test types to the production `module-include.xml` in the moduleTest compile when the test
  sources changed), points at `--restore-gradle-root` and `types/moduleTest.md` (plugin-version trap), and says
  whether the restore checked it out. Other files keep the generic row; the run stays FAIL with the same exit code.

### Changed — absolute-claim sweep of types/

- `types/logic-authoring.md`: a proxy-ext `write(Context)` that posts the write to a worker returns `true` (a write
  is pending; finish with `writeOk`/`writeFail`), per the devguide BProxyExt contract; the BQL `get()` comment names
  why the caller's Context matters (read permissions).
- `types/dashboard.md`: slot lexicon keys stay bare, and the one prefixed key is a type's `<Type>.displayName`.

### Added — count-retro-deltas

- New `toolbelt/count-retro-deltas.sh <retro.md>...` (G39): counts each retro's delta-table rows and compares them
  with its `**Delta count**` line, its `<!-- deltas: N -->` header and its generated `retros/INDEX.md` row, printing
  one OK/MISMATCH line per retro and a TOTAL line to quote; exit 1 on any mismatch, 3 when no `INDEX.md` is beside a
  retro (generate it with `toolbelt/kit-state.sh --write`).

### Added — dashboard scaffold: a route handler class and the ux-rt API contract

- A fresh dashboard's servlet delegates `GET /api/status` to `ux/route/StatusHandler` (router + one handler class
  per route group, DUX1).
- `<M>-ux/src/rc/js/contract.json` names each route's response fields; `StatusHandlerTest` (pure JUnit) checks
  the servlet side and `srcTest/rc/spec/<M>ContractSpec.js` (node, offline) checks that every field
  `services/api.js` reads is declared (types/dashboard.md DUX-CONTRACT1). The status component now prints the
  `status` field instead of the whole JSON body.

### Changed — the dashboard scaffold emits a split rc/js skeleton

- `scaffold-module.sh --type dashboard` emits `rc/js/config.js`, `js/services/api.js` (the only `fetch`, with the
  XHR header and an abort timeout), `js/store.js` (poll loop, state, subscribe, reload after `reloadAfterMs`
  without a good read), `js/components/status.js`, `js/pages/home.js` and `js/main.js`; `index.html` loads them in
  order and holds no inline script. The module map lists them in `frontend-data` and `frontend` rows. A fresh
  dashboard no longer WARNs `rc-scan` `fetch-no-signal` or `lint-spa-poll-no-recovery`.

### Changed — CI installs the kit's npm tools from a committed lockfile

- `build-n4-module-kit/toolbelt/eslint/package-lock.json` is now committed, and the CI bats job installs eslint and
  acorn with `npm ci --ignore-scripts` (exactly the locked tree, no dependency lifecycle scripts) instead of
  `npm install`; npm's download cache is keyed on the lockfile. Local installs use the same command.

### Added — ledger-check and a scaffolded client version ledger

- New `toolbelt/ledger-check.sh <VERSION-LEDGER.md>` (Go `go/cmd/ledger-check`) checks a client's version ledger
  against `templates/VERSION-LEDGER.md`, reading the contract from the template: title, sections, the Versions
  header, the Status vocabulary, real dates, classes declared in the ledger's own Root-cause classes table, and a
  defect with its root cause and class (FAIL); a template placeholder left in a row is a WARN.
- `scaffold-module.sh` emits `<out-dir>/<ModuleName>/odd/VERSION-LEDGER.md` for every type (the template with its
  sample row dropped); it passes `ledger-check.sh` clean.

### Added — lint-absolute-claims

- New `toolbelt/lint-absolute-claims.sh <types-dir>...` (G38): an INFO report of every line holding "never",
  "always", "silently", "forbidden" or "throws" whose bullet or paragraph cites only corpus or retro inference (no
  code, course, devguide, CERT or javap tag and no `.java:` line), for the K33 check against Tridium sources. It
  never fails (exit 0, `--strict` included); a root with no `*.md` exits 3. Not run by `report-module`.

### Added — lint-adapter-cfg-default

- New `toolbelt/lint-adapter-cfg-default.sh <module-root>...` (audit A7-D1): WARNs a pure-model `Cfg` field whose
  default (its literal initializer, or 0/false) differs from the `@NiagaraProperty` default of the slot the Baja
  adapter fills it from (`cfg.f = getSlot();`, through `.getMillis()`/`.getSeconds()` for a BRelTime), so a pure
  test of `new Cfg()` runs the configuration the station starts with. Not yet run by `report-module`.

### Added — lint-logger-name

- New `toolbelt/lint-logger-name.sh <module-root>...` (G40): WARNs a `Logger.getLogger("<x>")` whose name is neither
  the part's exact `moduleName` nor a `<moduleName>.<child>` logger, and a `Logger.getLogger(X.class.getName())`.
  The module name comes from the part's gradle `moduleName`, else the group gradle, else the part directory name.
  Not yet run by `report-module`.

### Fixed — lint-module-map symlinked directories and its exit-3 contract

- `lint-module-map` no longer descends into a symlinked directory under a part: a link back up the tree no longer
  loops to an ELOOP exit 3 (or grows the walk exponentially), and a symlinked sibling directory no longer lists its
  target's files a second time as unlisted or duplicate MM4 rows. A symlinked file is still listed.
- The exit-3 contract now reads the same in the wrapper header, the rule's doc comment and `types/structure.md`:
  usage, a root with no part, an unreadable directory under a part, or an unreadable map file; the MM3 row also
  names a directory, absolute or root-escaping entry.

### Added — lint-module-map and lint.ModuleMode

- `toolbelt/lint-module-map.sh` (Go rule `go/internal/lint/rules/modulemap`) checks a module root's `MODULE-MAP.md`
  (`types/structure.md` § Module map): MM1 no map (WARN), MM2 header/table (FAIL), MM3 listed path missing (FAIL),
  MM4 source file unlisted (FAIL strict, WARN legacy) or in two concerns (FAIL), MM5 concern without a test (FAIL
  strict, WARN legacy), MM6 test in no concern (WARN). `lint.ModuleMode(path)` returns the strict/legacy/none mode
  of the nearest map for per-file lints. report-module does not run it yet.

### Added — lint-permissions-xml

- New `toolbelt/lint-permissions-xml.sh <module-root>...` (G21): FAILs a `module-permissions.xml` whose root is not
  `<permissions>` (baja ignores every request under any other root) or whose `<niagara-permission-groups>` type is
  not `station`, `workbench` or `all`; WARNs a `<req-permission>`/`<opt-permission>` without `<name>` or
  `<purposeKey>` and a `new File("/…")` or `Paths.get("/…")` with an absolute literal in `-rt` code. A file that
  does not parse exits 3. Not yet run by `report-module`.

### Fixed — rc-scan map anchors ignore the directories above the module; MM7 count and duplicate concerns

- rc-scan's `fetch-outside-services` exemption, its third-party skip and its vendored-JS skip read the path below
  the artifact dir: a module checked out under a directory named `rc/js/services`, `rc/vendor` or `rc/ext` is no
  longer exempted or skipped as a whole.
- lint-module-map MM7: the row message counts the concern-folder NAMES it decides on (`model/x` + `adapter/x` count
  once) and lists them; a second row with the same concern name (case-insensitive) is its own MM7 row.

### Added — lint-structural-tests

- New `toolbelt/lint-structural-tests.sh <module-root>...` (G34): WARNs (FAIL under `--strict`) a `srcTest` test
  that reads a `.java` source as text (`Files.readAllBytes`/`readAllLines`/`readString`/`lines`, `Paths.get`/`Path.of`
  with a `.java` literal) when its part's `srcTest` has no `BTestNg`/`BTestNgStation` class. Not yet run by
  `report-module`.

### Added — lint-module-map MM7: one row per concern folder

- `lint-module-map` MM7: a concern folder (the directory right under a role folder: rt `model/` or `adapter/`;
  ux `route/`, `service/` or `model/`; wb and se `model/`) whose files two rows list, or a non-plumbing row
  listing files in concern folders of two names, is a WARN in a `modularity: legacy` module and a FAIL in a
  `modularity: strict` one. Concern folder names are free per module (rt `model/battery/` and
  `adapter/battery/` are one concern). A module with no concern sub-folder gets no MM7 row.

### Fixed — module-map tool: retryable add-concern, Tests cells inside the root, station tests by import line

- `module-map.sh add-concern`: a run that fails midway now also removes the directories it created (deepest first,
  never a pre-existing one) and its `MODULE-MAP.md.tmp`, so the same command can be retried; a module whose
  shallowest Java source is in the default package is refused (exit 3) instead of yielding a dot-prefixed package.
- `module-map.sh tests` (and its Tests-cell expansion) refuses an entry that is absolute or leaves `--root` (exit 3),
  as lint-module-map MM3 does; a station test is one with an `import javax.baja…` line, not any file naming
  `javax.baja`; a runner that cannot start or exits other than 0/1 is exit 3.

### Added — module-map tool: add-concern, where, tests

- `toolbelt/module-map.sh add-concern <Name> --root <module-root> [--part rt|ux|wb|se] [--adapter]` (also
  `scaffold-module.sh add-concern ...`): creates the concern folder under the part's role folder, a pure
  `<Name>Logic` class and its `<Name>LogicTest`, an rt `<Name>Adapter` stub with `--adapter`, and inserts the
  concern's MODULE-MAP.md row; an existing concern is refused with nothing written. A fresh scaffold plus
  add-concern stays clean under lint-module-map, strict L3 and lint-size.
- `module-map.sh where <file|concern>` prints the owning concern, its files, tests and "change here when";
  `module-map.sh tests <concern>` runs only that concern's pure JUnit tests through `run-pure-test.sh` and names
  the station tests and JS specs it skips.

### Fixed — ng-deploy: every check runs before the copy

- `scripts/ng-deploy.sh` runs the type-count verify, the `BUILD_ID` cache-buster check and the verify-module gate
  on the jars in `build/libs` BEFORE copying them to `STATION_MODULES_DIR` (the prechecks already ran there). A
  failing check (exit 12, 50, 51, 52, 53 or 54) now leaves the station's modules dir untouched; before, the verify
  and the gate ran after the copy, so an exit 50 still left the rejected jar installed.

### Fixed — pre-push gates every pushed ref; CI runs the npm-tool suites and asserts a SKIP allowlist

- `.githooks/pre-push` gated only the FIRST ref line of a push: `git push origin a b`, `--all` or a branch deletion
  listed first let every later ref through unchecked. Every ref line now runs the gate; the first failing ref stops
  the push with its status.
- The CI bats job installs the pinned npm tools of `build-n4-module-kit/toolbelt/eslint/package.json` (eslint
  10.11.0, acorn 8.18.0; installed versions asserted, npm cache keyed on that file), so the report-module ESLint and
  lint-vendor-floor acorn tests run in CI instead of skipping.
- New `scripts/check-bats-skips.sh <bats.tap> <allowlist>`: the CI bats step fails when a test skips for a reason no
  line of `tests/bats-skip-allowlist.txt` matches (one whole-reason ERE per line; exit 1 unlisted, 3 usage or no plan).

### Fixed — rc-scan innerhtml-server clears values computed from the page's own literals

- A field of a collection the page computes from its own const literal with `.map(u => ({ k: <expr> }))` (or a
  scalar `.map(u => <expr>)`), read directly, through a const arrow or through a function declaration with one
  return, is no `innerhtml-server` row when `<expr>` uses only literals, arithmetic and comparison operators,
  `Math.*`, members of literal elements, consts computed the same way, the callback's index and
  `toFixed`/`toString`-like methods. A `let`, an unbound name, a call, a template substitution, a loop-built
  array, a function with a second `return`, or a field the page writes into keeps the row.

### Fixed — rc-scan module-map checks: anchored services exemption, third-party files skipped

- `fetch-outside-services` exempts only files under `rc/js/services/`; a `fetch(` in `js/pages/services/` or
  `rc/services/` is now reported (it was exempt through any `/services/` path segment).
- `fetch-outside-services` and `js-file-size` skip third-party files: `rc/vendor/**` (its HTML included, where an
  inline-script `fetch(` was reported), `rc/ext/**` and `*.min.js`.

### Added — rc-scan fetch-outside-services and js-file-size under a module map

- `rc-scan` reads the module map mode above the artifact. Under a map, `fetch-outside-services` reports a
  `fetch(` in rc JavaScript (a `.js` file or an inline `<script>`) outside an `rc/js/services/` file, and
  `js-file-size` a `.js` file over 400 lines: WARN in a `modularity: legacy` module (FAIL under `--strict`), FAIL
  in a `modularity: strict` one. A module with no map gets neither check; its rows are unchanged.

### Added — rc-scan `status-default-ok` (advisory)

- `rc-scan` WARNs a status read that falls back to `"ok"` when the status is missing (`p.st || "ok"`,
  `?? 'ok'`, `||= "ok"`, `??= "ok"` after a status name: `st`, `stat`, `sts`, `status` or a `*Status` name).
  An unlinked or uncommissioned point then renders as healthy. Advisory: WARN under every mode, `--strict`
  included; `// rc-scan: allow status-default-ok` on the line silences it.

### Added — report-module runs lint-module-map and prints a modularity row

- `report-module` runs `lint-module-map` once per module after `lint-structure`: its FAIL rows fail the report,
  its WARN rows (MM1 "no MODULE-MAP.md" on every legacy module) do not.
- One uncounted `(module)  INFO  modularity  mode=<m> concerns=<n> unlisted=<n> largest=<path> (<n> hand-written
  lines)` row per module: the module map mode, its concern rows, the part sources in no concern and the largest
  part class by hand-written lines. For a module with no map or a clean map the verdict and exit code are
  unchanged; an invalid or FAILing map (MM2-MM5 FAIL rows) now fails the report (exit 1), and an unreadable map or
  part directory is an env fault (exit 3). The row prints `?` for the counts and largest class in those cases.

### Fixed — report-module modularity row and the strict size relay

- The `(module) INFO modularity` row reads the mode the per-file rules apply (`lint.ModuleMode`): a strict header
  over a broken table reads `mode=strict`, not `none`. Its largest class is taken over the same files
  `lint-module-map` walks (a symlinked part included), and with a map at the root a module with no part, or an
  unreadable map or part, prints `?` instead of counts of a partial walk.
- Only `lint-size` (the mode-aware advisory member) has an exit 1 with FAIL rows relayed as FAIL; any other
  advisory member exiting 1 is an env fault (ERROR, exit 3) as before 2e261afb.

### Added — report-module runs the run-3 module lints

- `report-module` now runs `lint-permissions-xml` (its FAIL rows fail the report; PASS `clean` otherwise),
  `lint-structural-tests`, `lint-logger-name` and `lint-adapter-cfg-default` (WARN rows relayed) once per module. A
  module without their sources gets a SKIP row instead of an env fault.

### Added — report-module runs lint-vacuous-tests

- `report-module.sh` runs `lint-vacuous-tests` once per module on the module root, after the driver lints: its
  FAIL rows (a srcTest `@Test` that cannot fail or never runs) fail the report; a module with no `srcTest/`
  `*.java` gets a `SKIP  lint-vacuous-tests  no *.java under a srcTest/` row, never an env fault.

### Fixed — lint-module-map MM7, module-map tests, rc-scan trailing-slash root

- `lint-module-map` MM7 compares rows by concern AND part (case-insensitive): a concern with one row per part is
  clean, `plumbing` rows may repeat, and a second row of one concern in one part is still a row. Concern folders are
  gathered per row, so rows of one concern in two parts no longer merge into a false "two names" row.
- `module-map.sh tests` treats a srcTest class as a station test when its code references `javax.baja` fully
  qualified, not only through an import line; a mention in a comment or a string still runs as a pure test. The
  skip row reads `uses javax.baja`.
- `rc-scan` given an artifact dir with a trailing slash reads the path below it for the vendored skip and the map
  anchors, so an ancestor directory named `rc` no longer stops `rc/vendor/**/*.js` from being skipped.

### Fixed — sweep ledger tags, pre-push ref names, bats skip allowlist

- `sweep-build-state` names the real opener of an open ledger tag (`unbalanced [deferred: tag` for a
  `[deferred:` tag, open or swallowing the next one) and no longer FAILs a closed tag whose text quotes `` `[ev:` ``
  as code. It reads citeparse's new typed `Unterminated` flag instead of the free-text note.
- `.githooks/pre-push` prints `pre-push: refused <local ref> -> <remote ref> (exit N)` when a ref fails, so a
  multi-ref push says which ref was refused.
- `scripts/check-bats-skips.sh` reads a skip whose reason quotes ` # skipped` (it was dropped, so the run passed
  unchecked) and passes each allowlist line to grep as its own pattern.

### Fixed — run-2 review fix-forward: ledger-check, rc-scan provenance, slotomatic-doc-line

- `ledger-check` LC9 no longer skips every row holding `<...>` text: only the template's exact sample row is a
  placeholder row (WARN, not checked further); a row with one placeholder cell left WARNs and is still checked by
  LC4-LC8. Rows after a blank, comment or prose line inside `## Versions` are checked up to the next heading.
- `rc-scan` innerhtml-server provenance: an object literal with a token that starts no expression no longer loops
  forever; a page function declaration no longer stands in for a local binding that shadows its name; a `.map`
  verdict taken while writes were still being marked is re-checked once every write is marked; the index parameter
  of an iterator callback counts as a number only on a literal array receiver (a `Map.forEach` key does not).
- `lint-slotomatic-markers` `slotomatic-doc-line` is an ADVISORY row (no exit effect, `--strict` included, so
  `report-module` no longer relays it), and a blank `//` separator or an indented list item before the slot is not
  read as a continued sentence.

### Fixed — run-2 review fix-forward: skip allowlist last line, CL19 node, build-verify mode text

- `scripts/check-bats-skips.sh` reads an allowlist whose last line has no trailing newline (that line was dropped,
  so its skip read UNLISTED).
- `tests/ci-local.bats` CL19 skips its lockfile pin check with "node not installed" on a host without node; CI's
  bats job sets up node, so the reason is deliberately not allowlisted there.
- `build-verify.md` § Known gap: mode B (the `-ux`-only build) and mode C (`-rt` only) never regenerate `-ux` slots;
  the sentences that called mode B silent and "the exception" are gone.

### Fixed — run-3 review fix-forward: permissions-xml, adapter-cfg-default, logger-name

- `lint-permissions-xml`: a `module-permissions.xml` with no root element (empty, declaration-only or comment-only)
  FAILs instead of passing clean; ISO-8859-1 and US-ASCII files are decoded (a non-ASCII byte used to fail the parse
  as invalid UTF-8), any other declared encoding is exit 3; the absolute-path row names a FILE_READ or FILE_WRITE
  grant.
- `lint-adapter-cfg-default`: Cfg fields and slots are matched by (class, name) through the receiver's declared type
  and the class holding the statement, so two Cfg classes or two components sharing a name no longer hide or
  fabricate a row; a link it cannot resolve to one field and one slot gets no row.
- `lint-logger-name`, `lint-permissions-xml`, `lint-adapter-cfg-default`: only a part's own `test/` and `build/`
  directories are pruned, so a Java package named `test` or `build` under `src/` is read.

### Fixed — run-3 review fix-forward: env-fault verdict, repeated ledger header, KCRD2 teeth

- `report-module`: a member env fault (an `ERROR` row, exit 3) with no FAIL now reads verdict `INCOMPLETE`
  instead of `CLEAN`; the exit stays 3 and a FAIL still reads `ISSUES`.
- `ledger-check`: a header row repeated in the Versions section (for example after a blank line) is no longer
  checked as a data row (it FAILed LC5/LC6/LC7 on the column names).
- `tests/kit-cite-resolve-deferred.bats` KCRD2 asserts exit 0 and the resolved `[deferred]` pointer row; it
  passed before on a missing resolver.

### Changed — the scaffold emits a strict module map and role packages

- `scaffold-module.sh` now emits `<module-root>/MODULE-MAP.md` (`modularity: strict`) for every type: logic
  (`control`, `station-test`, `plumbing`), dashboard (`control`, `api-routing`, `frontend`, `station-test`,
  `plumbing`) and driver (`codec`, `network-device`, `points`, `discovery`, `point-manager`, `plumbing`). A fresh
  scaffold passes `lint-module-map` with no row.
- The fixtures' Baja-free classes moved to role packages: `<MOD>Logic` and the driver's `<MOD>Codec` (and their
  tests) to `com.<vendor>.<MOD>.model`, now public; the dashboard's router and servlet to
  `com.<vendor>.<MOD>.ux.route` (the ux `module-include.xml` type class and bajadoc package follow).

### Added — lint-servlet handler-size

- `lint-servlet` reports a method of a servlet file that spans more than 80 lines (its name line to its closing
  brace) as `handler-size`: WARN with no module map or a `modularity: legacy` map (FAIL under `--strict`), FAIL in
  a `modularity: strict` module. The servlet stays a router; each route group's handler is its own
  `ux/route/<Route>Handler` class (`types/dashboard.md` DUX1).

### Added — lint-slotomatic-markers `slotomatic-doc-line` (WARN)

- `lint-slotomatic-markers` WARNs a `//` comment block of two or more lines directly above a `@NiagaraProperty`
  whose last line continues the line before it: Slot-o-Matic keeps only that last line as the slot's generated
  Javadoc. The row is a WARN (exit 0; exit 1 under `--strict`, as every WARN of this lint), relayed by
  `report-module`.

### Fixed — report-module relays a strict module's size FAIL rows

- `report-module` relays an advisory member that exits 1 with a FAIL row (`lint-size` in a `modularity: strict`
  module) as FAIL rows with no PASS verdict; before, it printed one `ERROR env fault (exit 1)` row and hid the
  size rows. An exit 1 with no FAIL row is still an env fault.
- `lint-size` `adapter-branches` counts a static import of a `model.` member (`import static …model.X.y;`) as the
  adapter's pure peer.

### Added — strict lint-size, strict slot wall and the adapter branch budget

- `lint-size` reads the module map mode per file. With no map it is unchanged. Under a map its method rule covers
  every method and constructor of every class (Baja classes too) and counts a wrapped signature from its first
  line; class and method rows WARN in a `modularity: legacy` module and FAIL in a `modularity: strict` one. New
  `adapter-branches` WARN (both map modes): a Baja class that imports a `model/` peer and keeps more than 15
  hand-written `if`/`case`/`for`/`while`/`catch` branches.
- `verify-module --src` `slot-wall`: more than 15 `@NiagaraProperty` in one class now FAILs the gate in a strict
  module (still a WARN elsewhere).

### Added — strict L3 and L3b in lint-structure

- In a module whose `MODULE-MAP.md` says `modularity: strict`, `lint-structure` replaces the advisory L3 directory
  WARN with per-file FAIL rows on every `-rt`/`-ux`/`-wb`/`-se` part: a Baja-free class outside its role package
  (`model/`; for `-ux` also `route/` or `service/`), a `model/` class importing a `javax.baja` or `com.tridium`
  type, and a `model/` class with no `<C>Test.java` under `srcTest/`. New L3b (strict only): a `model/` class may
  not import the module's own non-model packages. Legacy modules and modules without a map are unchanged.

### Added — sweep-build-state: ledger evidence tags, module-field WARN, `--age --strict`

- The content check FAILs an `[ev:` / `[deferred:` tag of a `ledger.d/` fragment that is left open (no closing `]`
  before the next tag or the end of the text), named as `ledger.d/<file>:<line>`; backticked code is not a tag.
- A module envelope (not `module: kit`) with no or an empty `module_root` or `source_of_truth` prints a
  `sweep: WARN` line on stderr; the exit is unchanged (`unknown` is a value).
- `--age --strict` exits 1 when a pending retro is older than `--max-age` days (the report is unchanged; one stderr
  line names the count). Without `--strict` the report still always exits 0.

### Fixed — `build.sh --werror` survives a build that replaces the compiler args

- `toolbelt/lib/werror.init.gradle` adds `-Xlint:all -Werror` at `taskGraph.whenReady`, only when absent, so a build
  script or plugin that sets `options.compilerArgs = [...]` no longer turns `--werror` into a silent no-op.

### Added — build.sh --werror

- New `toolbelt/build.sh --werror`: an opt-in build that compiles every part with `-Xlint:all -Werror` through the
  gradle init script `toolbelt/lib/werror.init.gradle`; a javac warning fails the build (exit 30) with the remedy
  printed (fix it, or a scoped `@SuppressWarnings`). Off by default.

### Added — corpus-index rows for the course recordings

- `corpus-index.md` § n4-dev-course lists B1300 (the course's command set), B1302 (study map), B1354 (verified
  course exemplars) and the recordings B1416-B1419 with B1420, which checked their open claims against code.

### Fixed — kit lines the N4 course evidence contradicts

- `types/logic-authoring.md`: a station restart at about 5000 subscribed points is reported, its cause unconfirmed;
  a bounded driver worker queue throws `QueueFullException` to the poster, it does not restart the station.
- `types/driver-authoring.md`: COV sensitivity is driver-specific (no COV slot on the base `BTuningPolicy`; BACnet's
  `covPropertyIncrement`, default 1.0, applies to COV-Property subscriptions only).
- `types/logic.md`: `stopped()` runs on `Nre:Console` only for a console quit or an Application Director stop; a
  platform reboot or rename uses the platform service thread.
- `types/structure.md`: `niagara.security.manager.disable` switches to a logging developer mode when licensed and only
  logs a WARNING otherwise; it never turns the security manager off.

### Added — N4 course rules the kit was missing

- `types/logic-authoring.md`: why the first read after `lease()` is stale and how to wait for the driver's read;
  relation IDs must come from a tag dictionary; where the System DB lives.
- `types/dashboard.md`: count alarm occurrences by record (return-to-normal keeps the UUID); the search service takes
  bare NEQL and cannot pipe into `|bql`.
- `types/logic.md`: engine watchdog default (Terminate after 1 minute) and `getPriority() - 1` on every platform; a
  pasted component gets a new id.
- `types/distribution.md`, `types/structure.md`, `types/diagnostics.md`, `types/lexicon.md`,
  `types/driver-authoring.md`, `BUILD-LOOP.md`, `build-verify.md`: install-time permission limits, non-`B` types,
  Spy pages, lexicon limits, proxy-ext write permissions, the restart a changed module needs, the Niagara Console
  environment for a non-standard home.

### Fixed — course-coverage run A review fix-forward

- `lint-structure` L17 FAILs `<symbol>:B<Type>` only when the symbol maps to this module; another module's symbol
  (or one the palette never maps) is not judged.
- `lint-charset-default` CD1 judges `getBytes()` only on a provable String receiver (a custom `getBytes()` is
  clean), and the byte/char ambiguity guard reads C-style `x[]` declarators on both sides.

### Added — lint-charset-default

- New `toolbelt/lint-charset-default.sh [--strict] <src-root>`: WARNs text bytes encoded or decoded with the platform
  default charset — a no-argument `getBytes()`, a `new String` over a byte array without a charset, a one-argument
  `InputStreamReader` / `OutputStreamWriter`, and any `FileReader` / `FileWriter` (no charset overload on Java 8).
  Not yet run by `report-module`.

### Added — lint-intrinsic-lock

- New `toolbelt/lint-intrinsic-lock.sh [--strict] <src-root>` (WARN, advisory): IL1 a `synchronized` instance method
  and IL2 a `synchronized (this)` block in a Niagara `B*` class (the framework locks the same monitor), IL3 a
  `synchronized` block nested in another lock in any class. Not yet run by `report-module`.

### Added — lint-lexicon-apostrophe

- New `toolbelt/lint-lexicon-apostrophe.sh [--strict] <module-root>` (WARN), the `lint-lexicon-ascii` companion: a
  lexicon value with a MessageFormat argument (`{0}`) and a single `'`, which MessageFormat drops; write `''`.
  Not yet run by `report-module`.

### Added — lint-lifecycle-guard

- New `toolbelt/lint-lifecycle-guard.sh [--strict] <src-root>` (WARN, advisory): LG1 a `changed`/`added`/`removed`
  callback whose first statement after `super` is not an `isRunning()` (or `Sys.atSteadyState()`) guard; LG2 a
  `stationStarted()` doing startup work in a class with no `started()`, `serviceStarted()` or `descendantsStarted()`.
  Not yet run by `report-module`.

### Fixed — lint rare cases found by new tests

- `lint-lifecycle-guard` LG2: two anonymous bodies of the same type no longer share one finding key.
- `lint-intrinsic-lock`: IL2 judges `<Outer>.this` inside an anonymous class; IL3 no longer counts a lambda or
  anonymous-class method body inside `synchronized` as a nested lock.
- `lint-charset-default`: a keyword before `(` (`return (s + "!").getBytes()`) is no longer read as a receiver.
- `lint-structure` L17: a commented-out `moduleName` in a gradle script no longer wins.
- The part-output prune list shared by three lint rules now lives in one helper.

### Added — lint-slot-default-type

- New `toolbelt/lint-slot-default-type.sh <src-root>`: FAILs an `@NiagaraProperty` of type `int`, `long`, `float`,
  `double`, `boolean` or `String` whose `defaultValue` literal has a Java type the slot refuses (for example `int` with
  `"false"`, `float` with `"1.5"`): the module compiles and the type fails to load at runtime. An `int` literal on a
  `long`/`float`/`double` slot stays clean. Not yet run by `report-module`.

### Fixed — `lint-structure` L17 knows its own module by its declared name

- The B-prefix FAIL resolves this module from the gradle `moduleName` (part directory, else module root, else the
  directory name) and compares ignoring case, so a part directory named differently from its module is still judged.

### Added — lint-structure L17: module.palette type specs

- `lint-structure.sh` reads each part's `module.palette`: a dotted Java class name in `t=` and a `B`-prefixed name of a
  type the module declares without it FAIL (Workbench "Invalid typespec" / "Type not found"); a later `m=` that maps a
  symbol to another module, and a palette with entries that is not well-formed XML, WARN. An identical repeated `m=` is
  clean.

### Fixed — `preflight.sh --toolchains` says which route it checked

- The `toolchains` row checks the WSL/POSIX `gradlew` route only; a root with a `gradlew.bat` gets a SKIP row saying
  the Windows route is not checked, instead of a PASS read as covering it.
- A `gradle.properties` pin of a full JDK 8 passes even when the root has no executable `gradlew`.

### Added — preflight --toolchains

- New opt-in `toolbelt/preflight.sh --toolchains` row for the raw or Windows `gradlew` route: PASS when
  `gradle.properties` pins `org.gradle.java.installations.paths` to a full JDK 8, else runs
  `gradlew -q javaToolchains -Porg.gradle.java.installations.auto-detect=true` and FAILs when no full JDK 8
  (Language Version 8, Is JDK true) is listed; SKIP when `gradlew` is not executable or lists nothing.

### Changed — report-module runs the course-coverage run B lints

- `report-module` runs `lint-lexicon-apostrophe` (per artifact, after `lint-lexicon-ascii`), `lint-lifecycle-guard`
  and `lint-intrinsic-lock` (per artifact `src/`, after `lint-slot-default-type`); WARN rows relayed, no verdict row.

### Changed — report-module runs the course-coverage lints

- `report-module` runs `lint-charset-default` (WARN rows) and `lint-slot-default-type` (FAIL rows) on every artifact's
  `src/`, after `lint-bformat-call`.

### Added — mcp-n4 history extensions in one batch
- `n4_add_history` adds one interval history extension per point (1 to 50), already configured (`historyName`, `interval`, `enabled`) under one dry run and one confirmation token; it refuses an existing child of the same name and a history name that would repeat across the batch.

### Changed — lint doctrine points at the Go engine

- `BUILD-LOOP.md` § 5 (shared helpers, no fork per input line, fail-closed input listing) and `skill/SKILL.md` no
  longer tell an author to use the removed `toolbelt/lib/awk-lint.sh` / `lib/scan-files.sh`: a new lint is a Go
  rule on `go/internal/lint` (`lint.Walk`, the shared runner). The owed NUL-separated listing is closed by the
  removal of the bash lister.

### Changed — three scripts/ checks ported to Go

- `scripts/check-client-source.sh`, `scripts/check-installed-line.sh` and `scripts/check-bats-skips.sh` are thin
  wrappers over `go/cmd/<tool>` (`go/internal/checkclientsource`, `checkinstalledline`, `checkbatsskips`): same
  arguments, rows, messages and exit codes; they now need `go` on `PATH` (exit 3 without it).
- `check-bats-skips`: an allowlist line is a Go POSIX ERE. A GNU grep extension (a back-reference, `\w`, `\<`) is
  an invalid line (exit 3), and an invalid line no longer prints grep's own error before the check's message.
- `check-installed-line`: the version read and compare are Go copies of `toolbelt/lib/vendor-version.sh`
  (`vv_source_version`, `vv_cmp`), pinned to that lib by `TestMatchesTheShellLib`.

### Changed — lint-vendor-floor is a Go command

- `toolbelt/lint-vendor-floor.sh` is now a thin wrapper over `go/cmd/lint-vendor-floor` (`go/internal/vendorfloor`),
  like the other lints: the arguments, the tool resolution (`KIT_NODE`, `KIT_ACORN`, `toolbelt/eslint`), the
  vendored-file list, the API-above-floor table (`apis.go`), the verdicts and the exit codes are Go. node + acorn stay
  external tools; they run the embedded `driver.js`, which only parses and tokenizes and reports JSON lines. Rows,
  stderr lines and exit codes are unchanged.
- A `KIT_NODE` that is a directory is now the `SKIP ... unavailable: node not found` row (exit 4) instead of bash's
  own exit 126, and a node that ends without the driver's result is an env fault (exit 3), never a clean exit.
- The table check (no method name in both the member and the qualified family) is pinned by Go tests
  (`TestDefaultTableIsDisjoint`, `TestSharedMethodNameIsAnEnvFault`); the bats case `VF-disjoint`, which edited the
  bash source, and its `# Mutation:` line are removed.
- Known gaps, kept for parity (not fixed here): a `vendor/` that is a symlink is not followed ("nothing to check",
  exit 0), and a sub-directory of `vendor/` that cannot be read prints find's error line on stderr and is skipped
  (exit 0 when the rest is clean). Both are silent passes of the bash lint.

### Added — mcp-n4 n4_copy_component
- `n4_copy_component` copies a configured component N times in one batch, under one dry run and one confirmation token.

### Added — mcp-n4 create_component with a driver proxy extension
- `n4_create_component` can create a point together with its driver proxy extension.

### Fixed — mcp-n4 copy, remove and simple encoding
- A `baja:Simple` takes a whole number as its decimal string; the Modbus enum tag is `signedInteger`.
- `n4_copy_component` and `n4_remove_component` accept station-escaped names for an existing copy or source.
- A copy leaves out name-only children, the derived Modbus address and alarm extension state.

### Changed — mcp-n4 n4_set_slot integers and encoded simples
- `n4_set_slot` writes `baja:Integer` slots and encoded `baja:Simple` values; `n4_invoke_action` allows `ping`.

### Changed — mcp-n4 n4_set_slot slot types
- `n4_set_slot` types a slot the station omits at its default from the type definition (registry channel) and accepts `baja:Format` and `baja:RelTime`, so a history extension's `enabled`, `historyName` and `interval` can be set.

### Removed — dead bash lint libs

- `toolbelt/lib/awk-lint.sh` and `toolbelt/lib/scan-files.sh` are deleted with their unit tests
  (`tests/awk-lint.bats`, `tests/scan-files.bats`, `tests/lint-empty-root.bats` ER3/ER4): every lint that sourced
  them is an `n4lint` wrapper, so no script called them. The walk, the argument modes and the empty-root row live
  in `go/internal/lint`.

### Fixed — atomicfile keeps private and mode-000 targets private
- An existing target with mode 000 keeps 000 (it was rewritten as 0644), and the temporary copy of an existing target is created 0600, so a private file's new content is never readable wider than the file while it is written.

### Changed — toolbelt/build.sh is a launcher over go/cmd/build-module
- The build chain (preflight, gradle on Java 8, Slot-o-Matic skip check, deployed-baseline drift gate, verify gate, report-module) runs in Go (`go/internal/buildmodule`) with the same rows, exit codes and command lines; `build.sh` only builds and execs the binary (exit 3 without a Go toolchain).
- Declared differences, all in diagnostics bash itself printed (exit codes unchanged): a command the chain cannot start (a missing `preflight.sh` or `report-module.sh`, a `gradlew` with a bad interpreter, a missing `cp`, `mkdir` or `mktemp`) is reported as `build.sh: <command>: <reason>` instead of `<script>: line N: ...`; a gradle killed by a signal no longer adds bash's job-status lines; a module root that cannot be entered no longer prints bash's `cd` error line before `cannot resolve module root`.
- New shared Go packages: `vendorversion` (module version reader and comparator, also used by check-installed-line), `slotomaticskip`, `fstype` and `findwalk` (both moved out of `preflight`).

### Added — verify-module.sh `station-save` row and the reference `ConfigSaveScheduler`
- `verify-module.sh --src` prints one `station-save` row per jar: the classes under `<profile>/src` that call `Sys.getStation().save()`, each at file:line. None or one is a PASS row, two or more a WARN row; without `--src` (or without a `src` dir) it is a SKIP row like every source check. It is reported and never a FAIL, with or without `--strict`, so an existing module keeps building; a call in a comment or in a string is not counted and a call split across lines is. Every run now prints one more row and its summary line counts it (one more `skipped`, `passed` or `warned`): a consumer that pins the summary line of a run must re-record it.
- `fixtures/MinimalPan` ships the kit's one reference for the debounced station save: `model/ConfigSaveScheduler` and `ConfigSaveDecision` (Baja-free) with `ConfigSaveSchedulerTest` (JUnit 4, `run-pure-test.sh`). `scaffold-module.sh --type logic` copies them into a new control module; the dashboard fixture has none.

### Changed — the debounced station save is a per-STATION rule
- `types/logic.md` § Restart & persistence: one operator change yields at most one station save however many modules observe it. The owner of the persisted slot requests the save, a module that forwards or mirrors the write requests none, and where two owners can both fire the ticket skips its save when a station save started after the change it covers. `types/dashboard.md` carries the facade half and `BUILD-LOOP.md` § 6.b item 10 the live check: one `Saved .../config.bog` per operator change in the station log, two is a FAIL.

### Changed — one git runner for the install tools, kit-resolve and the push gate
- `go/internal/gitcmd` gains `Command{Dir, Env, Stderr}.Run`, which returns git's stdout, its exit code and the error, and tells a git that could not be started from one that exited non-zero. `installtools.GitStatus`, the `kit-resolve` fetch and the git calls of `pre-push` run on it; no tool changes its output, its messages or its exit codes.
- `kit-resolve` passes `GIT_TERMINAL_PROMPT=0` to its fetch only, instead of setting and restoring the variable in its own process.
### Removed — scripts/lib/live-doctrine-set.sh
- The live-doctrine set is defined once, in `go/internal/prepush/paths.go`. The shell copy had no caller left; it is deleted with the test that compared the two copies, and the Go patterns are pinned by the paths they select.

### Changed — the install and bootstrap scripts are launchers over Go commands
- `scripts/install-skill.sh`, `check-skill-drift.sh`, `kit-resolve.sh`, `install-hooks.sh` and `install-client-hooks.sh` only build and exec `go/cmd/<name>` (`go/internal/skillinstall`, `kitresolve`, `hookinstall`, shared helpers in `installtools`). Options, messages, exit codes and the files and git configuration written are the same: a differential of 143 runs against the bash versions has 125 identical and the 18 below. Every one of them now needs a Go toolchain to build (exit 3 without it); `kit-resolve.sh` therefore no longer runs from a copy outside a checkout.
- `install-skill.sh` refuses (exit 3, nothing reported as installed) where the bash version printed `installed` and exited 0: the copy failed (a read-only target, a target that is a directory, an unreadable tracked launcher); `$BUILD_N4_KIT` could not be examined (it silently materialized the repository's kit instead); `toolbelt/kit-state.sh` could not be examined (it reported "no kit-state.sh"); and an empty `$HOME` without `--home` (it targeted `/.claude`).
- `install-client-hooks.sh` refuses (exit 3, nothing changed) when the default hooks directory cannot be read, instead of installing and silently stopping the hooks in it.
- `kit-resolve.sh` refuses (exit 2, `REFUSED unreadable`) a candidate it cannot examine instead of skipping to the next one, and a kit whose `git status` fails (a corrupt index, reached with `--offline-ok`) instead of reading it as clean and printing the KIT line.
- The sha256 tool check (`sha256sum` or `shasum`) stays in the two skill launchers although the digests are computed in Go, and now runs first: without either tool, a usage error or a "not installed" row is exit 3 with the missing-tool message.
- Diagnostics only (exit codes unchanged): `--help` of `check-skill-drift.sh` and `kit-resolve.sh` prints the script header, which gained an Implementation paragraph; a failed copy, a missing kit toolbelt directory and an unreadable installed copy are reported in the tool's own words instead of `cp`'s, bash's `cd` and `sha256sum`'s; the active hooks named by an `install-client-hooks.sh` refusal are in byte order whatever the locale.

### Fixed — mcp-n4: a copy keeps the flags of a frozen slot at its default value
- `n4_copy_component` dropped a frozen slot that carried only integrator flags (name plus `f`, value at its default) before the flags were carried, so the copy lost them while the verdict said `verified`. The add op now carries such a slot as name plus flags; proven live on a 4.15.6.20 station from the saved `config.bog`.

### Changed — live-smoke: tier opt-ins, failure reasons, report path checked first
- `live-smoke` accepts `--allow-tier-b NAME` and `--allow-tier-c NAME` and passes them to the server (the operator's opt-in; none is ever added by the runner). Without them it could not run on a station that is not tier A.
- A step that ran and did not verify prints its reason on the console, indented under its line.
- `--report PATH` whose directory does not exist is exit 2 before the server starts (it was exit 1 after the whole run, with the report lost).

### Added — n4call, a bench client for the mcp-n4 server
- `go/bin/n4call --calls FILE [--dry-run-only] [--call-timeout SECONDS] -- SERVER [ARGS...]` runs the tool calls of a JSON-lines file in one session; a line with `"write": true` is sent as a dry run and then executed with the returned token. One JSON line per request; credential values are masked; exit 1 at the first tool error.

### Changed — live-smoke: the rollback of the removed folder is a required step
- The rollback of a removed nested subtree gates the exit code of `live-smoke` (it was probe-only: recorded, never failing the run). It verified on a 4.15.6.20 station with its three links restored; the plan no longer prints a probe-only line.

### Added — mcp-n4: a test that a copy accounts for every source child
- `n4_copy_component` plans are pinned by an invariant: every child the station loaded is in the add op, named in the plan notes, or bare (name only, no flags). The defect it would have caught: a copy that lost the flags of a frozen slot at its default value (such a child was in none of the three). A copied child must also keep the flags of the source.

### Fixed — mcp-n4: the kept type contracts end with the station session
- The BOX client kept every type contract it loaded for its whole life. Opening a session now forgets them, so a station that restarted with another module set is asked again.

### Fixed — mcp-n4 n4_copy_component no longer drops configuration by slot name
- A copy leaves out runtime slots by flags instead of by a list of names, read from the station's type registry: a frozen slot by its declared flags (r readonly, t transient, d default-on-clone) or when made transient, a dynamic slot only when flagged t or R (remove-on-clone). Flags an integrator set are configuration and travel with the copy; a child named `status` or `health`, or a fallback's status, is copied.
- The plan notes list every node left out with its flags, and call out a readonly slot that holds a non-default value.
- The read-back loads each copy and compares its whole configured subtree with the plan, listing the paths that differ or are missing.
- A copy is refused when the client cannot read the station's type definitions or the station does not describe a type it must evaluate.
- Fixes #612.

### Changed — box client caches type definitions per session
- `LoadTypes` keeps every type entry the station sent; a request for types it already holds makes no registry call.

### Added — live-smoke replaces live_smoke.py
- `go/bin/live-smoke` (`go/cmd/mcp-n4/live-smoke`, logic in `go/internal/mcpn4/livesmoke`) is the live smoke runner: plan only by default; `--apply` starts `mcp-n4 --allow-writes` over stdio and runs the kitControl thermostat scenario, every write as dry run -> token -> execute -> verdict, with credential values scrubbed from all output. Same flags, plan, usage errors, report and exit codes as `tools/live_smoke.py` (differential: identical stdout, stderr and exit codes once the program name `live_smoke` reads `live-smoke`). The default server is the `mcp-n4` next to the `live-smoke` binary (`go/bin/mcp-n4`).
- `go/internal/mcpn4/readtools/readtoolstest/boxstation`: an in-memory fake BOX station over HTTP (the Go port of `tests/fake_station.py`'s FakeStation) that the Go tests drive the real server code against.

### Removed — mcp-n4-kit/tools/live_smoke.py and tests/test_live_smoke.py
- Use `go/bin/live-smoke`; its 37 tests are Go tests in `go/internal/mcpn4/livesmoke`.

### Fixed — live-smoke: a relative `--go-bin` starts the file that was checked
- `live-smoke --go-bin NAME` (or `MCP_N4_GO_BIN`) with a name that has no directory was checked as a file of the working directory and then started through `PATH`, so another `NAME` could run or the start could fail. The resolved path is now absolute.

### Changed — mcp-n4-kit mutation checks follow the Go rule
- A mutation check runs both the mutated and the restored test with `go test -count=1`, so a cached result is never read as RED or GREEN, and confirms the mutation is gone with `git diff --exit-code` on the mutated file before reading GREEN (`mcp-n4-kit/METHODOLOGY.md` section 8). The Python bytecode rule and `tools/mutation_check.py` are retired.
- `mcp-n4-kit/README.md` "Run the tests" is Go only: `go test -count=1 ./internal/mcpn4/... ./cmd/mcp-n4/...`; shuffled runs use `go test -shuffle`. The untested-branch audit uses `go test -coverprofile`.
- `blackbox.TestMethodologyCitesNoPythonTest` replaces the Python-citation check: METHODOLOGY.md may cite no `test_` Python name.

### Added — retro-draft --write replaces new_retro.py
- `go/bin/retro-draft --write --station NAME --state-dir DIR [--date D] [--since TS] [--out-dir DIR] [--force] [--dry-run]` writes the session retro and its `pending` row in `retros/INDEX.md`, with the same refusals, output and exit codes as `tools/new_retro.py` (differential: identical stdout, files, modes and exit codes; stderr differs only in the program prefix). `--date` defaults to today (UTC) in write mode only; draft mode still requires it.
- `go/internal/atomicfile`: the one atomic-write helper (temporary file in the same directory, O_EXCL name, rename; an existing target keeps its exact mode, a new file is 0644 under the umask; the temporary file is removed on any failure).

### Changed — gen writes through atomicfile
- Generated files (BUILD-STATE, lint index) now keep an existing target's exact mode: a group-writable 0664 file stays 0664 instead of being masked by the umask. The temporary file is a hidden O_EXCL name next to the target instead of `<path>.tmp.<pid>`.

### Removed — mcp-n4-kit/tools/new_retro.py
- Use `retro-draft --write`; the skill, README, METHODOLOGY and retros/INDEX.md name it.

### Changed — mcp-n4 retro header names `retro-draft --write`
- A drafted retro's header says it was generated by `n4_session_retro_draft` / `retro-draft --write`; it named `tools/new_retro.py`, which no longer exists.
- The recorded draft goldens are now Go-owned: an intended output change edits them by hand in the same commit (documented in `go/internal/mcpn4/golden`).

### Added — mcp-n4: n4_set_flags sets the flags of a slot
- `n4_set_flags(ord, slot, flags)` sets the complete flag set of one slot of a component (letters such as `h` hidden, `o` operator, `r` readonly; `""` clears them) with the station's set-flags sync op, under the same dry run, confirmation token, write scope, tier gate and journal as every write. The plan shows the current and the requested flags with their names; the inverse restores the previous flags exactly; the read-back loads the slot again and says `verified` only when it holds the requested set, else `mismatch` with both sets. `n4_rollback` of such a batch restores the flags after checking that nobody changed them since, and verifies by read-back.
- Refused: an unknown or repeated flag letter, a slot the component does not have, a set equal to the current one, and a slot whose current flags cannot be read (not sent by the station and its type contract unreadable), because no faithful inverse exists.
- Proven on a live 4.15.6.20 laboratory station for a dynamic slot, a frozen property and a frozen action: each set read back `verified`, the saved `config.bog` held the flags (`f="h"`, `f="ths"`, `f="ho"`), and each rollback read back `verified` and left no `f` attribute after a save.

### Changed — scripts/ng-deploy.sh is a launcher over go/cmd/ng-deploy
- The deploy (backup, slotomatic reminder and run, gradle build, version-bump and schema-risk prechecks, type-count and cache-buster verify, verify-module gate, copy, `.last-deploy-sha`) runs in Go (`go/internal/ngdeploy`) with the same lines, exit codes and command lines; `ng-deploy.sh` only builds and execs the binary (exit 3 without a Go toolchain). Sourced with `BATS_TEST_MODE=1` it still offers `verify_jar`, `verify_cachebuster` and `backup`, now over the same binary.
- **The env file is no longer sourced.** It is read as plain assignments (`KEY=value`, `export KEY=value`, single or double quotes, comments, blank lines; a missing file is still tolerated). `$NAME` and `${NAME}` expand in bare text and inside double quotes exactly as before (an earlier line of the file first, then the deploy's own option variables and the environment), and a leading `~` or `~/` in a value (also after a `:`) is still the home directory. Only non-plain constructs are refused, with exit 10 naming the file and line and never printing the line: a command or function, `$(...)`, backquotes, arithmetic, `${NAME<op>...}` forms (`:-`, `#`, `%`, `/`, length), the shell parameters (`$1`, `$@`, `$?`, `$$`), `$'...'`, `~user`, a shell operator, a multi-line value. A variable that is not set is exit 10 naming the variable (bash, which sourced the file under `set -u`: exit 1 `unbound variable`). An env file that cannot be read, or that assigns `SCRIPT_DIR`, `VERSION_FILE`, `SCRIPT_VERSION` or `IFS`, is exit 10 (bash: exit 1, or a changed output).
- False successes turned into refusals (exit 10 before anything runs): `--keep 0`, `--keep -1` or an empty `KEEP_N` deleted every backup including the new one and still printed `backup ok`; `--keep <word>` purged nothing; `EXPECTED_RT_TYPES=09` (any value bash could not read as a number) made the type-count test fail open and deployed any jar; `MODE=<invalid>` in the env file deployed nothing and ended `copy ok`. `KEEP_N` must be a whole number, 1 or more; the expected type counts must be whole numbers.
- "Could not check" now refuses: a deployed jar that cannot be examined stops the backup (exit 20) and the version-bump precheck (exit 52) instead of reading as "not deployed"; `-ux` sources that cannot be read stop `--with-slotomatic` (exit 15) instead of reading as "no annotation"; in strict mode (`--strict-slotomatic` or `SLOTOMATIC_DETECTION=strict`) an annotation detection that could not run (no git, not a repository, a failed `git diff`, an unreadable `.last-deploy-sha`) is exit 15 instead of passing like "no change". Outside strict mode the detection behaves as before.
- Declared differences in diagnostics (exit codes unchanged unless said): a switch value that is not a number (`FULL_BACKUP=yes`, `NO_GATE=true` in the env file) is exit 10 `<NAME> must be 0 or 1` before anything runs, where bash stopped with exit 1 and `unbound variable` when it reached the test; a non-numeric expected type count is exit 10 before the build (bash: exit 1 after it); a command that cannot be started (`gradlew`, `unzip`, `cp`, `mkdir`, the gate or schema-risk tool) and a `.last-deploy-sha` that cannot be written are reported as `ng-deploy.sh: <command or file>: <reason>` instead of `<script>: line N: ...`; a gradlew killed by a signal no longer adds bash's job-status line; the slotomatic output is no longer copied through a temporary file.
- New shared Go package `envfile` (plain-assignment env file reader). `toolbelt/lib/fs-type.sh` is deleted: no script sourced it, and `go/internal/fstype` keeps its answers as tests.

### Fixed — ng-deploy: the deploy record is validated and its failures are no longer silent
- **New exit 41.** After the copy, a commit that could not be read (`git rev-parse HEAD` fails, git is absent, or it prints something other than a full commit id) or could not be written to `.last-deploy-sha` is `[ng-deploy] ERROR jars copied, but the deployed commit was not recorded in .last-deploy-sha: <reason> (...)` on stderr and exit 41; before, the deploy ended with exit 0 and the next annotation check silently started at `HEAD~1`. The jars are in the station when this happens: the restart line is still printed, and nothing is rolled back. A deploy from a directory that is not a git checkout now needs `SLOTOMATIC_DETECTION=off`, which keeps the record best effort as before (exit 0; a file that cannot be written is still reported as `ng-deploy.sh: <file>: <reason>`).
- Only a full object name (40 or 64 lower-case hex digits) is written to `.last-deploy-sha`, and only such a record is handed to `git cat-file` / `git diff`. Any other content (an abbreviated id, a ref name, text starting with `-`, a trailing carriage return) never reaches git: stderr gets `[ng-deploy] NOTICE: .last-deploy-sha does not hold a full commit id; annotation changes are looked for since HEAD~1 only`, the diff starts at `HEAD~1`, and in strict mode the deploy stops with exit 15 (the detection could not run). A hand-written abbreviated id worked before and must be replaced by the full one; an empty file is still read as "no record" without a notice.

### Fixed — ng-deploy: every silent stop has an ERROR line and a documented exit code
- A built jar whose `module.xml` declares no type is counted as 0 types: with `EXPECTED_<RT|UX>_TYPES=0` the deploy goes on (`verify ok (<jar>: 0/0)`); with any other expected count it is the usual mismatch, exit 50 `verify failed: expected N <type entries in <jar>, found 0`. Before: exit 1 with no message in both cases.
- An `unzip` that fails while the type count is read, or is not installed, is exit 50 `verify failed: cannot read META-INF/module.xml in <jar> (unzip exit N)`. Before: unzip's own status (any number) or exit 1, with no ERROR line; a failed unzip that printed the expected number of entries no longer needs a second look.
- A build that passed and left no jar in `build/libs` is exit 52 `version-bump precheck cannot verify <jar>: the build left no jar at <path>`, before any jar is compared. Before: the version-bump precheck was skipped without a word and the deploy stopped later (exit 1 at verify, or exit 40 at the copy).
- **New exit 21.** After the backup is written, an older backup of the module that cannot be listed, examined or removed stops the deploy before the build: `backup written (<new backup>), but an older backup could not be purged: <path>: <reason>`. The new backup stays, the other old backups are still purged. Before: the purge failed silently and the deploy went on to `backup ok`.
- An env file path that is a directory (`--env-file <directory>`, or a directory named `.env.local`) is exit 10 `env file <path> is a directory`. Before: it was ignored like a missing file and the deploy ran on the ambient environment. A missing file is still tolerated.
- Outside strict mode a failed `git diff` is no longer silent: stderr gets `[ng-deploy] NOTICE: git diff <baseline>..HEAD failed (exit N); skipping annotation change detection` and the deploy goes on, as it does without git or outside a repository. In strict mode the same notice now precedes the exit 15 ERROR line.

### Fixed — ng-deploy: cosmetic and timing defects
- The `build:` and `gate:` lines name the tasks and the jars on one line, separated by spaces (`build: mode A — :m-rt:clean :m-ux:clean :m-rt:jar :m-ux:jar`, `gate: verify-module on m-rt.jar m-ux.jar`). Before: one name per line. The commands themselves are unchanged.
- Strict mode (`--strict-slotomatic` or `SLOTOMATIC_DETECTION=strict`) is satisfied by `--with-slotomatic`: the deploy runs slotomatic and goes on, whether annotation changes were detected or the detection could not run. Before: exit 15 even though slotomatic was about to run. Without `--with-slotomatic` strict mode is unchanged.
- A backup made in the same second as an existing one of the module is named `<MODULE>-pre-<time>-2.tar.gz` (then `-3`, ...). Before: the second tar wrote over the first backup. A first backup in a second keeps the plain name; the numbered ones are purged with the others.
- The purge never removes the backup just written: it counts as the newest whatever its modification time says. Before: older backups stamped at the same instant or later (a coarse file system, a clock set back) could push it out of the kept set, and the run still printed `backup ok` for a file that was gone.
- A CRLF env file is read: one trailing carriage return is stripped from every line, and stderr gets `[ng-deploy] WARN env file <path>: a carriage return was stripped from the end of N line(s), the first at line L (CRLF line endings; save the file with LF)`. Before: every value ended in a carriage return, and a blank line was refused with exit 10. An LF file is read as before, with no warning; a carriage return inside a value stays. `go/internal/envfile` strips only for a caller that passes `TrimCR`.

### Removed — scripts/ng-deploy.sh can no longer be sourced for single steps
- `scripts/ng-deploy.sh` is a plain launcher: it builds `go/bin/ng-deploy` on demand and execs it. The `BATS_TEST_MODE=1` branch and the `verify_jar`, `verify_cachebuster` and `backup` shell functions it offered to a sourcing test are gone, and the binary no longer reads `NG_DEPLOY_STEP` (`go/internal/ngdeploy/steps.go` is deleted). A deploy run as a command is unchanged: same lines, exit codes and command lines. One difference for a shell that exports `BATS_TEST_MODE=1`: the launcher used to define its functions and exit 0 without deploying; it now deploys, as documented.
- The eight `tests/ng-deploy.bats` cases that used those functions are Go tests of `go/internal/ngdeploy` (`verify_single_jar_test.go`, `backup_archive_test.go`); the three backup cases read the archive the host's `tar` wrote.

### Removed — the bash libs `toolbelt/lib/vendor-version.sh` and `toolbelt/lib/slotomatic-skip.sh`
- No script sourced them since the build and the deploy moved to Go. Their cases are tests of `go/internal/vendorversion` and `go/internal/slotomaticskip`; `tests/vendor-version.bats` and `tests/slotomatic-skip.bats` are deleted with them, and `toolbelt/lib` now holds only `werror.init.gradle`.

### Changed — .githooks/pre-push is a launcher over go/cmd/pre-push
- The push gate (client-identifier guard, live-doctrine citation gate, changelog gate, retro and ledger gate with the `Retro:` / `Changelog:` trailers, content sweep) runs in Go (`go/internal/prepush`) with the same decisions and messages. The hook builds `go/bin/pre-push` on first use; without a Go toolchain, or when the build fails, the push is refused.
- A client denylist that exists but cannot be examined now refuses the push when `client-guard` is not on PATH; the bash hook treated it as absent and went on unchecked.
- A `client-guard` that cannot be started and a sweep whose temporary directory cannot be created are reported as `pre-push: cannot run <tool>: <error>` instead of the shell's own line; the decision is unchanged.

### Changed — toolbelt/scaffold-module.sh is a launcher over go/cmd/scaffold-module
- The scaffolder (fixture copy, name substitution, station-test overlay, version ledger, `add-concern` passthrough) runs in Go (`go/internal/scaffoldmodule`) with the same emitted trees, messages and exit codes; the script only builds and execs the binary (exit 3 without a Go toolchain).
- An option value sed rejected as replacement text (an unescaped `|`, a newline, a trailing `\`, `\1`-`\9`) and the GNU-only case escapes (`\L \U \l \u \E` and `\c \d \o \x`) exit 2 and write nothing; the bash tool printed sed errors, left a broken tree and exited 0.
- A write fault while emitting (a read-only out-dir) exits 3 with one line; the bash tool exited 0 with "emitted".

### Changed — toolbelt/orient-guard.sh, mirror-niagara-home.sh and run-pure-test.sh are launchers over Go
- The three tools run in Go (`go/internal/orientguard`, `go/internal/mirrorhome`, `go/internal/puretest`) with the same rows, messages, exit codes and resulting trees; each script only builds and execs its binary (exit 3 without a Go toolchain).
- `orient-guard.sh`: a path that cannot be resolved (a missing directory before the last component, a path through a file, a link loop) no longer PASSes on its text alone: what does resolve of it must stay under the legal prefix. The bash tool PASSed `<prefix>/New/../../elsewhere`. The path is resolved by the tool itself, with or without `realpath` on PATH: a relative path whose last component is missing, which FAILed as raw text on a host without `realpath`, now PASSes as it always did with `realpath` present (a difference that is not a refusal).
- `mirror-niagara-home.sh`: a relative `<source_niagara_home>` is refused (exit 1) before anything is wiped, and so is a source or a `modules/` that cannot be listed; the bash tool printed "mirror ready" over a mirror of dangling links. A `<mirror_dir>` that exists and is not a directory gets the guard refusal (exit 20) instead of a bare `cd` error (exit 1): a difference that is not a refusal of a false success, it is the exit the header documents. The source is listed before the old mirror is wiped.
- `run-pure-test.sh`: with several copies of a jar under `~/.gradle` the first in name order is used (it was the first in directory order).

### Removed — bash libraries without a shell caller
- `toolbelt/lib/method-boundary.sh` (replaced by `go/internal/javasrc`; `tests/method-boundary.bats` MB1-MB9 and MB-e2e are cases of its Go tests; the awk-equivalence test `TestDifferentialAwk`, which ran the lib as an oracle over the fixture corpus, is deleted with it and `javasrc` with its own cases is the reference) and `toolbelt/lib/signing-block.sh` (`go/internal/signblock`, whose test now pins the table entry by entry; `tests/lint-structure.bats` LS15-sync fails on a second copy).

### Changed — changelog-gate refuses a line added inside [Unreleased] outside a release

- `scripts/changelog-gate.sh` (and the pre-push gate that runs it in process) now FAILS a range that adds a
  non-blank line inside `CHANGELOG.md`'s `## [Unreleased]` unless the range is a release: it deletes a
  `changelog.d/` fragment (the `gen-changelog.sh --write` fold) or changes `VERSION` (the bump). The refusal holds
  next to a valid fragment and in a range that changes no behaviour, and names `changelog.d/<YYYY-MM-DD>-<slug>.md`
  as the fix. Before, such a line was an accepted carrier, and four open PRs that each used it conflicted on
  `CHANGELOG.md`. The missing-entry message no longer offers an `[Unreleased]` line.

### Fixed — check scripts and lint-vendor-floor fail closed on the paths their ports kept open

- `scripts/check-installed-line.sh` reads git's file names NUL-separated: a group whose path git prints in quotes (a
  non-ASCII directory) is compared like any other. Before, such a group was skipped, so its downgrade was reported
  by no row and the exit stayed 0. Retro go-check-scripts-ports Δ1.
- `scripts/check-client-source.sh --build-state FILE`: a file that exists and cannot be read is an environment fault,
  exit 3 with `check-client-source: cannot read FILE: <reason>` and no row. Before, the same stderr line was followed
  by `WARN  source-of-truth  source_of_truth undeclared` and exit 0. Retro go-check-scripts-ports Δ2.
- `scripts/check-bats-skips.sh`: the header of `tests/bats-skip-allowlist.txt` states the pattern dialect (POSIX ERE as
  Go `regexp.CompilePOSIX` reads it), and a line Go compiles with another meaning than the author wrote is exit 3 with
  `check-bats-skips: allowlist line uses <construct>, outside the dialect (...): <line>`: a backslash before anything
  but an ERE operator (`\<`, `\>`, `\n`, `\t`, `\x41`), a backslash inside a bracket expression, `[[:word:]]`,
  `[[=a=]]`, `[[.a.]]`. Before, such a line was matched silently. A line Go does not compile keeps
  `allowlist line is not a valid ERE` and exit 3. Retro go-check-scripts-ports Δ3.
- `toolbelt/lint-vendor-floor.sh` follows a `vendor/` that is a symlink (as `find -H` follows a root; a symlink below
  it is still neither followed nor listed), so a library vendored through a link is linted. Before, the run printed
  `no *.js under <rc>/vendor — nothing to check` and exited 0.
- `toolbelt/lint-vendor-floor.sh` exits 3, with no row and before any tool is asked, when `vendor/` cannot be listed
  whole: `lint-vendor-floor: env fault: cannot list every file under <rc>/vendor: find: '<dir>': Permission denied`
  for each unreadable sub-directory (before: the bare `find:` line, then the verdict of the files that were listed,
  exit 0 when they were clean), and `lint-vendor-floor: env fault: <rc>/vendor is a symlink that leads nowhere` for a
  dangling link (before: `no vendor/ under <rc>`, exit 0). Retro go-lint-vendor-floor Δ1.
- Doctrine (`CONTRIBUTING.md` § 2, `BUILD-LOOP.md` § 5 Kit tooling doctrine): a committed test that edits a tool's
  source text to prove a guard is a finding; the guard is tested through the command's interface with the data
  injected. An observed mutation stays a review-time proof run by hand. Retro go-lint-vendor-floor Δ2.

### Added — lint for the template archives a module ships

- `toolbelt/lint-ntpl-manifest-deps.sh [--niagara-home <dir>] <module-root>` (Go lint G92): every `*.ntpl` under a
  module root (the extension in any letter case) must hold `template.bog` and `template-manifest.xml`, a well-formed
  `<template>` manifest with every `<dependency>` named and inside `<dependencies>`, and only dependencies that a
  `*.gradle.kts` of the module declares (`api`, `nre`, `implementation`) or that the Niagara installation ships; a
  template under `newStations/` must carry `isStation="true"` on its manifest root. Rows
  `FAIL  ntpl-entries|ntpl-manifest|ntpl-deps|ntpl-station`; exit 0 no FAIL (one PASS row when the module ships no
  `.ntpl`), 1 any FAIL, 3 usage or environment.
- The installation is `--niagara-home <dir>` or `NIAGARA_HOME` (the option wins): each `modules/<name>.jar` is a
  module it ships. Without one the lint cannot tell a stock module from a missing one: it prints one WARN row
  saying so, reports undeclared dependencies as `WARN  ntpl-deps`, and exits 0. A path that does not exist or has
  no `modules/` is exit 3.
- The archive is read as untrusted input: 32 MiB, 10000 entries and a 4 MiB manifest at most, and an unsafe entry
  name refuses it. A template behind a symlink (a linked file, or a file under a linked directory) is read
  through the link; a `.ntpl` link that leads nowhere is exit 3. A path with a control character is printed
  quoted, one row per line. `report-module` does not run the lint yet.
- `types/logic-authoring.md` § What a template upgrade keeps and what it wipes; `types/distribution.md` § 8 Check.

### Changed — four gate tools refuse inputs they used to pass

- `scripts/check-bats-skips.sh`: an allowlist line using `{,2}`, `{}`, an interval bound with a leading zero
  (`{02}`, `{1,02}`), any other unescaped brace that opens no `{n}`, `{n,}` or `{n,m}` interval, a star right after
  the start anchor (`^*`), or a class name in single brackets (`[:alpha:]`) is exit 3 naming the construct. Such a
  line was matched as Go reads it, which is not what GNU `grep -E` reads. Write `\{` for a literal brace, `\*` for
  a literal star and `[[:alpha:]]` for a class.
- `toolbelt/lint-vendor-floor.sh`: an rc directory the process may not search is exit 3
  (`env fault: cannot look for vendor/ under <rc-dir>: <reason>`). It was "no vendor/ under <rc-dir>", exit 0.
- `scripts/check-client-source.sh`: a BUILD-STATE file with CRLF or CR-only line endings, a doubled CR, a UTF-8
  byte-order mark, or blanks after the `module:` or `source_of_truth:` value is read as what it declares, and
  `--build-state ""` is the usage error (exit 2) unless `--declared` is given. All were
  `WARN  source-of-truth  source_of_truth undeclared`, exit 0. A value no longer keeps the blanks that end it.

### Fixed — a crash and two silent skips

- `toolbelt/report-module.sh`: a module with a part whose `src/rc` is a regular file crashed the report
  (`slice bounds out of range` in the module-map reader, exit 2); it gets its ordinary rows.
- `toolbelt/report-module.sh`: a `src/rc/vendor` symlink that leads nowhere gets the
  `ERROR  vendor-floor  env fault (exit 3)` row; it got no row and the report could read CLEAN.
- `scripts/check-installed-line.sh`: a working-tree entry literally named `<ref>:<group>/build.gradle.kts` no longer
  hides the group (its downgrade was not reported), and a group path holding a control character, a double quote
  or a backslash is printed quoted with C escapes, so a path with a newline is one row.

### Added — a size rule for the Go tree

- `go test ./...` now fails when a non-test Go file exceeds 200 lines, unless `go/internal/filesize/baseline.txt`
  lists it. The baseline holds the 98 files that were already larger, each with its line count: a listed file may
  shrink but not grow, no file may be added, and a stale line is itself a failure. Split a file by
  responsibility, then lower or delete its line.

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

### Added — lab-sim: a BACnet/IP front end, written from the standard and not yet validated against another stack
- `lab-sim` now also serves the devices of a scenario as BACnet/IP devices, one UDP socket per device (default `127.0.0.1:47809`), so a station's BACnet driver can be proven with no field device: Who-Is/I-Am, Who-Has/I-Have, ReadProperty with array indexes, ReadPropertyMultiple with ALL/REQUIRED/OPTIONAL and per-property errors, WriteProperty with command priorities 1-16, NULL relinquish and a relinquish default, and SubscribeCOV (confirmed or unconfirmed, lifetime, cancellation) when the scenario enables it. Any other confirmed service is rejected; there is no segmentation (an answer that does not fit is aborted with segmentation-not-supported), no routing, no MS/TP, no alarms, trends or schedules. The codec was written test-first from the rules of the standard and has exchanged packets only with its own probe: `go/internal/labsim/README.md` says which test frames are the standard's own examples and which are constructed.
- A device gains a `bacnet` section (device instance, listen address, vendor, model, maximum APDU, COV) and a point a `bacnet` mapping (object type among analog, binary and multi-state input, output and value; instance; units; commandable; texts; limits; COV increment), validated with one message per fault. The existing behaviours and faults apply: an offline device answers nothing, `delay_ms` holds the response, `fail_after_reads` gives Error device/operational-problem. Both example scenarios carry BACnet mappings, and `lab-sim check` prints the object list of each BACnet device (object type, instance, name, units, commandable, access).
- It answers Register-Foreign-Device and keeps a bounded foreign device table, so its I-Am reaches a registered station as a Forwarded-NPDU; it sends to a broadcast address only when `--bacnet-broadcast` names one, and binds outside loopback only with `--bacnet-listen`. Packet size, foreign devices, COV subscriptions, remembered answers, held responses and the packet rate per source are bounded.
- The request log gains BACnet entries (`"protocol":"bacnet-ip"`, with service, object, property, index, priority, values and the error, reject or abort); Modbus lines are unchanged. `lab-sim status` lists the address of every front end, `get` and `set` work on the same points (a control `set` changes the relinquish default of a commandable object and never a level of its priority array, and `get` gives the effective value), and `lab-sim bacnet-probe` (loopback only) sends who-is, read, read-multiple, write and subscribe to a simulator on this machine. The README describes three routes from a station on the same Windows machine, none of them tried yet.

### Added — lab-sim: a Modbus TCP device simulator for the laboratory
- `go/cmd/lab-sim` (`go/internal/labsim`, standard library only) simulates field devices for a laboratory station that has none, so device, proxy-point, polling and write work can be proven live. `lab-sim run <scenario.json>` serves several unit ids behind one Modbus TCP listener: function codes 1, 2, 3, 4, 5, 6, 15 and 16 with the quantity limits of the specification, exceptions 1 (illegal function), 2 (an address with no point, a write to a read-only point), 3 (illegal data value) and 4 (injected device failure), and an unknown unit id either silent or answered with exception 0x0B. Connections are bounded, each frame has a read deadline, and it stops on Ctrl-C, SIGTERM or `lab-sim stop`.
- The scenario file (JSON, validated on load with one message per fault) names the devices by unit id and their points in the four tables, as `bit`, `int16`, `uint16`, `int32`, `uint32` or `float32` in the byte orders `ABCD`, `CDAB`, `BADC` and `DCBA`, read-only or writable, with an optional behaviour (`constant`, `ramp`, `sine`, a seeded `random_walk`) and faults (`fail_after_reads`, `delay_ms`, device `offline` windows). `lab-sim check <scenario.json>` prints the register map with each address 0-based and in the 0xxxx/1xxxx/3xxxx/4xxxx notation. Two examples ship in `go/internal/labsim/examples/`: an air-handling unit and energy meters.
- The simulator listens on loopback by default; binding any other address needs an explicit `--listen`. `--log FILE` writes one JSON line per request (time, unit, function, address, quantity, values written, values read, exception or no response). `lab-sim get`, `set`, `status` and `stop` talk to the running simulator over a control port that is loopback-only and refuses any other address, and `lab-sim probe` sends one Modbus request to a simulator on this machine. Builds for Linux and Windows (`GOOS=windows`); `go/internal/labsim/README.md` has the scenario format, the control port and an outline for proving a Modbus network against it.

### Added — mcp-n4: commissioning report
- `n4_commissioning_report` answers "what is still wrong or unfinished below this part of the station?" in one call (a read tool, available without `--allow-writes`): a `summary` line per section first (`ok`, `N finding(s)`, `not checked: <why>`, `skipped: <why>`), then `requests` (the station reads made) and the sections. `points`: the control points per status flag (in fault, down, stale, disabled, overridden, in alarm, null output), the devices that are down, disabled or in fault listed first with their points counted and not listed, and the counts per driver network. `histories`: the points with no history extension (listed and counted as findings with `expect_histories`; without it a count with the note `pass expect_histories to list them`) and the extensions that are disabled, in fault or have never recorded. `alarms`: the points with no alarm extension (only with `expect_alarms`), the extensions whose alarm class does not exist, the classes with no recipient linked, and the open and unacknowledged alarms per class. `links`: the links whose source names nothing, the priority inputs several links feed, and the outputs of the dangling-output check. `naming`: display names repeated among siblings, names with escaped characters, palette default names (a point of the control palette, or a point called by the name of its own type; the fact says which). `station` (whole station only): the services in fault and a licence expiring soon; unsaved changes and the time of the last backup are marked `not_checked` with the reason, because the station exposes neither. Every check is a count, the first `limit` offenders (`path`, `name`, `fact`) and `more`; a check whose read failed is `{not_checked: why}` and is never counted as clean. Cost is independent of the station's size (11 station reads for the whole station, 8 for a subtree); each query reads at most `max_components` rows and a truncated table says `truncated at N components ... narrow the scope`, with the checks that would assert an absence from it not checked. README section "Finishing a job" and the skill's checklist describe the recipe.

### Added — n4_add_alarm creates configured alarm extensions in one batch

- `n4_add_alarm` adds one alarm extension per point (1 to 50) under one confirmation token: an
  `alarm:AlarmSourceExt` whose `offnormalAlgorithm` and `faultAlgorithm` follow the kind (`out_of_range`,
  `numeric_change_of_state`, `boolean_change_of_state`, `boolean_command_failure`, `enum_change_of_state`,
  `enum_command_failure`, `string_change_of_value`, `status`), picked from the point type unless given. It sets the
  alarm class, the offnormal, fault and normal texts, the source name, the delays, the enabled transitions (hex bit
  mask), inhibit, the hyperlink, and the algorithm's limits (enabling each limit given), deadband, limit texts or
  alarm values; for `out_of_range`, `fault_high_limit`, `fault_low_limit` and `fault_deadband` set the same slots
  on its fault algorithm, beyond which the point goes to fault with `toFaultText`. The add holds the plain slots; set ops in the same batch swap both algorithms by type and set the
  rest by path, since a station refuses a component inside an add body.
- The dry run refuses a kind that does not fit a point (naming the kinds that do), an argument the kind does not
  take, an alarm class the station does not hold (one BQL read, the refusal lists the classes) and a second
  extension of the same kind on a point unless `allow_second`. The read-back reads every slot it set and lists any
  value the station stored differently under `coerced`. The README gains the alarm extension recipe: kinds, slots,
  encodings and the text syntax. Before, an alarm took a create, two algorithm swaps and one set per slot by hand.
  Bench retro 2026-11-15, row 10.

### Changed — n4_add_history picks the extension from the point type and refuses a repeated history id

- `n4_add_history` reads each point's type from the station's type registry and creates the extension that fits it:
  a numeric, boolean, enum or string control point gets `NumericInterval`, `BooleanInterval`, `EnumInterval` or
  `StringInterval`, named after that type unless `ext_name` is given. An `ext_type` that does not fit the point, or a
  point that is not one of those, is refused at the dry run naming the type that fits. Before, a boolean point got a
  `NumericInterval` that failed only at execute.
- The dry run predicts each history id (`/<station>/<history name>`) and refuses one that repeats in the batch or that
  an extension on the station already holds, naming that extension (one BQL read of every `history:HistoryExt`).
  Before, the extension was created in fault "Duplicate history id" and reported `verified`.
- The read-back compares the history id with the predicted one and the fault cause with an empty one, so an extension
  in fault reads `mismatch`. The capacity note computes the duration from the requested interval (about 50 s at
  100 ms), where it always said "about 41 h at 5 min". Bench retro 2026-11-15, row 7.

### Added — n4_add_history change-of-value extensions and a capacity argument

- `mode=cov` adds a change-of-value extension (`history:NumericCovHistoryExt`, `BooleanCov`, `EnumCov`, `StringCov`)
  per point in the same one-token batch: it records each change of the point value, has no interval, and refuses
  `interval_ms`; `interval_ms` is now required only for the default `mode=interval`. Before, a COV extension took 3
  calls per point by hand. Bench retro 2026-11-15, row 8.
- `capacity` (records, 1 to 2147483647, or `"unlimited"`) is set by a set op on
  `<ext_name>/historyConfig/capacity` with a `history:Capacity` value `1:<records>` (or `0:0`), right after each add in
  the same batch, since `historyConfig` is a component a station refuses inside an add body. The read-back checks it
  and the note says how long it lasts at the interval (20160 records at 5 min: about 70 d). Before, capacity stayed at
  500 records, about 41.7 h at 5 min, too short for a dashboard's 7 d / 30 d views. Client-A history rollout retro,
  row 2.

### Fixed — copy, remove and rollback carry a constant block's configured out

- `n4_copy_component` and the `n4_remove_component` snapshot used to drop every Status `out`, so a copy of a
  `kitControl:NumericConst` with `out=10.0` held 0 yet read `verified`, and remove then rollback brought it
  back at 0. An `out` is now kept when the parent's type contract declares it without `r` or `t` (the
  constant blocks declare it summary-only); a computed `out` (a writable's, declared `rtso`) is still left out,
  as is any `out` whose type the station does not describe.
- The copy read-back compares the carried `out` with the source value, and the rollback read-back of a remove
  compares each carried `out` with the live slot, so a dropped value reads `mismatch`, never `verified`.
- The remove plan now reads the subtree's type definitions through the registry channel (one call).

### Fixed — mcp-n4: reads list declared properties still at their default, and a rollback to a default reads verified
- A BOX load leaves out a frozen property still at its type default, so a fresh component read `slots: []`. `n4_read_slots` now lists those properties after the loaded slots, and `n4_navigate` under each component or struct slot whose children it lists, from the type contract of the station's registry (`loadTypes`, read once per session for the listed types and their supertypes): each is marked `default: true` with the contract's default value, or `value: null` and a `note` when the contract gives no default value (a component default such as `proxyExt`); the sub-entries of a declared struct default are marked `default` too and may show `type: null`. No value is ever invented.
- `defaults_note` says when a contract could not be read and names a declared default that could not be decoded, which is left out instead of failing the read. A type the station answers as unknown, or leaves out of a reply that answered other types, is not asked for again in the session; after a failed call, or a reply that answered nothing, it is.
- The `n4_rollback` read-back counts a frozen property the load leaves out as its declared default, so restoring a default reads `verified` instead of `mismatch`; when the contract cannot be read, the read-back reports `readback_error` instead of a `mismatch`.

### Added — mcp-n4: n4_describe_session reports the server build and a stale-binary warning
- `n4_describe_session` adds a `server` block, with or without a session: `build` (the git revision, `+modified` for a dirty tree, `unknown` without a VCS stamp), `built_at` (the commit time, else the binary's mtime at start), `started_at` and `binary` (the executable's path).
- When the binary on disk was modified after the server started, the reply carries a `warning` to restart the MCP server: new tools reach a client only after a restart.

### Added — mcp-n4: licence information
- `n4_license_info` answers what the station's licence manager says in one call (one GET of the spy page `sysManagers/licenseManager`; a read tool, available without `--allow-writes`): `host_id`, `certificates` (name, expiry, valid), `licenses` (name, expiry, valid, `days_left`) with `license_kind`, `brand`, `owner` and `project`, `nearest_expiry` (the earliest dated expiry among licences, certificates and features), `expired`, and `features` (count, and each feature's name, expiry and attributes, capped by `max_features`, filtered by `feature`). Every `days_left` counts from the station's own date (`station_date`, read from its SystemService), never from the machine that runs the server. `maintenance` reports what the licence states of the software maintenance agreement (every `sma.*` attribute with its feature, grouped per feature and capped, a feature named after maintenance, and `sma_exempt_features`, the features whose `sma.exempt` is true, each once: an exemption per feature, not of the licence); the page carries no end date of the agreement, so `expires` is `null`. An unreachable page is a typed `status` (`forbidden`, `not_found`, ...), never an empty licence; a missing section and the lines that matched no known pattern are named in `note`.
- `n4_station_health` gains a `license` section, answered on its own like the others: `licenses` (name, expiry, `days_left` from the station's own date), `nearest_expiry`, `expired`, and a `warning` when the nearest expiry is 60 days away or fewer, or already past. An unreadable licence page is `{"error": "..."}` while the other sections still answer.
- New start option `--niagara-home NAME=PATH` (repeatable): the Niagara installation directory of a configured station, when it is reachable from the machine that runs the server. With it `n4_license_info` adds `generated` and `version` to each licence and, when a licence file states a `maintenanceExpiration`, `maintenance.expires` with its `days_left`, read from the attributes of the root element of `security/licenses/*.license` and nothing else of the file; a file issued for another host is not used. Without it the fields are omitted and `generated_note` says why.
- `n4_license_info` and the health `license` section after the independent verification, before any release. The reply is bounded whatever the page holds: every string taken from the page is stripped of control characters and cut with the marker `...[cut]`, a feature keeps 30 attributes, 100 certificates and 100 licences are read, every list stops at a count and at a byte budget with its whole count beside it, and a whole reply stays under 2 MiB for any page up to the read cap. A line of the page is read whole or counted as unread: two entries on one line, a name with a bracket or a brace and a section heading out of its place are never half-read. A licence that is not `valid` carries its `state` and health warns of it; an expiry that is neither `never` nor a date is returned as written with `days_left: null`, counted in `note`, and health warns of it; `days_left` counts calendar days with no longest span; the station date must be a whole timestamp. With `--niagara-home` only a regular file of exactly a licence's name is opened (never a link, never a FIFO), a file that matches no licence of the page is neither read nor named, and `generated_note` gives the reason per licence. The health error says which case it is instead of always naming permissions.

### Added — mcp-n4 finds components by tag and lists the tag dictionaries

- `n4_neql_query` runs one NEQL query (`<base_ord>|neql:<query>|view:file:ITableToCsv`) and returns the
  matching components' station ORDs, or with `bql` the columns and rows of a select run on the matches.
  `|` is refused, a station parse error is reported as HTTP 400 with the query, `max_rows` (default 200,
  at most 2000) bounds the reply.
- `n4_list_tag_dictionaries` lists the installed tag dictionaries (namespace, enabled, frozen, version,
  status) with their tag definitions, tag groups and relation ids, bounded by `max_items`.

### Fixed — mcp-n4: the commissioning report no longer flags a lone point named after its type
- `n4_commissioning_report` `palette_default_names` lists a control palette point name (`NumericWritable`, `NumericWritable1`) with or without the number the station appends, but the simple name of the point's own type only with that number (`Tstat1` of `kitControl:Tstat`): a lone `Tstat` is a meaningful name, not a forgotten default.

### Added — mcp-n4: read alarms and alarm classes
- `n4_read_alarms` returns the station's alarm records newest first, by default the open ones (unacknowledged or not back to normal): `timestamp`, `uuid`, `source`, `source_state`, `ack_state`, `alarm_class`, `priority` and `msg_text` as the station renders them. Optional filters, combined with and: `source` (a `station:|slot:/...` component, itself and below), `alarm_class`, `source_state` (`normal`, `offnormal`, `fault`, `alert`), `ack_state` (`acked`, `unacked`, `ackPending`), `since`/`until` (BAbsTime text); a state filter replaces the open default and `open_only: false` reads every record. `limit` defaults to 50, at most 1000; `truncated` says more records match, `total` is one `select count(*)` (`null` with `total_note` when it fails) and `where` echoes the clause sent. Every filter value is checked before any request, so no quote, wildcard or BQL of the caller reaches the station.
- `n4_list_alarm_classes` lists every alarm class with its priority and ack requirement per transition (`toOffnormal`, `toFault`, `toNormal`, `toAlert`) decoded, its alarm counts, and the recipients (console, email and other `alarm:AlarmRecipient` components) each class is linked to from its `alarm` topic to the recipient's `routeAlarm` action; `unresolved_links` names the links whose source is not a listed class, and at most 50 recipients are read.
- Both are read tools, available without `--allow-writes`: fixed queries over the `alarm:` ORD and at the station root. The README documents creating an alarm class with `n4_create_component`, `n4_set_slot` and `n4_create_link`.

### Added — mcp-n4: acknowledge alarms
- `n4_ack_alarm` acknowledges 1 to 100 alarm records by `uuids` (as `n4_read_alarms` returns them), with optional `notes`, through one `ackAlarms` call on the station's alarm channel; the station sets the acking user, the ack time and `ackPending`. The dry run refuses, naming them, uuids with no record, records whose source is outside the write scope (a source that is not one plain slot path, such as a list of several ORDs, is refused too) and records already acknowledged or ack pending. The read-back is `verified` when no record stayed unacked and each has a user, `mismatch` when one stayed unacked or the station names alarm classes the user may not acknowledge (`observed.permission_failed_classes`). It is destructive (`destructiveHint: true`, so `n4call` needs `--allow-destructive`): an acknowledgement cannot be undone and has no inverse.
- The write engine sends a planned `{channel, key, body}` op as one BOX call on that channel and journals it like any op; only the channel ops of its allow-list (`alarm` `ackAlarms`) are sent, and any other op, or a client that cannot make the call, is refused at the dry run, before the token or the journal (and again before sending, as defence in depth).

### Fixed — mcp-n4: read fixes from the second bench retro
- `n4_read_slots` decodes a slot whose type is `baja:Status` itself (a proxy extension's `status`, for example): `value` and `status` are its encoded bits, `status_flags` names the flags set (`fault`, `down`, `alarm`, `unackedAlarm`, ...) and `status_ok` is false when any is set. Before, such a slot read `value: null`, `status: "0"`, `status_ok: true` whatever the station showed, so a faulted point read as healthy. The reply for a Status complex (`baja:StatusNumeric` and its siblings) is unchanged.
- `n4_inventory` finds networks by type under any base: a network is a row of its `driver:DeviceNetwork` query, and a device or a point belongs to the deepest network above it. Before, the network was the first path segment below the base, so a base above the driver container collapsed every network into one entry carrying the type of the last network row and listed plain folders that hold points as untyped networks. With such a base a network is now named by its path below the base (`Plant/Drivers/BacnetNetwork`); with the driver container as base the names are unchanged. A device or point under no network has `network: null` and is counted in the totals only; a base that is itself a network lists that network.
- `n4_navigate` `types` keeps the entries of a given type and of its subtypes: `types: ["history:HistoryExt"]` now keeps every history extension, where only the concrete type spec matched before. The ancestry is the station's own, read from its type registry in one call for the types of the listing and kept for the session; a listing whose types are all given exactly makes no extra call. When the registry cannot be read the filter keeps exact types only and the reply carries `types_note` saying so.
- `mcp-n4 --help` describes the Go binary: the paragraph that told the operator to run the retired Python module (`python3 -m ...`, "Importing this module", `__main__`, `SUPPORTED_PROTOCOL_VERSIONS`) now says to run the `mcp-n4` binary and names `server.protocol_versions` of the registry.
- The session retro draft names Go files as the target of its proposed deltas for a refused action, a refused value type and a failed read-back (`go/internal/mcpn4/writeops/invoke.go · AllowedActions`, `go/internal/mcpn4/plan/values.go · ValueTypes`, `go/internal/mcpn4/plan/execute.go · read-back`), where it cited a file of the retired Python server.
- The `mcp-n4` usage line and every message the command prints on stderr name the binary (`usage: mcp-n4 [-h] ...`, `mcp-n4: error: ...`), where they named the module of the retired Python server; the usage is wrapped for the shorter name.

### Added — mcp-n4: read history records
- `n4_read_history` returns the last records of one history, newest first, so a rollout proves records arrive and not only that the history extension runs. Name the history by `history_id` (`/<station>/<history name>`) or by `ext_ord`, the extension's `station:|slot:/...` ORD (resolved to its id with one BQL select); give exactly one. `limit` defaults to 20, at most 1000. An optional range is `period` (`today`, `last24Hours`, `yesterday`, `weekToDate`, `lastWeek`, `last7Days`, `monthToDate`, `lastMonth`, `yearToDate`, `lastYear`) or `start`/`end` (BAbsTime text such as `2026-11-15T00:00:00.000-06:00`), not both.
- The reply carries `history_id`, `columns` (the station's headers), `rows` (`timestamp`, `value`, `status` as the station renders them), `row_count`, `order: newest_first`, `truncated` (the range holds more records than returned), `total_records` (one `select count(*)`; `null` with `total_records_note` when it fails) and `elapsed_ms`.
- It is a read tool, available without `--allow-writes`: fixed queries over a `history:` ORD, never a caller's query, no file output. A history id that is not two slot-name segments (`|`, `?`, `;`, spaces and controls included) is refused before any request; a history the station cannot read is an error naming the id and how to list the ids with `n4_bql_query`. `n4_bql_query` still refuses a `history:` base with its unchanged message.

### Added — mcp-n4: n4_read_schedule decodes a schedule of any kind
- `n4_read_schedule(ord)` reads one schedule in one load and answers plain JSON: for boolean, numeric, enum and string schedules `out`, `next_time`, `next_value`, `default_output`, `in`, `out_source`, the effective period, the weekly ranges per day (`HH:MM`, sorted, `24:00` for midnight) and the special events in priority order with their kind and rule (date, date_range, week_and_day, custom, reference); a calendar's dates; a trigger's times, dates and next trigger.
- Slots the station leaves out because they hold their type default are filled from the station's type registry; a slot that is neither sent nor described is refused by name instead of guessed. A range stored without a value is reported as ignored, as the station treats it.

### Added — n4_read_tags: direct tags, tag groups, relations and implied tags in one call

- `n4_read_tags(ord | ords, implied=true, relations="direct")` reads up to 50 components: the direct tags (metadata
  slots, the id unescaped from the slot name: `hs$3apoint` is `hs:point`), the tag groups (`n:tagGroup` relations),
  the direct relations and, through the session handler's `impliedTags` call (the one BajaScript makes), the
  implied tags with their types. `relations="all"` adds the implied relations; `implied=false` makes no session call.
- A client that cannot call the session handler answers `implied: null` with `implied_unavailable`, never an empty
  list. The kit README gains a Tags section (how a tag, a tag group and an implied tag are stored).

### Changed — n4_remove_component scans for outgoing links by default

- `n4_remove_component` always scans for links from the removed subtree into components outside it:
  by default the `--write-scope` prefix that holds `parent_ord`, to depth 6; `link_scan_ord` now
  narrows that root instead of enabling the scan, and refuses when it is not equal to or under
  that prefix. A root holding more than 100000 nodes to depth 6 refuses as too large to scan and
  asks for a narrower `link_scan_ord`. A link whose target is inside the removed component is
  never counted as outgoing. The note names the scanned root and says that links stored outside
  it are not detected.
- When the scan finds links, the dry run refuses and lists each one (`target (target_slot <-
  source_slot)`) until the call repeats with the new argument `acknowledge_links: true`; a remove
  that executed before now needs that acknowledgement. A subtree with no such links removes as
  before.

### Fixed — n4_remove_component on a simple slot plans an inverse that re-adds that slot

- Removing a simple or struct slot (a direct tag such as `tst$3acap`, a relation `r`) planned an inverse that
  re-added the *parent component* under the slot's name: the station answers a simple slot's ORD with the component
  that holds it, and the snapshot was taken from that load. A rollback would have created a copy of the parent.
- When the child's load comes back with the parent's handle, the snapshot now comes from the parent's load: the
  inverse re-adds that slot with its type, value and flags (`{"nm":"a",…,"b":{"nm":"p","t":"baja:Double","v":"4.5"},
  "f":"m"}`), so a rolled-back tag keeps its metadata flag. Components are planned as before.
- The links of a removed simple slot are checked like a component's: a link it sources into another component
  refuses the plan until `acknowledge_links: true`, is listed in `outgoing_links_broken` and is re-created by
  `n4_rollback` (only the links read from that slot: the holder's other links never refuse a tag remove); a link of
  the holder that writes the slot is recorded as an `unlinked_input` with the slot's last value (`prior`), which the rollback reports. A struct slot (a
  `StatusNumeric`) is snapshotted with its fields whatever depth the parent was loaded at.
- `n4_rollback` accepts the escaped slot names such a remove records (`tst$3acap`) in an outgoing link or an unlinked
  input, and its read-back of a re-created outgoing link now finds a `baja:ConversionLink` (a `Double` into a
  `StatusNumeric` input), which it reported missing.

### Fixed — n4_rollback of a create no longer silently removes later work

- The rollback of an `n4_create_component` batch is refused when a later batch (not itself rolled back) recorded
  a target equal to or under the created component, or when the component now holds a link or a child component
  its create did not make (a Workbench edit no journal records; a frozen slot its type declares does not count).
  Before, the plan was one remove op and the verdict `verified`: on the bench, rolling back the create of a folder
  removed it with 10 blocks, their history extensions and 6 links later batches had added.
- The refusal names the created ORD, each later batch as `batch_id (tool)` newest first and each extra child or
  link, and says to roll the later batches back first (newest first) or remove that work by hand. Nothing is
  removed and no rollback batch is journaled. Rollbacks of other batches are unchanged. Bench retro 2026-11-15 Δ1.

### Fixed — n4_rollback restores the outgoing links of a removed component

- An acknowledged `n4_remove_component` records in its inverse each link that leaves the removed
  component (stored on a target outside it) as an `outgoing_link`; `n4_rollback` re-creates those
  links after the component, through the same link writer and journaled intent as the links inside
  it, and refuses when an end is outside today's write scope.
- The rollback read-back looks for each re-created link on its target and reports
  `outgoing_links: {restored, missing}`; a missing one makes the verdict `mismatch`. Before, a
  remove then rollback of a constant block lost its links into other blocks and read `verified`.
- A link stored at the remove snapshot's last level now keeps its slot names: its plan note names
  them (it printed `(None <- None)`), `snapshot_truncated` no longer lists the link's own slots,
  and such an internal link is recorded as a `relink` that `n4_rollback` restores, where the
  rollback refused it as a malformed relink before.

### Fixed — mcp-n4: the rollback change guard compares StatusEnum and StatusString values
- `n4_rollback` restores a Status slot only while it still holds what the batch wrote. That guard read every Status as a boolean or a number, so a batch that set a `baja:StatusEnum` or `baja:StatusString` slot (an enum or string schedule's `defaultOutput`) was always refused as "changed since". It now compares their value text and status: an enum by its ordinal (a DynamicEnum loads as `ordinal@range`), an omitted value child as its type default, a slot of another type as changed.
- StatusBoolean and StatusNumeric keep the same guard. `n4_set_slot` and `n4_invoke_action` behave as before: neither writes those two types (their value types are StatusNumeric and StatusBoolean only), so the fix reaches only batches that do, such as `n4_set_schedule`.
- The rollback's read-back now checks a restored StatusEnum or StatusString the same way. Before, it raised "is not a StatusBoolean/StatusNumeric", so such a rollback executed but ended with verdict `failed`.
- A schedule day (a `schedule:DaySchedule` set by `n4_set_schedule`) is compared by its time ranges in both the change guard and the read-back: a day edited after the batch is refused as "changed since", and a day the station did not restore is no longer reported `verified`. Every other set keeps the plain-value comparison.

### Added — mcp-n4: n4_list_schedule_kinds says which schedules exist and how each works
- `n4_list_schedule_kinds` (no station, no session) lists the six schedule kinds an integrator creates (boolean, numeric, enum, string, calendar, trigger) with their station type, value type, default output, parts, purpose, behaviour and usual link, and the five special-event kinds (date, date_range, week_and_day, custom, reference) with their fields and allowed values, plus the priority, cleanup and export/import rules.
- The kit README gains a "Schedules / Kinds of schedules" section: how many kinds there are, what each is for, how `out` is chosen, and how a station stores schedules, with the facts confirmed live on a 4.15.6.20 station.

### Added — mcp-n4: a schedule that drives a point
- The README's schedule recipe creates the schedule, sets it with `n4_set_schedule` and links its `out` to the point's `in16` (the help's convention; any level is allowed), proved live: a special event for today drove a BooleanWritable to true @ 16 and removing it put the point back.

### Changed — mcp-n4 write scope: create the scope root, scope a link on its target

- `n4_create_component` may create the `--write-scope` root itself under its parent, so a scope can
  name a folder that does not exist yet; it is refused, naming the root, when that root already
  exists, because the station would rename the new component outside the scope. Retro
  niagara-research client A 2026-11-13 M5.
- `n4_create_link` checks the write scope on `target_ord` only (the link is stored on the target; the
  source is only read). The source must still be a plain slot ORD. A refusal now names the end:
  `target_ord: ord '…' is outside the write scope (…)` or `source_ord: ord '…' is not a plain station
  ORD (…)`. Before, a source outside the scope refused the link. Retro M10.

### Added — n4_set_flags add and remove

- `n4_set_flags` takes `add` and/or `remove` (flag letters) instead of `flags`: the letters change in
  the slot's current set and the others are kept, so making a slot a summary slot (`add: "s"`) no
  longer drops a readonly `r` the caller did not restate. One form per call; a letter in both `add`
  and `remove`, an unknown or repeated letter, or an empty edit is refused before the station is
  read. The plan note still shows the current and the resulting set, and the inverse stays the exact
  previous string. `flags` is no longer a required argument; the whole-set form is unchanged.

### Fixed — mcp-n4: n4_set_schedule and n4_rollback compare event and date rules
- A wrong-type or null value in a special event's ranges is named as a fault instead of an internal error, with every other fault kept.
- The write read-back compares each special event's rule, not only its name and ranges.
- `n4_rollback` compares the specialEvents list (names, order, rules, ranges) and each calendar date part (`yearSchedule`, `monthSchedule`, ...) by content: a list changed since the batch is not overwritten, and a list or date the station did not restore reads as a mismatch.
- `remove_special_event` and `remove_date` refuse a name given twice; a `reference` event must name an existing `schedule:CalendarSchedule`.

### Added — mcp-n4: n4_set_schedule changes calendar dates and trigger times
- `add_date` and `remove_date` change the dates of a calendar or of a trigger (a remove each; an add of the bare date type then a set of each part by path, so the rollback re-adds a removed date exactly); `times` replaces a trigger's firing times. Each is checked first and read back; replacing a date under the same name takes two calls.

### Added — mcp-n4: n4_set_schedule sets the default output
- `n4_set_schedule(ord, default_output)` sets the value `out` takes when nothing is active, of the schedule's kind (a calendar's is a boolean) or `null`; it is checked with the weekly request, rolled back exactly (a default output the station left out comes back as its declared default) and read back. A trigger schedule has none and is refused by name.

### Added — mcp-n4: n4_set_schedule adds and removes special events
- `add_special_event` ([{name, kind, rule, ranges}], every kind of `n4_list_schedule_kinds`) and `remove_special_event` ([names]) change the special events in one set of the whole list, keeping the order (the priority); new events go last. A date or date range that ends before today is refused unless `allow_expired: true`. The read-back compares each event's name and ranges; `n4_rollback` restores the list exactly.

### Added — mcp-n4: n4_set_schedule writes the weekly schedule
- `n4_set_schedule(ord, weekly)` replaces only the days given (`{"sunday": []}` clears one), names every fault of the request before anything is sent, and reads the schedule back to compare each day written. Each day is one set whose inverse restores it exactly through `n4_rollback`.
- Every overlapping pair of ranges in a day is named (a long range against each range inside it), and a schedule that is not there is named with the request faults that need no station.

### Added — n4_set_tags applies tag groups and relations

- `add_groups` / `remove_groups` apply or remove a tag group an installed dictionary defines (an `n:tagGroup`
  relation to the group's definition, flag `r`); the read-back lists the target's implied tags and any group
  tag missing from them.
- `add_relations` / `remove_relations` add or remove an outbound relation to another component; the inverse of
  an added relation removes the slot the station named.

### Fixed — n4_set_tags refuses duplicates and contradictions and restores relations as stored

- A tag, tag group or relation named twice in one list, or both added and removed, is refused at plan; so is
  adding a group or relation the target already has (each used to plan two slots or an in-doubt remove).
- Only simple value types are written as tags (Marker, Boolean, Double, Float, Integer, Long, String,
  DynamicEnum, Format, RelTime), each value checked for its type: `baja:Folder` or `nonsense` is refused.
- The rollback of a removed relation or tag group re-adds the fields the slot stored (the endpoint ORD as it
  was written), not the request's form.
- `n4_set_tags` is in the destructive class (`destructiveHint: true`): it removes tags and relations.

### Added — n4_set_tags writes direct tags checked against the installed dictionaries

- `n4_set_tags` adds, changes and removes direct tags (metadata slots named `ns$3aname`) on 1 to 50
  components in one batch of at most 200 ops, with the usual dry run, confirmation token, write scope,
  read-back and `n4_rollback`.
- A defined tag takes its definition's value type and default; an undefined one needs `allow_ad_hoc` and a
  namespace no dictionary installs (an installed namespace is refused, as the Tagging guide advises); a
  type change is refused. The read-back lists values the station normalized under `coerced`.

### Fixed — mcp-n4: n4_spy_page reads the registry's type pages
- `n4_spy_page` accepts one `module:Type` segment directly under `sysManagers/registryManager/types/` (e.g. `sysManagers/registryManager/types/control:NumericWritable`), the registry's page of that type; a `:` anywhere else (a scheme, an ORD, a segment after the type spec, a `..` trick) is still refused before any request.

### Added — mcp-n4: station health and spy pages
- `n4_station_health` answers the station's state in one call, each section on its own: `system` (the SystemService slots: CPU usage, memory as the station reports it plus `memory_used_percent`, version, host, time), `points`, `networks` and `devices` (`total`, `ok`, `null_only`, a count per status flag in `by_flag`, the not-ok entries named up to `max_not_ok` with a `truncated` marker; the status is read as `status.bits`, which no locale translates), `totals` (the `n4_inventory` totals of the same rows), `capacity` (the Global Capacity counts and licence limits and the recount times of the spy metrics page) and `elapsed_ms`. A section that fails carries `error` while the others answer. `capacity` is `null` with a `capacity_note` when the spy metrics page is unreachable or holds no Global Capacity table, never a guessed count; a status the station did not answer counts as `unknown`, never as ok. A status whose only flag is `null` (a point with no value) is not a fault: it is counted in `null_only` and not listed, so `ok + null_only + not_ok_count = total`. It is a read tool, available without `--allow-writes`.
- `n4_spy_page` reads one diagnostics page of the station's spy tree (`GET /ord?spy:/<path>`, the pages of the Workbench Spy view) and returns it as data: `title`, `tables` (title, header, rows of cells), `links` to the child pages as paths to pass back, the `actions` the page offers (named, never invoked; their form fields are never returned) and the `text` outside the tables. It is a read tool, available without `--allow-writes`, and it only issues GET requests under the spy root: a scheme or `:`, `|`, `..`, a query string, `%`, `;`, a backslash, a control character and the known action pages (`util/gc`, `util/forceThreadDump`) are refused before any request. `status` is `ok`, `forbidden`, `not_found` or `not_html`; `truncated_by` names the caps that cut a reply. Spy pages are diagnostics whose layout differs between Niagara versions (measured on 4.15).
- `n4_spy_page` hardening from the independent verification, before any release: the content of a form field (`textarea`, `select`/`option`, `button`) is never returned, also when the field sits in a table cell; a form action is reported only as a clean spy path, without its query or fragment, once, up to 100; the page reader is linear in the page size and gives up with `status: parse_timeout` after `timeout_s`; the reply is bounded as a whole (200 tables, 100 cells per row, 200 characters per link name or title, each with its `truncated_by` marker); the path rule refuses every Unicode control character and invalid UTF-8.
- `n4_station_health` after the independent verification and a laboratory measurement: `points` adds `proxy`, the points whose proxy extension is not `control:NullProxyExt` (a licence counts proxy points only); each capacity resource adds `live`, what the station really holds (networks, devices, proxy points; `null` otherwise), beside the licence tally `used`, and `capacity` adds `may_be_inflated` with an `inflation_note` when a `used` is above its `live` or the recount has never run, because the tally only grows until a recount or a station restart. The capacity is never silently partial: the metrics page is read with a row cap above what its byte cap can hold, `capacity_truncated` and `capacity_note` say when the page was cut, and rows that do not match the table's header are counted in the note. `totals` carries `rows_truncated` when a query feeding it was cut.

### Fixed — mcp-n4: explicit station refusals are journaled as refused, and dry runs pre-check creates and links
- A write the station answers with its own error reply on the first op (`Illegal parent`, `Slot ... already exists`, `Could not find Source Slot`, `Duplicate link already exists`) is read back; when nothing was applied, the batch is journaled with state `refused` (`verdict: "refused"`, `refused: true`, `station_error`, an empty inverse) instead of `in-doubt`, the reply says the station refused it and nothing changed, and the audit outcome is `refused`. `n4_create_component` and `n4_create_link` read back their targets for it; a transport failure, a timeout, a read-back that fails or shows a change, and every other tool stay `in-doubt`.
- `n4_list_batches` shows the `refused` state, `n4_rollback` refuses a refused batch with "nothing to roll back" and does not count a refused rollback as an earlier one, and the session retro draft lists refused batches as their own candidate.
- Station error text is decoded from HTML entities (`&quot;`, `&lt;`, `&amp;`) wherever a tool reports it.
- The `n4_create_component` dry run refuses an unknown `type` or `proxy_ext` by name (the station's type registry does not describe it) instead of the station's generic "Unable to process request". The `n4_create_link` dry run refuses a source or target slot that neither the component nor its described type contract has, and a link identical to one the target already has.
- A create whose name is already taken under the parent is not refused (the station renames
  on collision and the read-back and inverse follow it), but its dry run says so in a first
  note; the note is part of the hashed plan, so a name taken after the dry run, or a re-used
  create token, is refused at execute as a plan mismatch instead of renamed silently.

### Changed — n4call refuses destructive writes without --allow-destructive

- `n4call` refuses, with exit 2 and before the server starts, a calls file whose write calls a
  destructive tool (`n4_remove_component`, `n4_rollback`, `n4_save_station`, taken from the
  server's `destructiveHint`) unless `--allow-destructive` or `--dry-run-only` is given; the
  refusal names the line and the tool. A run with writes says on stderr whether they execute.
  A wrapper that dropped `--dry-run-only` could otherwise execute a rollback planned as a dry run.

### Fixed — ng-deploy: the copy is all-or-nothing
- Every jar of the mode is copied to `<jar>.ng-deploy-new` beside its target; then the jars the station holds are moved aside to `<jar>.ng-deploy-old`, the staged jars are renamed in, and the ones aside are removed. Each step is a rename within the modules directory, so a jar is replaced whole or not at all. Before: the rt jar was copied, then the ux jar, and a failure on the second left one new jar and one old with `copy failed for <ux jar>` as the only word.
- A step that fails is undone from the jars moved aside, with or without `--no-backup` (the backup of the run is not read): every jar aside is moved back, bytes and mode, and a new jar the station did not have is removed. Exit 40 then reads `copy failed for <jar>[: <reason>]; nothing was replaced: the station holds the jars it held before the deploy` (a jar that could not be staged or moved aside, or the first rename in), or `...; what was already replaced was put back: the station holds the jars it held before the deploy` (a later rename in).
- The one way left to a mixed pair is a jar that cannot be moved back or removed again. Exit 40 then says what each jar is and names the file that still holds the previous bytes, which is left in place: `...; the station does NOT hold the jars it held before the deploy: <jar> is the NEW jar and the previous one is in <jar>.ng-deploy-old (<reason>), <jar> is the previous jar; put each named .ng-deploy-old file back under its jar name before the station restarts` (also `<jar> is MISSING and the previous one is in ...`). When the stuck item is a new jar the station did not have, the line is `...: <jar>, which the station did not have, could not be removed (<reason>): remove it before the station restarts`, without the sentence about `.ng-deploy-old` files.
- **ng-deploy never deletes a `.ng-deploy-old` file it did not create, and refuses to deploy while one is in the station.** Such a file was left by a deploy that did not finish (killed, or a rename back that failed) and may be the only copy of the previous module, beside its jar or without it. Before anything is copied, and for the `-rt` and the `-ux` jar whatever the mode, the deploy changes nothing and is exit 40: `[ng-deploy] ERROR copy refused: a deploy that did not finish left previous jars aside in <modules dir>; nothing was copied or changed`, then one line per jar (`<jar> present, <jar>.ng-deploy-old present: the jar may be the new one of that deploy; the .ng-deploy-old file is the previous jar`, or `<jar> missing, <jar>.ng-deploy-old present: the .ng-deploy-old file is the previous jar`) and `to put the previous jars back, rename each .ng-deploy-old file over its jar; to keep what is there, delete the .ng-deploy-old files; then deploy again`. A stale `.ng-deploy-new` is written over.
- A finished copy that cannot remove a jar aside still succeeds (exit 0), with `[ng-deploy] WARN the jars are copied, but the previous jar <jar>.ng-deploy-old could not be removed (<reason>): delete it, the next deploy refuses while it is in the station`. Otherwise no `.ng-deploy-new` or `.ng-deploy-old` file is left after a deploy or after a failure that was undone.
- Something that is not a jar file under the name of a jar (a directory, a symbolic link to nothing or to a directory) is exit 40 before anything is staged or moved: `copy failed for <jar>: the station holds a directory under that name, not a jar file; nothing was replaced: ...`. `cp` refused these too; the staged copy never moves or removes them.
- A SIGINT or SIGTERM during the copy is honoured between two steps: the copy is undone as after a failure, the staged jars are removed, stderr gets `[ng-deploy] ERROR copy interrupted (SIGINT|SIGTERM); <the same outcomes>`, and the run exits 130 or 143. Outside the copy the signals end the run as before. A SIGKILL cannot be undone: the next deploy finds the `.ng-deploy-old` files and refuses as above.
- Consequences of renaming instead of writing into the existing jar: the modules directory itself must be writable (a read-only directory holding writable jars used to deploy; it is now exit 40 with nothing replaced), a read-only jar in a writable directory is now replaced, and a jar that was a symbolic link to a jar file is replaced by a regular file (the file it led to is left as it was). A replaced jar keeps the mode the station's jar had.

### Fixed — ng-deploy: a record that cannot be written stops the deploy before the copy
- After the checks and before the copy, a `.last-deploy-sha` that is a directory or cannot be opened for writing, or a working directory that takes no new file when there is no record yet, is exit 10 `the deployed commit cannot be recorded in .last-deploy-sha (cannot write <path>: <reason>); nothing was copied (fix the file, or SLOTOMATIC_DETECTION=off makes the record optional)`. Before: the jars were copied first and the same condition was exit 41. The look opens the record exactly as the write after the copy does, so a record that is a symbolic link is judged through the link: a link to a file not there yet deploys as before, a link into a missing directory is exit 10 (`No such file or directory`). The look writes nothing: an existing record is only opened, and a file it had to create, at the end of a link too, is removed again.
- Exit 41 stays for what can only be known after the copy: `git rev-parse HEAD` failing or printing no full commit id, and a record that could be written before the copy and no longer can. With `SLOTOMATIC_DETECTION=off` nothing is looked at and the record stays best effort, as before.

### Fixed — ng-deploy: a recorded commit git does not know is no longer a silent fallback
- A `.last-deploy-sha` that holds a full object name `git cat-file -e <id>^{commit}` does not accept (a commit of another clone, a history rewritten since) gets `[ng-deploy] NOTICE: .last-deploy-sha names a commit git does not know; annotation changes are looked for since HEAD~1 only` on stderr, and in strict mode (`--strict-slotomatic` or `SLOTOMATIC_DETECTION=strict`, without `--with-slotomatic`) the deploy stops with exit 15 `annotation change detection could not run; ...`, as it does for a malformed record. Before: the diff started at `HEAD~1` without a word and strict mode passed. A change the `HEAD~1` diff does show is still reported as a change.

### Changed — ng-deploy: a flag wins over the env file, and NO_GATE=1 / NO_BACKUP=1 are refused in the env file
- An option a flag set on the command line is no longer replaced by the option variable of the same name in the env file: `--mode C` with `MODE=B` in the file deploys mode C, `--keep 1` with `KEEP_N=4` keeps one backup, `--full-backup` with `FULL_BACKUP=0` backs the whole directory up, and a switch flag (`--no-deploy`, `--with-slotomatic`, `--strict-slotomatic`, `--first-deploy`, `--accept-lossy`, `--no-gate`, `--no-backup`) is not undone by a `0` in the file. Before: the file was read after the flags and silently replaced them. Without the flag the env file still sets the option, as before, and the commands the deploy runs still see the value the deploy settled on.
- `NO_GATE=1` or `NO_BACKUP=1` in the env file (any value bash read as 1) is exit 10 before anything runs: `[ng-deploy] ERROR env file <path>: line <N>: NO_GATE would switch the verify-module gate off, which an env file may not do: give --no-gate on the command line` (`NO_BACKUP` ... `the backup` ... `--no-backup`). With the flag given the key is accepted. Before: the key switched the gate or the backup off without a word. **An env file that carries either key must drop it and the deploy command must pass the flag.** `0`, an empty value and any other number leave the step on and are read as before. Only these two keys are refused: `MODE=B` (a mode without the gate) and `VERIFY_MODULE_BIN` in the env file still take effect, as before.
- The flags and their option variables are one table (`go/internal/ngdeploy/optiontable.go`).

### Changed — ng-deploy: NIAGARA_HOME must be a mirror, and a deploy that stops leaves its modules directory as it was

- **Behaviour change.** gradle's `:jar` step installs the built jars into `<NIAGARA_HOME>/modules` before any check of the deploy has run, and in a real installation that directory is a station's. `scripts/ng-deploy.sh` now refuses a `NIAGARA_HOME` that is a directory without `.niagara-mirror` (the marker `toolbelt/mirror-niagara-home.sh` leaves, the one `toolbelt/build.sh` reads), before the backup and before gradle runs, slotomatic and `--no-deploy` included: exit 10, `[ng-deploy] ERROR NIAGARA_HOME <path> is a real niagara_home (no .niagara-mirror): gradle's :jar step would replace the jars of its modules/ before any check runs; build against a mirror (toolbelt/mirror-niagara-home.sh <niagara_home> <mirror_dir>), or pass --install-into-home`. **A deploy configured with a real installation as `NIAGARA_HOME` must point it at a mirror or add `--install-into-home`** (new option, the name `build.sh` uses). `INSTALL_INTO_HOME=1` in the env file is refused without the flag, as `NO_GATE=1` and `NO_BACKUP=1` are. A `NIAGARA_HOME` that names no directory is left to gradle, as before.
- In a mirror, or with the opt-in, the module's own `-rt`/`-ux`/`-wb` jars in `<NIAGARA_HOME>/modules` are moved aside (`<jar>.ng-deploy-old`, the rename the copy uses for the station) just before the build. When the build, a precheck, verify, the gate or the copy fails, they are moved back and the jars the build added are removed, and stderr gets `[ng-deploy] NOTICE: the jars the build installed in <dir> were taken out again: it holds the module jars it held before the build`. Before: the jars of a refused deploy stayed there. The exit code and the ERROR line of the failed step are unchanged. A jar that cannot be moved back or removed is one more ERROR line in the copy's jar-by-jar form (`<dir> does NOT hold the jars it held before the build: <jar> is the NEW jar and the previous one is in <jar>.ng-deploy-old (<reason>), ...`). After a deploy that went through, and after `--no-deploy`, the built jars stay there as before; a jar aside that cannot be removed then, or moved back when gradle did not install that jar again, is a WARN and the exit code stays 0.
- A SIGINT or SIGTERM between that move aside and the copy no longer ends the process on the spot: the step in progress finishes (a build the signal reached stops by itself), `<NIAGARA_HOME>/modules` is put back, stderr gets `[ng-deploy] ERROR interrupted (SIGINT|SIGTERM) before the copy; nothing was copied to the station`, and the run exits 130 or 143 as before. A SIGKILL cannot be undone: the next deploy finds the `.ng-deploy-old` files there and refuses before gradle runs (exit 10, `build refused: a deploy that did not finish left previous jars aside in <dir>; nothing was built or changed`, then the per-jar lines and the two ways out of the copy's refusal); it never deletes one.
- New exit 10 before the build: a jar of the module in `<NIAGARA_HOME>/modules` that cannot be moved aside (`build not started: <jar> could not be moved aside in <dir> (<reason>); nothing was replaced: ...`).
- When `STATION_MODULES_DIR` is `<NIAGARA_HOME>/modules` itself (symbolic links resolved) the deploy says `STATION_MODULES_DIR is <NIAGARA_HOME>/modules (<dir>): the build installs the jars there, so the copy step only verifies them`. The version-bump precheck compares the build with the jar that was there before the build; before, it compared the build with the copy gradle had just installed and never fired. The copy step copies nothing and prints `copy: nothing to copy, the build installed <jars> in <dir> (the same bytes as build/libs)` before `copy ok`; a jar of the mode the build did not leave there with those bytes is exit 40 `copy failed for <jar>: STATION_MODULES_DIR is <NIAGARA_HOME>/modules and the jar the build left there is missing or is not the one in build/libs`, and the previous jars are put back. A `.ng-deploy-old` file left in that directory is now refused before the build (exit 10) instead of at the copy (exit 40).
- `go/internal/modjar` (new) holds what build-module and ng-deploy share: the mirror marker with `IsMirror`, and the expected-signer resolver that was in `go/internal/buildmodule` (moved, `build.sh` unchanged).

### Changed — ng-deploy: the gate is told the signer build.sh expects
- **Behaviour change.** `scripts/ng-deploy.sh` runs the verify-module gate with `--expected-signer <signer>`, resolved by the code `toolbelt/build.sh` uses (`go/internal/modjar`, `ExpectedSigner`): the `niagaraSigningAlias` gradle property (`ORG_GRADLE_PROJECT_niagaraSigningAlias` in the environment or the env file, else `GRADLE_USER_HOME/gradle.properties`, else `gradle.properties` beside `gradlew`), else the one signer of the module's jars of the mode that were in `<NIAGARA_HOME>/modules` before the build, never the default development signer `NIAGARA4`. A jar signed with another key now fails the gate and the deploy with exit 50; before, the gate passed any signature. The gate call is `verify-module.sh --src <module dir> --expected-signer <signer> <jars>`: a replacement gate named by `VERIFY_MODULE_BIN` receives the two new arguments.
- With no signer to expect the gate is called as before and the deploy prints, after the gate's rows, the warning `build.sh` prints, with the deploy's own remedy: `[ng-deploy] WARN no niagaraSigningAlias is configured and no deployed jar gives a signer, so the gate accepts any signature: ... (set it: ORG_GRADLE_PROJECT_niagaraSigningAlias=<alias> in the env file, or niagaraSigningAlias=<alias> in gradle.properties)`, or `... the deployed jar itself carries the signing plugin's default development signer (NIAGARA4), ...`. The exit code is unchanged. `--no-gate` and mode B run no gate and print neither.

### Fixed — ng-deploy: a rebuild of unchanged sources is no longer refused as "bytes changed" (exit 51)
- **Behaviour change.** Under the same vendorVersion the version-bump precheck of `scripts/ng-deploy.sh` now compares jar content instead of whole files: the set of entry names and the bytes of every entry, except the signature files directly under `META-INF/` (`*.SF`, `*.RSA`, `*.DSA`, `*.EC`, any case), the `*-Digest` attributes of the manifest's per-entry sections, and the values of the `buildMillis` and `buildHost` attributes of the root element of `META-INF/module.xml` (cut out of its start tag wherever they stand; every other byte of the descriptor is compared as it is). Two builds of the same sources differ in exactly those, so a second deploy with no source change goes on with `[ng-deploy] version-bump ok (<jar>: vendorVersion still <v>, and the same content as the deployed jar apart from the build time and host, the manifest digests and the signature files)`. Before: exit 51 `bytes changed but vendorVersion is still <v>` on every retry.
- A real change under the same version is still exit 51 with the same line: one changed class, an added or removed entry (a directory entry too), another attribute of `module.xml`, `buildMillis` or `buildHost` on a nested element, the manifest's main section, a per-entry manifest attribute that is not a digest.
- New exit 52 case: under the same vendorVersion, a jar whose content cannot be read (not a zip, no entry, two entries under one name, an entry that fails its CRC) is `[ng-deploy] ERROR version-bump precheck cannot verify <jar>: cannot compare its content with the deployed jar: <path> is not a readable jar (<reason>)` (or `<path> holds no entry`, `<path> holds two entries named <name>`, `<path>: <name> cannot be read (<reason>)`, `<path>: META-INF/module.xml cannot be read as a descriptor (<reason>)`). The comparison reads entries in 64 KiB pieces and never whole, and refuses to read without end, also exit 52: `<path> declares more than 2147483648 bytes of content, the most the precheck reads` (2 GiB, all entries of one jar) and `<path>: <name> is over 8388608 bytes` (8 MiB, for `MANIFEST.MF` and `module.xml`). Before, such a pair was exit 51. A higher, lower, unreadable or incomparable vendorVersion is judged as before, without reading the content.

### Fixed — ng-deploy: the backup step names an archive only when it writes one
- For a module with no `-rt`/`-ux`/`-wb` jar in `STATION_MODULES_DIR` (a first deploy) the backup step prints only `[ng-deploy] backup: no existing <MODULE_NAME>-{rt,ux,wb}.jar in <dir>; nothing to archive` and no longer creates an empty `_backups/` directory. Before: `backup: _backups/<MODULE_NAME>-pre-<stamp>.tar.gz` came first, naming an archive that was never written. A backup that is written prints `backup: <archive>` and `backup ok: <archive>` as before.
- `--help` has a `Backups:` section: where the archives are kept (`_backups/` under the working directory), their names, and the `tar` command that restores a lightweight or a `--full-backup` archive.

### Added — ng-deploy: a deploy that went through ends with what to do next
- After `station restart required (Java classes deployed)` or `browser hard-reload only (no station restart needed)`, a deploy that exits 0 prints on stdout: `[ng-deploy] done: <jars> copied to <dir>`; `[ng-deploy] signer: <jar> signed by <NAME>[ and <NAME>] | <jar> carries no complete signature, ...; the gate checks who signed a jar, not whether the station trusts that signer: the station decides when it loads the module`; in modes A and C `[ng-deploy] next: restart the station; ng-deploy did not look at whether one is running, and stops or starts none` and `[ng-deploy] next: after the restart, check the station console for the module: <kit>/toolbelt/triage-console.sh --package <the Java package of the module> --module-prefix <MODULE_NAME> <station-dir>/console*.txt (exit 1: rows to look at)`; and `[ng-deploy] undo: to remove the module again, take its components out of the station, stop the station and delete <jar paths>`, followed by ` (for the version before this one, restore the backup this run wrote: --help, Backups)` only when the run wrote a backup archive. The lines above them, the exit codes and stderr are unchanged; a script that read the last line of stdout as the restart reminder now finds it five lines up (three in mode B). `--no-deploy`, a failed deploy and exit 41 print none of them.
- The lines are one table (`nextSteps` in `go/internal/ngdeploy/nextsteps.go`). `BUILD-LOOP.md` § 6 gains the short path for the first deploy of a new module, a removal step, and the statement that the deploy does not look at whether a station is running.

### Fixed — ng-deploy: two runs on one modules directory no longer lose the module's jars
- **Behaviour change.** A run holds a lock file, `.ng-deploy.lock`, in `<NIAGARA_HOME>/modules` and then in `STATION_MODULES_DIR` (when that is another directory and the run deploys), from before it looks at either directory until it has ended. The file holds `pid <pid>`, `host <host>`, `started <UTC time>`. A second run is refused with exit 10: `[ng-deploy] ERROR another ng-deploy run holds <file> (pid <pid> on host <host>, started <time>, <age> ago): two runs on one modules directory lose jars. A lock is never removed for the run that made it: when no process <pid> runs on <host> (ps -p <pid>), that run was killed; delete the file and deploy again`. Before: of two failing runs started together, each put back or removed what the other had moved, and `<NIAGARA_HOME>/modules` could end with none of the module's jars while both runs said it held what it held before.
- A run removes its own locks on every way out: success, any failure, SIGINT and SIGTERM. From its first lock on it catches both signals and ends with 130 or 143 (before, a signal during the backup or slotomatic ended the process at once). **A lock a killed run left (SIGKILL, power loss) is never removed automatically:** every deploy refuses until the file is deleted by hand, after `ps -p <pid>` on the named host. A lock that cannot be made for another reason is exit 10 `cannot take the run lock <file> (<reason>); nothing was built or changed`; a directory the run is not allowed to write gets no lock. The lock is an exclusive file creation, not `flock`, so it holds on a Windows mount.
- The move aside (in the home before the build, in the station during the copy) never replaces a file that is already aside: it makes the second name with a hard link and then removes the first; where the file system makes no links it renames after looking. When a deploy stops, a jar `<NIAGARA_HOME>/modules` did not have is removed only when its bytes are those of the jar this run built; any other is left, with `[ng-deploy] WARN <jar> in <dir> was not there before the build and is not the jar this run built (build/libs): it is left where it is`.
- The NOTICE of a stopped deploy says `the jars the build installed in <dir> were taken out again` only when the build installed any; otherwise `NOTICE: the build installed no jar in <dir>; the module jars moved aside for it were moved back: it holds the module jars it held before the build`.

### Fixed — ng-deploy: STATION_MODULES_DIR is recognised as `<NIAGARA_HOME>/modules` however it is spelled
- A relative path (`../home/modules`), `./`, a path through a symbolic link above the directory, or another case on a mount that ignores case now name the same directory: the file system is asked (`os.SameFile`). Before, a relative spelling was taken for another directory and the deploy ended with exit 40 and a line about jars left aside by a deploy that did not finish.
- In that mode, something that is not a jar file under the name of a jar of the mode (a directory, a link to nothing) is refused before the build with the copy's own exit 40 line. Before: exit 0 with a WARN.

### Changed — pre-push comments and test titles describe the Go gate

- `scripts/changelog-gate.sh`'s header now says that ci.yml runs it and that the Go pre-push gate runs the same
  check in process. The H29 comment no longer describes the hook as bash, and the LC8 title names the early-pass
  exit instead of the bash function `pass_early`. Retro `2026-11-14-go-pre-push-port` Δ4.

### Changed — pre-push refuses a push it cannot place in a work tree

- When `git rev-parse --show-toplevel` fails, the pre-push gate now refuses the push with git's own error
  (`pre-push: git rev-parse --show-toplevel failed: <git error>; refusing to push unchecked`) instead of
  passing it silently with exit 0. A bare repository is the one exception: the push goes on and a one-line
  notice names it (`pre-push: bare repository <dir>: no work tree, nothing to gate`). Retro
  `2026-11-14-go-pre-push-port` Δ1.

### Fixed — pre-push reads a retro header flip with rename detection off

- The promotion exit's header-flip check now diffs the range with `--no-renames`. Before, a range that
  deleted a pending retro and added a similar retro born folded was a rename in git's default diff, so both
  header lines sat in one file's diff and counted as a flip. Now only a flip inside the same path counts. Retro
  `2026-11-14-go-pre-push-port` Δ2.

### Added — report-module separates NEW FAIL rows from a committed baseline

- `report-module.sh` reads `<module-root>/.report-baseline` when the module has one: the FAIL rows are printed in
  two groups, NEW and BASE, the summary reads `f FAIL (n new, b base in <file>)`, and the verdict and the exit
  code read the NEW rows only. A module whose FAIL rows were all there before the change exits 0, so `build.sh`
  no longer reports it as "not hand-off-ready". Before, a client module with 37 FAIL rows, none of them from the
  change, ended every build at exit 50.
- `--write-baseline` writes the file from the run's FAIL rows (one `<artifact>  <check>  <detail>` key per line,
  line numbers and counts masked, sorted); `--no-baseline` ignores it for one run.
- The floor is never baselined: verify-module gate rows and schema-risk rows stay NEW and are left out of a
  written file, and a SKIPped floor check still reads INCOMPLETE. A baseline line no FAIL row matches is a WARN
  `stale baseline line` row.
- A member that exited non-zero without a FAIL row ("exited N with no FAIL row") is never baselined either: its
  row stays NEW and is left out of a written file, so a baseline cannot hide a crashed lint.
- Without the file and without the options the report is unchanged; only the usage text (exit 2) is longer.

### Changed — scaffold-module writes option values as typed

- `--target-version` and `--plugin-version` are literal text: `&`, `|` and `\` are written as typed. Before, a
  value was read as sed replacement text (`&` was the replaced token, `\&` a literal `&`, `\n` a newline) and a
  value with an unescaped `|`, a trailing `\` or `\1`-`\9` was refused. A value that holds a newline is exit 2 with
  `scaffold-module: <option> value must be a single line (it holds a newline): <value>` and nothing written; the
  message `<option> value cannot be substituted (...)` is gone. `--vendor` keeps its own rule (one Java package
  segment once lowercased, exit 2). Retro `2026-11-14-go-scaffold-module-port` Δ1.
- Inserted text is never rewritten by another substitution: `--plugin-version 4.14 --target-version 4.15` keeps
  plugin `4.14` (it became `4.15`), and a ModuleName that holds the fixture vendor (`AngelesBox --vendor Acme`)
  keeps its name (it became `AcmeBox`). Ordinary values emit the same tree byte for byte.

### Fixed — scaffold-module removes the partial tree of a failed run

- A run that fails after it created `<out-dir>/<ModuleName>` (a failed write, a missing station-test or
  version-ledger template) removes that directory and adds one stderr line, `scaffold-module: removed the partial
  <path>`; when the removal fails the line is `scaffold-module: could not remove the partial <path> (<error>): its
  files stay`. The exit code stays 3. Before, the partial tree stayed and the next run was refused with "destination
  already exists". A destination that existed before the run is refused as before and never touched; the out-dir
  itself is never removed. Retro `2026-11-14-go-scaffold-module-port` Δ4.

### Removed — install-skill and check-skill-drift no longer require a sha256 tool

- `scripts/install-skill.sh` and `scripts/check-skill-drift.sh` no longer exit 3 with `<tool>: no sha256 tool found
  (sha256sum or shasum)` when neither tool is on PATH: the digests are computed by the Go command. The exit 3 that
  remains before the command runs is the Go one (`<tool>: go is not on PATH; it is needed to build <bin>` when the
  command has to be built). `scripts/lib/sha256.sh` stays for its one caller, `scripts/ci-local.sh`. Retro
  `2026-11-14-go-install-tools-port` Δ1.

### Fixed — the close bats no longer run install-skill against the real home

- `tests/c8-close.bats` to `tests/c11-close.bats` run `install-skill.sh --dry-run` with `--home` on the case's
  temporary directory; the c8 and c9 cases no longer read `$HOME/.claude/skills` to decide whether to skip.
  `tests/shell-hygiene.bats` SH2 fails on a bats case that runs a script taking `--home` with neither `--home` nor a
  `HOME=` override on the command. Retro `2026-11-14-go-install-tools-port` Δ3.

### Fixed — the PF-jarlock cases no longer depend on the shared go/bin during a parallel run

- `tests/preflight.bats`: the cases that narrow `PATH` (no `go`) run a private copy of `preflight.sh` and of its
  binary, made once per file. Before, they ran the launcher over the shared `go/bin/preflight`, so a Go source that
  changed during the run (an edit, a checkout, another build) made the launcher exit 3 and the cases fail on a row
  that was never printed. `PF-jarlock-timing` now asserts that the check ran and that its one `lsof` call slept (its
  fake `lsof` called a `sleep` the narrowed `PATH` did not have), with 9 s of margin instead of under 2 s. No lock
  path or port was shared: each case already had its own temporary directory. Retro
  `2026-11-14-go-scaffold-module-port` Δ2.

### Changed — scaffold-module refuses a --vendor that is not one Java package segment

- `--vendor` is checked before anything is written: lowercased, it names the vendor's Java package directory,
  so it must be an ASCII letter or `_` first, then ASCII letters, digits or `_`. Anything else (`/`, `.`, `..`,
  whitespace, a newline, a leading digit, a non-ASCII byte, `&`, `|`, `\`, empty) exits 2 naming the value. Before,
  such a value emitted a broken or escaping package path. Retro `2026-11-14-go-scaffold-module-port` Δ3.

### Added — lint-structure accepts a declared theme or library module

- A module declares its kind with an optional line in the header of `<module-root>/MODULE-MAP.md`: `type: theme` or
  `type: library`. A declared theme's zero-Java `-ux` part passes L9 when it holds at least one `.nss` file, and
  FAILs without one; a declared library's `-rt` part passes L5 with no `module.palette` (a palette it ships is still
  judged). Before, both shapes FAILed exactly as `types/theme.md` and `types/utility-lib.md` prescribe them.
- The kind is read from the map only, never inferred from missing files: an undeclared part FAILs as before. The
  undeclared `-ux` L9 row now ends with how to declare a theme; the L5 text and the `-wb` L9 text are unchanged.
- `types/structure.md` lists every rule the lint runs (L12 to L16 added, L8 marked as `verify-module.sh`'s, L10 as a
  WARN or a FAIL by key and tree), and the header of `toolbelt/verify-module.sh` every check the gate runs, in run
  order. Two Go tests fail when either list and the code differ.
- Corrected against the code: where the station test runner runs (`METHODOLOGY.md`, `BUILD-LOOP.md`,
  `types/moduleTest.md`), the seven Spanish level names the console triage matches (`types/diagnostics.md`), and the
  corpus-index range in `skill/SKILL.md`.

### Added — verify-module --expected-signer, passed by build.sh

- `verify-module.sh --expected-signer ALIAS` FAILs the `signed` row unless the jar's complete signatures are exactly
  that one signer, compared as jarsigner names its files (upper-cased, 8 characters, other characters as `_`:
  `acmesigner` is `META-INF/ACMESIGN.SF`). Before, a client module signed with the default development key passed
  the gate and a station's Software Manager rejected it as self-signed. Without the option nothing changes.
- `build.sh` passes it with the configured `niagaraSigningAlias` (`--gradle-prop`, `ORG_GRADLE_PROJECT_`, or a
  `gradle.properties`), else with the signer of the deployed jars its drift gate snapshots; with neither it prints
  one `WARN — no niagaraSigningAlias is configured` line and keeps its exit code. External session retro, delta B5.
- The deployed-jar signer is never the signing plugin's default development signer (`NIAGARA4`): such a deployed jar
  gives no expected signer and the WARN names it. A verify gate FAIL now restores the pre-build `modules/` jar the
  drift gate backed up (exit 50 as before), so a bare re-run fails for the same reason; `--no-drift-check` has no
  backup and no deployed-jar signer.
- On a first deploy (no jar of the profile in `modules/` before the build) a gate FAIL removes the jar gradle just
  installed, so a later run cannot take its signer as the expected one; the development signer is recognised in any
  case (`niagara4.SF`).
- `verify-module.sh` reads signature files in any case of the extension (`.sf`, `.rsa`, `.dsa`, `.ec`), as the JDK
  verifier does, with or without `--expected-signer`: a jar signed only as `NAME.sf` + `NAME.rsa` now PASSes `signed`
  (it was "no META-INF/*.SF"), and a second signer filed in lower case is reported and FAILs under the option.

### Added — alarm guidance: the return to normal rewrites msgText

- `build-n4-module-kit/types/logic.md` LC9: the toNormal transition rewrites the same alarm record's `msgText`
  with `toNormalText` (empty by default), and alarm texts resolve against the record; a value that must outlive
  the event lives in a persisted read-only slot, reset by acknowledge and quoted by both texts through an
  `alarmData` key. `mcp-n4-kit/METHODOLOGY.md` § 3 carries the same rule as one live-rules row.

### Changed — check-bats-skips and check-client-source read what they used to refuse or misread

- `scripts/check-bats-skips.sh`: `^*` is refused only where it opens a branch (the line's start, after `(` or
  `|`, after anchors only), where GNU `grep -E` warns "* at start of expression"; `a^*`, `(a)^*` and `b|a^*`,
  which Go and GNU grep read alike without a warning, are allowed again. `^*a` stays refused.
- `scripts/check-client-source.sh --build-state`: a BUILD-STATE saved as UTF-16 with its byte-order mark (either
  byte order) is decoded and declares what it declares; UTF-16 that cannot be decoded (no mark, so NUL bytes, or
  an odd byte count) is exit 3 with a message saying to save it as UTF-8. A UTF-8 byte-order mark that opens a
  line other than the first is exit 3 naming the line. Each of these read "undeclared", exit 0.

### Fixed — build.sh: a mirror keeps the jar a build replaced; a missing `.deploy-baseline` names its command (retro row B6)

- `build.sh` against a mirror: gradle's `:jar` step replaces `<mirror>/modules/<MOD>-<p>.jar`, so the mirror (also the
  drift baseline of what the station runs) stopped matching the deployed jar. A run that ends with a `modules/` jar
  different from the drift gate's pre-build snapshot now keeps the snapshot as
  `<mirror>/.mirror-replaced/<MOD>-<p>.jar.<sha12>` (never overwritten or deleted) and ends stdout with one line:
  `==> mirror: <MOD>-<p>.jar in <mirror>/modules replaced (vendorVersion <old> sha256:<old> -> <new> sha256:<new>);
  first replaced this cycle: vendorVersion <v> sha256:<s>; restore with: <toolbelt>/build.sh --restore-mirror <kept>`.
  After two builds the restore command still names the jar from before the first one; a cycle ends only at
  `--restore-mirror`, so after a deploy delete `<mirror>/.mirror-replaced/<MOD>-<p>.jar.first`. A jar or a cycle
  pointer that cannot be written is one WARN (and then no jar is named as first). Printed paths are shell-quoted. Exit codes are unchanged; nothing is kept for a real `niagara_home`, under `--no-drift-check`, on a first
  deploy, or when the run already restored `modules/`.
- New `build.sh --restore-mirror <kept>`: copies a kept jar back into its mirror's `modules/` and builds nothing; it
  refuses a path not under `.mirror-replaced/` (exit 2), a missing kept jar, one whose sha256 prefix is not its
  `<sha12>`, or a home without `.niagara-mirror` (exit 10, nothing written); a failed copy is exit 1. It ends the cycle.
- `report-module.sh`: the `schema-risk` SKIP row of an artifact without `.deploy-baseline` names the exact command that
  creates it from the deployed commit (archive to a temporary tar, then `mkdir <artifact>/.deploy-baseline` and
  extract, so a wrong ref or a pasted `<deployed-ref>` leaves no empty baseline); the verdict stays INCOMPLETE (exit 4, `build.sh` 53), and
  `build.sh`'s INCOMPLETE line points at that row only when an artifact lacks `.deploy-baseline`.

### Fixed — `ci-local.sh --only "Go"` no longer fails the vendorfloor tests on a host without acorn

- `--only` of the Go step skipped CI's npm step but kept `export KIT_REQUIRE_JS_TOOLS=1`, so a host without node
  or acorn failed `go/internal/vendorfloor` on the missing tool. When the npm step is not part of the run and
  acorn cannot be resolved, the replay now runs the step with the variable unset and prints a NOTE; a host with
  acorn, or a full replay, keeps CI's strict setting (`scripts/lib/ci-local-js.sh`).

### Fixed — the pre-push shellcheck closure reads compound source lines and falls back on any it cannot resolve

- `ci-steps changed` now recognises a `.`/`source` command after a separator or keyword on its line
  (`[ -f x ] && . x`, `then . x`, `{ . x; }`, `a; . b`), not only at the start of a line, and follows it.
- A source line anywhere in CI's shellcheck set that names no file of the set now checks the whole set (it might
  be the only sourcer of a changed lib); before, only a line inside the closure did.
- A `# shellcheck source=` directive holds over comment and blank lines up to its command.
- A changed path the set does not hold (a deleted script) is left out of the closure instead of being read.

### Added — `-update` re-records the golden cases of a Go port

- The shared golden runner (`go/internal/lint/rules/golden`) takes `go test ./internal/<pkg> -update`: a case
  whose exit code, stdout or stderr differs gets its `want` and `stderr` sections rewritten instead of failing,
  and every other byte of the case file is kept. The case directory is resolved to an absolute path first, so
  `CHDIR=1` cases (report-module) are rewritten in place. Without the flag nothing is written. Documented in
  `go/internal/lint/rules/golden/record/README.md`.

### Added — install-skill and check-skill-drift for pi and gentle-shell

- `install-skill.sh --harness <claude|pi|gentle-shell|all>` (default `claude`, unchanged) installs a skill launcher
  into `<agent-dir>/skills/<skill>/SKILL.md` for pi (`$PI_CODING_AGENT_DIR`, else `~/.pi/agent`) and gentle-shell
  (`~/.gentle-shell/agent`), plus the prompt file `<agent-dir>/prompts/<skill>.md` that gives those harnesses a bare
  `/<skill>`. The prompt file is rendered from the kit's `skill/prompt.md` beside the tracked launcher and follows the
  launcher's current / diverged / `--force` rules. An agent directory that does not exist is an `absent` row (exit 0),
  never created. Retro 2026-11-15 H1, H2.
- `check-skill-drift.sh --harness <h>` / `--all` checks the launcher and the prompt file of each harness; the skill
  column reads `<skill>[<harness>]` and an absent harness is a `SKIP` row. Without the option the row is the one it
  always was. Retro H3 (the session card line is a separate change).
- `--home ""` is a usage error (exit 2) in both scripts instead of falling back to `$HOME`. Retro S3.
- `install-skill.sh` refuses (exit 3) to write through a symlinked `SKILL.md`, skill directory or prompt file; before,
  `--force` overwrote the link's target. Retro S4.
- The close gates' `CLOSE-install-skill` cases install both launchers into a throwaway home in every harness, compare
  them byte for byte with the kit and run the drift check; before, `--dry-run` exit 0 passed for any state. Retro S5.
- The mcp-n4 launcher no longer assumes `~/.claude/skills/` or `.mcp.json`; the kit README gains "Registering the
  server per harness" (pi tool names, `"exposure": "direct"`). Retro H4.

### Fixed — install-skill --harness review follow-ups

- An explicit `--home` now wins over `PI_CODING_AGENT_DIR` in `install-skill.sh` and `check-skill-drift.sh`, so
  `--home <scratch>` keeps every pi write under the scratch directory. Retro 2026-11-15 T3 R1-001, T4 (1).
- `install-skill.sh` checks every file of every selected harness before the first write: a diverged copy without
  `--force` (exit 1) or a symbolic link on the way (exit 3) stops the run with nothing written, and the refusal comes
  before any `--force` line. A symlinked `skills/` or `prompts/` directory is refused like a symlinked target.
  T3 R3-symlink-scope, T4 (2) and (3).
- A diverged prompt file names its template as `template: <path> (rendered for this harness)` with the
  `rendered sha256`. T3 R1-002.
- The prompt file always carries the absolute path of the installed `SKILL.md`, also under a relative `--home`.
  T3 R3-relative-home-prompt.
- Printed paths keep the `--home` spelling (a trailing slash is not cleaned), as before `--harness`.
  T3 R3-path-join-output.
- A run whose every selected harness is absent no longer materializes the kit state. T3
  R3-materialize-when-nothing-installed.
- `--home ""` prints the usage line in `install-skill.sh` too. T4 (4).

### Added — merge-when-green, ci-local --changed, the here-document bats lint, the host-budget free-space floor
- `scripts/merge-when-green.sh <pr> [--rebase|--squash|--merge]` (`go/cmd/merge-when-green`, `go/internal/mergewait`): waits for the check runs of the PR's exact head commit and merges it with `gh pr merge --match-head-commit`; refuses a draft (exit 4) and a branch that lacks the current base (exit 5), stops when the head moves (exit 6), re-runs once a check that failed on a transient download error, prints the log tail of a failed check (exit 1), 7 on timeout, 8 when the PR is not merged after the merge call. Every gh error is printed; a network error is retried (`--retries`, default 3); the PR state read after the merge call decides, so a merge that went through despite a failed call is exit 0 with its merge commit, as is a PR already merged at start. Every error, a gh error or an unreadable reply, is printed once. It reads every page of the check runs, ends a gh call that hangs after 2 minutes (retried like a network error) and polls again after a failed poll, up to the retry budget. Runs under bash and zsh.
- `scripts/ci-local.sh --changed [--range <from>..<to>]`: the pinned shellcheck (CI's whole file set when one of its files changed), lint-bats-errexit, the new here-document lint and `gofmt -l` over the changed files only (`ci-steps changed`); the pre-push gate runs it on every pushed range.
- `ci-steps lint-bats-heredoc <file|dir>...`: flags a here-document line in a .bats file that starts with the test keyword (bats 1.10 reads it as a duplicate case); a new `lint-sweeps` step of ci.yml runs it over `tests/`.
- host-budget `--disk-min-free <size>` (40G, 500M, …): when given, a mount FAILs below that much free space instead of above `--disk-pct`. Not used by default.

### Changed — host-budget disk rows and the ci-local disk limit
- Every host-budget disk row names the percent used, the free space in GB and in bytes, and the limit.
- `scripts/ci-local.sh` fails a disk above 95% used (`--disk-pct 95`), not the 90% default.

### Changed — CI on drafts, the pinned local bats, the retro refusal text, preflight.bats speed
- ci.yml runs every job on a draft pull request too.
- `scripts/ci-local.sh` runs a bats step with the bats-core version ci.yml installs (sha256-checked, cached) when the bats on PATH is another version; its helpers moved to `scripts/lib/ci-local-tools.sh` and `scripts/lib/ci-local-changed.sh`.
- The pre-push retro refusal names the directories it reads (`build-n4-module-kit/retros/`, `ledger.d/`) and the two ways out.
- `tests/preflight.bats` runs a stub `lsof` in every case: serially 847 s → 15 s on the WSL host.

### Fixed — ci-local --changed kit-links fails loudly (retro row T32 of the 2026-11-15 coordination stretch)

- The kit-links step of `scripts/ci-local.sh --changed` exits 3 when bats cannot be provisioned at the version
  ci.yml pins, instead of running whatever bats is on PATH: the download error, or one line when the provisioned
  bats is not the pinned version.
- A pinned-tarball download (shellcheck, bats) is bounded by curl `--connect-timeout 10 --max-time 120`, so a
  blackholed network exits 3 instead of hanging.
- A git error while detecting whether the change touches a kit doc is reported (exit 3), never read as no change.
- The generated state of the snapshot is written by the snapshot's own `toolbelt/kit-state.sh`.

### Fixed — ci-local --changed runs kit-links (retro row T25 of the 2026-11-15 coordination stretch)

- `scripts/ci-local.sh --changed`, which the pre-push hook runs, also runs `tests/kit-links.bats` when the change
  touches a kit doc, a `ledger.d` or a `changelog.d` file, on a snapshot of the change with the generated
  BUILD-STATE.md rebuilt there; a fragment that names a missing doc is refused at push, not first in CI.
- The host-budget gate of `ci-local.sh` moved to `scripts/lib/ci-local-host.sh` without a behaviour change.

### Added — lab-sim: a `replay` behaviour that plays a recorded series into a point
- A point's `behaviour` can be `{"kind": "replay", "file": "<csv>"}`: the records of a CSV file beside the scenario play into the point, each holding until the next (no interpolation). `speed` plays it faster than recorded (60: a recorded minute per second), `end` is `hold_last` (default) or `loop` (the last record holds as long as the one before it, then the series starts again), and `start_offset_ms` starts it that far into the recorded series.
- The file is a plain `time,value` CSV or the `Timestamp,Value,Status` columns of a history read as ITableToCsv, newest or oldest first; times carry their zone (`2026-11-15T08:00:00.000-06:00`, `...Z`), values are plain numbers, so a history is read with `timestamp.encodeToString, value.encodeToString, status`. Records whose status is not ok (`{stale}`, `{fault,down}`, ...) are skipped and counted; `lab-sim check` shows the records played and skipped. A bad file is one fault naming the row and column; a replay keeps the rules of the other behaviours (not on a bit, not on a writable point, its values must fit the type).
- `examples/replay.json` replays two small committed series. The existing behaviours, examples and messages are unchanged, except that the unknown-behaviour message now lists `replay`.

### Added — logic rule LC10: seed persisted state before the first control cycle

- `build-n4-module-kit/types/logic.md` LC10: Niagara can call `changed()`, and so run a control cycle, after a
  component starts and before `atSteadyState()`. A module that loads its persisted latches, counters or
  accumulators into memory only at steady state writes the defaults over them in that window. The rule: seed
  first in the cycle and write nothing while unseeded (or skip the cycle before steady state), for every
  persisted group the cycle writes, with a source-structural test pattern and a live restart check.

### Added — logic rule LC11 and the LP1 lab restart proof

- `build-n4-module-kit/types/logic.md` LC11 (extends LC10): a control cycle that runs before `atSteadyState()` can
  command outputs before the boot safety holds (power-on stagger, minimum off across a restart, anti-short-cycle)
  are armed, and can see a config "change" on every restart when its "last valid" state is not seeded. The rule:
  arm the holds first in the cycle, write OFF while held, keep output commands TRANSIENT, and seed the derived
  state with the persisted slots; with a source-structural test pattern.
- `build-n4-module-kit/build-verify.md` § Lab restart proof (LP1): the restart recipe that exposes both (non-default
  timers, every demand input linked, inputs written during boot, timestamped reads from before `Station Started`,
  each backup file missing once, proven cleanup).

### Fixed — logic series: BOM header, out-of-range time, ASCII tolerance

- `n4kit.series` ignores a UTF-8 byte order mark before the header (a series saved by a Windows editor).
- A row time beyond the long range is refused with its line (`time '…' is out of range (ms)`), never a raw
  `NumberFormatException`.
- A mismatch message shows the tolerance as `+/-<tol>` (ASCII), so a console without UTF-8 prints it intact.
- The header parser reads each column in its own method (readability; behavior unchanged).

### Changed — LP1 lab restart proof traps

- `build-n4-module-kit/build-verify.md` § Lab restart proof (LP1): set two holds far apart so the proof can tell
  which one held an output; grep the module's `config.bog` type prefix in the cleanup check, not its name; a
  remove plus `n4_rollback` creates a new instance and is not a restart; touch a jar swapped under the same file
  name so the registry rebuilds.

### Fixed — mcp-n4: an unknown-type refusal names the stale-registry route; lab module-swap step (retro rows M12, M14)

- `n4_create_component`'s dry-run refusal of a type the station's registry does not describe now also says what to
  do when the module is installed and running: the station's registry may be stale, so stop the station, touch the
  module jar, start it until its log shows "Rebuilding registry", then reconnect with `n4_connect`. On a lab
  station a module jar removed and re-added under the same name left the registry "Up-to-date" without the module's
  types; the pre-check (#659) refused a client module type that a binary older than the pre-check had created.
- `mcp-n4-kit/METHODOLOGY.md` § 3 adds the lab module-swap step: touch the jar before the start and check the start
  log for the registry rebuild.

### Added — `mcp-n4 install` and `mcp-n4 register`

- `mcp-n4 install [--prefix DIR]` copies the running binary to `DIR/mcp-n4` (default `~/.local/bin`)
  atomically (temporary file, fsync, chmod 0755, rename), so station launchers stop depending on a worktree.
- `mcp-n4 register --station NAME --harness claude|pi|gentle-shell [--env PATH] [--bin PATH] [--project DIR]
  [--force] [-- SERVER-FLAGS]` writes `~/.config/mcp-n4/NAME-run.sh` (sources the env file, holds no
  credential) and the `mcp-n4-NAME` MCP entry for the harness, changing only the bytes of that entry; re-running
  is a no-op, and a differing launcher or entry, or a `--bin` inside a git checkout, is refused without `--force`.
- Without server flags, `register` keeps a launcher only when it execs the installed binary, and an explicit
  `--bin` or `--env` must match what it execs or sources (never ignored); symlinked configs
  stay links, malformed files (trailing data, duplicate keys) are refused, and an entry equal in value is unchanged.

### Added — `mcp-n4 --version`

- `mcp-n4 --version` prints `mcp-n4 <version> (git <sha>)` and exits 0 (it printed the usage before), from the
  same build stamp `n4_describe_session` reports as `server.build`.

### Added — n4_add_history sets a numeric COV change tolerance and the full policy

- `change_tolerance` (a number of at least 0, in the point's units) sets `changeTolerance` of each
  `history:NumericCovHistoryExt` in its add body, for every point of the batch: a change smaller than it records
  nothing. It is refused for mode interval, for an `ext_type` other than the numeric COV one and for a point that is
  not numeric, naming the point. The read-back checks it.
- `full_policy` (`roll`, the extension default, or `stop`) is set by a set op on
  `<ext_name>/historyConfig/fullPolicy` (`history:FullPolicy`) right after each add in the same batch; the capacity
  note names the policy and the read-back checks it.
- Three COV histories with a 0.5 A tolerance, 50000 records and a full policy are now one dry run and one token
  (proven live on a 4.15 lab station: 9 ops, `verified`, a 0.3 change recorded nothing). Before, the tolerance took a
  `n4_set_slot` dry run and token per point. `capacity`, `mode=cov` and N points under one token were already there
  (#670). Client-A overamps retro 2026-11-16, row M4.
- The plan notes that `full_policy=stop` has no effect with `capacity="unlimited"`, and the type-mismatch and
  tolerance refusals say "an enum point" instead of "a enum point".

### Added — mcp-n4: n4_alarm_text_check, the live LC9 alarm-text check (retro row P2)

- `n4_alarm_text_check` reads every alarm source extension below `base` (one BQL select, subtypes included) and
  flags `parent-path` (a `%...%` token rooted at no alarm record field, such as `%parent.out.value%`: the texts
  resolve against the record, never the source) and `empty-tonormal` (a `toOffnormalText` with an empty
  `toNormalText`, which blanks `msgText` on every return to normal). Rule names, slots and rows match the offline
  `bog-nav alarm-text`, so the same extension gives the same finding from a bog or a live station. METHODOLOGY's
  LC9 row is now mechanical. The 2026-11-13 client A over-amps text failed silently; it is now one read-only call.

### Fixed — mcp-n4: n4_alarm_text_check splits tokens as BFormat does (retro row P2b)

- `n4_alarm_text_check` now splits an alarm text into `%...%` tokens with a port of `BFormat.doParse`: `%%` is
  a literal percent, a token ends at the next `%` outside parentheses, and a `%` that never closes makes
  BFormat show the whole text as `ERROR <text>`, so no token of it is reported. Before, a lone `%` paired
  with the next one: `Load at 90% of %parent.out.value%` reported `% of %` as a parent-path token, and a
  `%` inside `%lexicon(...)%` split the call. The offline `bog-nav alarm-text` (niagara-go) has the same
  tokenizer and the same test table. The select's columns are now built from the text slot list, so the
  two cannot drift apart.
- New rule `bad-format`: a text BFormat cannot parse (a `%` that never closes) shows on the station as
  `ERROR <text>` and none of its tokens is evaluated. It is one WARN row naming the slot (token empty);
  `bad_format` counts the rows, `hint` advises `%%` for a literal percent, and the summary names
  bad-format only when a text has it. bog-nav alarm-text reports the same row.

### Added — mcp-n4: behaviour snapshot and diff around a deploy (O6)
- `n4_behavior_snapshot(name, scope)` stores the out value and status bits of every control point below a scope in `<state-dir>/snapshots/<name>.json` (0600, never overwritten), with one BQL read.
- `n4_behavior_diff(before, after?, tolerance, limit)` compares it with a second snapshot or the live station: `status_changed` (flags by name), `value_changed` (numeric tolerance reads a locale decimal comma and a trailing unit), `missing`, `new`, `summary`. A truncated side leaves missing/new `not_checked`, never zero.

### Added — mcp-n4: `n4_bql_query` reads the alarm: and history: bases
- `base` may now be `alarm:` (the alarm database) or `history:/<station>/<history name>` (one history), also after `station:|`. The query over those bases is `select <columns> [where ...] [order by ...]`, with no type and no `from` clause (refused: live, `history:` ignores it and `alarm:` answers 404), e.g. `select timestamp, source, sourceState where source like '*slot:/Lab/P/*' order by timestamp desc` or `select timestamp, value where timestamp.millis >= 1794808800000`. One `GET /ord/` of the same `view:file:ITableToCsv`; the reply has the station-base shape (`columns`, `rows`, `row_count`, `truncated`, `elapsed_ms`, optional `output_file`).
- Read-only: a non-select, `|`, a control character or an over-long query is refused before any request; a base in another scheme (`file:`, `ip:`) is refused naming the three accepted forms, and a history id that is not two slot names names the `base`. Station-subtree queries and their refusals are unchanged; `n4_inventory` still reads station subtrees only.

### Added — mcp-n4: `n4_event_report`, alarm times and the source's history peak in one call (retro row P3)

- `n4_event_report(source | source_glob, history_id, since, until, margin_minutes=15, max_events=10)` returns, per
  alarm event of the source (newest first), the alarm database times (`alarm_time`, `normal_time`, `ack_time` as
  ISO-8601 with the station's offset, `acked_by`) and the `n4_read_history` summary (count, min, max with its time,
  first, last) of the source's history over `[alarm_time - margin, normal_time + margin]`, up to now while the alarm
  is active. The history is `history_id` or the history extension beside the alarm extension, found once per source;
  a source with no history gets a null `summary` and a `summary_note`, never a failed report. Read-only.
  "A cuanto llego" took six steps in the 2026-11-16 client A over-amps retro.

### Fixed — mcp-n4: history summaries leave {null}-status records out of min/max (retro row M16)

- The summary of `n4_read_history` `summary: true` and of `n4_event_report` counted the `0.0 {null}` records
  a new COV history writes at start in min and max, so min read `0.0` (live on a lab history). A record whose
  status has the `null` flag holds no value (`BStatusValue` renders it `-`), so it is now counted in `count`
  and `status_counts` but left out of `min`, `max`, `first` and `last`, as `nan`/`inf` records already were
  for min and max. `excluded_from_extremes {null, nan_inf}` counts both causes, a `note` says so, and
  `first_record` / `last_record` keep the raw ends when they differ. Other status flags keep their value.

### Added — mcp-n4: whole-range history summary and alarm state counts (retro row live M2)

- `n4_read_history` `summary: true` returns count, min and max with their ISO timestamps, first, last and
  per-status counts over the whole range, whatever `limit` cut the rows, from one read of every record with
  machine-encoded time and value; a range above `summary_max_rows` (default 100000) says so and adds the
  station's whole-range `select min(value), max(value)`. `start`/`end` also take a span back from now (`30m`,
  `24h`, `7d`), and a bare history name is a history of the connected station.
- `n4_read_alarms` `state_counts: true` returns every matching record per source and ack state; rows add `timestamp_iso`,
  `normal_time` and `ack_time`; a `source_glob` like-pattern filter and relative `since`/`until`.
  "A cuanto llego" was six manual steps in the 2026-11-16 client A over-amps retro; it is now one call.

### Security — mcp-n4: BQL values are checked as BQL decodes them

- Niagara decodes `$xx` / `$uxxxx` escapes inside a BQL body (`BqlQuery.make`), so `n4_read_alarms`
  `source` and `alarm_class` holding `$27` could close the quoted literal (live: `alarm_class` matched 27 of 27
  alarm records), and `n4_bql_query` / `n4_neql_query` selects could pass a local check through an escape
  (`select uuid$20from$20control:NumericPoint` on `alarm:` passed the no-from rule). `bql.DecodeForBql` (one pass
  of `EscUtil.unescape`, a malformed escape refused) now feeds every one of those checks. ORD paths and history
  ids are matched escaped by the station and are never decoded. `source_glob` takes no `$`.

### Fixed — mcp-n4: `n4_read_slots` reads a struct slot's own fields and decodes AlarmTimestamps (retro row M3)

- A BOX load answers with the nearest component, so `n4_read_slots` on a property path such as
  `.../OutOfRangeAlarmExt/toOffnormalTimes` listed the extension's own slots (live on a lab station). An ORD
  whose trailing segments name a property of the loaded component (a run of loaded or declared-default
  nodes without a handle, confirmed by one extra load of the component that must answer the same handle)
  now lists that property's fields: loaded ones, then its declared defaults from the type contract
  (every field is marked `default` when the property itself is at its default). A property with no fields
  (a simple) is listed itself. The reply adds `component_ord`, `property` and `property_type`.
- An `alarm:AlarmTimestamps` (`BAlarmTimestamps`: `alarmTime`, `ackTime`, `normalTime`, `count`) adds
  `alarm_timestamps {alarm_time, ack_time, normal_time, count}`: times as the station's ISO text with its
  offset, `null` when unset (`BAbsTime.NULL`, epoch 0), `count` as an integer. Component reads are unchanged.
- The test station now loads the nearest component for a property ORD, as N4 does, and can hold a frozen
  property at its default that resolves but is never sent.

### Changed — BREAKING: mcp-n4 refuses unknown arguments and enum violations in every tool (retro row M11)

- BREAKING: every mcp-n4 tool now refuses an argument its `inputSchema` does not declare with JSON-RPC -32602
  `unknown argument '<name>' for <tool>`, plus `; did you mean '<name>'?` when one declared argument is close
  (`capacity_records` -> `capacity`), and a value outside an `enum` with `argument <name> must be one of a, b, c;
  got '<value>'` (an enum list names the item, `sections[1]`). Before, an unknown argument was silently dropped
  (`capacity_records: 50000` on `n4_add_history` kept the 500-record default) and enums were checked only by some
  tools, as a tool error with their own wording. A caller that sent a dead argument now gets an error instead of a
  call that ignored it. One shared validator (`go/internal/mcpn4/jsonrpc`) does it before the session gate, so
  nothing reaches a station. Objects a tool keeps open stay open (`alarm_values` through `additionalProperties`;
  `expect_histories`, `expect_alarms`, `wire_sheet`, `weekly` and the `n4_set_schedule` / `n4_set_tags` list items
  declare no properties); a nested object is checked only where its schema declares `properties`. A refusal echoes at
  most 64 characters of the value or argument name, then `…`. A did-you-mean never offers a key the unknown one only
  starts unless it covers half of it (`all` is not `allow_ad_hoc`). `n4_read_alarms` keeps `source_state: ""` and
  `ack_state: ""` as no filter: the empty string is now declared in their enums.
- `n4_neql_query`: the no-from rule of its `bql` select now matches `from` between any BQL space (a decoded tab,
  newline or no-break space, `$09from$09`, `$0afrom$0a`, `$u00a0from$u00a0`) and at the end, as the table-base check
  does; before only ` from ` was refused (the station answered 404 anyway).

### Fixed — mcp-n4 register: kept launcher resolved and nothing written on a duplicate name (retro row T17 of the 2026-11-15 coordination stretch)

- A launcher kept without server flags whose exec target is the installed binary path (`~/.local/bin/mcp-n4` by
  default) is now refused when that path is a symlink into a git checkout, with the same message as the server-flags
  case; `--force` still keeps it.
- A duplicate entry name (or any other unreadable JSON on the entry path) in the harness config is refused with
  "; nothing written", and every file stays byte-identical.
- When writing the harness entry fails after the launcher was written, the launcher is put back (or removed when it
  is new) and the error ends with "; nothing written"; the "written launcher" line is printed only once both writes
  succeeded.
- A dangling `~/.local/bin/mcp-n4` (a binary in a worktree since removed) is judged by its link text, followed
  relative to the link's directory through the whole chain: a target inside a checkout, or under a checkout whose
  worktree is gone, is refused like a live one; a dangling link outside any checkout is kept as before.

### Fixed — mcp-n4: a failed type-registry call is retried once (retro row M13)

- Right after a station starts or a module first loads, one type-registry request (`reg` / `loadTypes`) could get
  no answer, and the first `n4_set_slot` of a slot still at its declared default failed with it. `box.Client.LoadTypes`
  now asks again once, after 0.5 s, when a request gets no answer (an HTTP error status, a transport failure or a
  malformed reply), so every caller inherits it: `n4_set_slot` and the other declared-default writes, the create,
  link, alarm, history and flags prechecks, and the `n4_read_slots` / `n4_navigate` listings.
- An answer is never retried: the station's own error reply, an `isErr` entry for a type the station does not know
  (still refused as "not known to the station" with the stale-registry route) and an AuthError.
- Two requests without an answer refuse with "the station's type registry did not answer (it may have just
  started): retry", naming the last failure.

### Fixed — mcp-n4: a remove's confirmation token survives runtime slots ticking (retro row M15)

- `n4_remove_component` no longer hashes the values of runtime slots into the plan its confirmation token
  binds. A runtime slot is transient (never persisted): its declared or its instance flags hold `t`, such as an
  alarm extension's `timeInCurrentState` (flags `rt`). The dry-run inverse marks it `"runtime": true` and shows it
  without its value, so its ticking between the dry run and the execute no longer refuses the execute with
  "confirmation_token does not match".
- Readonly alone is not runtime: a readonly struct's children (a schedule's `yearSchedule/year`) and persisted
  readonly state (`alarmState`) still bind the token.
- The token still binds the remove op, the subtree's structure (every slot and child name and type) and every
  configuration value: a child added or a configured value changed after the dry run is still refused.
- The journal and `n4_rollback` keep the whole snapshot, runtime values included, as before.

### Changed

- mcp-n4 kit: the session checklist (METHODOLOGY section 4 step 1 and the skill) now says to restart the MCP server after rebuilding the mcp-n4 binary, and never to conclude a tool is missing before checking `server.build` in `n4_describe_session` (retro row live M10).

### Added — mcp-n4: runbooks served as MCP prompts and resources (O8)
- The server answers `prompts/list`, `prompts/get`, `resources/list` and `resources/read`, and `initialize` advertises the `prompts` and `resources` capabilities. Three runbooks ship: `post-deploy-check` (read-only check after a module deploy and restart), `alarm-triage` (open alarms grouped, sized and acknowledgements proposed) and `commissioning` (the commissioning report recipe through dry runs and the save).
- A prompt fills its `{{argument}}` placeholders (required arguments refused when missing, unknown ones refused); a resource `runbook://<name>` returns the Markdown as written; an unknown URI answers `-32002`.
- The runbooks are Markdown files under `go/internal/mcpn4/runbooks/files/`; tests refuse a runbook naming a tool the registry does not hold or a placeholder that is not a declared argument.

### Fixed — mcp-n4: n4_set_schedule with empty date lists
- `add_date: []` and `remove_date: []` on a calendar or trigger schedule are refused as "nothing to set" (when nothing else is given) instead of failing with an internal error.

### Fixed — mcp-n4: n4_set_slot's missing-slot refusal states the declared-default rule
- The refusal for a slot the station's load does not list said the station omits only type defaults (0.0, false, an empty string). The station omits any frozen property whose value equals the default its type declares (a 10 s `RelTime`, `false`), and `n4_set_slot` already fills such a slot from the type registry; the refusal now says so and is left for a name no type contract declares (a misspelled name, a dynamic slot at its default).

### Added — mcp-n4: n4_connect warns when the binary on disk is newer than the running server
- A successful `n4_connect` carries the same one-line `warning` as `n4_describe_session` when the binary on disk is not the one serving: restart the MCP server to serve it.
- Both tools now compare the git revision stamped in the binary on disk with the server's own first (`the binary on disk (<path>) is build <disk>, but this server runs build <running>: ...`); the mtime against the server start decides only when either binary has no stamp. A missing or unreadable binary never warns and never fails the tool.

### Added — mcp-n4 n4_station_runtime: uptime, loaded modules and jars on disk

- `n4_station_runtime` reads, in one call, the station's uptime (spy `sysInfo`, seconds from its boot and current
  ticks), the module parts it has loaded with their versions (spy `sysManagers/moduleManager`) and, when the server
  has `--niagara-home` for the station, the jars of the installation's `modules` directory. `mismatches` names each
  difference (`version_differs`, `modified_after_boot`, `loaded_not_on_disk`, `not_loaded_modified_after_boot`) and
  `restart_needed` says whether a restart is needed to run what is on disk: `null`, with `restart_note`, when a part
  is unknown. Lists are capped by `max_rows`. Research retro row M4.

### Fixed — mcp-n4: the shared argument validator bounds its did-you-mean and keeps the closing quote (retro row M11b)

- An unknown argument name is compared by edit distance only when it is at most 64 bytes long and its length is
  within the allowed distance of a declared key: a 1 MB unknown name is now refused at once, in a bounded number
  of allocations (it cost about four million before).
- A refusal clips the caller's value or argument name before quoting it, so the echo keeps its closing quote:
  `got 'xxx…'`, not `got 'xxx…`.
- `n4_add_history` (`mode`, `ext_type`, `full_policy`), `n4_add_alarm` (`kind`), `n4_read_tags` (`relations`) and
  `n4_commissioning_report` (`sections`) drop their own enum checks, unreachable since the shared validator
  refuses these values with -32602; a test pins each schema enum to the table the tool maps it with.

### Changed — mcp-n4: the write budget never blocks the save

- `n4_save_station` neither needs nor spends a `--max-writes` write, so a bulk job that spent the
  budget can still persist its work; the budget refusal says so.
- Every write reply (dry run and execution) carries `write_budget` `{executed, max, remaining,
  save_counts}`.

### Changed — merge-when-green settles a merge before deciding, and stays inside --timeout

- After `gh pr merge`, the PR state is read up to 1+`--retries` times whether the call returned or failed, and a
  read that fails (an HTTP 502, an unreadable reply) is printed and read again instead of ending with exit 3.
- A merge call that returned while the PR stays OPEN (a merge queue, auto-merge, or GitHub's lag), or whose
  state could not be read, now exits 9 ("check it by hand") instead of 8, which stays "gh pr merge refused".
- Each poll call (check runs, `pr view`) has its own failure budget, so a `pr view` that keeps failing stops the
  wait even while check runs succeed.
- No network retry starts after `--timeout`; a gh call whose child holds its output pipes returns within a
  second of its deadline.

### Changed — merge-when-green decides a refused merge at once and names the state it settled on

- A merge GitHub refused outright (`GraphQL:`, an HTTP 4xx, "is not mergeable", "Head branch was modified") is
  decided on one state read and exits 8 at once, instead of after 1+`--retries` reads `--interval` apart; any
  other merge error still gets the reads, since the merge may have gone through.
- The "merge queue or auto-merge" message (exit 9) is printed only for a PR still OPEN, with how many reads
  answered; a PR CLOSED after the merge call exits 6 ("nothing merged").
- `--timeout` bounds the network retries while the checks run; the merge call and its state reads keep theirs,
  so a network error on an authorized merge near the deadline is retried instead of ending with nothing merged.

### Fixed — merge-when-green no longer reports "nothing merged" for a merge that went through

- A merge call sent again after a network error and then refused ("not mergeable", because its first send
  merged) gets 1+`--retries` state reads, and MERGED at the checked head is the success; only a refusal of a
  merge sent once is decided on one read.
- A rate limit (HTTP 429, "rate limit"), a request timeout (HTTP 408) or GitHub's GraphQL server error
  ("Something went wrong") is not a refusal: it gets every state read too; the HTTP status matches in any case.
- New tests pin the network retry of a state read after the merge call past `--timeout`.

### Fixed — ng-deploy: review advisories T15 and T18 of the 2026-11-15 coordination stretch

- A SIGINT, SIGTERM or SIGHUP that arrives while the probe before the build looks at a station jar it does not
  move (one the station does not have yet) ends the run with 128 + the signal before the build, instead of being
  dropped when the probe stops holding signals.
- A `<jar>.ng-probe` in the station that is not a regular file (a directory, a symbolic link) is no longer moved
  to the jar's name as a probe file a killed run left: the run stops with exit 40 naming what it is, nothing changed.
- The "takes no hard link" NOTICE of a run lock made in two steps prints once per run, and only when the run holds
  its lock, not before a refusal. A link that fails for another reason than a lock that is there (EMLINK and the
  like) also falls back to the two steps instead of failing the run with exit 10.
- A `.ng-deploy.lock.<random>.tmp` older than a minute, left by a run killed while it took its lock, is removed
  with a NOTICE when the next run takes the lock of that directory.
- Two runs that sweep the same temporary lock file print no WARN for the one that finds it gone, and a lock whose
  own temporary file was swept is taken without the "takes no hard link" NOTICE.

### Fixed — ng-deploy: the run lock says whose it is (retro rows D5 and D6 of the 2026-11-15 coordination stretch)

- The run lock is written in a temporary file beside it and hard-linked to `.ng-deploy.lock`, so it holds its
  content from the moment it is there: a run that loses the lock race reads who holds it instead of "it does not
  say which run made it; the file is 0s old" for a live run. A file system without hard links gets the lock in
  two steps, as before, and a NOTICE says so.
- The run lock records the program of the run (`command ng-deploy`), and the refusal names the check that tells a
  killed run from a live one, `ps -p <pid> -o comm=`, and what it must print, instead of `ps -p <pid>`, which a pid
  given again to another program passes. A lock of this host whose pid runs another program gets a second line,
  "pid <pid> runs <other>, not an ng-deploy run: ...". A lock an earlier version wrote (three lines) is still read.

### Fixed — ng-deploy: retro rows D1-D4 and D7-D9 of the 2026-11-15 coordination stretch

- A jar that cannot be replaced or moved (exit 40 in the station, exit 10 in `<NIAGARA_HOME>/modules`) with `Device
  or resource busy` or `Text file busy`, or with `Permission denied` or `Operation not permitted` in a directory this
  user may write (how a WSL mount of a Windows drive answers for a jar a running station holds open), is followed by
  `a running station or Workbench holds its module jars open; stop it and run again`. The same errors in a directory
  this user may not write get a line about its permissions instead.
- A directory or a dangling link under the name of a jar of the mode in `<NIAGARA_HOME>/modules` is refused before the
  build (exit 40, `build refused for <jar>`) in every case, not only when the station is that directory: the put-back
  could not rename over it.
- After a deploy that stopped, the NOTICE about `<NIAGARA_HOME>/modules` says it holds the jars it held before the
  build only when every jar was put back or removed; a jar of another origin left in place gets a NOTICE that says so.
- A deploy from a working directory that is not a git repository ends 0 with `NOTICE: not a git repository; the
  deployed commit is not recorded in .last-deploy-sha: the next annotation check has no starting point` instead of an
  ERROR and exit 41. Only git's "not a git repository" (read in the C locale) with no `.git` there or above counts;
  a repository git refuses to open (dubious ownership, a corrupt `.git`) or whose HEAD cannot be read is still exit 41.
- Before the build, each station jar the run will replace is renamed to `<jar>.ng-probe` and straight back: a jar a
  running station holds is exit 40 with the line above and no gradle call, instead of after a minute of building. A
  jar without write permission still deploys (a rename needs the directory's permission). A jar that cannot be moved
  back is exit 40 naming both paths and the rename to make. SIGINT, SIGTERM and SIGHUP are held across the two
  moves. A `.ng-probe` file a killed run left is moved back when its jar is missing (NOTICE), and refused with the
  manual fix (exit 40) when the jar is beside it; it is never written over.
- The backup and its purge run just before the copy, after the build and every check (with `--no-deploy`, after the
  build): a run that fails at the build or a check leaves `_backups/` as it was, instead of one more archive of the
  same jars per refused run. Exit 21 still means nothing was copied. When `STATION_MODULES_DIR` is
  `<NIAGARA_HOME>/modules` the backup holds the jars that were there before the build, under their jar names; only this
  module's own jars are renamed or left out (another module's file aside is archived as it is). The
  `--no-backup` warning is printed where it was, before the build.
- A mirror `NIAGARA_HOME` gets the isolation `build.sh` applies, from one shared rule (`go/internal/gradleenv`, which
  `build.sh`'s `isolateMirror` now calls too): `JAVA_TOOL_OPTIONS += -Duser.home=<mirror>/.mirror-user-home` (and the
  real maven repository), `GRADLE_USER_HOME` pinned and `--no-daemon` on every gradlew call, so the signing plugin
  cannot read the real `~/.tridium` keystore. A mirror path or `$HOME` with whitespace is exit 10 before gradle.
- `ng-deploy --gradle-prop k=v` (repeatable) is forwarded to gradlew as `-Pk=v`, as `build.sh --gradle-prop` is, so
  `niagaraSigningAlias` and `niagaraSigningProfile` no longer need `ORG_GRADLE_PROJECT_*` set by hand; the gate expects
  the last `niagaraSigningAlias` given. `build.sh` is unchanged: same gradle arguments and environment.

### Added — the template lint checks what a template uses, and reads .napl archives

- `toolbelt/lint-ntpl-manifest-deps.sh`: new sub-check `ntpl-uses`. A module that `template.bog` uses (each
  `m="<alias>=<module>"`, each `module://` ORD) or a `*.px` entry uses (each `<import><module name>`, each
  `module://` ORD) must be named by a `<dependency>` of the manifest (a part counts: `control-rt` lists
  `control`), unless the installation ships it (`--niagara-home`, else `NIAGARA_HOME`), it is `baja`, or it is
  one of the module's own parts. One row per entry, names sorted: FAIL with the installation's module list, WARN
  without it. A bog or Px entry that is not well-formed XML, or inflates past 32 MiB, is a FAIL row naming it.
- The bog and Px entries of one archive are read within 128 MiB inflated in all (32 MiB each): past it one
  `FAIL  ntpl-uses` row names the limit and the first entry not read, and nothing more of that archive is read.
- `.napl` application templates (the same zip) are checked like `.ntpl`, the extension in any letter case.
- `report-module` relays the new rows through its existing template lint member.

### Changed — lint-ntpl-manifest-deps is bounded for the whole run

- `lint-ntpl-manifest-deps.sh`: every template of a template module jar and every manifest, template.bog and Px
  entry of every archive now inflate from one budget of 128 MiB for the whole run, instead of 128 MiB per
  archive. The entry the budget leaves no room for is one FAIL row (`the templates and their entries inflate
  past 134217728 bytes in all, the bound of one run -- <entry> and everything after it not read`) and nothing
  more is read: a small jar of many amplifying templates ends in seconds instead of minutes.
- Behaviour change: a module or a template module jar with more than 1000 templates now FAILs (exit 1). The
  first 1000 are checked, in path order (entry-name order in a jar), and one FAIL ntpl-entries row names the
  first template not read.
- Behaviour change: an unreadable jar over 32 MiB is now the size-bound FAIL row (exit 1); it was exit 3. A jar
  over 32 MiB is refused from its metadata before it is opened, as a .ntpl already was.

### Added — lint-ntpl-manifest-deps: a template module jar as input, and the guide's manifest spelling (retro row K4)

- A template archive without `template-manifest.xml` is read through `template.manifest.xml`, the spelling of the
  Templates guide ("Example template.manifest.xml file"); with both, the hyphenated one alone is read.
- `lint-ntpl-manifest-deps.sh <template-module.jar>` checks each `*.ntpl` and `*.napl` entry of a template module
  jar as a shipped template, each row naming `<jar>!/<entry>`, within the bounds of a `.ntpl` (32 MiB per template
  inflated, 128 MiB for all). A jar declares no build dependencies: dependencies are judged against the
  installation alone. The lint engine gains `Spec.FileRoot`, an opt-in that only this rule sets; every other lint
  still refuses a file root ("not a directory", exit 3).
- Every ntpl-manifest and ntpl-uses row names the manifest entry it read (`template.manifest.xml` when the
  fallback was used), and the usage line names both inputs: `<module-root | template-module.jar>`.

### Changed — the template lint stays inside the module and runs in report-module

- `toolbelt/lint-ntpl-manifest-deps.sh`: a symlink is followed only when its real target is inside the module
  root. A link to an ancestor (a sibling module), to `/` or to a file of another module is not followed and is one
  `WARN  ntpl-link  <link>: a symlink to <target>, outside the module root -- not followed, nothing behind it is
  checked` row, exit 0; a module whose only finding is such a link prints no "nothing to check" row. A link inside
  the module is pruned by its target too, so `out -> build` no longer re-exposes a part's build output, and a file
  reached under two spellings is read once.
- `--niagara-home ""` is a usage error (exit 3, the lint's usage exit) instead of "no installation" (which turned
  every `ntpl-deps` row into WARN). A `modules/<name>.jar` ships its module only when it is a regular file the lint
  can open: a dangling link, a directory or an unreadable file under that name ships nothing. A `.ntpl` that is
  not a regular file (a FIFO, or a link to one) is exit 3 naming it, never "nothing to check".
- `toolbelt/report-module.sh` runs the template lint once per module root (a new member) and takes
  `--niagara-home <dir>`, passed on to it; without the option the lint reads `NIAGARA_HOME` from the environment
  report-module inherits; `toolbelt/build.sh` passes its `niagara_home` as that option. Its rows are relayed whole (`(module)  FAIL  ntpl-deps  <file.ntpl>: <reason>`), a
  `FAIL` row fails the report, and a module with neither a build file nor a template is a SKIP row. Every report
  gains the member's row (`PASS  lint-ntpl-manifest-deps  clean` or SKIP).
- `verify-module --src` reads the declared dependencies through the shared `lint.DeclaredDeps` (no behaviour
  change).

### Added — logic series: check a pure control core against scripted inputs offline (O12)

- `build-n4-module-kit/toolbelt/java/n4kit/series` (Java 8 source, no dependency): `LogicSeries` steps a pure model
  through a CSV of timed inputs and expected outputs (`time_ms`, `in:<name>`, `out:<name>[~<tol>]`, `-` skips a
  check) through a small `SeriesModel` adapter, and stops at the first mismatch with its file, line, step, time,
  output, expected value, tolerance and actual value. A malformed series is refused before any step.
- `run-pure-test.sh` puts the kit's Java helper root on the javac source path, so a module test imports
  `n4kit.series.*` without copying it.
- `build-verify.md` § Logic series (recipe, results, how it complements the LP1 lab restart proof) with a worked
  example (`tests/fixtures/logic-series/thermo-rt`) pinned by `tests/logic-series.bats`.

### Changed — n4call executes a write only with --execute, a destructive one only on its own line (retro row M17, opportunity O9)

- **Breaking for n4call scripts**: without `--execute`, every write in the calls file is a dry run
  only and the stderr mode line says so; a run that executed writes before needs `--execute` now.
  `--dry-run-only` stays and excludes `--execute`. A flag lost on the way to n4call can no longer
  execute a write: a wrapper's dry-run-only flag did not reach n4call and an `n4_remove_component`
  executed on the lab station.
- A destructive write (`destructiveHint`: remove, rollback, save, ack_alarm, set_tags) executes only
  with `--execute`, `--allow-destructive` and its own line's `"execute": true`; a run with
  `--execute` that misses either is refused with exit 2 before the server starts, naming the line.
  `"execute"` on a line without `"write": true` is a usage error.
- The server already executes only with a token a dry run issued in the same process and connect
  session; two new black-box tests pin a restart and a token no dry run issued.

### Changed — the pre-push CI lints shellcheck the change with its sourced files, not CI's whole set

- `scripts/ci-local.sh --changed` (run by `.githooks/pre-push`) shellchecked all ~285 files of CI's shellcheck set
  whenever one shell file changed (about 30-43 s per push). It now checks the changed files of that set together
  with the files they source and the files that source them, followed transitively through `.`/`source` lines,
  so a sourced lib still gives its sourcer's context (no false SC2034).
- When a source line names a path that matches no file of the set, it falls back to the whole set and names
  `<file>:<line>` on stderr.

### Fixed — `report-module.sh --ready` review advisories (retro row P4b)

- A module root that cannot be listed is now `report-module: --ready: cannot list <root>: <error>` (exit 3),
  no longer hidden behind "no built jar under ...".
- An artifact of the module (`module-include.xml`, its own `<artifact>.gradle.kts` or a `build.gradle[.kts]`)
  with no built jar is one `FAIL  jar` row naming it, so the module reads NOT READY instead of leaving the
  artifact out; a `docs/` or `gradle/` directory beside the artifacts gets no row.
- A baja floor that cannot be compared with `--expect-niagara` (`4.14-rc` against `4.15`) FAILs the
  `niagara-target` row instead of WARNing, so the jar never reads READY with its target unchecked.
- The lint-ntpl-manifest-deps header and `types/logic-authoring.md` now name the 100,000 entries-read bound of a
  run and its FAIL row next to the 128 MiB bound (retro row T34 doc follow-up).

### Added — `report-module.sh --ready`, the ready-to-load report of a built module

- `report-module.sh <module-root> --ready [--niagara-home <mirror>] [--expect-niagara <ver>] [--expect-signer <alias>] [--json]`
  prints one row per check for each built jar: module, niagara-target, niagara-home, signer, verify-module,
  schema-risk, sha256 and mirror. It ends with `READY TO LOAD: <MOD> <version> for Niagara <x> (sha256 <12>)`
  and the read-back to run on the station after loading, or `NOT READY` with the failing rows. Exit 0 ready,
  1 not ready, 3 env. It never contacts a station. See `build-verify.md` § Ready to load.
- `--ready` refuses every lint-run option (`--lint-only`, `--write-baseline`, `--no-baseline`, `--console-dir`,
  `--profile`, `--legacy`, `--wiring-map`) with exit 2 and a first line naming it, instead of ignoring it.

### Fixed — report-module --ready: non-part gradle directories and the unbuilt artifact line

- `report-module --ready` no longer reports a gradle directory beside the module parts (`buildSrc/`, a shared
  conventions build) as an unbuilt artifact: an artifact is a directory with `module-include.xml`, or a
  `<MOD>-rt/-ux/-wb/-se/-doc` directory with its gradle build file.
- The NOT READY line of an artifact with no built jar says `(not built)` instead of a `?` version.

### Fixed — scaffold-module: retro rows S1 and S2 of the 2026-11-15 coordination stretch

- A SIGINT, SIGTERM or SIGHUP while scaffold-module emits a module removes the partial `<out-dir>/<ModuleName>`
  the way a write fault does ("interrupted (SIGINT)", then "removed the partial ...") and ends with 128 + the
  signal, so the next run is not refused with "destination already exists".
- `tests/scaffold-module.bats` pins the literal substitution of option values (TC-S2a: a `--plugin-version`
  holding the fixture's target version is written as typed) and the discard of a failed emit (TC-S2b: a kit
  without its version-ledger template leaves the out-dir empty); reverting either fix turns its case red.
- `toolbelt/scaffold-module.sh` lists the exit codes 129, 130 and 143 (SIGHUP, SIGINT, SIGTERM while the module
  is emitted), and `tests/scaffold-module.bats` is split by concern into `tests/scaffold-module*.bats` files under
  200 lines with a shared `tests/lib/scaffold-setup.bash`; every case keeps its name and body.

### Changed — the SH2 home lint catches more runs and has its own file

- `tests/shell-hygiene.bats`: the SH2 lint (a bats case that runs a `--home` script on the real home) moves into
  `tests/lib/home-lint.awk`. It now flags a run after `if`, `elif`, `while` or `until`, the first word of a
  `$( )` or backticks given as a variable, and a script variable assigned after `local` or `export` or inside a
  one-line function. A trailing comment no longer satisfies the `--home` or `HOME=` test, and an assignment
  inside a function body is no longer reported as a run.
- The forms the lint is held to are a table, `tests/lib/home-lint-forms.tsv`, checked by `SH2-forms`; the
  evasions the lint accepts are listed in the lint header. `SH2-marker` checks that the `--home` marker finds
  both scripts that take `--home`.

### Fixed — install-skill: review findings of #694 (retro row T10 of the 2026-11-15 coordination stretch)

- A diverged prompt of a skill whose kit has no `skill/prompt.md` names its template as "the built-in default (the kit
  has no <path>)" instead of a `skill/prompt.md` path that does not exist.
- New tests pin the dry-run digest labels (rendered for a prompt, tracked for a launcher), the refusal of a whole
  `--harness all` run before any harness is written when one harness is diverged or holds a symbolic link, and
  that a refused `--home ""` writes nothing anywhere.

### Fixed — install-skill: a file that cannot be placed is refused before the first write (retro rows T3 and T4)

- `install-skill.sh` now also checks, before the first write, that every file of every selected harness can be
  placed: a non-directory where its `skills/`, skill or `prompts/` directory must be, or a target that exists and is
  not a regular file, refuses the run with `refusing to install: not a directory: <path>` or `not a regular file:
  <path>` (exit 3, "move it away, then re-run"). Before, the launcher of a harness was installed and its prompt then
  failed, leaving `SKILL.md` without its prompt. `--dry-run` output is unchanged. Retro 2026-11-15 T4 (3) residual;
  the other T3 and T4 findings were already answered by #694 and T10.

### Changed — merge-when-green exit 10 for a merge call lost on the network (retro row T33b of the 2026-11-15 coordination stretch)

- `scripts/merge-when-green.sh` exits 10, not 8, when every send of `gh pr merge` failed on a network error and the
  PR is still OPEN: GitHub refused nothing and nothing merged, so run it again. Exit 8 stays GitHub's refusal; every
  other exit is unchanged.

### Fixed — the ntpl lint run is bounded by entries read too (retro row T34 of the 2026-11-15 coordination stretch)

- `n4lint lint-ntpl-manifest-deps` stops a run after 100,000 entries read, as well as after 128 MiB inflated:
  empty or tiny entries drew nothing from the byte budget, so many of them could keep a run going for minutes. The
  stop is one FAIL row, "100000 entries read in all, the bound of one run -- <entry> and everything after it not
  read" (exit 1).

### Fixed — ci-local --changed kit-links runs the bats it checked (retro row T36 of the 2026-11-15 coordination stretch)

- The kit-links step of `scripts/ci-local.sh --changed` runs the bats binary whose version it compared with the
  ci.yml pin, by its absolute path, so a relative PATH entry cannot swap in another bats inside the snapshot.
- A bats version mismatch names the pinned version, the version found and the binary's path.
- A pinned-tarball download (shellcheck, bats) starts no retry after 60 s (`--retry-max-time 60`), so a blackholed
  network exits 3 within about 3 minutes instead of up to ~8.

### Fixed — build.sh keeps a jar that passed the gate when interrupted during the report

- A signal while report-module ran rolled back the jar that had already passed the gate; the gate now marks
  modules/ as verified when it passes.
- A gradle whose child kept the output open past 10 s no longer fails a passing build (the `WaitDelay` added in
  #795 is removed; only the signal handler's wait is bounded).

### Fixed — build.sh no longer keeps a stale mirror copy when interrupted

- On a mirror home, the step that keeps a replaced jar (`.mirror-replaced/`) now waits for an interrupt's
  restore to finish, so it no longer keeps a copy or prints a "replaced" row for a jar the restore puts back.

### Added — build.sh takes the schema-risk baseline from a git ref

- `build.sh --schema-baseline-ref <deployed-ref>` runs the archive command the schema-risk SKIP row prints, for the
  run: before gradle, every artifact under `<module-root>/<MOD>` is extracted as it is at the ref into a temporary
  directory, `<artifact>/.deploy-baseline` links to it, and both are removed when the run ends. An already-deployed
  module whose deployed jars are not at hand no longer ends at exit 53 after a manual SAFE run.
- Exit 2 before gradle for a ref that is not a commit, a module directory absent at the ref, an artifact that already
  has a `.deploy-baseline` (never moved or overwritten; a dangling link a killed run of the option left is removed with an INFO line, an operator link is refused and kept), or `--first-deploy`; an artifact new since the ref gets an
  INFO line and keeps its SKIP row. The INCOMPLETE line now names the option.

### Changed — build.sh help names what an interrupt does

- `build.sh --help` says that SIGINT/SIGTERM/SIGHUP before a passing chain restore the pre-build `modules/` jars
  (exit 128 + signal) and that the process group must be interrupted, not build.sh's pid alone.

### Fixed — build.sh passes gradle an absolute niagara_home

- `build.sh … ../mirror-415` (a relative `niagara_home`) failed in gradle, which runs in the gradle root where the
  relative path names another directory. build.sh now hands gradle the absolute path; its own messages still name
  `niagara_home` as given.

### Fixed — build.sh stops gradle before it puts modules/ back on an interrupt

- A SIGINT, SIGTERM or SIGHUP sent to build.sh's pid alone used to leave gradle running, and it could install
  its jar after the restore. The handler now takes the restore lock, passes the signal on to the gradle the run
  started and waits for it (killing it after 10 s, with a bounded wait after the kill), and no gradle starts once
  a signal arrived. A gradle daemon (outside a mirror home) cancels on its own.

### Added — build.sh signal handler ordering test (waits for the chain's own restore)

- `TestASignalWaitsForTheChainsOwnRestore` pins that a signal arriving while the build chain restores `modules/`
  or cleans up waits for it before touching anything (a mutation that drops the handler's lock fails it).

### Fixed — kit-cite-resolve no longer reads the text between two code spans as a repo path

- In live prose, a backtick that closes one code span no longer opens a `repo-path` pointer: in
  `` `--src`go/x.sh`--y` `` the `go/x.sh` between the spans is prose, not a cited path.

### Changed — field commissioning notes

- `BUILD-LOOP.md` § 6: save the station and stop running write batches before a module install (the install
  restarts the station).
- `types/driver-authoring.md` § 11.2: the BACnet Discover Networks filter; a point's object instance is not the
  controller terminal. § 11.3: one writable Modbus proxy point per register per device (a duplicate is reset to
  `Hex:-1`); use the stock Modbus Async managers.

### Added — kit-cite-resolve OUTSIDE_REPO and live repo paths

- `kit-cite-resolve` reports an `[ev: …]` token naming a `~/` or absolute path as `OUTSIDE_REPO` (a strict finding
  on live doctrine, `HISTORICAL_DRIFT` on a history card) instead of resolving it by existence on one machine.
- On live doctrine, a backticked `toolbelt/`, `scripts/`, `go/` or `tests/` path of prose must exist under the kit
  or the repository (`MISSING_FILE` otherwise), so deleting a file fails `--strict` until the doctrine naming it
  is reworded; the generated `toolbelt/INDEX.md` and `go/bin/` pass. Live docs naming deleted libs were reworded.

### Added — client-identifier guard at commit time; servlet write template and lint-servlet-write-guards

- `.githooks/pre-commit` and `.githooks/commit-msg`: niagara-go's `client-guard` now also runs on the staged
  changes and on the proposed commit message, so a client name is refused before a local commit exists (the
  pre-push guard is unchanged). The hooks hand the guard a snapshot commit written to a throwaway object
  directory; the repository's objects and index are never written. `scripts/install-hooks.sh` activates them
  with the other hooks and names them.
- `build-n4-module-kit/types/servlet-write.md`: a copy-ready servlet `doPost` that verifies the Niagara CSRF
  token (`CsrfUtil.verifyCsrfToken`) and the caller's permission on the written TARGET
  (`target.getPermissions(cx)`), never on the servlet or the user.
- `build-n4-module-kit/toolbelt/lint-servlet-write-guards.sh` (Go `go/internal/lint/rules/servletwriteguard`,
  a report-module member): FAIL `self-permission` for `getPermissionsFor(this)`, a permission read off the user
  and similar; WARN `no-target-permission` for a slot write with no target check before it, and `no-csrf` for a
  writing `BWebServlet` with no CSRF check.

### Fixed — path guards of orient-guard and mirror-niagara-home

- `orient-guard.sh` resolves the longest existing ancestor of the module root and judges the rest by its text,
  so a new directory several levels deep inside the prefix passes, spelled absolute or relative. The FAIL row
  names the prefix that was compared (the seam one when set), every final slash of `HOME` is dropped, and an
  argument starting with `-` is a usage error (exit 3; spell such a path `./-name`).
- `mirror-niagara-home.sh` runs guard A on a mirror dir that is not there yet before it creates anything, so a
  refused mirror dir leaves no directory behind; one that cannot be resolved is refused (exit 20). A mirror dir
  that passes is made where guard A judged it, so `<src>/new/../../out` no longer creates `<src>/new`, and
  guard B judges that same path, so `missing/../dir` no longer writes into a non-empty `dir` without the marker. A symlinked
  mirror dir is refused (exit 20) whatever its spelling: `link`, `link/` and `link/.` no longer replace the link
  or empty its target.

### Fixed — path guards: resolution after "..", nothing made before the checks

- `orient-guard.sh`: once a `..` after a missing directory climbs back into existing directories, their
  symlinks are followed again (as `realpath -m`), so `<prefix>/missing/../linkout/x` with `linkout` leaving the
  prefix FAILs where it leads. A final newline is part of a name: a sibling `Cliente\n` is not the prefix.
- `mirror-niagara-home.sh`: every spelling of the mirror dir is judged by that same walk, and the guards, the
  wipe and the rebuild all use the judged path, so `ml/../o` (with `ml` a link to `<out>/o`) refreshes the
  mirror at `<out>/o` instead of deleting it and failing. Nothing is created before every check has passed
  (a refused `missing/../srclink/a/b/c/x` or a relative source leaves no directory). A source or mirror dir
  starting with `-` (`-` alone too) is a usage error (exit 1).

### Fixed — kit rare-case follow-ups (lint-logger-name, L17, preflight version-drift)

- `lint-logger-name` reads `moduleName` with gradle comments blanked: a commented-out old name is no declaration.
- `lint-structure` L17 WARNs when a gradle file ends inside an unclosed `/* */` comment or text block (no
  `moduleName` after it is read), and pins that a Groovy `'/*'` string is not a comment.
- `preflight` version-drift skips `<repo>-worktrees/` checkouts and the gradle root's own ancestors (the repository
  root), which are not sibling modules.

### Fixed — client guard at commit time and lint-servlet-write-guards: the verifier's follow-ups

- `.githooks/commit-msg` (via `.githooks/pre-commit --message`): comment lines are guarded too (`-m`, `-F` and
  `--cleanup=verbatim` keep them; a comment naming a client is now refused even where git would strip it), and a
  scissors-shaped line in a `-m`/`-F` message no longer hides the lines below it: below a `commit -v` cut only the
  lines of the diff git appends are dropped.
- `lint-servlet-write-guards`: FAIL `self-permission` also for `super.getPermissions(cx)` and for a local bound to
  the user or the servlet (`BUser caller = op.getUser(); caller.getPermissions(op)`, `self = this`);
  `no-target-permission` WARNs unless the check names the written receiver and its result is used, counts
  `invoke(…)` as a write, and accepts a `requireTargetWrite` declared in the file only when it checks its first
  parameter; `no-csrf` needs a `verifyCsrfToken(` call or `CsrfProtectedFilter`, not a variable named `csrfToken`.
- `types/servlet-write.md`: fact #3 names the cached-permissions early return of `BComponent.getPermissions`
  (proxies only).

### Added — mcp-n4: acknowledge alarms by source, class or time

- `n4_ack_alarm` takes `select: {source?, alarm_class?, before?, max?}` instead of `uuids` (exactly one of the two; at least one of `source`, `alarm_class`, `before`). `source` matches a component and everything below it in both stored spellings (`slot:/a` and `local:|station:|slot:/a`), `before` is a BAbsTime or a span such as `24h`, `max` is 1 to 100 (default 100). The dry run resolves the selection to exact uuids with the records query and where clause `n4_read_alarms` uses, unacknowledged records only, newest first, and returns them in `plan.selection.records` (`{uuid, source, alarm_class, timestamp}`) with `matched` and `truncated`. Each resolved record goes through the existing checks (write scope, already acknowledged).
- The confirmation token is bound to that exact uuid list: the execution acknowledges the uuids its dry run resolved and never re-runs the selection, so an alarm raised in between is left alone; an execution without its dry run is refused. A selection that matches nothing is a dry run with no token. Calls with `uuids` answer exactly as before.
- A newer selection dry run that matches nothing (or is refused) drops the held list, so an older token for the same selection can no longer execute.

### Added — mcp-n4 alarm health (station-audits S4)

- `n4_alarm_health(window, chatter_threshold, unacked_days, top)`: chattering sources, alarms still out of normal,
  alarms unacknowledged for more than N days and the busiest sources of a window, from two fixed reads of `alarm:`
  grouped by the server. Read-only.

### Added — mcp-n4 alarm routing audit (station-audits S5)

- `n4_list_alarm_classes(audit=true)`: the classes no recipient is linked to, and each recipient's settings and the
  problems that stop delivery (no or invalid email address, no remote station, empty transitions or days, an empty
  webhook/Telegram endpoint, a last delivery failure). Credential slots are named, never returned. Without `audit`
  the reply is unchanged.

### Added — mcp-n4: n4_bql_query hints the $24 form on an empty escaped-literal result

- An empty `n4_bql_query` result whose quoted literal holds a `$xx` escape carries a `hint`: BQL decodes the
  escape (`$20` is a space) while slot names are stored escaped, so `name = 'A$2420B'` matches `A$20B`.

### Added — mcp-n4: `n4_connect` `expected_version` (field retro delta U14)

- `n4_connect(expected_version=...)` refuses the session unless the detected version equals it or
  starts with it by whole dotted parts (`4.14` matches `4.14.0.162`, `4.1` does not); an undetected
  version never matches. Without the argument a connect is unchanged. `expected_station` already
  pinned the station name.

### Fixed — mcp-n4 n4_copy_component of a writable Modbus point can keep a valid address
- A Modbus client device allows one writable point per register: a copy of a writable Modbus point keeps its source's address, and the station resets it to `Hex:-1` ("Duplicate Writable point for register address"). Proven on a 4.15.6.20 lab station; the copy body was not the cause.
- New optional `copies[]` (`name`, `data_address`, `address_format`): each listed copy gets its own `proxyExt/dataAddress` inside the same add, so the read-back can verify it. Refused for a source whose proxyExt declares no `dataAddress`, a name outside `new_names` or given twice, and an empty address.
- The plan notes each copy that would share a register with its source or an earlier copy; the read-back of such a copy adds a `cause` next to its differences.
- Docs: a Modbus proxy created with `proxy_ext` starts at `hex` address `0` (set it through the proxy's ORD with `slot="dataAddress/..."`); a 4.15 lab station needs `--allow-tier-b NAME`.

### Fixed — mcp-n4: dangling outputs count alarm and history extensions as consumers (OD007)
- `n4_find_dangling_outputs` no longer reports a point whose `out` is read by its own alarm or history extension (no link involved); it lists it in `consumed_by_extension` with the extension type. The commissioning report's `dangling_outputs` check skips it too (client A had 13 such false positives).

### Added — mcp-n4: `n4_deploy_template` deploys a component template (.ntpl) with its settings

- `n4_deploy_template(template_file, parent_ord, name?, settings?)` adds the template component of a local `.ntpl`
  under `parent_ord` in one op, with a dry run and a confirmation token like every write. `settings` fills the
  manifest's exposed settings (`num` checked against `min`/`max`, `bool`, `str`); a required setting the template
  leaves empty must be given; an unknown or metadata setting, a station or application template and a type the
  station's registry does not describe (a missing module) are refused before a token.
- The plan notes the template, its dependencies, each setting's value and slot, and what is not carried: the
  template configuration and every reference by template handle (links, ORD values), passwords, slot facets and
  categories. The read-back checks every planned node; the inverse removes the deployed component.
- Live-certified on the 4.15.6.20 lab station (deploy, read-back, rollback).

### Changed — mcp-n4: n4_set_schedule and n4_invoke_action are destructive (M19)
- Both are now listed with `destructiveHint: true`: `n4_set_schedule` replaces every event of the schedule (`n4_rollback` still restores it exactly), and `n4_invoke_action`'s `active`, `inactive` and `auto` have no inverse. `n4call` therefore executes them only with `--execute`, `--allow-destructive` and the line's own `"execute": true`.
- benchcall's runner field `dryRunOnly` is now `writesExecute` (it held the effective mode, not the flag) (M17b).

### Added — n4_station_health device_detail (down-device report)

- `n4_station_health drivers=true device_detail=true` adds, per bad device, `fault_cause`, `last_fail_cause`,
  `last_ok`, `last_fail`, `address` (known per-type slot only) and `stale_points`, and per network `port` and a
  `pattern` hint (`bus` when all but at most one device are down, `single` when one device alone is down).
  Two more BQL requests at most, whatever the device count; unreadable values are `null`. `device_detail`
  without `drivers` is refused; `drivers=true` alone is unchanged. Field retro delta U12.
- The `bus` hint names wiring, termination and polarity only on a network with a serial port; on
  any other (BACnet/IP) it names the shared network path. A network with no serial port reports
  `port: null`, and a network with no pattern reports `pattern: null` (found in the live check).

### Added — mcp-n4: driver health per network

- `n4_station_health(drivers=true)` adds `drivers`: per driver network, its field devices counted as `down`, `fault`, `disabled` (the flag or `enabled` false) and `unknown`, and its proxy points (a control point with `control:NullProxyExt` is no driver point and counts nowhere) counted as `stale`, `fault`, `down`, `disabled` and `unknown`, with the bad ones listed up to `max_bad` (default 20) and `*_bad_truncated` markers; `drivers.totals` sums them and adds `unassigned_points`, `proxy_unknown` and `rows_truncated`. A status with several bits counts once in each of its categories. It makes no extra request: it groups the rows the networks, devices and points sections already read (`status.bits`, `enabled`, `proxyExt.type`). Without `drivers` the reply is unchanged.

### Added — mcp-n4: `n4_histories_audit`, what is wrong with a station's histories (station-audits S1)

- `n4_histories_audit(base, exclude_services=true, stale_factor=3, stale_min=10m, cov_max_age=24h, capacity_warn=0.9, max_histories=500)` lists, in one
  read-only call, the control points with no history extension (`no_history`), the disabled history extensions
  (`disabled`), the stale histories (`stale`: last record older than the larger of `stale_factor` x the collection interval and
  `stale_min`, or than `cov_max_age` for COV, or never recorded; service points left out of `no_history` by default), the record-capped histories close to capacity (`near_capacity`, warning
  for full policy `stop`, info for `roll`) and the history ids several extensions share (`colliding_ids`). Each list
  keeps 100 items; `counts` holds the totals. Two fixed BQL reads plus one `select count(*)` per distinct
  record-capped id that has recorded (`count_reads`); a cut read sets `truncated` and leaves `no_history` unchecked.

### Added — mcp-n4: `n4_export_history`, histories to one CSV for client reports (station-audits S2)

- `n4_export_history(history_ids, period | start/end, max_rows=100000)` writes 1 to 20 histories to ONE CSV
  file, `<state-dir>/exports/<station>-<UTC time>.csv` (directory 0700, file 0600, never overwritten; the
  server names it, the caller never passes a path). Columns `history_id,timestamp,value,status`, oldest first,
  with machine-encoded time and value (ISO-8601 with the station's offset, `21.0` not `21,0`). The reply is
  `{path, rows_total, per_history [{history_id, rows, truncated}], bytes}`. Ids are checked like
  `n4_read_history`'s before any request; the station is only read.

### Added — mcp-n4: `n4_read_history` `cov_hint`, a COV tolerance suggested from real data (station-audits S3)

- `cov_hint: true` suggests a COV `changeTolerance` from the same machine-encoded read the summary uses
  (both together still read once); report only, nothing is written. It returns the |delta| distribution of
  successive records (`median`, `p90`, `p99`, `noise_floor`), the tolerances tried and
  `suggested_tolerance`: the smallest 1-2-5 value at or above the median non-zero |delta| for which a
  simulated COV (`|value - last kept| > tolerance`) keeps at most half the records, with `reduction_estimate`.
  A non-numeric history is `cov_hint: null` with `cov_hint_note`. The reply without `cov_hint` is unchanged.

### Added — mcp-n4: `n4_inventory_snapshot` and `n4_inventory_diff` (station-audits S8)

- `n4_inventory_snapshot(base, depth=6, max_components=20000, slots)` writes the components of a subtree (slot path,
  type, handle, configured property values or the chosen slots) to `<state-dir>/exports/inventory-<station>-<time>.json`
  with one BOX load. The type contract decides what is runtime: transient slots (instance or contract, such as `out`,
  the inputs other than `in1` and `in8`, `lastRecord`), a computed Status `out` and linked inputs are left out; a Status config slot such as
  `fallback` is kept.
- `n4_inventory_diff(a, b, limit=200)` compares two snapshots by slot path from the local files only: `added`,
  `removed`, `changed`, `type_changed`, `handle_changed`, `counts`, `truncated`, `notes`. A name with `..` or a path
  separator, or one not under the exports dir, is refused.

### Added — mcp-n4: licence planner in `n4_license_info`, device limit check in `n4_create_component` (field retro delta U8)

- `n4_license_info` adds `usage`: per global capacity rule (`devices`, `points`) `used`, `limit`,
  `headroom` and `excluded_modules`. `used` counts as the station's capacity recount does
  (`driver:Device` / `driver:ProxyExt` components, less the modules the `tridium:globalCapacity`
  feature lists in `excludedDevices` / `excludedPoints`), with one BQL select per rule. An unlimited
  or absent limit is `limit: null`; a count that cannot be read is `used: null` with a `note`.
- `n4_license_info(plan: {add_devices?, add_points?})` adds `plan_result: {fits, over: [{rule, by}]}`;
  `fits` is `null` when a named rule's count or limit is unknown, never guessed. Without `plan` the
  reply keeps its fields.
- `n4_create_component` refuses, in the dry run, a `driver:Device` subtype whose creation would take
  the counted devices over `device.limit`. Non-device types are not checked; an unreadable licence or
  count is never a refusal (a fixed plan note says the limit was not checked).

### Added — n4_create_link batch form (field retro delta U11)

- `n4_create_link` takes `links`: 1 to 50 entries `{source_ord, source_slot, target_ord, target_slot}` under one
  dry run and one confirmation token (105 links had taken ~41 min one by one). Each entry is planned, inverted and
  read back exactly as the single form; the batch refuses a mix with the single-form arguments, the same link twice
  (keyed on the station's handles, so another ORD spelling of the same component is caught), an entry identical to
  an existing link, and any entry whose target is outside the write scope. The batch spends one write of
  `--max-writes` per link and is refused before its token when the remaining budget cannot hold it. The read-back
  reports `{source_ord, source_slot, target_ord, target_slot, requested, observed, verdict}` per entry; the batch
  takes the entries' verdict when they all agree, else `partial`. `n4_rollback` removes every link of the batch.
- The single form's plans and replies are unchanged. Its four arguments are no longer `required` in the schema, so
  a missing one is refused as a tool error with the same text (`missing required argument: source_ord`) instead of
  a -32602 JSON-RPC error.

### Added — mcp-n4: platform capacity and expiry

- `n4_station_health(platform=true)` adds `platform` (station-audits S13): `heap` (used and committed from `spy:/sysInfo`, the maximum from the heap pools of `spy:/util/JMX Info`, and the fraction), `cpu` and `memory` (the system section, which now loads the SystemService once more when its CPU usage slots are still at -1, not yet polled), `history_db` (histories and records summed from `spy:/localHistory`; bytes `null`, no read-only page gives the file size), `licence` (the nearest expiry of the license section, `never_expires` when nothing is dated), `certificates` (the station's own key store certificates from the `CertificateExpiryPoint` of each `Services/SecurityService/certificates` entry, then the user and system trust store certificates of `spy:/securityInfo/Cryptography Info`, with `days_left`, nearest first per store), `modules` (the loaded parts' count, the list with `include_modules`) and `warnings` (heap at 90% or more, the licence and each station or user certificate expired or within `expiry_warn_days`, default 30; the system trust store's JDK roots as one informational line). An item that cannot be read is `null` with a `*_note`. Without `platform` the reply is unchanged.

### Added — mcp-n4: `n4_px_binding_audit`, the Px bindings whose ORD does not resolve (station-audits S15)

- `n4_px_binding_audit(root, file_ord, base, max_views=50, max_bindings=500)` audits, read-only, every Px view
  below `root` (relative ORDs read against the component the view is on) or one `file_ord` with an optional
  `base`. Each distinct target is loaded once: `resolves` when it loads, `missing` (listed under `broken`) only
  when the deepest ancestor that loads holds no slot by that name, loaded or declared; everything else
  (another scheme, a chained ORD, a relative ORD with no base, a handle, a failed load whose parent still holds
  the name) is `unknown` with a `why` and never counted as broken. `n4_px_summary` is unchanged.

### Fixed — mcp-n4: whole-range history extremes leave out null-status records (M16b)
- When a range is larger than `summary_max_rows`, `range_min` / `range_max` now come from `select min(value), max(value) where status.null = false`, matching the summary's rule (a new COV history's `0.0 {null}` records no longer pull `range_min` to 0). A `nan` or `inf` record still counts in the station's aggregate, and the note says so.

### Added — mcp-n4: `n4_read_audit`, who changed what from the station's AuditHistory (station-audits S7)

- `n4_read_audit(target | target_glob, slot, user, operation, period | start/end, limit=100)` reads
  `history:/<station>/AuditHistory` and returns its records newest first, each
  `{timestamp, operation, target, slot, old_value, value, user}` (timestamp ISO-8601 with the station's offset), with
  `truncated` and `total_records`. `target` matches a component and everything below it; `slot` and `user` are exact;
  `operation` is one of the AuditEvent operations the audit history keeps. Every filter value is checked on its
  decoded form before any request. Read-only.

### Added — mcp-n4: n4_refrigeration_faults, refrigeration fault rules over history (O1)
- Read-only report over history data: `short_cycling`, `defrost_termination` (ended by temperature / time / early / open), `setpoint_not_reached` (with an optional exclusion history such as defrost), `over_amps` per clock hour, `door_drift`. Every threshold is a required argument; each history is read once per call with ISO timestamps and locale-free values.
- Findings per rule as `{count, items, more}`, a `summary` line per rule, and a rule whose history could not be read is `not_checked`. The rules are pure functions in `go/internal/mcpn4/refrig`.

### Fixed — n4_replicate_points keeps a write for its rollback and folds a trailing-slash target

- `n4_replicate_points` refuses a replication that would leave no write of `--max-writes` for its own
  `n4_rollback` (one that used the last write could be rolled back only after a server restart).
- Two targets that differ only by a trailing `/` are one target and are refused as a duplicate (they put two
  copies under one parent).

### Added — mcp-n4: `n4_replicate_points` (bulk replication under one token)

- `n4_replicate_points(source_parent_ord, points, targets)` copies M points of one parent onto N
  target parents: N x M adds in one batch under one confirmation token (at most 200), each the
  body `n4_copy_component` would add (address and `dataType` kept), with per-target `copies[]`
  name and `proxyExt/dataAddress` overrides. Each add spends one `--max-writes` write; the plan
  notes the count and a replication larger than `write_budget.remaining` is refused before any
  token. A client B field job had spent 630 calls and 39 minutes on 112 points.

### Changed — mcp-n4: copy and replicate dry runs note a non-null fallback

- `n4_copy_component` and `n4_replicate_points` note every writable point whose `fallback` is not
  null: each copy commands its device with that value on first communication. A note, never a
  refusal.

### Fixed — mcp-n4: n4_rollback restores the configuration of frozen children

- The rollback of an `n4_remove_component` now writes back the configuration of a frozen child (a
  component the station creates with its parent, e.g. an alarm extension's `offnormalAlgorithm` or a
  writable's `proxyExt`): the dry run plans `frozen_config` on the component spec from the station's type
  contracts (type swap plus set ops for the captured configuration slots, never readonly, transient or
  status slots), so the token covers it; a value the station refuses stays in
  `frozen_config_not_restored` with the station's reason in `refused`. The first remove + rollback of a
  freshly alarmed point now brings back the algorithm type and its `highLimit`/`lowLimit` (retro
  2026-11-16 row M18); what the depth-3 snapshot did not capture (`limitEnable` bits) still cannot.

### Added — mcp-n4: n4_save_station backup=true zips the station directory (O5)
- With `backup: true` and a `--station-home`, after the save the station directory is zipped to `<state-dir>/backups/<station>-<UTC time>.zip` (0600, never overwritten) without what the BackupService excludes by default (`*.hdb`, `*.adb`, `*.lock`, `*backup*`, `console.*`, `config.bog.b*`, `config_backup*`; the `history`, `alarm`, `webFileCache` directories). The reply's `backup` lists path, files, bytes, sha256 and the excluded entries, or `{error}` with verdict `mismatch`. A symlinked station home is resolved; a directory without `config.bog` and an open or linked `backups` directory are refused.
- Without a station home the dry run refuses the backup. A Fox/platform `.dist` backup for remote stations is a later campaign.

### Changed — n4_save_station proves persistence without --station-home, and retries a config.bog caught mid-rewrite

- Without `--station-home`, `n4_save_station` now reads the station's own save job: the `baja:StationSaveJob`
  children of `Services/JobService` are listed before the save and polled after it. A new job that ends `success`
  gives `persisted: true` (`evidence.route: station save job`, `evidence.job`); `failed` or `canceled` gives
  `false`; no new job, a job still running at the timeout or an unreadable JobService stays `unknown` with a hint
  (overamps retro row M5). The file servlet never serves `.bog` files, so config.bog metadata is not readable
  remotely.
- With `--station-home`, a `config.bog` that is unreadable right after the save is read once more after 1 s
  (`evidence.retry`), so a save caught between the backup move and the rename of `config.bog.working` is no longer
  reported `unknown` (overamps retro row M9).

### Added — mcp-n4: `n4_schedule_audit`, every schedule's next event, holidays and links (station-audits S12)

- `n4_schedule_audit(base, horizon="7d", max_schedules=500)` reads, in one read-only call, every boolean, numeric,
  enum and string schedule and every calendar below `base` (`select slotPath, knobCount from
  schedule:ControlSchedule`): per schedule `out_value`, `next_time`, `next_value`, `special_events` `{count,
  upcoming}` (the events, or a calendar's dates, whose first day falls in the horizon from the station's date, with
  references followed to their calendar and `calendars: [{ord, resolves}]`) and `linked_targets` (the knob count).
  `unlinked` lists the non-calendar schedules nothing reads, `no_next_event` those whose next time is null; a value
  that cannot be read is null with a note, and a rule the audit does not evaluate is noted, never guessed.

### Security — the scope-root rollback exception covers create batches only

- The same-scope rollback of the write-scope root (added the same day) also admitted copy, replicate and
  deploy batches that named a component after the root, and could delete the root with hand-made children
  inside. It now applies only to `n4_create_component` batches, whose rollback refuses when anything else
  sits inside the root.

### Changed — n4_rollback can undo a create of the write-scope root under the same scope

- A create of the scope root itself was allowed by that scope but its rollback was refused as outside it
  (the rollback checked the parent). The rollback is now checked on the created path, as the create was,
  while nothing was written inside the root later; `n4_remove_component` of the root stays refused.

### Added — mcp-n4: n4_semantic_export, Haystack JSON of the hs: tags (O10)
- Read-only export of the components a NEQL query matches (default sites, spaces, equips and points) as a Project Haystack grid in the Haystack 4 JSON encoding: ids from slot paths, `dis`, every `hs:` tag and outbound `hs:` relation as a ref. Values that cannot be converted are listed in `skipped`; `save_as` writes a private file under `<state-dir>/exports/`.
- The conversion is pure functions in `go/internal/mcpn4/semantic`. Brick output is not produced.

### Added — n4_set_slot batch form (OD040)

- `n4_set_slot` takes `writes`: 1 to 50 entries `{ord, slot, value, value_type, status?, simple_type?}` under one
  dry run and one confirmation token (overamps retro row M6, backlog OD040). Each entry is planned and read back
  exactly as the single form; the batch refuses a mix with the single-form arguments, the same (component, slot)
  twice, and any entry outside the write scope. The batch spends one write of `--max-writes`, as `n4_set_tags` does. The read-back
  reports `{ord, slot, requested, observed, verdict}` per entry, `verified` only when all are; `n4_rollback`
  restores every entry; `incoming_links` carries each entry's links with its `target_ord`.
- The single form's plans and replies are unchanged. Its four arguments are no longer `required` in the schema, so
  a missing one is refused as a tool error with the same text (`missing required argument: ord`) instead of a
  -32602 JSON-RPC error.

### Added — mcp-n4: n4_set_slot dry run lists the links that will overwrite the value (OD001)
- When links write the target slot, the plan carries `incoming_links` (`source_ord`, `source_slot`, `target_slot`, `enabled`) and a note that the written value is overwritten at the link's next update. The token covers the list, so a link added after the dry run refuses the execute. The write itself is still planned.

### Changed — mcp-n4 skill: rename, template deploy and write-budget rules

- `mcp-n4-kit/skill/SKILL.md` Hard Rules: renaming stays a Workbench step (never create + remove: history and
  links are lost); `n4_deploy_template` for repeated equipment (what it drops); plan bulk jobs against
  `write_budget.remaining`, the save being outside the budget.

### Added — mcp-n4: Spyder section of n4_commissioning_report

- `n4_commissioning_report` section `spyder` (asked in `sections`, or added by `spyder_spec`): lists each
  Honeywell BACnet Spyder left disabled, with `deviceStatus` other than 1, `compileStatusFlag` other than 0
  or a non-empty `sErrorStatus`; with `spyder_spec` (`{<Spyder>: {<point>: {sensor_type, input_low,
  input_high, output_low, output_high, terminal}}}`) it compares each IOComp with the table. The default
  sections are unchanged.

### Added — mcp-n4: `n4_tag_coverage`, how far a station's Haystack tagging reaches (station-audits S9)

- `n4_tag_coverage(base, dictionary?, max_points=2000, exclude_services=true)` lists, in one read-only call, the
  control points missing `hs:point` or an outbound `hs:equipRef` relation (`points_without_tags`, with what each
  misses), the `hs:equip` components no point references (`equips_without_points`) and the direct tags no installed
  dictionary defines (`unknown_tags`, with count and an example; a namespace with no dictionary installed is one
  `ns:*` entry, so a station without a Haystack dictionary says so once). Implied tags and relations count. Lists
  keep 100 items; `counts` holds the totals; a read cut at `max_points` sets `truncated`.
- Bulk tagging (S10) needs no new tool: `n4_set_tags` already adds and removes on up to 50 ords under one dry run and
  token; a new test pins N points with add and remove together.

### Added — mcp-n4: `n4_users_audit`, the station's user accounts without a credential (station-audits S14)

- `n4_users_audit(unused_days=90, max_users=500)` lists, read-only, every `baja:User` under
  `Services/UserService` with its name, enabled, lock-out and expiration state, roles (resolved against
  `Services/RoleService`, `admin_role` when a role is `baja:AdminRole` or grants `super`), auth scheme and
  `last_login` (the newest `Login` record of `history:/<station>/SecurityHistory`), and flags `disabled`,
  `expired`, `locked`, `no_recorded_login`, `unused`, `admin`, `unknown_role`, plus `unused`, `counts`,
  `masked_slots` and `notes`. Fail closed: only allowlisted slots are returned, `email` and `cellPhoneNumber`
  show only `<set>` or `""`, and the authenticator, password hash, permissions and any other slot are only
  named in `masked_slots`. A verdict the evidence cannot support is null, never guessed.
- An allowlisted user slot typed `baja:Password` or holding a non-scalar value is masked instead of echoed.
- `no_recorded_login` (was `never_logged_in`) and `unused` no longer trust a rolled SecurityHistory: they claim only what the records read cover, and `logins_since` names that window.

### Added — mstp-plan: MS/TP address planner

- `build-n4-module-kit/toolbelt/mstp-plan.sh` (Go `go/internal/mstpplan`): per device the MAC, the DIP switches
  to turn ON and the device ID, as a markdown or CSV table; refuses a reserved MAC, one above maxMaster, one the
  switches cannot set, or a MAC / device ID already on the trunk (`--existing`).

### Fixed — ng-deploy: station-first expected signer, NIAGARA_HOME must be a directory

- With no `niagaraSigningAlias` configured, the deploy gate's expected signer is now taken from the module's jars
  the station holds before the copy, and from the jars that were in `<NIAGARA_HOME>/modules` only when none of the
  station's is signed; the two sources are never mixed.
- A `NIAGARA_HOME` that names no directory, or names a file, is refused with exit 10 before the backup and before
  gradle runs, with a line that names the value.

### Fixed — ng-deploy: the env file may not replace the schema-risk tool

- `SCHEMA_RISK_BIN` in the ng-deploy env file is refused (exit 10, file and line): it would put any executable in
  the place of the schema-risk precheck. The environment of the command still sets it.

### Fixed — ng-deploy: the home's signer is kept when the station gives none

- With no `niagaraSigningAlias`, the deploy gate again expects the signer of the jars that were in
  `<NIAGARA_HOME>/modules` whenever the station's jars give none (unsigned, two signers, or only the development
  signer); before, a station jar signed with the development key or two station signers left the gate with none.
- A station jar whose signer cannot be told (unreadable, not a zip, a `.SF` without its block file) gives no
  expected signer and a WARN naming it, and the home's signer is not taken in its place.
- `NIAGARA_HOME` reaches gradle as an absolute path, and one under a directory the deploy may not search is
  refused as a permission error.

### Fixed — ng-deploy keeps the signer check on when a station jar cannot be read

- A station jar whose signer cannot be told (an interrupted copy, a damaged jar) no longer makes the gate accept any
  signature: the jars moved aside from `<NIAGARA_HOME>/modules` give the expected signer, with a WARN. Before
  #781 the home's signer was enforced in this case; #787 had switched the check off.

### Fixed — build and deploy safety gates

- **Behaviour change.** `toolbelt/build.sh`: a SIGINT, SIGTERM or SIGHUP between the baseline snapshot and a passing
  chain puts the pre-build `modules/` jars back (and removes a first-deploy jar) before the temporary directory is
  removed, with `build.sh: interrupted (<signal>) before the gate passed; …`; the exit stays 128 + the signal.
- **Behaviour change.** `scripts/ng-deploy.sh`: `--no-deploy` (or `NO_DEPLOY=1`) when `STATION_MODULES_DIR` is
  `<NIAGARA_HOME>/modules` is exit 10 before the backup and before gradle (`--no-deploy refused: …`).
- **Behaviour change.** `scripts/ng-deploy.sh`: `MODE=B` in the env file is exit 10 with file and line unless
  `--mode` is given, and `VERIFY_MODULE_BIN` unless `--no-gate` is given, as `NO_GATE=1` already was; the
  environment of the command still sets `VERIFY_MODULE_BIN`.

### Added — preflight remote-backup row

- `preflight.sh` WARNs (`remote-backup`) when the gradle root's git checkout has no remote, or its branch has no
  upstream, so its commits exist on one disk only; pushing or backing up stays the operator's decision. Silent
  outside git, on a detached HEAD and with an upstream.

### Changed — mcp-n4: rollback outside the write budget, in-doubt replication rollback, held-register note (retro mcpn4-replicate-points Δ5, Δ6)

- **Every `n4_rollback` is outside `--max-writes`**, like `n4_save_station`: it runs with the budget spent and spends
  no write, nested components and links included. `write_budget` gains `rollback_counts: false`; the budget refusals
  say `n4_save_station and n4_rollback still run` / `are outside the budget`. A replication of N adds now needs N
  writes (was N + 1: #777's write kept for the rollback is gone).
- `n4_rollback` of an in-doubt `n4_replicate_points` batch whose failed add is journaled removes only the copies the
  station accepted (by the name its reply assigned) that the read-back finds with the type the batch added; the
  failed add, the adds never sent changed copies and copies a later batch wrote under are kept, each with a verdict in the plan notes and in
  `observed.copies`; a rollback that keeps a copy is `partial`.
- The replication dry run reads each target four levels deep and notes a copy at the register a writable Modbus
  point already on that device holds.

### Fixed — n4_replicate_points no longer flags Hex:-1 as a shared register

- The dry run's shared-register note treated `-1`, the address the station resets a duplicate writable point
  to, as a register; points and copies at `-1` are no longer noted.

### Changed — scaffold-module: `--vendor` is required and version options are shape-checked (retro build-kit-bench-session Δ4, Δ8, Δ11, Δ17, Δ18)

- **Breaking:** `scaffold-module.sh` no longer guesses the vendor. A run without `--vendor` exits 2 with
  `scaffold-module: --vendor is required: …` and the usage, before anything is written. `--target-version` must be
  `<major>.<minor>` and `--plugin-version` `<major>.<minor>.<patch>` (digits only); another shape is exit 2 naming
  the option. Every kit caller, bats case and the CI scaffold-diff steps pass `--vendor`.
- Every scaffold type passes `report-module.sh --lint-only` with zero FAIL and zero WARN, and verify-module's
  `facets-req` and `palette-root` checks with no WARN: MIN facet and floor guard on the logic `interval`, unit
  facets on the `setpoint` slots, `b:UnrestrictedFolder` palette roots, `<Type>.displayName` lexicon keys.
- Every fixture, so every scaffold, carries a `.gitignore` (`.gradle/`, `build/`, `_backups/`, `.last-deploy-sha`).
- `skill/SKILL.md` points only at files that exist, holds no per-user path, maps each decision-table row to a
  scaffold `--type`, and describes the scaffold once; `BUILD-LOOP.md` § 1.a matches what the scaffold emits.

### Fixed — scaffold-module picks the gradle plugin of the target release

- Without `--plugin-version`, `scaffold-module.sh` now bakes the Niagara gradle plugin that the `--target-version`
  release ships (4.13 7.3.40, 4.14 7.6.17, 4.15 7.6.22). A 4.15 scaffold used to pin 7.6.17 and failed preflight
  `plugin-pin` on a 4.15 niagara_home.

### Added — mcp-n4: BACnet discovery and Honeywell Spyder commissioning over BOX

- `n4_rename_component` renames up to 50 dynamic slots under one dry run and token (RenameOp), with an
  inverse and a read-back.
- `n4_discover_devices` runs the network's Device Manager discovery (Who-Is, "Send Global") and lists each
  answer with `mac_display` exactly as Workbench's MAC Addr column prints it (`BBacnetAddress.bytesToString`:
  MS/TP in decimal, BACnet/IP port in hex) and `objects` (`listSize`), marked `registered`, `new` or
  `address_differs`; registered devices that stayed silent go under `not_answering`. Default wait 20 s
  (Workbench's 10 s missed MS/TP Spyders).
- `n4_discover_points` runs a device's point discovery job and lists every object with name, value and
  description, the station point that already proxies it, and for a Spyder a `hardware` block (hw id from
  `analogValue:0`, model, I/O).
- `n4_invoke_action` accepts, on a `honeywellBacnetSpyder:BacnetSpyder` only: `writeDeviceInstance` (read-back
  asks the controller its instance), `modelChange` (arg = model display name; inverse back to the previous
  model; read-back checks the model and `compiledFwModel`), `compile` and `spyderDownload` (arg `full` or
  `quick`; the read-back waits for the job and returns its state and log, and flags a download that skipped
  the compile step).
- The server reconnects once when the station drops the BOX session ("Could not find Server Session"): a
  read or dry run is retried, an execution is not (its token died with the session) and says to redo the
  dry run.
- `modelChange` calls its undo partial: changing the model back does not restore the Sylk devices or terminal
  assignments the first change dropped.

### Fixed — mcp-n4: BACnet octet strings are read back by their octets

- A `BacnetOctetString` slot (a MAC) is compared by its octets, so a write of the same address no longer
  reads back as a mismatch.

### Fixed — mcp-n4 link batch recovery and licence check follow-ups

- `n4_create_link` batch form: when the station's replies cannot be trusted and one entry's lookup
  fails, that entry carries `lookup_error` and the other entries keep their inverse, so
  `n4_rollback` still removes the links that were found (it used to journal an empty inverse).
- `n4_create_component`: a device create under a licence with no `device.limit` no longer runs the
  station-wide device count in its dry run.

### Fixed — build.sh and report-module skip a module's docs/ folder; a FAILed report restores the mirror jar

- An artifact is a directory with `module-include.xml`, or a `<MOD>-rt/-ux/-wb/-se/-doc` part with its build file,
  for report-module (lint run and `--ready`) and build.sh (`--schema-baseline-ref`, the INCOMPLETE pointer) alike.
  A `docs/` folder beside the parts gets one INFO line instead of a schema-risk ERROR (build exit 10) or an
  INCOMPLETE verdict (build exit 53).
- When the report-module punch-list FAILs (build exit 50), build.sh puts the pre-build jar back into the mirror's
  `modules/`, as a failed verify gate does, so a fix rebuilt at the same version is not reported as drift (exit 51).

### Added — report-module `bundle-gzip`; static serving of a pre-compressed bundle; parallel SPA writer briefs

- `report-module.sh` adds a `bundle-gzip` row on every `-ux` part that has a `src/rc/dist/`: a `*.js` there over
  300 KB with no `<name>.gz` next to it is a WARN row (advisory, never fails the report), none is one PASS row,
  and a part with no `src/rc/dist/` gets no row (retro kit-gzip-bundle-writer-briefs Δ1).
- `types/dashboard.md` § Static serving: the bundle step writes deterministic `.gz` twins and the static handler
  serves them with `Content-Encoding: gzip` and `Vary: Accept-Encoding`; `types/frontend-standard.md` § 5 and
  § Enforcement point to it.
- `ORCHESTRATION.md` §3.c and the `BUILD-LOOP.md` § 2 parallel writer checklist: a brief names the module-map row
  that owns each path, and each writer runs `lint-module-map.sh` and the whole `npm test` before reporting done
  (retro kit-gzip-bundle-writer-briefs Δ3).

### Added — lint-dashboard-write-save; lint-delays reads a Math.max floor in either argument

- `build-n4-module-kit/toolbelt/lint-dashboard-write-save.sh <module-root>` (Go `go/internal/lint/rules/dashboardsave`,
  a report-module member run once per module): FAILs every write
  line of a `-ux` part (`.set(`, `.add(`, `.remove(`, `.invoke(` whose last argument is the request
  `Context`/`WebOp`, and `addSpecialEvent(`) when nothing in the module requests a station save
  (`BStation.save`, `Sys.getStation().save()`, `Station.save*(`) and `MODULE-MAP.md` declares no
  `persistence:` line. Issue #821: a dashboard lost its writes on restart and a driver re-sent a stale value.
  A module with no `-ux` part is a SKIP row in the report.
- `build-n4-module-kit/types/dashboard.md` § Write path: the debounced station-save exemplar and the
  `persistence:` declaration; BUILD-LOOP §6.b item 6 links the rule.
- `lint-delays`: `Math.max(1L, x)` is a floor like `Math.max(x, 1L)` (inline, local binding and helper body);
  it used to FAIL `zero-floor cannot determine Math.max floor`.

### Fixed — mcp-n4: `n4_copy_component` keeps the status of Status values

- A copy carries the status of a Status value (StatusBoolean, StatusNumeric, StatusEnum, StatusString)
  whole with its value whenever it is off its declared default, as `n4_set_slot` writes a Status slot.
  The status child is declared readonly, so it was left out and every copy fell back to `{ok}`: a schedule
  whose `in` and `defaultOutput` held `{null}` was copied with a live `out` (false / 0.0) that would have
  switched units OFF and set setpoints to 0 once linked (delta P2, live near-miss 2026-11-19).
- The read-back compares that status on each copy: a copy whose status differs from the source is a
  `mismatch`, never `verified`. `n4_replicate_points` shares the same capture and gains both.
- A status at its default (`{ok}`) is still left out, so a plain copy plans the same ops as before.

### Added — mcp-n4: `n4_read_history` window, status filter and history type; `n4_histories_audit` flat histories

- `n4_read_history` takes `from` / `to` (the same window as `start` / `end`, one pair or the other) and
  `status` (`not-ok`, `ok`, or one flag such as `fault`), filtered by the station so `limit`, `truncated` and
  `total_records` count only the matching records: the `{fault}` / `{ok}` pairs a comm drop leaves in a COV
  history are one call, not a hand scan of timestamps (delta M2).
- `history_type=true` states whether the history records on change of value (`cov`) or every interval (with
  its period, e.g. `5m`), from the collection interval of the extension that records into it.
- `n4_histories_audit` `flat=true` lists the histories holding one value for more than `flat_days` days
  (default 2) as a possible device placeholder, with the point and the value: a device that sends 0.0 for
  "no sensor" no longer hides behind `{ok}` records (delta M3).
- Without the new arguments both tools send the same requests and answer the same reply as before.

### Added — mcp-n4: `n4_create_link` dry run states what each link will command

- The dry run lists `link_sources`, one entry per link (single and batch form):
  `{source_ord, source_slot, target_ord, target_slot, value, status}`, the value and status the
  source slot holds now, which the link drives into the target as soon as it is made.
- An `{ok}` source linked into a writable's control input (`in1`..`in16`, `fallback`) adds a
  `note`: "this link will command <target> to <value> {ok} as soon as it is made". Linking 28
  schedule copies at false / 0.0 `{ok}` would have switched units OFF and set setpoints to 0
  (delta P2, live near-miss 2026-11-19); only a manual read of `out` caught it.
- `link_sources` is volatile and outside `plan_hash` (a new dry-run-only plan preview), so a
  source whose value moves between the dry run and the execute keeps the token valid. The
  source is now loaded at depth 2 (the same single read) to see its slot's value and status.

### Fixed — mcp-n4: a second lost session is reported, and BQL flags a column that is no slot

- After a station restart the station answers "Could not find Server Session". A read or dry run
  already reconnected once with the last `n4_connect` and was retried once; a second loss in a row now
  answers a named error (reconnected, lost again, check the station and call `n4_connect`) instead of the
  raw BOX text, and is never looped. A write execution that loses its session answers
  "session lost: ... redo the dry run" and is not retried (delta U4).
- `n4_bql_query` adds a `warnings` entry for a projected column that is `NULL` on every row and names no
  slot of the queried type, read from the type contract (e.g. `maxOverrideDuration`, a facet key, gave
  135 rows of `"NULL"`). A column null on some rows only, or a real slot null everywhere, gives no warning;
  a normal reply is unchanged (delta M1).

### Fixed — mcp-n4: `n4_read_history` ok / not-ok status filters

- The `ok` and `not-ok` filters returned no records on a live station (`status.ok` is no BQL field: zero rows, no error); they now filter on `status.isOk`, live-checked on 4.14.0.162.

### Changed — build-module does its file work itself

- `build.sh` (`go/cmd/build-module`) no longer runs `cp`, `mkdir`, `mktemp`, `unzip` or `df`: the pre-build jar copy
  keeps the mode, times and (where the user may) the owner, a temporary path is made under `$TMPDIR` (else `/tmp`), the
  module version of a jar is read with `archive/zip`, and the WSL 9p/drvfs probe reads the mount table (a mount
  matches by whole path components, so `/mnt/cc` is not under a mount at `/mnt/c`). A host without those commands
  builds the same; none of them is on the `PATH` the chain needs any more (`git` and `tar` stay, for
  `--schema-baseline-ref`).
- Diagnostics reworded, exit codes unchanged: a temporary path that cannot be made is
  `build.sh: cannot make a temporary directory in <dir>: <reason>` (exit 1; it was `mktemp: failed to create ...`); a copy that fails names the path
  (`cannot read <src>: ...` / `cannot write <dst>: ...`); a directory that cannot be made is
  `build.sh: cannot create directory <dir>: <reason>`. A copy onto the file it copies, or onto a directory, is refused
  and never truncates the source.
- The bats cases that stubbed `unzip` on `PATH` (`BS-drift-unverifiable-unzip`, `-restore-fail`,
  `BS-drift-empty-listing`) are Go tests with the fault injected into the jar. Retro `2026-11-14-go-build-module-port` Δ3.

### Changed — ci-local reads the bats pin in Go

- `scripts/ci-local.sh` reads the bats pin through `ci-steps bats-pin`; an "Install bats" step that sets `BATS_VER=` or `BATS_SHA256=` twice, or malformed, now exits 3 instead of taking the first value.

### Added — a module is installed into the Niagara version it was compiled for

- `ng-deploy.sh` exit 55: after the build and the version-bump precheck, a target installation (the `modules/baja.jar`
  of `STATION_MODULES_DIR`) whose Niagara major.minor differs from the one a built jar was compiled against (the highest
  Tridium dependency floor in its `module.xml`) is refused before the schema-risk precheck, the gate and the copy; the
  line names both versions and the installation. A jar with no recorded floor and a `baja.jar` that cannot be read are
  exit 55 too. A station directory with no `baja.jar`, `--no-deploy` and a station that is `<NIAGARA_HOME>/modules` are
  not checked.
- `toolbelt/check-target-version.sh <jar> <niagara_home>` (`go/internal/niagaraversion`): the same comparison for a copy
  made by hand; one PASS or FAIL row with both versions, exit 0 / 1, exit 2 when a version cannot be read.
- `build.sh` against a mirror ends with one line naming the installation the mirror was made from, its Niagara version,
  that the jars built were NOT installed there, and the command that installs them (`modjar.MirrorSource` reads the
  marker's `source=`). Nothing is printed for a real `niagara_home`, a marker without a source or a failed build.

### Changed — a held jar says which station to stop

- The hint after a copy or move a running station refused (`ng-deploy.sh` exit 40 or 10) and the line after `build.sh`
  exit 32 now say to stop the station and close Workbench of that installation first, and that on Windows the station is
  a service or `station.exe`. The deploy text of `build.sh` and `ng-deploy.sh --help` list the new exit and line.

### Added — two lints on the repository's own tests and doctrine

- `go/internal/cardwidth`: every live doctrine card keeps its lines to 120 characters where a line can be wrapped;
  `BUILD-LOOP.md` and `skill/SKILL.md` are wrapped (no text change), and the other cards are held at the overlong-line
  count in `go/internal/cardwidth/baseline.txt`, which may only go down. Runs with `go test ./...`. Retro
  `2026-11-14-go-toolbelt-small-port` Δ5.
- `go/internal/batshome`: a bats suite that runs a Go launcher under a changed or unset `HOME` must build the binary in
  `setup_file` (or in a helper it loads); `build-sh`, `check-skill-drift` and `kit-resolve` gained that prebuild.
  Retro `2026-11-14-go-toolbelt-small-port` Δ4.

### Added — a throwaway signing profile for a first build

- `build-n4-module-kit/templates/signing-profile.xml`: a commented profile with the fully qualified `profileType`, the store keys and the `keytool` command with the code-signing key usage; the scaffold's signing line and `BUILD-LOOP.md` § 1.a point at it.

### Changed — smaller kit messages and checks

- `toolbelt/verify-module.sh` usage opens with its own title, without the shellcheck directive and its rationale.
- `toolbelt/count-retro-deltas.sh` says on stderr to regenerate the index when it has no row for a retro; `tests/count-retro-deltas.bats` CRD4 runs it over every retro dated after 2026-10-24 (two named mismatches excepted).
- `tests/bats-skip-allowlist.txt` header: `^*` is refused because POSIX leaves it undefined, not because GNU grep reads it differently; compare with `/usr/bin/grep`, since `grep` may be ugrep.

### Changed — lint-servlet-write-guards scope, interrupt cleanup, preflight delivery row, contrast rules

- `lint-servlet-write-guards` reads the servlet's dispatch class: a servlet that names a type of its own directory
  that verifies the CSRF token (types/dashboard.md DUX1) no longer gets the `no-csrf` WARN. It also narrows what a
  write is: a one-argument `set(v)`, a numeric first argument (JDBC `setInt(1, v)`, `List.set(0, x)`) and a variable
  declared only as a `PreparedStatement`, `List`, `Atomic*`, `Calendar` or `ThreadLocal` are no longer
  `no-target-permission` or `no-csrf` WARNs.
- `build.sh`: an interrupt ends what gradle itself started (a single-use daemon forked by the wrapper's client)
  after gradle ends and before `modules/` is put back, so it cannot install a jar after the restore.
- `preflight.sh`: a PASS `delivery` row in a kit checkout names what pre-push asks of a kit change (changelog
  fragment, ledger fragment, pending retro, or the trailers).
- `types/frontend-standard.md` § 10: contrast is checked per token pair (WCAG AA check beside the Node tests), a
  theme change re-maps every colour token, and faded state is a muted colour, never `opacity` on text.

### Fixed — kit-cite-resolve reads a backticked file with a colon-repeating line list

- A prose citation such as `` `toolbelt/build.sh:15,:82-88` `` (the colon repeated before each range) is a pointer now,
  checked like `:15,82-88`; before it was not read at all, so a stale line went unreported. Retro
  `2026-11-14-go-build-module-port` Δ4.

### Added — lint-gradle-userhome

- `build-n4-module-kit/toolbelt/lint-gradle-userhome.sh [--strict] <root>` warns when `gradle.properties` sets `user.home` (`org.gradle.jvmargs` with `-Duser.home=`, or `systemProp.user.home`); advisory, not in `report-module`.

### Added — lint-mqtt-tls-port

- New `toolbelt/lint-mqtt-tls-port.sh [--strict] <station-dir>...` (WARN, advisory, ledger item G87): a
  `GenericMqttAuthenticator` of the `abstractMqttDriver` module that connects over TLS (the default
  `UserLoginOverSSL`, or `AnonymousOverSSL`) and leaves `brokerPort` at 1883 (`types/cloud-connector.md` § 9). Reads
  every `*.bog` under the station directory. Not run by `report-module`.
- `xmlsrc.Doc` gains `SlotPath` and `SlotValue`, which `lint-bacnet-status-flags` now uses instead of private copies.

### Added — lint-rdbms-dialect-descriptor

- New `toolbelt/lint-rdbms-dialect-descriptor.sh [--strict] <module-root>...` (WARN, advisory, ledger item G88): a
  `new B(SqlServer|MySQL|Oracle|HsqlDb)HistoryExport(` in `-rt` source fixes the database dialect in code, and a
  `jdbc:postgresql:` literal in a module with no class extending `BRdbms` reaches for a dialect the platform does not
  ship (`types/cloud-connector.md` § 11). Not run by `report-module`.

### Changed — mcp-n4: read timeouts retried once, slot filter, status flag names, terminal outputs, set-flags note

- A read that times out is asked again once after 0.5 s: the `loadSlots` request of a load and the GET of a BQL or ORD read. A write (`syncTo`, `invokeAction`, `save`) is never sent twice by the client, and a failure that is not a timeout (an HTTP status, a refused connection, a login) is final.
- `n4_read_slots` takes `slots` (1 to 100 names) and returns only those slots, in station order, with `slots_not_found` for the names the component neither holds nor declares. Without `slots` the reply is unchanged.
- `n4_set_slot` `status` accepts flag names (`fault`, `down,overridden`, `{alarm,unackedAlarm}`, `ok`) besides the hex bits, written whole with the value; the refusal of a bad status names the flags. The recorded plan goldens carry the new refusal text.
- `n4_find_dangling_outputs` lists a writable point with a proxy extension and a `schedule:CalendarSchedule` under `terminal_by_design` (with the reason) instead of `dangling`, only when there is one; the commissioning report skips them too.
- `n4_set_flags` adds a plan note when a slot with no flags that no contract declares sits under a type chain that ends at a module type the station answered with `{}`: the slot is taken as dynamic, but may be a frozen slot of an undescribed type, and the inverse `""` would not restore it. A `baja` type answered with `{}` ends every normal chain and adds no note.
- The `n4_bql_query` refusal of a `|` in the query names the fix: put the subtree in `base`. The recorded bql goldens carry the new text.
- The four pending retros of 2026-11-10 to 2026-11-15 are folded: every delta is APPLIED, ALREADY LANDED, DECLINED or DEFERRED (live station) in its row; README, METHODOLOGY section 3 and the skill say so, and the skill adds that what a wire sheet or a Px page draws only a person can confirm.

### Changed — mcp-n4: token refusal after a lost session, failed Spyder download detail

- A confirmation token issued in an earlier station session (the station dropped the session, or `n4_connect` ran since) is refused naming that cause and the way out: call the same tool with the same arguments and `dry_run=true`, check the new plan, execute with the new token. The refusal sends nothing and issues no token. A write execution that loses its session says the same.
- A failed or canceled `spyderDownload` / `compile` read-back adds `log_error` (the station's error when the job log could not be read), `device_state` (`deviceStatus`, `compileStatusFlag`, `sErrorStatus`) and a hint naming a renamed `BacnetNIFolder` slot as the known cause. A successful job is unchanged.
- A write whose path is below a Spyder `ControlProgram` that the station refuses names the cause: the program unloads on its own and must be loaded again.

### Added — ng-deploy mode D and --plugin-version

- `scripts/ng-deploy.sh --mode D` deploys the `-rt` and `-wb` jars of a module with a workbench part (the `driver` scaffold's pair): the same build, backup, type-count verify, gate and all-or-nothing copy as mode A, with `EXPECTED_WB_TYPES` in the env file. Mode C still copies `-rt` only.
- `scripts/ng-deploy.sh --plugin-version <major>.<minor>.<patch>` forwards `-PniagaraPluginVersion` to every gradle call, as `build.sh` does; another shape is exit 10.

### Changed — four tools say more or refuse more

- `scripts/ng-deploy.sh`: the exit list of `--help` now names code 1 (the backup directory could not be made); the invalid-mode messages read "A, B, C, D".
- `toolbelt/preflight.sh`: a `plugin-pin` FAIL names the plugin versions the installation ships and the option that passes one; the version-drift walk stops at the filesystem root, the temporary directories and the home directory, and skips a directory it cannot read without a `find:` line (it took about 28 s under a temporary directory).
- `toolbelt/report-module.sh`: given a gradle root (a directory whose children hold the module parts) it exits 3 naming `<root>/<MOD>` instead of reporting INCOMPLETE; a verify-module row whose jar path holds a space keeps it out of the detail.
- `toolbelt/sweep-fold-audit.sh --strict`: a pending retro dated since `--deltas-since` whose every delta is cited now fails with `pending, but all N deltas are cited`.

### Fixed — run-pure-test: default-package classes, a symlinked Gradle cache, the jar copy it picks

- `run-pure-test.sh <module-rt-dir> <Class>` runs a test in the default package
  (`srcTest[/test]/<Class>.java`); it looked for `<Class>/<Class>.java` and always ended "test source not found".
  The not-found message names the path it looked for.
- A `~/.gradle` that is a symlink (a shared or relocated cache) is followed, as `find -H` does; before, JUnit was
  reported missing (exit 3) from a cache that held it.
- With several copies of `junit-4.13.2.jar` or `hamcrest-core-1.3.jar` under `~/.gradle`, a NOTICE line on stderr
  names the jar, how many copies there are and the one used (still the first in name order). A cache with one copy
  prints nothing new. Retro `2026-11-14-go-toolbelt-small-port` Δ3.

### Added — scaffold-module --type library

- `toolbelt/scaffold-module.sh <ModuleName> <out-dir> --vendor <v> --type library` emits a library module of
  `types/utility-lib.md` Shape A (an `-rt` part with `<types/>`, no Java, no palette) from the new
  `fixtures/MinimalLibrary`; its `MODULE-MAP.md` declares `type: library`, so `lint-structure` passes it clean. The
  fixture has not been built with gradle. The usage line and the unknown `--type` message name the four types.

### Fixed — lint-servlet-write-guards no longer drops one-argument generated setters

- `lint-servlet-write-guards`: the "fewer than two arguments" exemption now applies to the bare `set(v)` only
  (`AtomicReference`, `ThreadLocal`). A generated property setter takes one argument and is a component slot write,
  so `comp.setDouble(42.0)` / `n.setBoolean(true)` again get `no-target-permission` and `no-csrf` (the narrowing of
  the previous fold had hidden them). A numeric first argument still drops a call only when a value follows it
  (JDBC `setInt(1, v)`, `List.set(0, x)`).

### Added — signing-doctor: check the signing material before the build

- `build-n4-module-kit/toolbelt/signing-doctor.sh` (Go `go/internal/signingdoctor`): `--cert` FAILs a CA certificate (`CA:TRUE`), a certificate with an extended key usage that lacks code signing, a key usage without digitalSignature and an expired certificate, and WARNs on a certificate with no usage extension; `--key` FAILs a PBES2 AES-GCM private key export (the Workbench export, unreadable by OpenSSL 3) and says to convert it with Java. Exit 0 pass, 1 a FAIL row, 2 usage.

### Changed — lint-structure L5 row says how to declare a library

- The undeclared `-rt` row of L5 now ends with `(a library module declares `type: library` in its MODULE-MAP.md header,
  types/utility-lib.md § 6)`, as the `-ux` L9 row does for a theme. `lint.ReadMapHeader` reads the modularity and the
  `type:` kind of a module map in one pass (`lint.MapHeader` is unchanged), and `lint-structure` uses it for the kind.

---

### References

- Fold campaign build-n4-module-kit/odd/tasks/fold-pending-2026-11-23.md (PRs #833-#844); engram topic odd/fold-pending-2026-11-23/tasks.

## [v0.33.0] - 2026-10-25

### Added — `run-niagara-test`, the guarded station-test runner

- `run-niagara-test run` builds into a disposable copy of the Niagara home, never `--install`, refuses an install a
  running station holds, and proves the install unchanged after the run (#556). `verdict --testng` reads the TestNG
  stderr summary and the -1 exit of `test.exe` (#555).
- The NRE user home is sandboxed through `WSLENV`; writes to the real user home fail the run (#557).
- New options: `--module-verification low` for the disposable copy (#559), `--exclude-jar` to keep stale install jars
  out of the copy (#563), `--reuse-home` to boot again on a kept home (#579).
- A run whose build changes the gradle root fails, with an opt-in restore (#564).
- Ignored TestNG methods fail the run unless allowed by name (`--allow-ignored`) or, when the source cannot name the
  set, by count (#571, #573, #575).
- A failed run keeps a redacted log with the TestNG results, full failure traces and full stderr (#566, #572, #581).

### Added — lints and scaffold

- A production `module-include.xml` must not register test types (`test-type-in-prod`, also checked by
  verify-module) (#569).
- lint-structure L16 fails a declared signing profile whose refresh margin re-mints its key (#574).
- New lints for bats assertions that `set -e` cannot see (#576) and for test methods that cannot fail or never run
  (#577, pinned in bats by #580).
- `scaffold-module.sh` emits a station-test skeleton for the logic and dashboard types, plus a signing-profile
  preflight gate (#570).

### Fixed

- LTF1 reads a class javadoc that names a link target past `@NiagaraType` (#565).
- triageconsole treats the remaining Spanish JDK 8 log levels as headers (#553).
- `go_tool_bin` walks only what `go build` reads, and now sees edits to embedded non-Go assets (#562).

### Changed — CI, tests and docs

- CI runs parallel jobs, bats with `-j 4` (fail-closed on the test count) and a Go build cache (#554); superseded PR
  runs are cancelled and draft PRs are skipped (#560).
- The mcp-n4 Go box live differential replays from a recorded golden (#552).
- The c8-close SC16 pin is retired because its snapshot fixture is gone (#578).
- Station-test campaign retro and round-10 tail retros folded into the kit docs (#561, #567, #568).

---

## [v0.32.1] - 2026-10-25

### Fixed — mcp-n4 Go server closes the station session on Ctrl-C/SIGTERM

- The Go `mcp-n4` server (the default) died by SIGINT, SIGTERM or SIGPIPE and left its BOX session open on the
  station. It now cancels the station requests in flight, deletes the session (bounded by `--http-timeout`, 20 s by
  default, even right after `make`) and exits 0 like the Python server; a second signal forces the exit, and a client
  gone mid-reply ends with exit 1 and the session closed. See `mcp-n4-kit/docs/go-port-plan.md` section 22.

---

## [v0.32.0] - 2026-10-25

### Highlights — quality rounds 8-10

- **Go ports.** The kit's parsing tools and lints run on the Go module (`go/`): each `toolbelt/*.sh` name is kept as a
  thin `go_tool_bin` wrapper, and every port was proven byte-identical to its bash before its truth fixes landed.
- **mcp-n4 defaults to the Go server**, with the Python server kept for one release as a fallback (details below).
- **Driver authoring kit DR1-DR5.** Driver doctrine (`types/driver-authoring.md`), a driver scaffold, driver lints,
  a driver mode of the verify gate and a worked driver exemplar. [PR #498, #502, #503, #510, #534]
- **One golden Runner.** Go ports replay their goldens through the shared `golden.Runner` and record them with one
  `golden/record/record.sh` (hermetic environment, the whole post-run tree); the private loops and recorders are
  gone. [PR #516]
- **kit-cite-resolve and citation fixes.** False-positive classes fixed in the citation parser and resolver, DRIFT
  for anchored guide pointers, and dotted section paths read whole. [PR #499, #509, #514, #515, #517]
- **Local client gate findings.** `lint-slotomatic-markers` FAILs a `TYPE`/`getType()` declared by hand outside
  the generated region [PR #540], and the `report-module` summary names a skipped floor check beside a FAIL
  [PR #539].
- **Retro fold.** Every pending kit retro of the round is folded into the kit doctrine (`BUILD-LOOP.md` § 5,
  `build-verify.md`, `types/moduleTest.md`); no retro is left pending.

### Changed — mcp-n4 defaults to the Go server

- `go/bin/mcp-n4` is now the default mcp-n4 server: every read and write tool is ported, and the in-doubt matrix is
  proven on both servers. The Python server stays selectable for one release as a fallback (`MCP_N4_IMPL=python`,
  `live_smoke.py --impl python`, and the README `.mcp.json` fallback example); it will be removed after that release.
  CI gains a bounded `mcp-n4-go` job (15 min) that runs the black-box suite against the Go server; the Python run
  stays as the fallback check. See `mcp-n4-kit/docs/go-port-plan.md` section 20.

### Added — client-identifier guard in `.githooks/pre-push`

- The pre-push hook now runs niagara-go's `client-guard` first, with the hook arguments and the whole stdin:
  a push whose added lines or commit messages match the private client denylist is refused (report:
  `file:line: rule #N`, never the matched text). Without `client-guard` installed the hook refuses when a
  denylist exists and warns otherwise. `tests/pre-push-client-guard.bats`; the existing hook suites pin
  `NIAGARA_CLIENT_DENYLIST` to an absent path to stay hermetic.

### Fixed — build-n4 and mcp-n4 kits (retro deltas from the 2026-10-06 CompPan HMI support session)

- `toolbelt/build.sh` resolves ROOT to an absolute path before the gradlew walk-up (a relative `.` looped forever) and
  fails fast, naming `chmod +x`, when a `gradlew` exists but is not executable; the build-n4-module SKILL documents the
  `chmod +x gradlew` step after `git worktree add`. Behaviour change: the FIRST non-executable `gradlew` met on the walk-up now exits 10 instead of being walked past to an executable ancestor one. (sdd-investigacion#1924, #1925)
- `toolbelt/kit-ticket.sh`: with `gh` present but no known host / no auth, the ticket is written to
  `retros/tickets/` and the script exits 4 (typed, not a silent success); any other `gh` failure keeps the file too and exits 3. (sdd-investigacion#1928)
- mcp-n4 SKILL: read an operator-made config value back live before proceeding; prefer the station log over oBIX
  `about/serverBootTime` for restart timing. (sdd-investigacion#1926, #1929)

---

## [v0.31.0] - 2026-10-03

### Added — build-n4 kit deferred lint candidates (`odd/tasks/deferred-lints-2026-10-03.md`)

The three lint candidates that earlier folds recorded as DEFERRED, and the project-level split-package check of #142.
Each one was written test-first with fixtures and an observed mutation per guard, landed as its own PR with a granted
native RDD review, and was smoke-run read-only on real trees before release. Each review's fail-open was fixed in a
sub-task (anti-cascade policy); every other advisory is parked on #226.

- **D2** `toolbelt/lint-size.sh` (advisory): a class over 800 hand-written lines (slot-o-matic regions excluded) or a
  pure-class method over 80 lines; unclosed, nested or stray region markers are their own WARN rows. Run by
  `report-module.sh` through a new `advisory_member` helper (WARN rows relayed, any non-zero member exit is an ERROR row
  with the member's reason). [PR #235, #237]
- **D3** `toolbelt/lint-license-isoperational-gate.sh` (advisory): a licensed class whose `changed()`, action or
  servlet-write callback acts without an unqualified `isOperational()`/`isFault()`/`isFatalFault()` gate in its first
  statement. Run by `report-module.sh`. [PR #237, #239]
- **D4** `toolbelt/lint-set-null-ord.sh` (advisory): a `getSlotPathOrd()` result (null on an unmounted component)
  reaching a setter with no null guard, directly or through a same-method local; guards are tracked per receiver. Run
  by `report-module.sh`. [PR #239, #242, #244]
- **D5** `toolbelt/split-package-check.sh [--strict] <project-root>` (project level, #142): a Java package declared in
  two or more modules of one checkout; `--strict` exits 1 for a client-CI gate. Where it runs automatically is an open
  operator decision on #142. [PR #242, #244]

### Fixed — build-n4 kit

- **D1** the six T3/T4/T5a hot-path lints fail closed (exit 3) when `find` cannot enter a sub-directory, through the
  shared `toolbelt/lib/scan-files.sh` (#226 lead item); the lister walks a `.` / dot-named root and rejects a file root.
  [PR #230, #233, #235]
- **D1** folds the polish close retro: anti-cascade policy up front (`ORCHESTRATION.md` §7), structural rules for
  heuristic parsers (`METHODOLOGY.md`), the bash 5.2 `patsub_replacement` and bats `! cmd` traps (`CONTRIBUTING.md` §2,
  new guard `tests/shell-hygiene.bats` SH1), and the `candidate_context_unavailable` recovery (`ORCHESTRATION.md` §4).
  [PR #230, #233]

Close retro: `build-n4-module-kit/retros/2026-10-03-deferred-lints-2026-10-03-close.md` (5 Δ, pending).

### Fixed — `mcp-n4-kit` v0.5.1: token station binding and plain-ORD write scope

- A confirmation token is bound to the station session it was issued on: the plan names `station: {name, base_url, session_id}`, so a token cannot execute after a reconnect or on another station (audit F1). [PR #229]
- The write scope accepts only plain slot ORDs: `|h:`, `|slot:..` and any other hop after `station:|slot:` are refused, and `--write-scope` prefixes are checked at startup (audit F2). [PR #229]
- `--token-ttl` and `--max-writes` must be >= 1 (audit F16). [PR #229]

### Fixed — `mcp-n4-kit` v0.5.2: pause station calls after an authentication failure

- Any `AuthError` (login, oBIX about, `/ord` GET, BOX call) pauses every station call for `--auth-cooldown` seconds (default and minimum 30), with the reason and the seconds left. Only other credentials may reconnect during the pause, which keeps a model from walking the account into the station lock-out (audit F4). [PR #231]

### Fixed — `mcp-n4-kit` v0.5.3: nested set_slot, compare-before-restore rollback

- `n4_set_slot` on a nested slot (`grp/sp`) reads its real previous value. Before, it read the type default, which made the journaled inverse wrong (audit F3). [PR #232]
- `n4_rollback` refuses to restore a slot or fallback that changed after the batch wrote it, and shows both values. A later change is never overwritten silently (audit F6). [PR #232]
- The "slot not found" refusal of `n4_set_slot` explains that the station omits a slot at its type default and names the routes (audit F10). [PR #232]
- The authentication latch also records a second rejected credential on clients without the auth hook (H2 review). [PR #232]

### Fixed — `mcp-n4-kit` v0.5.4: write-branch test debt

- Tests cover the previously untested write branches (audit F13): status on a plain type, missing targets, component and relink journal-intent failures, malformed relink records, `config.bog` lost after a save, and startup exit 2 for a chained `--write-scope`. [PR #234]
- A rollback that stops because its component intent cannot be journaled now says so. Before, it said only `ToolError`. [PR #234]

### Added — `mcp-n4-kit` v0.6.0: BQL output_file; duplicate and localized headers

- `n4_bql_query` takes an optional `output_file` (`<name>.csv` or `.json`). The rows are written privately to `<state-dir>/bql/<name>` (0600, never overwritten) instead of the reply. [PR #236]
- A repeated CSV header is kept as `Type#2`, `Type#3` instead of losing a column (audit F7). [PR #236]
- `n4_inventory` and the `name` decoding of `n4_bql_query` resolve columns by their position in the select list, so a station with localized display headers works (audit F12). [PR #236]

### Added — `mcp-n4-kit` v0.7.0: n4_list_batches, snapshot truncation report, timing flags

- New read-only tool `n4_list_batches`: lists the journaled write batches, newest first, with state, verdict and rollback links (audit F14). [PR #238]
- `n4_remove_component` reports what lies deeper than its depth-3 snapshot in `snapshot_truncated`. A rollback cannot bring those back, and before this the cut was silent (audit F8). [PR #238]
- `--load-wait` and `--http-timeout` make the 3 s load window and the 20 s HTTP timeout operator settings (audit F11). [PR #238]
- An empty BQL answer no longer fails `n4_inventory`, and a failed `output_file` write leaves no partial file. [PR #238]

### Fixed — `mcp-n4-kit` v0.7.1: process-scoped write budget, docs drift

- `--max-writes` bounds the whole server process. Before, a reconnect reset it (audit F5). [PR #240]
- `modbusCore:FlexAddress` and `bacnet:BacnetAddress` are treated as slot values, not components (audit F17). [PR #240]
- The `snapshot_truncated` note gives a lower bound ("at least N"). [PR #240]
- README, SKILL and METHODOLOGY drift fixed: status, layout, `--station-home`, timing flags, and BQL-first read order (audit F15). Retro for the 2026-10-03 hardening audit added (`retros/2026-10-03-hardening-audit.md`, pending). [PR #240]

### Added — `mcp-n4-kit` v0.7.2: retro 2026-10-03 fold
- `tools/mutation_check.py`: mutation checks without stale bytecode (clears `__pycache__`, `PYTHONDONTWRITEBYTECODE=1`).
- README: stdlib `trace` untested-branch audit before each minor release.
- METHODOLOGY section 8: explicit per-PR review of medium units under one-PR-per-unit delivery.

### Fixed — `mcp-n4-kit` v0.7.3: `tools/mutation_check.py` exits 3 (environment error, with a `mutation_check: cannot launch` stderr line) when the command cannot be launched, instead of raising `FileNotFoundError`, so a failed launch is never read as a RED. Adds tests for launch failure and the `--root` not-a-directory branch, and clarifies the README `cp -p` restore instruction.

## [v0.30.0] - 2026-10-02

### Fixed — build-n4 kit polish after the 2026-10-02 retro-fold campaign (`odd/tasks/polish-2026-10-02.md`)

This release clears the advisories that the approved campaign reviews left on #199 and #151. Every fix was written
test-first and pinned with an observed mutation, and each landed as its own PR with a native RDD review. From P2d on,
an anti-cascade policy applied: only a fail-open or a regression introduced by the feature became a sub-task. Every
other advisory is parked in #226.

- **P1/P1b/P1c** `report-module.sh` runs `lint-link-target-flags.sh` (it fails closed on a non-zero exit with no FAIL row and names the wiring map), surfaces the slot-coverage facade FAIL and relays ADVISORY rows; `commissioning-verify.sh` gets the same member; bash 5.2 `patsub_replacement` quoting. [PR #207, #208, #209]
- **P2a-P2e** `lint-delays.sh` reports the MIN floor in ms, flags an unclosed schedule call and strips comments with string awareness; `verify-module` `has_nu`; arbitrary-ord markers inside literals; LTF1 handles self-set Table 2 rows (whole-cell, backtick-safe sentinels, header-row Source column); write-path literal-aware stripping; CS4 camelCase, unit-suffixed cutouts and the status/counter vs setpoint suffix split. [PR #209-#213]
- **P3-P3d** `obix-link-audit.sh` escapes credentials in the curl config, checks the MATCH source component and warns on an `http://` base; the `commissioning-verify.sh` values-owed parser became a named-state, Slot-anchored, fail-closed block parser; `triage-console.sh --site` duplicate-id token. [PR #214-#217]
- **P4** `gen-lint-index.sh`: Auto column from real invocations, every Usage/Exit line, and an awk failure is exit 3. [PR #218]
- **P5/P5b** spa-poll success-path `lastOk`, the rc-scan disabled-gate `auth` token (OAuth/reauth/unauthorized kept), hmi-sweep input checks, the vendor-floor sloppy-script tokenizer, report-module "PASS" after WARN rows plus the eslint major gate; `ui_profile` single source, profile tags, B34 alarm fidelity. [PR #219, #220]
- **P6** folds the campaign close retro: per-entry ledger merge, serial bats / count what ran, trivial trailer for feature-doc pushes, and advisory routing with the anti-cascade limit. `sweep-build-state.sh --age` disposition skip is bounded (0-delta rows, allowed values). `sweep-fold-audit.sh --deltas-since` (in CI) checks every Δ of each folded retro; `lint-manual-labels` header fixes. [PR #221]
- **P7a** the six T3/T4/T5a hot-path lints fail closed on an awk failure; persist-hot-write follows every callee; session-store scheduled purge is one hop; the ORD resolve is anchored; inert-coordination handles lowerCamel stagger and units() calls; spa-poll handles arrow-function polls. [PR #222]
- **P7b/P7c** `build.sh` drift gate: exit 52 "cannot verify" for an unlistable jar or an unreadable vendorVersion (it was a silent pass), and a failed restore is reported; `preflight.sh` jar-lock is SKIP on an empty lsof listing. [PR #223, #225]

Issues: #199 and #151 are closed with per-advisory mappings. Parked items are tracked in #226.

### Changed — `mcp-n4-kit` docs

- Retro for the 2026-10-02 tests and tier-gate work units: `mcp-n4-kit/retros/2026-10-02-tests-and-tier-gate.md`. [PR #224]
- That retro's D1-D3 are folded into `mcp-n4-kit/README.md`: an "Upgrading to v0.5.0" note, a "Versioning" section, and the test isolation rules. [PR #227]

## [v0.29.0] - 2026-10-02

### Changed — retro-fold campaign 2026-10-02: fold the 23 pending build-n4 retros (147 Δ)

The 23 retros of 2026-09-24..2026-10-01 (commissioning, dashboard frontend reliability, deploy/backup, logic
persistence, process tiers) were landed (#183) and folded into the build-n4-module kit core as one work-unit PR each,
branched from `main`, reviewed by the native RDD review when due, and merged after CI. `retros/INDEX.md` again shows
**0 pending delta-retros**; every Δ1..Δn of each campaign retro is cited as `[ev: retro <stem> Δn]` in the kit core.

- **WU0** generated lint index `toolbelt/INDEX.md` + `toolbelt/gen-lint-index.sh` (`--check` in CI). [PR #185]
- **WU1** `types/frontend-standard.md` and the `ui_profile` field (hmi/lan/both). [PR #186]
- **WU2** dashboard runtime doctrine: poll-loop reliability, write confirmation, `/api/version`, HMI shell. [PR #188]
- **WU3** `rc-scan.sh` frontend checks (browser floor, fetch-no-signal, data-URI budget, orphan pages …) + success-time `lint-spa-poll-no-recovery.sh`. [PR #191]
- **WU4** `lint-vendor-floor.sh`, kit ESLint config, `hmi-sweep.js`, `report-module.sh --profile/--legacy`. [PR #192]
- **WU5a/WU5b** logic persistence, restart, safety and structure doctrine (`types/logic.md`, `METHODOLOGY.md`). [PR #193]
- **WU6a** rt-lint false positives (`makeNumeric` units, multi-line schedules, reviewed-ord marker). [PR #194]
- **WU6b** `lint-link-target-flags.sh`, CS4, facade slot coverage, wiring-map display names, console-only and write-audit checks. [PR #195]
- **WU7** build/preflight fixes, `scripts/check-client-source.sh`, `scripts/check-skill-drift.sh`, `source_of_truth` field. [PR #197]
- **WU8** deploy, backup and release gates; roll-forward recovery doctrine. [PR #196]
- **WU9** `obix-link-audit.sh`, values-owed rows, `triage-console.sh --site`, commissioning checklist. [PR #198]
- **WU10** process doctrine: blast-radius tiers P0-P3, behavior-decisions gate, orchestration clauses, incident journal. [PR #202]
- **WU11a** client version ledger template (`templates/VERSION-LEDGER.md`), operator-manual lockstep gate + advisory `lint-manual-labels.sh`, corpus-index retitle + B33/B34/B39/B48. [PR #204]
- **WU11b** ledger reconcile (BUILD-STATE), final INDEX flips, `sweep-build-state.sh --age` skips rows with an INDEX `<!-- disposition: … -->` (the CXF proposal #122, the 2026-09-20 apply worklist), campaign close retro (4 proposed Δ), this release. [PR #206]

Deferred (BUILD-STATE open_issues): the out-of-repo preview harness (frontend-standard Δ8, deployment-profiles Δ4),
`lint-size.sh`. Review advisories: issue #199.

### Added — `mcp-n4-kit` v0.5.0: enforced version-tier write gate (MWU3) [PR #205]

- `n4_connect` / `n4_describe_session` report the detected station `version` (oBIX
  `productVersion` from `/obix/about/`), `version_source`, `tier` (A 4.13/4.14, B 4.15/4.3,
  C other or unknown), `tier_writes` and, when detection failed, `version_error`.
- Every mutating tool refuses execution on tier B unless the server was started with
  `--allow-tier-b NAME`, and on tier C (including an undetected version) unless
  `--allow-tier-c NAME`; tier A is unchanged and reads are never gated. Dry runs stay
  allowed and carry `tier_gate` when the execution would be refused.
- Refusals name their routes and start with `safety.REASON_TIER`, which the session
  retro classifies. METHODOLOGY section 5 no longer says the gate is manual.

### Changed — `mcp-n4-kit` tests

- Gap-error scrubbing and rollback verdict precedence covered (closes #190). [PR #201]
- Destructive-test patches are undone deterministically (test patch hygiene). [PR #203]

## [v0.28.1] - 2026-10-02

### Fixed — `mcp-n4-kit` v0.4.1: rollback read-back precision (issue #179 T5, v0.28.0 advisory review)

- **A frozen-child read error is no longer reported as `missing`** (R4-001/R2-001/R3-001): when the
  read-back cannot load a frozen child, its `frozen_config_not_restored` entry carries `readback_error`
  (scrubbed) and no `missing`, and the rollback verdict is `unverified` instead of `partial`. `partial`
  asserts what was not restored; a read error establishes nothing. Precedence: `mismatch` > `unverified` >
  `partial` > `verified`.
- **An omitted nested-default slot is equal** (R4-002): a captured struct slot (e.g. a StatusNumeric
  `readValue` with `value`/`status`) that the station omits now counts as equal when every captured child is
  at its type default, as for scalars. A child whose default the kit does not know still counts as different.
- Internal: the in-doubt error builder `_partial` is renamed `_in_doubt_with_created` (it collided with the
  `partial` verdict, R2-003); `_link_input_values` is a plain loop sharing `_prior_kind` with
  `_hashed_inverse` (R2-004); finding-ID comments name their review (`PR #184 review R3-001`,
  `PR #187 blocking review R3-001`) instead of an ambiguous `issue #179 R3-001` (R2-002).

## [v0.28.0] - 2026-10-02

### Changed — `mcp-n4-kit` v0.4.0: rollback fidelity (issue #179 T4)

- **New rollback verdict `partial`** (R4-002, B1200-G1): a rollback is `verified` only when it is faithful.
  A frozen child of a re-created component (e.g. a writable's `proxyExt`) is still never re-added, but the
  read-back now compares it with the snapshot (type, captured slots; a slot the station omits counts as its
  default). Any difference, and any descendant of a frozen child that is missing, is listed in
  `frozen_config_not_restored` (path, snapshot and live type, differing slots, captured config) and the
  verdict is `partial`. Before, the rollback reported `verified`. The configuration is not written back:
  a fresh frozen child can be of another type (`control:NullProxyExt` vs a driver proxy extension), which a
  set op cannot change, and set ops on frozen children are not live-certified.
- **Link-driven inputs are no longer lost silently** (R3-002, B1200-G2): `n4_remove_component` records the
  last value of every input that is a link target (configuration form: status bits kept, facets dropped) on
  its relink (`prior`), and records inputs linked from outside the removed subtree as `unlinked_input`
  inverse entries. `n4_rollback` lists every input it did not restore in `link_inputs_not_restored`
  (`target`, `target_slot`, `source_slot`, `reason`, `value`). A skipped relink stays `mismatch`; an input
  linked from outside the subtree makes the verdict `partial`. The value is reported, not written: the live
  station rejected link-target values in the re-create body (B1200 finding 3). Batches journaled before
  v0.28.0 carry no recorded value (`value: null`). The value is captured when the confirmed remove runs
  and journaled from that re-plan; it is kept out of the confirmation hash (a link source keeps changing
  it, so hashing it would reject a valid confirmation by timing alone, R3-001). The dry run shows it as a
  `link_input_values` preview outside the hashed `plan`.
- **Type contract table** (B1200-G3): `box.COMPONENT_TYPES` maps type specs to component/value and wins over
  the naming heuristic, which stays as the documented fallback. Seeded from the N4 declarations:
  `baja:Component`, `baja:Folder`, `baja:UnrestrictedFolder`, `control:NullProxyExt` (components),
  `control:PriorityLevel`, `baja:Link`, `baja:WsAnnotation` (values). A live `reg.loadContract` lookup is
  not built: it needs a new BOX channel certified on a station.
- `retro.py` turns a `partial` verdict into a MEDIUM candidate delta.

### Fixed — #184 review advisory

- `tools/new_retro.py` reads the umask from `/proc/self/status` without changing it, falling back to the
  set-and-restore pair elsewhere (R3-002).
- `n4_navigate`: `has_children` ignores the `types` filter, now stated in the tool description (R3-003);
  `_types_arg` docstring says it returns the list (R2-001).
- `box.py` documents the load-counter assumptions: `ssc` raises before send or on an error reply (R3-001),
  and loads are answered in order (R4-001). No behavior change.

### References
- ODD feature document: `odd/tasks/mcp-n4-issue179-hygiene.md` (T4).
- Issue: angeles725/niagara-tools#179; niagara-research B1200-G1/G2/G3.

## [v0.27.0] - 2026-10-02

### Added — `mcp-n4-kit` v0.3.0: navigate `types` filter, issue #179 hygiene

- **`n4_navigate` `types`** (retro 2026-10-02 D3): an optional list of type specs (e.g. `["bacnet:BacnetDevice"]`).
  Only children of those types are kept, plus the path to them; each kept entry carries `matched: true/false`
  and the reply echoes `types`. Absent or empty means no filter, so existing calls are unchanged.
- **METHODOLOGY** (no code gate, **manual**):
  - section 7: a long-running client takes a progress file (e.g. `--progress-file`) and never relies on stdout
    piped through `tail`;
  - section 3: when the harness permission classifier blocks a live write, even from a scoped writer, surface it
    to the operator and stop; never route around it through another tool, a subagent or a peer session.

### Fixed — issue #179 (T6e+T8b review advisory)

- **Load counter** (R4-001): a request that is rejected or never sent no longer stays counted, and an answered
  load clears the count left by an earlier timed-out one (replies come in request order). A single timeout no
  longer disables the stale-handle early abort for the rest of the session.
- **Duplicate guard** (R1-001, R2-001, R3-001, R4-003): `write.check_state_files()` runs once per confirmed write.
- **Retro file modes** (R3-003, R4-004): `tools/new_retro.py` keeps the existing file's mode (e.g. `INDEX.md`)
  and gives a new retro `0644` under the umask, instead of `mkstemp`'s `0600`.
- **Readability** (R2-002, R2-003): the drain loop in `box.py` parenthesizes its conditional iterable; `retro.py`
  builds `by_batch` once, in `_window`.

Still open in #179: rollback fidelity (R4-002, R3-002, B1200-G1/G2/G3), planned as T4.

### References
- ODD feature document: `odd/tasks/mcp-n4-issue179-hygiene.md`.
- Issue: angeles725/niagara-tools#179.

## [v0.26.0] - 2026-10-02

### Added — `mcp-n4-kit` v0.2.0: bulk reads, folding the first remote-session retro (D1-D8)

The first live use against a remote station showed the gap: a navigate/read crawl of `/Drivers` ran for more than
16 minutes over a ~100 ms WAN and never finished. Projected BQL took the same inventory in about 3 s. Retro:
`mcp-n4-kit/retros/2026-10-02-customer-remote-01-session.md`.

- **`n4_bql_query`** (D1): one read-only projected BQL `select` over a subtree, sent as
  `GET /ord/<url-encoded ORD>|view:file:ITableToCsv` (path form; `/ord?` answers 400). It returns rows plus
  columns. Guards:
  - `select` only, and no `|`;
  - a row cap (default 5000, `truncated`);
  - a timeout;
  - control characters and the BOM are stripped;
  - `$xx` names are decoded by `bql.unescape_slot`.

  The read is recorded as a session observation, and as an `audit.jsonl` line with outcome `read` when writes are
  enabled. Pure helpers live in `mcp_n4/bql.py`; `BoxClient.get_ord` is the transport.
- **`n4_inventory`** (D7, D8): networks, devices and points through three BQL queries, counted per network and per
  device. A network's built-in `localDevice` is flagged `local: true`, and `local_devices` is counted apart from
  `field_devices`. Each step reports `progress` counts, and the new `--progress-file PATH` flag receives the same
  lines.
- **Visible latency** (D2): `n4_navigate` and `n4_read_slots` return `elapsed_ms`.

### Changed

- **Adaptive load polling** (D2): `BoxClient.load_tree` polls at 0.1 s, then backs off 0.2, 0.4, 0.8 s (capped).
  The total wait per window is bounded at 3 s (`box.poll_delays`); before, it was a fixed 0.5 s delay.
- **`n4_read_slots`** (D4): adds `value_display`, and returns the display string as `value` for complexes it cannot
  decode. Examples: a Modbus `FlexAddress` reads `Decimal:302`, and a `BacnetAddress` reads its address, where
  both used to read `None`.
- **Actionable auth errors** (D5): HTTP 401 names `HTTPBasicScheme`, the password, and "do not retry (lockout)".
  HTTP 403 names permissions. This holds on both the BOX and the BQL paths.
- **Docs** (D3, D6, D7, D8):
  - METHODOLOGY section 7 "Inventory and long reads": BQL first, the progress-file rule, local devices;
  - the README covers the per-project `.mcp.json` registration and a `git worktree add <path> origin/main` recipe;
  - when the kit is missing, the skill launcher says so and offers that worktree (or an installed copy) instead
    of searching.

### References
- ODD feature document: `odd/tasks/mcp-n4-bulk-read-deltas.md`.
- Retro: `mcp-n4-kit/retros/2026-10-02-customer-remote-01-session.md` (folded).

## [v0.25.0] - 2026-10-01

### Added — `mcp-n4-kit/`: an MCP server + skill that lets an AI agent read AND write any Niagara N4 station safely

A new kit, sibling of `build-n4-module-kit/`, built under ODD + RDD as a chain of 20 work-unit PRs.
Every executable slice passed the native review, one of them through a bounded correction cycle. The kit was
validated live five times against a real N4.14 station: the full scenario, rollback included, passed on run 5.

- **BOX JSON client** (`mcp_n4/box.py`, route R-A, no station module needed):
  - HTTPS only; no login retry (lock-out safe);
  - no redirects carrying credentials;
  - one op per `syncTo`;
  - a single ORD grammar (`child_ord`/`join_ord`).
  [PRs #159, #160, #161, #175]
- **MCP stdio server** (`mcp_n4/server.py`, stdlib JSON-RPC 2.0), read-only by default. The operator fixes the station
  policy (`--station NAME=URL`, `--credential-env`, `--insecure-tls NAME`), and every session is identity-checked against
  `stationName`. [PRs #162, #165]
- **Read tools:** `n4_connect`, `n4_describe_session`, `n4_navigate`, `n4_read_slots`, `n4_list_links`,
  `n4_find_dangling_outputs`, `n4_session_retro_draft`.
- **Guarded write tools** (`--allow-writes`): `n4_create_component` (with wire-sheet placement), `n4_set_slot` (Status
  written whole), `n4_invoke_action`, `n4_create_link`, `n4_remove_component`, `n4_rollback` and `n4_save_station`
  (persistence proven by the `config.bog` sha256/mtime). Every write passes these layers:
  - dry run → single-use HMAC confirmation token;
  - identity check;
  - ORD write scope;
  - per-session budget;
  - write-ahead journal;
  - redacted audit;
  - read-back verdict (`verified`/`mismatch`/`failed`/`unverified`).
  [PRs #163, #164, #166, #167, #169, #171, #177]
- **Live smoke runner** (`tools/live_smoke.py`): drives the real server end to end on a kitControl thermostat
  scenario, with bounded writes, secret scrubbing and cleanup on any failure. [PRs #170, #172]
- **Usage retros:** `retros/INDEX.md`, the `mcp_n4/templates/retro.template.md` template, `tools/new_retro.py` and an
  evidence-backed draft from the session's audit and journal (propose, never apply). [PRs #173, #174]
- **`METHODOLOGY.md`:** route ladder, safety layers mapped to code, live rules and the session checklist. A test gates
  every cited test name. Plus the **`skill/SKILL.md`** launcher. [PRs #168, #178]
- `scripts/install-skill.sh --skill <build-n4-module|mcp-n4>` (the default is unchanged). [PRs #168, #178]
- Hygiene from accumulated review advisory. [PRs #176, #178]

### Fixed — four live station rejections on the rollback re-create path, each modeled in the fake station first

1. A nested-component `a` op is rejected → one op per component. [#171]
2. `Illegal double slashes` → ORD grammar. [#175]
3. `"t": null`, status facets and link-driven inputs → omitted. [#177]
4. `Illegal child "control:NullProxyExt"` → frozen children are never re-added. [#177]

### References
- ODD feature document: `odd/tasks/mcp-n4-kit.md`.
- Evidence: niagara-research blocks B1177, B1179, B1192, B1197, B1199 (AM20 PoC) and B1200 (live runs 1–5).
- Engram: `odd/mcp-n4-kit/progress`, `research/niagara/n4-agent-mcp/progress`.
- Follow-ups: #179.

## [v0.24.0] - 2026-09-24

### Added / Changed — retro-fold campaign 2026-09-21/23: fold the 5 pending kit retros (22 Δ)

Five retros from the 2026-09-21 module work and the 2026-09-23 PANCCADIA field session were landed
(#145, #146, #147) and folded into the kit core in a chain of one PR per retro under ODD + RDD. Every
executable change passed the native 4-lens review. `retros/INDEX.md` again shows **0 pending delta-retros**.

- **T1** secure-authoring-isoperational-gate (1Δ) → `types/security.md` §5.1 ongoing-work `isOperational()` gate. [PR #148]
- **T2** wb-mapping-ord-npe-and-wsl-windows-jdk (2Δ) → `types/issues-and-gotchas.md` §H3 null-ORD `set()` NPE, §D5 WSL gradlew picking a Windows JDK. [PR #149]
- **T3** live-diagnosis-hardening (6Δ) → new lints `lint-changed-hot-write.sh`, `lint-persist-hot-write.sh`, `lint-session-store-lazy-evict.sh` + issues §I1/§I2/§F2 (the PANCCADIA CompPan hours-at-18/s case). [PR #150]
- **T4** apillm-wb-subscription-refresh-and-points (5Δ) → `types/wb-widgets.md` `registerForComponentEvents`/subscribe-depth rules and create-point recipe, `types/logic-authoring.md` rt point read-path card, new lint `lint-wb-external-ord-value.sh`. [PR #152]
- **T5a** panccadia-defrost-sequencing-hmi-reload Δ1/Δ3-Δ6 → new lints `lint-spa-poll-no-recovery.sh`, `lint-inert-coordination.sh`; BUILD-STATE deployed-baseline fields; HOA precedence contract; cycle-anchor checklist. [PR #153]
- **T5b** panccadia Δ2/Δ7/Δ8 → `build.sh` deployed-baseline version drift gate (exit 51, retry-safe: restores the deployed jar), 9p/drvfs build-location WARN (`toolbelt/lib/fs-type.sh`), `preflight.sh` single-pass `lsof`. RDD found and bounded-corrected one CRITICAL (drift gate not retry-safe). [PR #154]

Deferred (BUILD-STATE open_issues): optional `lint-license-isoperational-gate`, `lint-set-null-ord`.
Follow-ups: #151 (43 non-blocking review advisories on the new lints/gates), #49 (the deployed ColdRoomPan
defrost stagger is inert, found by `lint-inert-coordination`). Suite 628 → 689 bats.

## [v0.23.0] - 2026-09-20

### Changed — apply-deltas campaign (#123): fold all 114 pending kit deltas into the core

The 10 propose-never-apply delta retros from the 2026-09-19/20 module-corpus research campaign
(114 Δ tokens) were folded into the kit guidance, one FULL-promotion PR per retro (#124–#133),
each with a `Retro: promotion` trailer + INDEX/marker/BUILD-STATE anchors, `sweep-fold-audit --strict`
clean, and CI green. `retros/INDEX.md` now shows **0 pending delta-retros**.

- **T1** wb-field-editors (3Δ) → `types/wb-widgets.md` Field-editors § (targetType facet, Manager point-creation, filtered BComponentChooser). Also greened pre-existing red CI: `cxf-to-n4-translator-proposal.md` + `APPLY-PLAN-*.md` were non-retro docs in `retros/` with no INDEX row (`sweep-build-state.sh` requires one). [PR #124]
- **T2** wb-manager-framework (4Δ) → `types/wb-widgets.md` Manager-recipe § (6-object anatomy, MgrColumn taxonomy, Manager-vs-BWbComponentView table, non-driver template). [PR #125]
- **T3** our-dashboard-audit (5Δ) → `types/security.md` §7 header checklist, `types/dashboard.md` (SERVICE_ORD/ROOMS portability, palette, WSL-test contract), `commissioning-verify.sh` MANUAL step. [PR #126]
- **T4** module-hardening-reqexec-closed (7Δ) → security §7/§8 (global header filter, CORS), distribution §5/§10 (signing-profile triage, boot recovery), driver-authoring §5 (configFatal/configFail card), logic-authoring (dynamic-slot prune + started()-migration), issues D4/G1. [PR #127]
- **T5** module-hardening-reference-cards (8Δ) → `types/logic.md` (BQL card, BFormat security, isValid-vs-isOk gate, timer-rearm) + `types/logic-authoring.md` (BFacets/BEnumRange/SlotPath/BRelTime). [PR #128]
- **T6** honeywell-wb-rt-wb (8Δ) → wb-widgets/driver-authoring/security/logic-authoring/distribution/observability (device-model plugin SPI, per-user prefs, fail-closed auth, credential redaction, chunked OTA, AtomicBoolean guard). [PR #129]
- **T7** wb-vendor-ux-wave3-vendor-drivers (14Δ) → driver-authoring §9 (oBIX/LonWorks/OEM-on-stock/BStationMgrCommand/M-Bus/OPC), wb-widgets (view-target taxonomy, platform-service/daemon-file/SNMP), security §9 (SHA-256 digest + lint), issues §H1. [PR #130]
- **T8** module-hardening-failure-modes (16Δ) → logic/logic-authoring/security §2.4/distribution/issues/wb-widgets (RUN2-7, PER2/3/5/6, BLD1-6, UXS6; 4 documented lint candidates). [PR #131]
- **T9** apillm-headless-servlet-rt-4.14 (24Δ) → dashboard/security/wb-widgets/structure/logic/logic-authoring (servlet-in-rt contract, facet-JSON, dual-path live view, UnrestrictedFolder scaffold, JDK-pin, poller gate; Δ19 folded as forward-reference to superseding Δ20/PD-FE1). [PR #132]
- **T10** wb-vendor-ux-rt-wb-pattern (25Δ) → wb-widgets/driver-authoring/logic-authoring/security §10/`METHODOLOGY.md` (WB archetypes table)/dashboard/cloud-connector §7-8/issues §H2 (rt↔wb data channels, outbound-vs-inbound security axes, COV columns, cloud-connector SPI). [PR #133]

### References

- Umbrella issue #123 (closed); feature doc `build-n4-module-kit/odd/tasks/apply-build-n4-module-deltas.md`.
- kit-docs promotion only — no toolbelt logic behavior change, no client jar bump; `sweep-fold-audit --strict` = 159 folded / 159 cited / 0 uncited.
- Engram topic `research/niagara/build-n4-module/apply-deltas-campaign`.

---

## [v0.22.0] - 2026-09-06

### Changed — Campaign 11: lint precision (T1-T4)

- **T1** shared `lib/method-boundary.sh` (function-only `MB_AWK`, PEAK depth via `BASH_SOURCE`) — fixes the one-liner method false-negative; shared parser + accessor skip across lint-timers / lint-silent-protection / lint-ext-writable-shape via three awk invocation mechanisms (inline, adjacent-quote, second `-f`). [ev: retro campaign11-shared-method-boundary]
- **T2** `tests/lib/client-root.bash` single-sources the client read tree — 10 hardcoded sites → 0 (5 C9_CLIENT_ROOT + 2 C9_CLIENT_REPO + 3 live-checkout); LD5 flips to exit 0 and c8-close SC1-smoke verdict tightened (both pinned a bug/vacuity, not a rule). [ev: retro campaign11-client-root]
- **T3** `lint-write-path.sh` DRIFT advisory — a `[concept]` marker on a slot that became real is flagged; true concept rows (slot absent from source) stay silent; `--strict` promotes STALE or DRIFT to exit 1. [ev: retro campaign11-concept-row-drift]
- **T4** `toolbelt/lint-guard-pins.sh` — every lint header's `# Mutation: <id> -- <desc>` declaration must name an existing bats fixture (9 lints + itself, 15 `# Mutation:` lines total); 'clean' = MATCH count == lint count, never 0 found. [ev: retro campaign11-lint-guard-pins]

### Tests

- Suite: **454/454** (`bats tests/` env-unset; c11-close.bats 11/13 pre-tag — CLOSE-tag and CLOSE-harness-run pending).

### References

- SDD change: `build-n4-module-campaign11` (PR1 T1, PR2 T3, PR3 T2, PR4 T4, PR5 close; v0.21.0 → v0.22.0; kit-only — no client jar bump).
- Engram topics `sdd/build-n4-module-campaign11/*`.

---

## [v0.21.0] - 2026-09-06

### Changed — Campaign 10: lint precision (S21-S25) + client hygiene (S26)

- **`lint-timers.sh` companion-flag method-boundary fix (S21) [#89]:** companion-flag lint now requires a class-scope FIELD (depth-1 declaration) + a `Clock.schedule` call in the SAME method body, using the section-D method-boundary parser (`brace_depth>=2` guard); method-locals are excluded; `@NiagaraProperty(` annotation lines no longer fire as false candidate method-open events. Real-tree ColdRoomPan-rt: `anyNoHardware` false positive absent; CompPan-rt/DashboardPan-rt: exit 0. `[ev: retro campaign10-lint-timers-scope]`
- **`lint-ext-writable-shape.sh` per-slot action-body exemption (S22):** exemption is now per-slot — a `do<Action>()` body must write THAT slot (B831-G1 convention: action `x` → `doX()`; `brace_depth>=2` scope filter); class-level `has_action` heuristic removed. EW10 CompPan-rt re-pins 0→1 (`faultReset` by NAME — false negative fix, not regression). `[ev: retro campaign10-ext-writable-per-slot]`
- **`lint-silent-protection.sh` recognises Pattern B surfaces (S23):** Pass 0b ALARM_CLASSES index + adapter→pure follow (`"B"+this_class` naming pair); CP-1 low-suction shed in CompressorControl no longer false-WARNs — the `BCompressorControl` adapter (implements `BIAlarmSource`, `newOffnormalAlarm`, `AlarmSupport`) is now visible to the lint via the cross-file class-name index. Real-tree CompPan-rt: 0 WARN; ColdRoomPan-rt Pattern A intact. `[ev: retro campaign10-silent-protection-pattern-b]`
- **`run-pure-test.sh` cwd-independent + 2-arg usage fix (S24):** the `java` call at `:62` now runs in a subshell `( cd "$rt" && java … )`, making the runner cwd-independent; `Paths.get("src/…")` in structural WiringTests resolves regardless of caller cwd. Usage doc corrected to 2-arg form `<module-rt-dir> <pkg.TestClass>`. `[ev: retro campaign10-run-pure-test-cwd]`
- **`lint-write-path.sh` STALE advisory + `[concept]` exemption (S25):** STALE advisory class (per-row) flags matrix rows whose slot name is absent from all source `@NiagaraProperty`/`@NiagaraAction` names in the matrix root; `[concept]` on a row exempts that row only; `--strict` promotes STALE to exit 1 (uncovered FAIL exit unchanged); covered set is matrix-root-wide. Real-tree: 5 STALE rows at ff1b659 resolve after S26 `[concept]` marks. `[ev: retro campaign10-write-path-stale]`
- **Client hygiene: gitignore build caches + 5 `[concept]` matrix marks (S26, docs-only):** `.gitignore` extended with `**/build/classes/` and `**/build/tmp/` patterns (51 untracked cache files removed from tracking); 5 `[concept]` rows added to `docs/write-path-matrix.md` (hoaMode ×3, inhibit, freezeEnabled); no `build.gradle.kts` change, no version bump — client module versions carry over (Compresores 2.2.0, Paccadia 2.1.0, Dashboard 2.2.0). `[ev: retro campaign10-close-process-meta-lessons]`

### Tests

- Suite: **418/418** (`bats tests/` with `C9_CLIENT_ROOT=Leon-Guanjuato-worktrees/main-ff1b659`; c10-close.bats 10/12 pre-tag — CLOSE-tag and CLOSE-harness-run pending).

### References

- SDD change: `build-n4-module-campaign10` (PR1 #89, PR2 S22, PR3 S23, PR4 S24, PR5 S25, PR6 S26; v0.20.0 → v0.21.0).
- Engram topics `sdd/build-n4-module-campaign10/*`.

---

## [v0.20.0] - 2026-09-06

### Added — Campaign 9: kit lints + doctrine (PR2, PR3, PR10, PR12)

- **`lint-demand-scope.sh` (PR2):** `toolbelt/lint-demand-scope.sh [--strict] <java-src-dir>` — demand-in-scope lint; a control/staging method that reads a process variable (suction/pressure/discharge/temp/cv/coil/head) in a comparison with NO demand-shaped input ({demand*, *call*, enable, loopEnable, *count, BStatusBoolean in*}) in its parameters or enclosing-class fields → WARN "pressure without demand"; WARN-only (--strict promotes to exit 1); two-pass awk: Pass 1 collects class-level demand fields, Pass 2 applies rules 1-4 with brace-balanced body extraction; dot-dirs pruned (D9b); DS1-DS7 + DS-smoke bats; real PASS: CompressorControl.step(demandCount) gates demand — zero WARN.*step rows confirmed [ev: corpus B820 §820.2-820.4; retro campaign9-demand-scope].
- **`lint-silent-protection.sh` (PR3):** `toolbelt/lint-silent-protection.sh [--strict] <java-src-dir>` — silent-protection lint (B824): a protection trip (a boolean latch that inhibits an output) with no SUMMARY/alarm/adapter surface → `WARN  lint-silent-protection  <file>:<line>  <reason>`, exactly one per trip site; effect-slot exemption, one-level field→slot adapter follow, private fields never a surface; WARN-only (`--strict` → 1), exit 3 on usage/no sources (K20), D9b prune. Real tree at a109249: CP-1 `CompressorControl.java:215` and CR-3 `BEvaporatorUnit.java:1287` flagged, CP-2/defrostSkipped clean, DashboardPan 0/0. RED `qa/c9-silent-protection` 3b281e0 (SP1-SP9 + exact-count SP-smoke). Routed in BUILD-LOOP §5 + SKILL; `report-module.sh` §5.5 member row.
- **`lint-ext-writable-shape.sh` (PR10):** `toolbelt/lint-ext-writable-shape.sh [--strict] <java-src-dir>` — ext-writable-shape lint (B823/B826): a `BStatusNumeric`, `BStatusBoolean`, or `BStatusEnum` OPERATOR property whose declaring class has no `@NiagaraAction` → WARN with child-leaf note; a conformant oBIX PUT is rejected for complex slots and a hand-crafted wrapped-`<obj>` PUT silently writes 0.0 when the `value` child is absent; the safe write path is the oBIX child-leaf bare `<real>` at `…/<slot>/value` (B826-G2, LIVE-CONFIRMED); WARN-only (`--strict` → 1), exit 3 on usage/no sources (K20), D9b prune. Real tree a109249: DashboardPan-rt 1 WARN (`BRoomPanel.setpoint` line 124), CompPan-rt/ColdRoomPan-rt/DashboardPan-ux 0. RED `qa/c9-ext-writable-shape` 269be48 (EW1-EW11). Routed in BUILD-LOOP §5 + SKILL; `report-module.sh` §5.6 member row [ev: corpus B823 §823.2; corpus B826; retro campaign9-ext-writable-shape].
- **Doctrine fold (PR12):** `types/logic.md` §Protection anatomy (alarm patterns A/B), `types/logic-authoring.md` ext-writable anti-shape line, `BUILD-LOOP.md §5` lint routing + §K22 real-tree smoke cross-ref, unified write-audit doctrine line [ev: retro campaign9-doctrine-fold].

### Client / tunnel (referenced; not in this repo)

- PANCCADIA client (angeles725/niagara-panccadia-leon): rotation PR#12 (S20 — time-slice compressor rotation, CompPan 2.0.3→2.1.0), servlet guards PR#10 (DashboardWriteGuards real-Context audited write, Dashboard 2.1.1→2.2.0), config login PR#13 (in-module station re-auth + sliding session, Dashboard 2.2.0), alarm CR-3 PR#11 (freeze alarm BAlarmSourceExt Pattern A, ColdRoomPan 2.0.7→2.1.0), alarm CP-1 PR#15 (low-suction BIAlarmSource+AlarmSupport Pattern B, CompPan 2.1.0→2.2.0), write-path rows PR#14 — CompPan 2.0.3→2.2.0, ColdRoomPan 2.0.7→2.1.0, Dashboard 2.1.1→2.2.0.
- Tunnel write-server (angeles725/pancaddia-leon-tunnel): config login w/ TTL PR#1 (buildServer seam + step-up login/logout), change_log canonical row + spool + replaySpool PR#2, AuditHistory mirror flag OFF PR#3 (blessed, awaiting Cristian's merge).

### Tests

- Suite: **382 @test across 36 files** (368 at a45fea1 + 14 from cherry-picked `tests/c9-close.bats`).

### References

- SDD change: `build-n4-module-campaign9` (PR2 #86, PR3 #87, PR10 #88, PR12 #90 + SP-smoke re-pin #91; v0.19.0 → v0.20.0).
- Engram topics `sdd/build-n4-module-campaign9/*`.
- Retros (campaign-9): campaign9-demand-scope, -silent-protection, -ext-writable-shape, -doctrine-fold, -close-process-meta-lessons.

---

## [v0.19.0] - 2026-09-05

### Changed — Campaign 8 close Wave 1: delay lint, console triage, timer extensions, facets, per-slot, rc-scan, report integration (PR1–PR8)

- **`slot-coverage.sh` empty-lexicon escalation (D6a):** an empty `module.lexicon` with at least one declared type now exits **1** (FAIL) instead of 0 (WARN/`--strict`-only). Every operator slot renders raw camelCase in operator views (T8 footgun is a ship-blocker); a `--strict`-only gate left the very module that motivated the check (`chihuahua`) passing the aggregate. `report-module.sh` renders this as a `FAIL  slot-coverage  empty lexicon` row [ev: corpus B788; D6a].

### Added

- **`lint-delays.sh` (PR1):** `toolbelt/lint-delays.sh <java-src-dir>` — Clock.schedule/schedulePeriodically delay-floor lint; detects call sites with no strictly-positive (>0) floor (e.g. `Math.max(x, 0L)`); D2b cross-file helper resolution; exits 0/1/3; LD1–LD9 bats; real smoke: BDefrostController `:556/:566/:620/:664` FAIL on pre-fix ColdRoomPan-rt [ev: retro campaign8-lint-delays].
- **`triage-console.sh` (PR2):** `toolbelt/triage-console.sh [--package PKG] [--console-dir DIR] <console.txt>…` — own-module exception triage over station console logs; three attribution channels C1 (own frame), C2 (own logger tag), C3 ([sys]/[sys.xml]/[sys.registry]); digit-normalised dedup; exits 0/1/3; TR1–TR9 bats; real smoke: PANCCADIA/REFLOW/MX60 SEVERE + WARN rows [ev: retro campaign8-triage-console].
- **`lint-timers.sh` C/K/N extensions (PR3):** appended three new checks — `companion-flag` (uncleared companion in `stopped()` path), `jdk-thread` (raw `ScheduledExecutorService`/`new Thread()` in BComponent), `changed-sched` (Clock.schedule in `changed()` without guard in scheduling body); TC-A/B/C bats; TL1–TL4 not regressed [ev: retro campaign8-lint-timers-ext].
- **`verify-module.sh --src` facets-req + ord-literal (PR4):** `facets-req` WARN: OPERATOR numeric slot without facets key, or name-pattern slot without UNITS/PRECISION; `ord-literal` WARN: hardcoded `station:|local:|slot:/` string under src; F1–F8 bats; 218/218 total [ev: retro campaign8-facets-lint].
- **`slot-coverage.sh per-slot` subcommand (PR5 / D6):** `slot-coverage.sh per-slot <module-include.xml> <module.lexicon> <src-dir>` compares `OPERATOR`-flagged `@NiagaraProperty` slots against lexicon keys; emits `pct=<n.n> (per-slot)`, `MISSING <slot>`, and `STALE <key>` rows; exits 1 when any slot is missing; dot-dirs pruned (D9b) [ev: corpus B788; D6].
- **`rc-scan.sh` (PR6):** `toolbelt/rc-scan.sh <artifact-dir> [--strict] [--servlet]` — browser-resource lint over `rc/` assets (`*.html *.js *.css`); checks: `ord-literal` (FAIL), `host-literal` (FAIL), `bare-catch` (WARN/`--strict` FAIL), `null-branch` (WARN/`--strict` FAIL); excludes `rc/ext/**`, `*.min.js`, `srcTest/**`, dot-dirs; RC1–RC9 bats; 227/227 total [ev: retro campaign8-rc-scan].
- **`report-module.sh --console-dir` integration (PR8):** `toolbelt/report-module.sh <module-root> [--target-version x.y] [--console-dir <dir>]` gains three new member integrations: per-artifact `lint-delays.sh <artifact>/src` (SKIP when no `src/`; FAIL rows aggregate to exit 1); per-artifact `schema-risk.sh <artifact>/.deploy-baseline <artifact>` (SKIP when no `.deploy-baseline/`; exit 2 OUTAGE maps to FAIL row per D9a); once-per-run `triage-console.sh --console-dir <dir>` (SKIP row when flag absent; FAIL rows aggregate); dot-dir pruning fix in `schema-risk.sh` parse_slots; RM1–RM8 bats; 234/234 total [ev: retro campaign8-report-integration].

### Added — Campaign 8 close Wave 2: station snapshot, bog audit, wb + servlet lints (PR9–PR12)

- **`station-snapshot.sh` (PR9):** `toolbelt/station-snapshot.sh <station-dir> <out-dir>` — pre/post-deploy audit-surface snapshot; copies `config.bog` + `console*.txt`, records history/alarm db pointers by path+size (never the db files), writes `manifest.json` with per-file sha256; source dir never opened for write; the NTFS/0777-mount guard strips `+x` from outputs and `cp -p` preserves the Windows mtimes (output mtime is not an ordering signal); exits 0 ok / 1 copy failure / 3 usage; SN1–SN5 bats; real smoke on PANCCADIA [ev: retro campaign8-station-snapshot].
- **`bog-audit.sh` (PR10):** `toolbelt/bog-audit.sh <config.bog|file.xml> --module <MOD>… [--source-dir <dir>] [--strict]` — station `config.bog` auditor, CHECK1–CHECK12 over an embedded python3 D10 BOG-XML grammar engine; runs from the bog alone for CHECK1/8/9/10/11/12, `--source-dir` adds the source-coupled CHECK2–CHECK7; CHECK11 proxy-link-safety is **FAIL** (own-module output linked to a Boolean/NumericWritable with no explicit fallback holds last state on stop/reload), CHECK12 dashboard-write-to-link-target is advisory WARN, and a frozen slot inherited from a framework superclass is CHECK5 WARN "possibly inherited" not a ghost FAIL; exits 0 clean / 1 any FAIL / 3 usage/python3-absent; real smoke on PANCCADIA (17 CHECK11 FAIL) [ev: retro campaign8-bog-audit].
- **`lint-wb-threading.sh` + wb conformance (PR11):** `toolbelt/lint-wb-threading.sh <wb-src-dir> [--strict]` — two heuristic WARNs over a `-wb` src tree: `ui-thread-traversal` (a `doInvoke` body calling `getNavChildren`/`getNavNodes`/`BqlQuery` without `invokeLater`/`BJobService`/`JobThread`) + an agent-breadth heuristic; WARN-only, exit 1 only under `--strict`. Ships with `slot-coverage.sh` WB-LEX1 (missing-lexicon exit-1 path), `verify-module.sh` WB-SCAFFOLD1 + WB-DEP1 (`check_wb_scaffold` + `check_phantom_dep`), and the `types/wb-widgets.md` DWB1 10-rule doctrine + chihuahua-wb exemplar; K19 routing; lint-wb-threading.bats 9 tests [ev: retro campaign8-wb-audit].
- **`lint-servlet.sh` (PR12):** `toolbelt/lint-servlet.sh <ux-profile>/src` — BWebServlet security lint following callees ~depth-3; six checks: `auth-gate`, `input-400` (a numeric parse not inside a try/catch that returns 400), `unbounded-set`, `cache-nofinger`, `log-in-handler`, and `csrf-xrw-only` (an `X-Requested-With` guard with no `CsrfUtil`/`csrfToken` → WARN, B813); ships the `CsrfXrwOnly.java` fixture + LSV4 pin; exit 0 clean/WARN-only / 1 any FAIL / 3 usage; lint-servlet.bats 11 tests; K19 routing [ev: retro campaign8-lint-servlet].

### Changed — Campaign 8 close Wave 2: build + deploy doctrine (PR13–PR15)

- **Post-deploy verification subsection (PR13):** `BUILD-LOOP.md §6.a` — an ordered post-deploy verification block (pre-snapshot → hot-reload watch → console triage → `schema-risk.sh`/`bog-audit.sh` re-run), gated on the CHECK11 proxy-link-safety result; the five step scripts are hard-pinned in `tests/kit-links.bats L7` [ev: retro campaign8-post-deploy-checklist].
- **Build-pipeline documentation (PR14):** `BUILD-LOOP.md §4.a` Gradle `niagara-module` task matrix (what each task does + the safe `clean slotomatic jar` combination) and `§4.b` `vendorVersion`/`bajaVersion` version-bump checklist + station-reload consequence; `tests/build-sh.bats` gains the `BS-lock` + `BS-lock-hint` exit-31 station-lock + `mirror-niagara-home.sh` regression pins; build-sh.bats 12 tests [ev: corpus B807] [ev: corpus B795] [ev: retro campaign8-build-pipeline].
- **RT-control doctrine promotion (PR15):** the campaign-8 RT-control doctrine promoted into the `types/` core (doc-only): `types/logic.md` §RT-control-logic (PID anti-windup/NaN-to-fault/deadband, the §805.9 flowchart template — B805/B808), `types/logic-authoring.md` §history-ext (B804) and §"Slot types for externally written values" (the value-class→slot table: `double`/`BStatusNumeric` write shapes, the oBIX child-leaf bare `<real>` preferred form + silent-zero fallback, the frozen-enum-carries-range rule — B823/B822/B825/B826/B828/B816), and a `types/dashboard.md` pointer [ev: retro campaign8-rt-doctrine].

### Added — Campaign 8 close Wave 3: retro loop, structure lint, write-path lint, station-logic checks (PR16–PR20)

- **`new-retro.sh` + `kit-ticket.sh` (PR16, retro-loop):** `toolbelt/new-retro.sh <module-or-kit> [--ticket]` creates `retros/<date>-<slug>.md` from a template (line 1 `<!-- review-status: pending -->`, 4 sections), appends one `retros/INDEX.md` row, and sets `retro_pending: true` in `BUILD-STATE.md` — all three atomically, idempotent on re-run; AWK-scoped BUILD-STATE flip never marks sibling module sections; 6-char slug floor (sweep-fold-audit drops tokens < 6 chars); `toolbelt/kit-ticket.sh "<title>" --from <retro-file>` opens a kit-repo GitHub issue (`gh issue create`, labels `kit,from-run`), SKIP+exit-0 when `gh` is absent/unauth (never fails a run); `skill/SKILL.md` close-of-run step + `BUILD-LOOP.md §7` retro gate (CD1); retro-loop.bats 9 tests [ev: retro campaign8-retro-loop].
- **`lint-structure.sh` (PR18, structure):** `toolbelt/lint-structure.sh <module-root>` implements the L1–L11 structure lint (L1 package naming, L2 one @NiagaraType/file anchored at `^[[:space:]]*@NiagaraType`, L3 pure-model advisory, L4 lexicon non-empty, L5 palette non-empty for rt, L6 module-include.xml consistent, L7 3-part dep floors, L9 no empty skeleton, L10 no absolute host paths in tracked gradle.properties with upward walk to `.git` sentinel, L11 mixed pure+Baja srcTest declares both); profile discovery by direct-child `*-(rt|ux|wb|se|doc)` or build-script presence — NOT by module-include.xml (so a profile lacking it is still visited); row `FAIL|WARN  lint-structure  <path>  L<n>: <reason>`; exits 0/1/3; lint-structure.bats 10 tests [ev: retro campaign8-structure].
- **`lint-write-path.sh` (PR19, write-path):** `toolbelt/lint-write-path.sh <module-root> [--bog <config.bog>]` — every `Flags.OPERATOR` `@NiagaraProperty` MUST have a row in `docs/write-path-matrix.md` (slot · writer · timing · existing `src/test` test name); walk-up matrix resolution + `--matrix` override; `--bog` adds bog-derived dashboard slots; a MISSING matrix is ERROR exit 3, never a silent pass; real smoke on ColdRoomPan-rt: 6 uncovered OPERATOR slots (exit 1); CompPan-rt: matrix covered; lint-write-path.bats 11 tests [ev: retro campaign8-write-path] [ev: corpus B816].
- **`bog-audit.sh` CHECK13–CHECK19 (PR20, station-logic):** extends `toolbelt/bog-audit.sh` with CHECK13 relay-double-source (FAIL: two sources into one output relay slot), CHECK14 own-output-unlinked (WARN, suppressed when `hasDefrost=false`), CHECK15 sensor-crossed-by-name (WARN), CHECK16 hasDefrost↔DefrostController sibling (FAIL, both directions), CHECK17 roomN-index-mismatch (FAIL), CHECK18 HOA-tile≠freeze-tile (FAIL), CHECK19 control-writes-setpoint (WARN); pins SL13–SL19 + SL16r + SL-smoke-panccadia + SL-smoke-mx60 consolidated in `tests/bog-audit.bats` (25 total); real smoke PANCCADIA: CHECK11=17 FAIL, CHECK13=0, CHECK14=1 WARN (EvaporatorUnit2/evapOut), CHECK15=0, CHECK16=0, CHECK17=0, CHECK18=2 FAIL (EU_1 and EU_3), CHECK19=0; MX60: all CHECK13–CHECK19 = 0 [ev: retro campaign8-station-logic].

### Changed — Campaign 8 close Wave 3: orchestration doctrine (PR17)

- **Orchestration doctrine (PR17, doc-only):** `build-n4-module-kit/ORCHESTRATION.md` created (8 sections: roles, model table, delegation triggers, escalation gate, artifact store, evidence discipline, retro/ticket loop, recovery); `skill/SKILL.md` gains steps 1b (Explore shard, audit-first >3 files, sonnet), 1c (opus design shard for schema/new-slot/facade), 5b (peer QA RED-by-branch-name session before every code PR); `tests/kit-links.bats L8` verifies every ORCHESTRATION-named script exists at `toolbelt/<script>.sh` (L7 already existed from PR13 §6.a; kit-links.bats 8 total tests) [ev: retro campaign8-orchestration].

### References

- SDD change: `build-n4-module-campaign8` (PR1 … PR20 + close; v0.18.0 → v0.19.0, single minor — tagged `v0.19.0` on the final merged commit per tasks.md C.3/C.5, no 0.19.1 bump).
- Engram topics `sdd/build-n4-module-campaign8/*`.
- Research blocks folded (niagara-research): B788/B795/B800/B801/B803/B804/B805/B807/B808/B809/B810/B813/B816/B817/B821/B822/B823/B825/B826/B828/B829.
- Retros (campaign-8): campaign8-lint-delays, -triage-console, -lint-timers-ext, -facets-lint, -slot-per-slot, -rc-scan, -doctrine-fold, -report-integration, -station-snapshot, -bog-audit, -wb-audit, -lint-servlet, -post-deploy-checklist, -build-pipeline, -rt-doctrine, -retro-loop, -orchestration, -structure, -write-path, -station-logic, -close-process-meta-lessons.

---

## [v0.18.0] - 2026-09-05

### Added — Campaign 7 close: 9 retros folded, scaffold/schema-risk/plano/logic-split/report tools (PR1–PR8 #52–#59)

- **Retro fold (PR1 #52):** 9 campaign-6 retros folded into METHODOLOGY.md (K11–K14), BUILD-LOOP.md §7, CONTRIBUTING.md §SDD-ledger; all 9 INDEX rows flipped pending→folded; `sweep-fold-audit.sh --strict` promoted in CI.
- **Tool integration (PR2 #53):** BUILD-LOOP.md §0.b/§5/§7 and `skill/SKILL.md` §References updated for all 10 toolbelt scripts (preflight, lint-timers, slot-coverage, sweep-build-state, sweep-fold-audit, scaffold-module, schema-risk, verify-module+plano, report-module); launcher v0.5 installed.
- **Dashboard B796 exemplar (PR3 #54):** `types/dashboard.md` gains DashboardPan-ux write-surface 4/5 gate score; gate 4 REQUIRED-but-absent cited as issue #49.
- **scaffold-module.sh + fixtures (PR4 #55, size:exception):** `toolbelt/scaffold-module.sh` emits a pre-slotomatic module tree from `fixtures/MinimalPan`; TC1–TC6 + TC-K8 RED-first with named mutations. Tests: scaffold-module.bats.
- **schema-risk.sh (PR5 #56):** `toolbelt/schema-risk.sh` two-snapshot slot diff; verdict SAFE/LOSSY/OUTAGE (B795 CSV embedded as heredoc); SR1–SR9+SR-CSV bats; 7 B799 fixture pairs.
- **verify-module --plano (PR6 #57):** `verify-module.sh --plano <index.html|jar>` aspect-ratio cross-multiplication (Rc/Ri/Rv); PL1–PL5 bats; `tests/fixtures/plano/ok/index.html` 2×3 inline PNG.
- **types/logic-authoring.md split (PR7 #58):** `types/logic.md` lines 91–136 (Author-side SPIs) extracted to `types/logic-authoring.md`; kit-links L4/L5/L6 guards added; corpus-index.md retargeted; launcher v0.6 installed.
- **report-module.sh (PR8 #59, stretch):** `toolbelt/report-module.sh <module-root>` aggregated conformance report composing verify-module --src + slot-coverage + dup-keys + lint-timers + --plano per profile; RM1–RM3 bats; real-tree smoke ColdRoomPan-rt 9 PASS · 1 FAIL · 1 WARN · 1 SKIP → ISSUES exit 1; launcher v0.7 installed.

### References

- SDD change: `build-n4-module-campaign7` (PR1 #52 … PR8 #59; v0.17.0 → v0.18.0).
- Engram topics `sdd/build-n4-module-campaign7/*`; research block B798 (ColdRoomPan-rt real-tree smoke).

---

## [v0.17.0] - 2026-09-05

### Added — Campaign 6 close: research fold + conformance lints (PR7 #42, PR8 #43, close PR)

- **Research fold (PR7 #42):** 27 exemplar-backed deltas from the `/research-sdd` lanes into `types/logic.md`
  (author-side SPIs idiom + instances B778/B782/B785 with the B773 analytics exception; point-extension authoring B772;
  child-tree containers B779; grouping/relating surfaces B781; templates are artifact production B783; background jobs
  B774; watchdogs and timers B775; action protection B776; minimal module B790 proven by the B793 build),
  `types/dashboard.md` + `types/wb-widgets.md` (B780 palette/lexicon/@AgentOn; web-tier pointer table B791),
  `METHODOLOGY.md` (B784 profile/dependency conventions; B775 watchdog note; permissions reconciled — one source
  `module-permissions.xml`, one inlined `<permissions>` element, two child kinds [B777/B636]; lintable-vs-advisory rule +
  human-review checklist), `corpus-index.md` (27 rows). `sweep-fold-audit.sh` promoted to `--strict` in CI after the two
  legacy uncited folds were cited (T7.5).
- **Conformance lints (PR8 #43):** `toolbelt/lint-timers.sh` (a class owning a `Clock.Ticket` / calling
  `Clock.schedule*` without a cancelling `stopped()` → FAIL; discarded `Clock.schedule*` return → FAIL) — fires today on
  ColdRoomPan `BEvaporatorUnit` [B787]; `sweep-build-state.sh --age --today <date> [--max-age N]` retro-debt aging
  (E5 model; ESCALATED strictly past the threshold, N/A on zero pending); `preflight.sh` JDK 8 detection falls back to
  `bin/java -version` when the JDK ships no `release` file (WSL openjdk-8, B793). Tests 104 → 152, every new check
  RED-first with a named mutation.
- **Close:** campaign-6 close retro (`retros/2026-09-05-campaign6-close-process-meta-lessons.md`, 11 meta-lessons),
  sdd-verify report, final openspec state.

### References

- SDD change: `build-n4-module-campaign6` (PR1 #35 … PR8 #43; v0.16.0 tagged at PR6 #41, this entry covers what landed after it).
- Engram topics `sdd/build-n4-module-campaign6/*`; research blocks B762–B793 (niagara-research).

---

## [v0.16.0] - 2026-09-05

### Added / Changed — Campaign 6 close: tracked launcher, openspec durability, toolbelt completion (PR1–PR6)

**PR1 — Marker-index coherence sweep**
- `toolbelt/sweep-build-state.sh` reads `<!-- review-status: … -->` markers on line 1 of retro files and checks them against the INDEX row status; exit 1 on mismatch; absent marker tolerated. Named mutation M1/M4/M5/M6 proved. CI runs this on every push.
- Tests: `tests/build-retro-sync.bats` (25 tests, merged from QA branch `cb0dd7d`). PR1 retro filed.

**PR2 — CI pure-test enforcement**
- `.github/workflows/ci.yml` now runs `run-pure-test.bats` with the Java 8 Temurin JDK and pre-fetched JUnit 4.13.2 + hamcrest-core 1.3 (pinned sha256); `grep "# skip"` log step to make zero-SKIPs visible. CI cannot be bypassed by a clone that skipped hook installation.
- `tests/run-pure-test.bats`: `${CI:-}` guard converts skip→fail when `$CI` is set (P1–P6 produce zero-SKIPs in CI). Named mutation P2 proved.

**PR3 — Doctrine: K1–K10, M1/M2, LC7**
- `METHODOLOGY.md`: K1–K9 kit-maintenance rules (gate-exit taxonomy, mutation-prove discipline, high-signal/promotable-unit criteria, worktree origin/main pattern, grep-kit-before-fold, HOME-nonexistent, set-e empty-cache lessons); §Multi-session-coordination (M1); §Live-verify-safety (M2); LC7 what-to-test-where table by module type.
- `CONTRIBUTING.md`: K10/A10 (pin any linter a CI gate depends on); §8 "No CI" fixed (CI exists since campaign 5).
- 7 retros flipped `pending`→`folded` in the same commit that folds their content. PR3 retro filed.

**PR4 — Types fold: LC1–LC6, B762/B763 seams, mode-B slotomatic gap, ledger corrections**
- `types/logic.md`: A1/L1 annotation-only sufficient; A13/L2 MOUNT/ORD is integrator-placed; LC1–LC5 (corpus B536–B552); LC6 scheduler seam DI `Sched{at(delayMs);cancel(t)}`; "GROWING" header removed.
- `types/dashboard.md`: A15/D1 off-station derived keys; DUX1 route()→RouteAction -ux servlet seam; DUX2 purity gradient; DWS1 5-gate write-surface checklist; DWS2 canWrite(boolean) pure-RBAC test seam; DJS1 inline SPA JS + module.exports shim.
- `types/wb-widgets.md`: DWB1 wb/model Predicate-injection seam (turns seed into exemplar-backed).
- `build-verify.md`: "Known gap" retitled; A17/V1–V4 mode-B slotomatic gap documented. `build-verify.md` now carries zero "rt only" occurrences.
- `SOURCES.md` + `corpus-index.md`: A18/S1–S2.
- DashboardPan ledger: `handleSetpointWrite` gated by `DashboardRbacHelper.checkCanWrite` (B763); `profiles: rt,ux` (not `rt,ux,wb`); DashboardPan-wb scaffold noted. PR4 retro filed.

**PR5a — sweep-fold-audit.sh + verify-module coverage%**
- `toolbelt/sweep-fold-audit.sh`: audits that every folded INDEX.md row has an `[ev: retro <slug>]` citation in a core kit file; hyphen-aligned token matching; `retros/` excluded from corpus; exit 1 under `--strict`; CI runs non-strict (visible WARN, not blocking). Named mutations F3/F4/F6 proved.
- `toolbelt/verify-module.sh coverage <P> <F> <W> <S>`: integer-tenths gate-coverage percentage; `N-A` when all counts zero. Named mutations MM1–MM8 proved.
- Tests: `tests/sweep-fold-audit.bats` (F1/F2 + named mutations); `tests/verify-coverage.bats` (8 pins, merged from QA `d7e52a8`). PR5a retro filed.

**PR5b — preflight.sh + slot-coverage.sh**
- `toolbelt/preflight.sh`: JDK-8 check (under `--jvm-dir`/`JAVA_HOME`, never `$HOME`), plugin-pin in `etc/m2`, jar-lock via lsof (SKIP if absent), Windows-path FAIL with `/mnt/c/…` remedy. Named mutations PF1/PF2 proved.
- `toolbelt/slot-coverage.sh`: MM2 exposed-set coverage; pure `set-coverage <declared-csv> <required-csv>` subcommand (exact pin vectors SC1–SC5); parse subcommand reads `module-include.xml` + `module.lexicon`; empty-lexicon WARN (CompPan T8 footgun); dup-keys detection (B759/B780). Named mutations SC1–SC5 denominator/extra/N-A proved; SC6-parse mutation proved. PR5b retro filed.

**PR6 — Tracked launcher + openspec durability + v0.16.0 (this release)**
- **NEW `build-n4-module-kit/skill/SKILL.md`** — tracked canonical copy of the Claude skill launcher. Edits vs v0.3: `state` column removed from decision table (growing/mature/seed not actionable); explicit wb-builder note pointing to corpus B751–B753 and B762/B780 (`types/wb-widgets.md` is exemplar-backed but still thin); §Execution step 1 aligned with BUILD-LOOP §0 orient-then-corpus-nav order; frontmatter version bumped to "0.4".
- **NEW `scripts/install-skill.sh`** — copies the tracked skill into `<home>/.claude/skills/build-n4-module/SKILL.md` with sha256 comparison; `--home`/`--dry-run`/`--force` flags; exits 0/1/2/3; VCS-free. Tests IS1–IS4 + named mutation proved (drop last line → cmp fails). Suite identical under `HOME=/nonexistent`.
- **`openspec/` tracked** — the entire `openspec/` directory is committed for the first time (hybrid-store durability). `openspec/changes/niagara-tools-slotomatic-integration/` moved to `openspec/changes/archive/` with `archive-report.md` (100% superseded as of v0.15.1; all 40 tasks live in ng-deploy.sh + ng-deploy.bats; 0 applied through the openspec flow). `.gentle-ai-instance` added to `.gitignore` (runtime state, not repository content).
- **`ci.yml`** — `actions/setup-java@v4` → `@v5` (deprecation notice from PR2 CI run 33956501710).
- **`slot-coverage.sh`** — parse mode pct= line labeled `(type-set)` so `100.0` cannot be misread as per-slot coverage (B788: DashboardPan-rt type-set 100% but per-slot ~25%).
- **`BUILD-STATE.md`** — ledger correction: the false "CompPan-rt/module.lexicon is empty (T8)" open_issue replaced with real B788 findings (ColdRoomPan-rt partial 32 keys; DashboardPan-rt ~25% per-slot; DashboardPan-wb palette empty scaffold). Kit self-envelope updated: version 0.16.0, last_commit Campaign 6 PR6. `[ev: corpus B788]`.

### References

- SDD change: `build-n4-module-campaign6`; campaigns 1–5 closed prior releases (v0.13.x/v0.14.0/v0.15.x).
- Engram topic: `sdd/build-n4-module-campaign6/*`
- PRs: #35 (PR1) · #36 (PR2) · #37 (PR3) · #38 (PR4) · #39 (PR5a) · #40 (PR5b) · this PR (PR6)

---

## [v0.15.1] - 2026-09-04

### Added — CI: server-side enforcement of the checks (Campaign 5 AG-PR2)

- **NEW `.github/workflows/ci.yml`** — a GitHub Actions workflow that runs on every push to `main` and every
  pull request: `shellcheck` (scripts + toolbelt + tests, matching CONTRIBUTING §6), `bats tests/*.bats`
  (all 104 unit tests), and `sweep-build-state.sh` (BUILD-STATE ↔ retros/INDEX coherence). Least-privilege
  `permissions: contents: read`.
- **Why it complements the pre-push hook** (AG-PR1): the hook is client-side and opt-in (a clone that did not
  run `scripts/install-hooks.sh` bypasses it); CI runs server-side for everyone on every PR, so the same
  checks cannot be bypassed. Client hook + server CI = enforcement that is both immediate (local) and
  un-bypassable (remote). Repo-infra only — no kit behavior change.
- **shellcheck pinned to v0.10.0** (downloaded in the workflow) so CI is reproducible and does not drift with
  the runner's apt package; `actions/checkout@v5` (Node 24).
- Fixes the first CI run surfaced (all no behavior change, calibrated to the pinned shellcheck):
  `verify-module.sh` disables `SC2317` alongside `SC2329` (the indirectly-invoked `check_*` false positive);
  `build.sh` uses an explicit `if` arg-guard (clears `SC2015`); `ng-deploy.bats` disables `SC2154`
  (`$stderr` is a bats `run` var); and **`tests/kit-links.bats` L1 no longer resolves `SKILL.md` via a
  machine-specific `$HOME/.claude/skills/` path** — the launcher is external by design, so L1 now treats it
  as a known external pointer (this removed an environment coupling that made L1 pass locally but fail on a
  clean checkout — exactly what CI is for).

### References

- SDD change: `build-n4-module-continuity` Campaign 5 AG-PR2 (CI). Repo-infra addition → PATCH.

---

## [v0.15.0] - 2026-09-04

### Added / Changed — activate the retro-enforcement gate (Campaign 5 AG-PR1)

- **`.githooks/pre-push` hardened** — the promotion exit now accepts a `Retro: promotion (folds …)` trailer
  when there is an in-range `retros/INDEX.md` diff (a full promotion flips a row) **OR** an in-range
  `BUILD-STATE.md` diff (a PARTIAL promotion folds content while its source retro stays `pending`, so it
  flips no row — it stamps the owed `open_issue` instead). This fixes the false-negative on legitimate
  partial promotions found dogfooding C4-PR2. The blanket-escape guard is kept: a promotion trailer with
  NEITHER anchor still FAILS. Either anchored path now runs `sweep-build-state.sh` for ledger coherence.
- **NEW `scripts/install-hooks.sh`** — opt-in, per-clone, reversible activation of the gate: sets
  `git config --local core.hooksPath .githooks` (idempotent); `--uninstall` restores the default; it
  REFUSES to clobber a pre-existing custom `core.hooksPath` (your own hooks are never silently overwritten)
  unless `--force`. It lives under `scripts/` (not `toolbelt/`) because it uses git — the kit-links L2 rule
  bans git only in `toolbelt/*.sh`.
- **Docs** — activation documented in `build-n4-module-kit/BUILD-LOOP.md` §7 and `CONTRIBUTING.md` §6.1;
  the §7 promotion-exit contract updated to the INDEX-OR-BUILD-STATE structural anchor.
- `tests/build-retro-sync.bats`: H8/H9/H10 (QA, RED-first) green — H10 proves a BUILD-STATE-anchored partial
  promotion PASSES. `tests/install-hooks.bats`: I1-I4 (git-real on a throwaway repo) green.
- The gate still ships **inert** — this PR makes activation a one-command step; the live activation is a
  deliberate operator action (`scripts/install-hooks.sh`), not automatic.

### References

- SDD change: `build-n4-module-continuity` Campaign 5 AG-PR1 (gate activation). New behavior + new script → MINOR.

---

## [v0.14.1] - 2026-09-04

### Added — Campaign 4 close retro (process meta-lessons)

- `retros/2026-09-04-campaign4-close-process-meta-lessons.md` (+ INDEX row, `pending`): captures the three
  genuinely-new process lessons Campaign 4 surfaced — (1) the **stale-checkout trap** (verify coverage against
  a worktree off origin/main, never a local `main` that can sit a whole campaign stale; a line-number mismatch
  is the tell); (2) the **partial-promotion 4th-gate-shape** (a fold that flips no INDEX row is legitimate but
  the §7 promotion exit false-negatives it — accept a BUILD-STATE diff as an alternative anchor, tracked as a
  gate-hardening open_issue); (3) **grep every kit file before folding** (a lesson's mined target isn't the
  only place it may live — T4 was nearly double-folded because only its target file was grepped). This closes
  Campaign 4; the 42-lesson mined corpus is exhausted.

### References

- SDD change: `build-n4-module-continuity` Campaign 4 close. New retro filed → PATCH.

---

## [v0.14.0] - 2026-09-04

### Added — corpus-index.md: a curated map of the authoring corpus (Campaign 4 C4-PR3, terminal)

- **NEW `build-n4-module-kit/corpus-index.md`** (T7a) — a curated nav of the niagara-research authoring
  corpus (B729–B760): one row per block (block → what it gives the builder → layer), grouped **priority-first**
  (P0 read-before-building → P1 → P2) with layer as the sub-sort, plus a "start here" pointer (B744 rt / B760
  our-module / B751 -wb / B752 -ux). It serves the `corpus-nav FIRST` hard rule: `corpus-nav` for a term, this
  index for what to read by layer/priority. It is a curated pointer, not a dump — each block lives in
  niagara-research and is cited, not copied.
- **Wired in-repo** (minimal, no churn): a one-line pointer from BUILD-LOOP §2, METHODOLOGY top, and
  types/logic.md start-here.
- `retros/INDEX.md`: **corpus-index-rt-authoring-and-organization-blocks → `folded`** — all eight of its
  proposed deltas are now in core (corpus-index + pointers here; WB/UX growth in C4-PR2; lexicon in C4-PR1;
  composition B737 at logic.md §Composition · L21; schema/versioning B739/B754 at METHODOLOGY §Schema/upgrade
  safety · S1/S2/S3; build-target B756 at build-verify.md). This flips the row cleanly (in-range INDEX diff).
- **This closes Campaign 4 and exhausts the 42-lesson mined corpus.** Every mined lesson (logic L, UX U,
  build/deploy/schema B/S/D, testing/process T) is now folded, and every promotion was fidelity-graded.
- Companion (out-of-repo, not gated/CHANGELOG'd): the launcher `~/.claude/skills/build-n4-module/SKILL.md`
  `corpus-nav FIRST` rule + step 2 point at `$KIT/corpus-index.md` and at B760 for our-module work.

### References

- SDD change: `build-n4-module-continuity` Campaign 4 C4-PR3 (terminal). New KB artifact + wiring → MINOR.

---

## [v0.13.4] - 2026-09-04

### Changed — T7 b/c: grow the two weakest type guides from the WB/UX corpus (Campaign 4 C4-PR2)

- **types/wb-widgets.md** (T7b, B751) — folded the "how much wb is enough" ladder (rung 0 nothing → 1
  FieldEditor → 2 Manager → 3 custom View; our components sit at rung 0), the FieldEditor recipe, and the
  "a Honeywell Wizard is a tabbed `BWbComponentView`, not a `BWizard`" rule. The seed guide now carries the
  authoring ladder.
- **types/dashboard.md** (T7c, B752/B753) — folded the three-way serving-recipe decision (servlet-SPA vs
  bajaux `@AgentOn` vs PX; our DashboardPan is servlet-SPA, don't migrate) and the vendor-contrast RBAC
  caveat (Honeywell React SPAs ship `permissions="unrestricted"` — don't copy; the browser is never the
  security boundary). The RBAC core, pure router, traversal guard, and Home-Page footgun of B752 were
  ALREADY in the guide, so only the new parts were folded (no double-fold).
- `retros/INDEX.md`: unchanged — **corpus-index-rt-authoring-and-organization-blocks stays `pending`**
  (split-retro: its T8 half folded in C4-PR1, T7 b/c here, and T7a's corpus-index nav doc is owed to C4-PR3;
  the retro flips to `folded` only when T7a lands).

### References

- SDD change: `build-n4-module-continuity` Campaign 4 C4-PR2. Doc folds → PATCH. (Partial promotion — no
  INDEX flip yet; the source retro flips in C4-PR3.)

---

## [v0.13.3] - 2026-09-04

### Changed — T-group doc folds T5/T6/T8 (Campaign 4 C4-PR1)

- **BUILD-LOOP.md §3** (T5): operator preview + explicit OK is now a REQUIRED gate before building any
  `-ux` change — seed the mock with the state that triggers the new behavior, and note the mock does NOT run
  `-rt` logic (the preview approves the DASHBOARD, not the physical rt effect).
- **METHODOLOGY.md** new "Debugging" section (T6): on a log WARNING, classify caught-and-cosmetic
  (swallowed by `changed()`/`try`) vs propagates-and-aborts before treating it as the root cause — a captured
  stack trace up to `Station.startStation` is not itself a failure.
- **METHODOLOGY.md** lexicon item (T8): reinforced — a missing `module.lexicon` key silently renders raw
  camelCase via `toFriendly`; keys are module-global (shared slot names collide).
- `retros/INDEX.md`: **self-retro-preview-gate-wsl-tests-live-verify** → `folded` — its four lessons are now
  all in core (T5 §3 here, the pure-test recipe at build-verify.md, T4 live-anchor smoke already at
  build-verify.md, T6 here). **corpus-index-rt-authoring-and-organization-blocks stays `pending`** — its T8
  half is folded but its T7 body (corpus-index + WB/serving folds) is owed to C4-PR2/PR3.

### References

- SDD change: `build-n4-module-continuity` Campaign 4 C4-PR1. Doc folds → PATCH.

---

## [v0.13.2] - 2026-09-04

### Added — Campaign 3 close retro (process meta-lessons)

- `retros/2026-09-04-campaign3-close-process-meta-lessons.md` (+ INDEX row, `pending`): captures the three
  genuinely-new process lessons Campaign 3 surfaced — (1) a new gate/verify check must be high-signal /
  low-false-positive, scoped to the reported defect (empty-palette-with-types, rc/-backup files), not a
  literal completeness rule; (2) when a lesson removes a safety guard, prefer a frictionless WARN over a
  silent removal; (3) when a retro's prose and its PROPOSED-delta marker disagree, the propose-never-apply
  marker is what promotes. This closes Campaign 3.

### References

- SDD change: `build-n4-module-continuity` Campaign 3 close. New retro filed → PATCH.

---

## [v0.13.1] - 2026-09-04

### Changed — kit doc tidy pass (Campaign 3 close, doc-only)

- **METHODOLOGY.md**: added a new "Kit maintenance — retro promotion discipline" section folding two
  process rules from the Campaign 2 close retro — the **doc-vs-script folded-completeness** rule (a
  script/gate lesson is only PARTLY folded by its prose; the impl stays owed and tracked) and the
  **adversarial fidelity-grading** rule (a promotion's correctness is faithfulness-to-source +
  fold-completeness, so budget an independent per-lesson fidelity pass — a green suite can't verify it).
- **types/dashboard.md** (U1): spelled out the detail-render height nuance — give the frame a definite
  `height:100%` (not `height:auto`), else a near-square image overflows and `overflow:hidden` clips the
  bottom zone label; the SVG `meet` then letterboxes inside a definite height.
- **Tag-convention normalization**: the two mixed-format retro citations that put a retro-local `#N` after
  the `·` separator (`comppan-fase3 · #2`, `process-timers · #2 + L16`) now use the full retro slug — neither
  lesson has a global L/U/B tag (fase3 #2 emerged during folding; process-timers #2 folds under L16), so the
  honest fix is the slug, not an invented tag. The ~16 `rt-hardening #N` / `5rooms #N` citations are
  legitimately pre-global-tag and left as-is.
- `retros/INDEX.md`: flipped **campaign2-promotion-process-meta-lessons** → `folded` (its three meta-lessons
  are now all in core: split-retro rule in BUILD-STATE, doc-vs-script + fidelity-grading in METHODOLOGY).

### References

- SDD change: `build-n4-module-continuity` Campaign 3 close (tidy pass). Doc-only → PATCH.

---

## [v0.13.0] - 2026-09-04

### Added — verify-module.sh: empty-palette gate check (Campaign 3 C3-PR4)

- `build-n4-module-kit/toolbelt/verify-module.sh` (stays VCS-free):
  - **palette check** (default-on, no `--src`) — a module that declares types but ships an **empty
    `module.palette`** (only the `b:Folder` root, zero `<p n=…>` component entries) builds, passes the whole
    gate, deploys, and then shows **nothing to drag in Workbench** — commissioning is silently broken.
    The check now WARNs (or FAILs under `--strict`), naming the empty palette and the declared type count.
    It SKIPs when the jar has no `module.palette`, and PASSes a populated one. A **typeless** module with an
    empty palette does NOT warn (the type-count guard keeps it high-signal, not a nag).
- `tests/verify-module.bats`: V13–V17 (QA, RED-first) now green; V1–V12 unchanged.
- `retros/INDEX.md`: flipped **module-palette-and-build-target** → `folded` — its last pending half (the gate
  check) is now implemented; the palette authoring rule (B5, METHODOLOGY) and the per-module build target
  (B6, build.sh) were already folded.
- This closes the owed-script-implementation backlog: B4, B6, B7, B8, B10, soft-start, and palette are all
  implemented and folded.

### References

- SDD change: `build-n4-module-continuity` Campaign 3 C3-PR4. New default-on gate check → MINOR.

---

## [v0.12.0] - 2026-09-04

### Changed — ng-deploy.sh: type-count fix, lightweight-backup default, --no-backup WARN (Campaign 3 C3-PR3)

- `scripts/ng-deploy.sh`:
  - **B8** (`ng-deploy-type-count`) — `verify_jar` counted module.xml `<type` entries with `grep "<type"`,
    which ALSO matched the `<types>` wrapper element → off-by-one (a correct jar reported N+1 and failed).
    Fixed to `grep "<type "` (trailing space): only real entries count.
  - **B10** (`ng-deploy-backup-liviano-y-autopurga`) — backup is now **lightweight by default**: it archives
    only this module's own `<MODULE>-{rt,ux,wb}.jar` present in `STATION_MODULES_DIR`, not the whole modules
    dir (the old default grew `_backups/` ~240 MB/deploy without bound). A **keep-N autopurge** (default 3,
    `--keep N`) prunes older backups after each successful one.
  - **`--no-backup` gate removed → WARN** — no longer requires `--i-know-what-im-doing`; it is a plain opt-in
    that prints a one-line rollback reminder (`backup skipped … committed to git for rollback`).
    `--i-know-what-im-doing` stays as an accepted no-op for backward compatibility.
  - **`--full-backup` / `FULL_BACKUP=1`** restores the old whole-modules-dir backup.
  - Exit code **20** now means a genuine backup failure only (no longer the removed gate).
- `tests/ng-deploy.bats`: B8 + B10 i/ii/iii/iv (QA, RED-first) now green; T1–T33 unchanged.
- `retros/INDEX.md`: flipped **ng-deploy-type-count** (B8) and **ng-deploy-backup-liviano-y-autopurga** (B10,
  promoted rule = lightweight-default + keep-N autopurge + `--no-backup` opt-in + `--full-backup`) → `folded`.
  kit self-section: owed B8/B10 → DONE; last owed = the verify-module palette check (→ C3-PR4).

### References

- SDD change: `build-n4-module-continuity` Campaign 3 C3-PR3. New ng-deploy.sh behavior → MINOR.

---

## [v0.11.0] - 2026-09-04

### Added — build.sh: auto-detect plugin, gradle-root walk-up, clean-lock message (Campaign 3 C3-PR2)

- `build-n4-module-kit/toolbelt/build.sh` (stays git-free):
  - **B6** — when no `--plugin-version` / `$NIAGARA_PLUGIN_VERSION` is given, auto-detect the module's own
    pinned `niagaraPluginVersion` from its `gradle.properties` (or `settings.gradle.kts` getOrElse) and
    forward it; an explicit flag/env still wins.
  - **B7** — if `<ROOT>/gradlew` is absent, walk UP to the gradle root (client multi-project layout) and run
    the `:MOD-p:` tasks there; profiles stay resolved under `<ROOT>/<MOD>/<MOD>-p`.
  - **soft-start** — capture gradle output; a `:clean` failure matching `Unable to delete …/modules/…jar`
    (a running-station lock) prints an actionable message (free the lock / mirror / use build/libs) and exits
    a distinct **31** (documented in the header), instead of the generic 30.
- `tests/build-sh.bats`: B8/B9/B10 (QA, RED-first) now green; B1–B7 unchanged.
- `retros/INDEX.md`: flipped **dashboardpan-ux-direct-build** (B7) and **soft-start-staggered-startup** (its
  build.sh clean-lock delta) → `folded`. **module-palette-and-build-target stays `pending`** — B6 is done, but
  its retro also owes a verify-module palette check (→ C3-PR4). kit self-section: B6/B7/soft-start owed → DONE;
  owed now B8/B10 (ng-deploy → C3-PR3) + the palette check (→ C3-PR4).

### References

- SDD change: `build-n4-module-continuity` Campaign 3 C3-PR2. New build.sh behavior → MINOR.
- Engram: #8120 (Campaign 2 complete).

---

## [v0.10.0] - 2026-09-04

### Added — verify-module.sh rc/ editor-backup check + `--strict` (Campaign 3 C3-PR1, B4 impl)

- `build-n4-module-kit/toolbelt/verify-module.sh`: new default-on `rcbackup` check — editor/backup files
  (`*~`, `*.orig`, `*.bak*`) packaged under `rc/` (bloat + servable) emit a `WARN rcbackup` row NAMING the
  offenders; the WARN does NOT increment the failure count (exit stays 0 for an otherwise-valid jar). New
  `--strict` flag promotes it to `FAIL rcbackup` (exit 1). A `WARN` severity was added to the reporter and
  a `warned` count to the summary. Implements the owed B4 (its rule was folded into build-verify.md in
  Campaign 2 PR-C).
- `tests/verify-module.bats`: V10 (WARN, exit 0) / V11 (clean rc silent) / V12 (`--strict` → FAIL), RED-first.
- `retros/INDEX.md`: `dashboardpan-detail-render-doors` → `folded` (its last owed lesson B4 is now
  implemented; U1/U4/D3 were folded in Campaign 2). kit self-section: B4 moved owed → DONE (5 impls owed).

### References

- SDD change: `build-n4-module-continuity` Campaign 3 C3-PR1. New capability (WARN check + `--strict` flag) → MINOR.
- Engram: #8120 (Campaign 2 complete).

---

## [v0.9.4] - 2026-09-04

### Added — Campaign 2 close retro (promotion-process meta-lessons)

- `retros/2026-09-04-campaign2-promotion-process-meta-lessons.md` (+ INDEX row): captures the three
  process lessons Campaign 2 proved — (a) the fold-vs-revert (split-retro) rule; (b) the doc-vs-script
  folded-completeness rule (a script/gate lesson is only partly folded by documenting its rule; impl owed
  → `pending` + open_issue); (c) conservative adversarial fidelity grading catches folded over-claims a
  green bats suite alone misses (5 caught across the promotion PRs). The mechanism documenting its own
  process — the honest Campaign 2 close.

### References

- SDD change: `build-n4-module-continuity` Campaign 2 close. New-retro exit; documentation only → PATCH.
- Engram: #8120 (Campaign 2 complete).

---

## [v0.9.3] - 2026-09-04

### Changed — Campaign 2 PR-C: fold build/deploy/schema lessons (Campaign 2 complete)

- `BUILD-LOOP.md`: §0.b free-the-lock-first (B2 — folding the last mirror-first instance) + read THIS
  module's gradle target (B6); §6 -ux-only = no restart (D1), station-on-another-device → verify live via
  oBIX (D2), power-cycle the panel after redeploy (D3), cd to the gradle root before ng-deploy (B9), the
  ng-deploy no-backup-default rule (B10), EXPECTED_*_TYPES = real+1 (B8).
- `build-verify.md`: nested/multi-project gradle layout — run :project:task from the gradle root (B7);
  keep src/rc asset-only (B4); a new §Module versioning & release (S4).
- `METHODOLOGY.md` §Schema/upgrade safety: the SAFE/LOSSY/OUTAGE saved-data survival matrix + bump
  vendorVersion / back up config.bog (S2); an additive slot change needs a version bump to auto-install (S3).
- Flipped 3 retros to `folded` (pure-doc, fully in core): ux-only-deploy, station-corre-en-atlas-snap,
  coldroompan-fan-mode-defrost (S3 + the delete-type-and-registration rule B12). soft-start stays
  `pending` (its build.sh clean-lock-message delta is owed). SIX rule-folded lessons whose IMPLEMENTATION
  is a script/gate change (B4/B6/B7/B8/B10 + soft-start build.sh) stay `pending`, tracked as kit
  self-section open_issues (future MINOR PRs); the T/process group stays pending (out of Campaign 2 scope).

### References

- SDD change: `build-n4-module-continuity` Campaign 2 PR-C (build/deploy/schema — Campaign 2 complete).
  Promotion of already-filed lessons → `Retro: promotion` exit; documentation only → PATCH.
- Engram: #8118 (Campaign 2 progress), #8101 (42-lesson mining).

---

## [v0.9.2] - 2026-09-04

### Changed — Campaign 2 PR-B: fold UX lessons U1–U9 into the core

- `types/dashboard.md`: promoted the dashboard/UX [CERT] lessons — overlay raster + vectors as ONE SVG
  (the WebView ignores CSS aspect-ratio/object-fit · U1); the 2-column no-scroll rule for the Control/HOA
  panel (U2); a reusable output row with custom buttons + special-cased prefill (U3); a per-room link-in
  boolean is SUMMARY not OPERATOR (U4); the reader's group arrays are the authority for the HMI surface
  (U6); the dashboard contract as an external port spec (U7); `dashboard-preview.py` --editor/--mock + the
  preview→Playwright→PDF loop (U9); the reusable add-a-tab + wire-data SPA recipe (U10); and the
  process/defrost timer facade+reader path — a READONLY BAbsTime
  anchor + a reader AbsTime type-reader emitting derived elapsed/remaining keys (process-timers #2 + L16).
- `METHODOLOGY.md`: the anchored-Python-`str.replace` edit method for a base64-heavy SPA (U8).
- Flipped `retros/INDEX.md` to `folded` for the 5 retros now fully in core (hmi-1280x800,
  dashboard-servlet-write-surface, dashboard-contract-port-spec, editing-base64-heavy-spa, and
  process-timers — its split completed). kit self-section open_issue narrowed to the remaining B / D / S groups.

### References

- SDD change: `build-n4-module-continuity` Campaign 2 PR-B (UX group complete: U1-U9; U5 was folded in
  Campaign 1). Promotion of already-filed lessons → `Retro: promotion` exit; documentation only → PATCH.
- Engram: #8118 (Campaign 2 progress), #8101 (42-lesson mining).

---

## [v0.9.1] - 2026-09-04

### Changed — Campaign 2 PR-A2: fold logic-group lessons L16–L22 into the core

- `types/logic.md`: promoted 7 [CERT] control-logic lessons — a polled-UI time is a READONLY `BAbsTime`
  anchor, not a stored `BRelTime` (L16); reserve `Flags.HIDDEN` for engine callbacks, expose a public
  force/run action for operator/oBIX (L17); guard a link-reachable `Clock.schedule` with
  `if(!Sys.atSteadyState())return`, set the anchor before scheduling, `period>0` (L18); an internal-only
  discrete selector is a `BFrozenEnum`, the double rule is cross-module only (L19); a time-gated auto
  control needs a run-now action + surfaced preconditions + an anchored first fire (L20); a new
  §Composition & organization — compose past ~12–15 slots, config separate from live-state (L21); the HOA
  manual-override slot pattern (L22).
- Flipped `retros/INDEX.md` to `folded` for the 3 retros now fully in core (hidden-actions,
  self-firing-timer, hoa-manual-override); process-timers stays `pending` (its logic lessons L16/L20 are
  folded here, but its dashboard/reader AbsTime lesson #2 folds in PR-B). kit self-section open_issue
  narrowed to the U / B / D / S groups.

### References

- SDD change: `build-n4-module-continuity` Campaign 2 PR-A2 (logic group complete: L3-L22). Promotion of
  already-filed lessons → `Retro: promotion` exit; documentation only → PATCH.
- Engram: #8118 (Campaign 2 progress), #8101 (42-lesson mining).

---

## [v0.9.0] - 2026-09-04

### Added — §7 close-gate third exit: `Retro: promotion` (MINOR)

- `.githooks/pre-push`: a THIRD accepted close for a build-relevant kit change, parallel to the
  new-retro path and the trivial waiver — a `Retro: promotion (folds <ids> from existing retros)`
  commit trailer, valid ONLY when the range also carries a `retros/INDEX.md` diff (the registry move).
  A promotion trailer with no INDEX diff is rejected (fail-closed), so it cannot be a blanket escape.
  Closes the gap where a lesson-promotion PR (Campaign 2) matched neither existing exit and would
  otherwise be forced into a FALSE `trivial` label. `sweep-build-state.sh` is unchanged (VCS-free,
  kit-links L2) — the trailer is a commit/diff concept the hook owns.
- `build-n4-module-kit/BUILD-LOOP.md` §7: documents exit (c).
- `tests/build-retro-sync.bats`: H8 (promotion + INDEX diff → pass) / H9 (promotion, no INDEX → fail),
  RED-first, mutation-checked. 18 cases total.
- Retro `2026-09-04-gate-exit-taxonomy-promotion.md` (+ INDEX row) captures the exit-taxonomy lesson.

### References

- SDD change: `build-n4-module-continuity` Campaign 2 (gate hardening; MINOR — a new hook behavior).
- Engram: #8114 (Campaign 1 complete). Sequenced after PR-A (0.8.2); PR-A2/B/C follow as 0.9.1/0.9.2/0.9.3.

---

## [v0.8.2] - 2026-09-04

### Changed — Campaign 2 PR-A: fold logic-group lessons L3–L14 into the core

- `types/logic.md`: promoted 11 [CERT] control-logic lessons from `retros/` into the living guide —
  invalid-reading-vs-bad-value / assume-commanded on a dead sensor (L3); latched fault + operator
  `faultReset`, never auto-retry rotating equipment (L4); proof-of-run + stuck-contactor alarms (L5);
  persist-only-survival + `resetTransient()` in `stopped()` (L7); arm the heartbeat tick in its own try
  before the first `execute()` (L8); the no-reentrancy invariant — outputs ∉ `changed()` filter (L9);
  cancel every ticket on one shared path at both edges + mode enter/exit (L10); a decoupled output needs
  its own apply method wired into every setter (L11); ship an additive phase dormant behind a default,
  old tests guard for free (L12); stub an unvalidated safety-critical data function to NaN, never invented
  numbers (L13); self-order by a guarded sibling tree-walk, not a coordinator component (L14); and the input-only-phase boundary rule (fase3 #2) — a phase changing only an INPUT pushes new logic to the adapter boundary, leaving the proven core and its tests untouched.
- Flipped `retros/INDEX.md` to `folded` for the 3 retros now fully in core (comppan-fase1/2/3); updated the
  `kit` self-section open_issue to the remaining L16-L22 / U / B / D / S groups. Excludes L6 (already folded
  in Campaign 1).

### References

- SDD change: `build-n4-module-continuity` Campaign 2 (PR-A of PR-A/A2/B/C). Documentation only → PATCH.
- Engram: #8114 (Campaign 1 complete), #8101 (42-lesson mining).

---

## [v0.8.1] - 2026-09-04

### Changed — promote proven retro lessons into the kit core (PR3)

- **Lesson folds (documentation):** promoted the highest-value [CERT]-grade lessons trapped in `retros/`
  into the living kit files —
  - `METHODOLOGY.md`: never retype a live slot / bog boot crash (S1) + NEW-safety-slot-defaults-SAFE
    carve-out (L15) under a new §Schema/upgrade safety; `0`-means-DISABLED on a protection limit (L2);
    `module.palette` one entry per `@NiagaraType` (B5); the timer `started()`+`atSteadyState()` pointer;
    the 4-layer assurance pointer.
  - `types/logic.md`: `Long.MIN_VALUE` sentinel guard (L1); the slot-default SAFE carve-out (L15); a
    §Pure-class extraction section (extract-test-then-wire, stateful `step()` adapter · T2/L6); cross-refs.
  - `types/dashboard.md`: a generic write endpoint must whitelist / check `Flags.OPERATOR` (U5).
  - `build-verify.md`: free-the-lock-first vs mirror-fallback (B2) + build/libs-still-signed (B1) +
    first-build-no-stop (B3); the `find ~/.gradle` + `run-pure-test.sh` recipe and niagaraTest-is-docs
    (B11); a §How you know it's good 4-layer assurance stack (T1/T3/T4).
- **Contradictions resolved to ONE rule each:** jar-lock (free the lock first; mirror only for a live
  production station) → `build-verify.md`; slot default (preserve current on a deployed path, but a NEW
  safety-gating slot defaults SAFE) → `METHODOLOGY.md` + `types/logic.md`.
- **`kit` self-section** added to `BUILD-STATE.md` (`module: kit`) so kit-infrastructure work has a row to
  update; `BUILD-LOOP.md` §7 notes it. Dogfood: this campaign filed TWO retros (the campaign retro + the
  run-pure-test `set -e` empty-cache lesson) with `retros/INDEX.md` rows, and marked 4 fully-folded retros
  `folded` — the mechanism recording its own evolution.

### References

- SDD change: `build-n4-module-continuity` (PR3 of 3; follows PR1 v0.6.0 ledger, PR2 v0.7.0 gate, v0.8.0
  run-pure-test). Documentation + schema-section only, no new script → PATCH.
- Engram: #8095 (campaign), #8101 (42-lesson mining), #8105 (tasks), #8111 (chain status).

---

## [v0.8.0] - 2026-09-04

### Added — one-command pure-test runner (`run-pure-test.sh`)

- `build-n4-module-kit/toolbelt/run-pure-test.sh` (NEW): runs a ZERO-Baja pure decision class +
  its JUnit test standalone in WSL, resolving junit-4.13.2 + hamcrest-1.3 from `~/.gradle` and
  compiling into a TEMP dir (never the module tree — a parallel session's working tree is
  off-limits). Removes the per-session jar hunt documented in the
  `2026-09-04-junit-standalone-cached-jar-locations` retro. Exit 0 pass / 1 test failed / 2 usage /
  3 env. Proven against the real CompPan `CompressorControlTest`: 31 tests OK in 0.015s.
- `tests/run-pure-test.bats` (NEW): 6 cases, each mutation-checked to bite — P2 (a failing pure
  test must exit non-zero, i.e. the runner does not swallow a JUnit failure) and P6 (an empty
  `~/.gradle` cache exits 3 with an actionable message) both caught real behaviour, the latter a
  `set -e`/pipefail bug in the runner itself.

---

## [v0.7.0] - 2026-09-04

### Added — retro-enforcement gate (sweep + pre-push hook + retro index)

- `build-n4-module-kit/toolbelt/sweep-build-state.sh` (NEW): content-only validator (VCS-free —
  kit-links L2) for BUILD-STATE.md + retros/INDEX.md. Exit 0 clean / 1 named integrity violation /
  3 usage. Checks column-0-anchored `build-state.v1` markers, required boolean fields
  (module/retro_required/retro_pending), a tolerated multi-line `open_issues` list, and INDEX
  integrity (every retro has a row, every row a real file, review-status ∈ {pending,folded}).
- `.githooks/pre-push` (NEW): opt-in gate (enable with `git config core.hooksPath .githooks`).
  Blocks a push that changes build-relevant kit files (`build-n4-module-kit/**` except BUILD-STATE.md
  and `retros/`, plus `scripts/**`) unless the range also carries a BUILD-STATE.md update + a pending
  retro + its INDEX row, or a `Retro: none (trivial: <reason>)` trailer. Delegates the content half
  to `sweep-build-state.sh` (the diff half owns the only VCS calls).
- `build-n4-module-kit/retros/INDEX.md` (NEW): the promotion registry — one row per retro (30
  seeded; 3 folded, 27 pending), with a `deltas` count column.
- `tests/build-retro-sync.bats` (NEW, RED-first): 16 cases (9 sweep + 7 hook), each
  mutation-checked to bite.
- `build-n4-module-kit/BUILD-LOOP.md` §7: parenthetical updated to point at the now-live gate.

### References

- SDD change: `build-n4-module-continuity` (PR2 of 3; follows PR1 v0.6.0 continuity ledger).
- Engram: #8095 (campaign), #8101 (recon), #8105 (tasks), #8107 (PR1 landed).

---

## [v0.6.0] - 2026-09-04

### Added — build-session continuity ledger (BUILD-STATE.md)

- `build-n4-module-kit/BUILD-STATE.md` (NEW): a "where did we leave off" registry for the build
  loop — one `build-state.v1` envelope per module (ColdRoomPan, CompPan, DashboardPan, chihuahua),
  seeded from evidence. The build-loop analog of research-sdd's FOCUSES + RESEARCH-STATE registries,
  collapsed into one file. Fields split GATED (`retro_required`, `retro_pending` — kit-local only)
  vs DECLARED (jar/module/src state, which lives in separate repos and is recorded, not gated).
- `build-n4-module-kit/BUILD-LOOP.md`: new `§0.a Orient from BUILD-STATE` (prints a one-line
  leave-off per module, with the meta-work exemption) and a reworded `§7` HARD close gate — update
  BUILD-STATE and write a retro, or declare `Retro: none (trivial: <reason>)`; print the `retro:`
  line in the Output Contract.
- Launcher `SKILL.md` (in `~/.claude/skills/build-n4-module/`, outside this repo): orient from
  BUILD-STATE at step 1, update it at step 6, print the retro line in the Output Contract.

### References

- SDD change: `build-n4-module-continuity` (PR1 of 3 — continuity registry; PR2 = retro-enforcement
  gate, PR3 = promote proven lessons). Tasks: `openspec/changes/build-n4-module-continuity/tasks.md`.
- Engram: #8095 (campaign kickoff), #8101 (recon: continuity design + 42-lesson mining), #8105 (tasks).

---

## [v0.5.0] - 2026-09-01

### Added — ng-deploy verify-module gate and `-ux` slotomatic

- `scripts/ng-deploy.sh`: runs the `build-n4-module-kit/toolbelt/verify-module.sh` gate on the
  built jars during the verify step, default ON for mode A/C. New `--no-gate` flag to skip it and
  `VERIFY_MODULE_BIN` env var to override the gate binary (used by the tests).
- `scripts/ng-deploy.sh`: runs `:MODULE-ux:slotomatic` (in addition to `:MODULE-rt:slotomatic`)
  under `--with-slotomatic` in mode A when the `-ux` profile source carries a
  `@Niagara(Type|Property|Action|Topic|Singleton)` annotation (presence-based scan).
- `tests/ng-deploy.bats`: 7 new cases — T27–T29 (`-ux` slotomatic) and T30–T33 (verify-module
  gate). Each was mutation-checked to fail when its target behavior is broken. 33 total.

### Changed — ng-deploy verify-module gate and `-ux` slotomatic

- `scripts/ng-deploy.sh`: exit 50 ("verify failed") now also covers a failed `verify-module.sh`
  gate, alongside the existing type-count mismatch and missing-BUILD_ID cases.
- `scripts/ng-deploy.sh`: mode-B `--with-slotomatic` warning enriched to point at
  `--mode A --with-slotomatic` or `toolbelt/build.sh` for regenerating `-ux` slots (behavior
  unchanged — mode B still warns and skips).

### References

- SDD context: build-n4-module-kit v0.2 retro §6 follow-ups P1 (ng-deploy gate) + P2 (`-ux` slotomatic).
- Engram: #7953 (this change).

---

## [v0.4.0] - 2026-09-01

### Added — `build-n4-module-kit-v0.2`

- `build-n4-module-kit`: fold-in of 41 proven lessons from 3 retros (ColdRoomPan rt-hardening,
  DashboardPan 5-rooms, DashboardPan HMI touch UX) and the concurrent session bitácora into
  7 kit files (`types/dashboard.md`, `types/logic.md`, `build-verify.md`, `METHODOLOGY.md`,
  `BUILD-LOOP.md`, `SOURCES.md`, `types/wb-widgets.md`). Evidence markers (`[CERT]`,
  `[CERT-live]`, `[INFER]`) preserved from source retros.
- `toolbelt/verify-module.sh`: THE verify gate — bytecode major 52, `NIAGARA4.SF` presence,
  `module.xml` type-to-class resolution; opt-in `--target-version`, `--stored`, `--src` checks.
  POSIX-only (no JDK). Exit codes 0/1/2/3.
- `toolbelt/build.sh`: rewrite — source-based profile selection (skips stub `-wb`), Java 8 +
  clean + slotomatic + jar, runs `verify-module.sh` on every produced jar. Exit codes 2/10/30/50.
- `toolbelt/mirror-niagara-home.sh`: safe writable mirror of a live Niagara install for builds
  against a running station (refuses real install or any non-mirror dir, exit 20).
- `toolbelt/stored-repack.sh`: STORED repackage for the Workbench re-sign path (B7 recipe);
  manifest-first ordering, `zip -0`, verifiable with `verify-module.sh --stored`.
- `tests/verify-module.bats`, `tests/build-sh.bats`, `tests/mirror-niagara-home.bats`,
  `tests/stored-repack.bats`, `tests/kit-links.bats`: 5 bats suites (27 tests; 53 total with ng-deploy.bats) with generated-jar
  fixtures helper `tests/helpers/n4-fixtures.bash`.

### Changed — `build-n4-module-kit-v0.2`

- Kit doctrine rewritten to three explicit roles: `verify-module.sh` (THE gate),
  `build.sh` (recommended WSL build, runs the gate), `ng-deploy.sh` (station deploy wrapper —
  backup → build → copy → type-count verify; slotomatic guard rt-only). Replaces the ambiguous
  primary/fallback framing in `build-verify.md` and `BUILD-LOOP.md`.
- `toolbelt/build.sh` exit codes: 2 usage error, 10 env/path error, 30 gradle failure,
  50 verify-module gate failure (supersedes previous informal codes).
- `CONTRIBUTING.md`: bats-core install step added to the test-runner prerequisites section.

### Fixed — `build-n4-module-kit-v0.2`

- Dangling kit links (`checklist-common.md` → `METHODOLOGY.md`;
  `type-dashboard.md` → `types/dashboard.md`) removed from kit files.
- `build.sh`: stub-profile selection now skips profiles with a `build.gradle` but no source files
  (was silently including them).
- `build.sh`: all-classes bytecode major check now correctly catches a later `.class` at major 65
  when the first `.class` in the jar is major 52 (false-pass fixed).

### Notes

- Launcher `~/.claude/skills/build-n4-module/SKILL.md` updated to v0.2 (three-role doctrine,
  verify-module.sh in Output Contract). This file lives outside the git repository; it is not
  included in this tag but is documented in the PR3 body and in engram.

### References

- SDD slug: `build-n4-module-kit-v0.2`
- Engram: explore #7937, proposal #7938, spec #7939, design #7940, tasks #7943.
- Tag: `v0.4.0`.

---

## [v0.3.0] - 2026-05-18

### Added — `niagara-tools-slotomatic-integration`

- `scripts/ng-deploy.sh`: `--with-slotomatic` flag — invoca `:MODULE-rt:slotomatic` con los
  3 overrides `-P` ANTES de `build_jars` (modos A/C; ignorado en modo B con WARN).
- `scripts/ng-deploy.sh`: `--strict-slotomatic` flag — aborta con exit 15 si se detectan
  cambios de anotación sin `--with-slotomatic`. No implica `--with-slotomatic` (cero magia).
- `scripts/ng-deploy.sh`: `SLOTOMATIC_DETECTION` env var (`warn`|`strict`|`off`, default `warn`) —
  controla la heurística pasiva de detección de cambios en anotaciones `@Niagara*`.
- `scripts/ng-deploy.sh`: `detect_annotation_changes()` — heurística pasiva que corre entre
  backup y build. Lee `.last-deploy-sha` como baseline (fallback `HEAD~1`); filtra diff con
  `grep -E '^[+-][[:space:]]*@Niagara(Type|Property|Action|Topic|Singleton)'`.
- `scripts/ng-deploy.sh`: `run_slotomatic()` — invoca gradlew con los mismos 3 `-P` overrides
  que `build_jars`; `die 15` si gradlew retorna distinto de cero.
- `scripts/ng-deploy.sh`: `write_last_deploy_sha()` — escribe `git rev-parse HEAD` en
  `.last-deploy-sha` post-verify exitoso; silencioso si git falla.
- `scripts/ng-deploy.sh`: `read_baseline_sha()` — lee `.last-deploy-sha`, valida con
  `git cat-file -e`, fallback a `HEAD~1` si ausente/vacío/SHA inválido.
- `scripts/ng-deploy.sh`: `warn_slotomatic_recommended()` — heredoc multi-línea a stderr.
- `scripts/ng-deploy.sh`: exit code 15 nuevo — slotomatic falló O cambios de anotación
  detectados en modo strict. Ningún code path existente (0-50) fue modificado.
- `tests/ng-deploy.bats`: 9 tests nuevos (T18–T26), total 26. git fakebin en setup().
  Refactor gradlew stub: `gradlew.calls.log` (log acumulativo) + `gradlew.args` (backward-compat).
- `.env.local.example`: sección `SLOTOMATIC_DETECTION` + nota sobre `.last-deploy-sha` en `.gitignore`.
- `docs/knowledge-base/slotomatic.md`: Card 4 — ng-deploy.sh integration (flags, detection
  mechanics, decision table, false-positive edge case, `.last-deploy-sha` gitignore note).
- `docs/GOTCHAS.md`: fila anti-pattern "Deploy with stale slotomatic".
- `tests/smoke-checklist.md`: paso opcional `--with-slotomatic` en modo A; nota no-op en modo B.

### Changed — `niagara-tools-slotomatic-integration`

- `scripts/ng-deploy.sh` `print_usage()`: documenta `--with-slotomatic`, `--strict-slotomatic`,
  `SLOTOMATIC_DETECTION` env var, y exit code 15.
- `CLAUDE.md` §1 tabla: fila "Slot/Property/Type/Action" actualizada →
  `A --with-slotomatic (or :slotomatic separately first)`.
- `scripts/ng-deploy.sh` header comment: añade `--with-slotomatic`, `--strict-slotomatic` a Usage.

### References

- SDD slug: `niagara-tools-slotomatic-integration`
- Engram: spec #1927, design #1928, tasks #1929.
- Tag: `v0.3.0` (pendiente de commit del operador).

---

## [v0.2.0] - 2026-05-18

### Added — `niagara-tools-versioning-and-contributing`

- `VERSION` file at repo root as the single source of truth for the version string.
- `CHANGELOG.md` (this file) in Keep a Changelog format, seeded with retroactive `[v0.1.0]`
  (bootstrap) and the current `[v0.2.0]` entries.
- `CONTRIBUTING.md` (solo+agents scope): release recipe, TDD gate, KB authoring rules,
  conventional commits, pre-commit checklist, versioning policy, content-boundary matrix,
  and known limitations.
- `scripts/ng-deploy.sh`: `--version` flag prints `SCRIPT_VERSION` and exits 0. Value is
  resolved CWD-agnostically via `BASH_SOURCE[0]`-relative read of `VERSION`
  (`cat "${SCRIPT_DIR}/../VERSION"`), with an `"unknown"` fallback when the file is missing.
- `tests/ng-deploy.bats`: 2 new tests (17 total) — `--version` exit/output contract and
  anti-drift regression guard for the path resolution.
- `CLAUDE.md` §9 "Release process": one-liner bump rule + pointer to `CONTRIBUTING.md`
  §Release process. (Decision tree only — no policy duplication; see proposal-decisions #1846.)
- `README.md`: "Versioning" section linking to this changelog.
- Git tags: `v0.1.0` retroactive on bootstrap commit `8d9a396`; `v0.2.0` on this commit.

### Changed — `niagara-tools-versioning-and-contributing`

- `scripts/ng-deploy.sh` header Usage comment: added `--version` next to `--help` in the
  metadata-flags Usage line.

### References

- SDD slug: `niagara-tools-versioning-and-contributing`
- Engram: explore #1843, proposal #1844, proposal-decisions #1846, spec #1848, design #1847.
- Tag: `v0.2.0` (this commit).

---

## [v0.1.0] - 2026-05-17

### Added — `niagara-tools-bootstrap` (retroactive entry)

This release documents the initial bootstrap of the `niagara-tools` repo. The actual commit
(`8d9a396`) shipped before any versioning infrastructure existed; this entry is the retroactive
`v0.1.0` record, tagged after the fact on the bootstrap commit.

- `scripts/ng-deploy.sh`: bash wrapper for the Niagara N4 module deploy cycle
  (backup → build → copy → verify). Flag API: `--mode A|B|C`, `--env-file PATH`,
  `--no-deploy`, `--no-backup --i-know-what-im-doing`, `--help`. Exit codes: 0 (success),
  10 (env/path), 20 (backup), 30 (build), 40 (copy), 50 (verify).
- `.env.local.example`: config schema with all 8 required env vars (`MODULE_NAME`,
  `GRADLEW_PATH`, `NIAGARA_HOME`, `NIAGARA_USER_HOME`, `JAVA_HOME`, `STATION_MODULES_DIR`,
  `EXPECTED_RT_TYPES`, `EXPECTED_UX_TYPES`) and optional `BUILD_ID` cache-buster.
- `tests/ng-deploy.bats`: 15 bats-core unit tests using PATH-injected fakebin stubs for
  `gradlew`, `unzip`, `tar`. No real station or build dependency.
- `tests/smoke-checklist.md`: manual integration checklist for modes A, B, C.
- `docs/GOTCHAS.md`: cross-project anti-patterns index linking the KB topic files.
- `docs/knowledge-base/`: four topic files seeded from chihuahua learnings —
  `bql-gotchas.md` (BQL N4.14 confirmed bugs + persistent-ack pattern),
  `wsl-build-gotchas.md` (WSL build overrides, gradlew path, slotomatic-in-WSL myth),
  `hot-reload-rules.md` (Java = station restart; JS/CSS = browser hard-reload),
  `slotomatic.md` (when to run, slot removal coordinated edit, AUTO region rules).
- `CLAUDE.md`: agent guide with deploy decision table, invariants
  (backup → build → copy → verify), onboarding, gitignore exception note, test runner setup,
  engram `topic_key` conventions, cross-project search hints, KB index.
- `README.md`: human-first quick-start with consumer-module integration pattern.
- `.gitignore`: `_backups/`, `.env.*` with explicit `!.env.local.example` exception.

Bootstrap commit: `8d9a396` (retroactively tagged `v0.1.0`).

### References

- SDD slug: `niagara-tools-bootstrap`
- Engram: init #1806, proposal #1811, delivery decision #1817, apply #1820, verify #1823,
  archive #1824.
