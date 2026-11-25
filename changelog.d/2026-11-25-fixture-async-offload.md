### Added — MinimalPan fixture: an async-offload component (S6) and an RBAC station test (B3)

- `fixtures/MinimalPan/.../BPanOffload.java`: a `B*` component with an `ASYNC | OPERATOR` action whose
  `post(Action, BValue, Context)` override hands `new Invocation(this, action, arg, cx)` to a module-owned
  single-thread executor (named thread factory `MinimalPan-offload-N`), refuses the post when `!isRunning()` or
  before `started()`, reports a `RejectedExecutionException` instead of letting it escape, and calls `shutdownNow()`
  in `stopped()`. It is the shape of the course Lab 9 solution (`BExecutorServiceExecutor.java:46-70`) with the
  three corrections the lab lacks: `new Invocation(...)` instead of `this::doExecute` (the lab's `:66` skips the
  action pipeline), the not-running guard, and the named thread factory. Registered in `module-include.xml`, the
  palette and the lexicon; `module-permissions.xml` now requests `MANAGE_EXECUTION` (type `station`; it was the
  empty `<permissions/>`), which `lint-timers` `jdk-thread` needs for the `shutdownNow()`/`setDaemon`.
- `srcTest/.../BMinimalPanOffloadTest.java` (the action runs on the named thread, not the caller's; a never-started
  or removed component refuses the post) and `BMinimalPanPermissionsTest.java`
  (`userPermissionsScenario(...).actAsUser(...)`: a read-only user gets `PermissionException` on a `set` and on an
  ASYNC `invoke` and nothing runs; an admin `rwiRWI` writes and the action runs). API from `BTestNgStation.java:749`
  and `PermissionsScenario.java:64-81`; both tests are registered in `moduleTest-include.xml` and `MODULE-MAP.md`.
- `types/actions.md` (the sanctioned-offload bullet) and `types/moduleTest.md` (TEST-G9) point at the fixture.
- `scaffold-module.sh --type logic --with-offload` (new, opt-in; `go/internal/scaffoldmodule/offload.go`): emits the
  component, its two tests, their `moduleTest-include.xml` entries, the `MANAGE_EXECUTION` request and the palette,
  lexicon, `module-include.xml` and `MODULE-MAP.md` rows, all renamed like the other fixture files. Without the flag the
  output is byte-for-byte what it was before the component (a new module never requests a permission it did not ask
  for); any other `--type` with the flag is exit 2 (`--with-offload needs --type logic; <type> has no async-offload
  component`). The fixture declares the offload parts itself: three offload-only files, `offload:begin`/`offload:end`
  marker lines (whole trimmed lines in the file's comment syntax, and a ` <!-- offload:line -->` row suffix at line end)
  in the shared files, and the empty `<permissions/>` as the no-flag text of `module-permissions.xml`; the markers
  are never emitted, and an unclosed or nested block or an end without a begin is exit 3 with the file and line. The
  scaffold-diff round trip (`ci.yml`, `ci-local`) runs with `--with-offload` and compares against a fixture copy
  stripped by the internal `scaffold-module.sh --strip-offload-markers <dir>` mode, the one definition of a marker.
  Tests: `tests/scaffold-module-offload.bats` (no-flag output, flag output and its lints, refusal), TC3/TC-K8,
  `go/internal/scaffoldmodule/offload_test.go`.
- Not run: the fixture build and the station tests (no Niagara install here). The structural checks ran: the
  kit lints over the fixture and a javac pass of the new sources against the decompiled corpus (only the
  slotomatic-generated accessors are missing, as for the existing `BMinimalPan`).
