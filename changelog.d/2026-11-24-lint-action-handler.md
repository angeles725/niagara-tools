### Added — lint-action-handler and lint-async-blocking: the owed Go lints G1 and G28

- New `toolbelt/lint-action-handler.sh [--strict] <src-root>...` (Go rule `actionhandler`). FAIL a class whose
  `@NiagaraAction(name = "x")` has no public, non-static `do<X>` method of the shapes the station's introspector
  looks up (`doX()`, `doX(P)`, `doX(Context)`, `doX(P, Context)`), or whose return type differs from the action's
  `returnType` (void when none). The station refuses such a type at load ("Missing action do method", "Do action
  method must be not be static", "Action method and do method have mismatched signature";
  `ComplexIntrospector.java:316-343`) while Slot-o-Matic and the jar task succeed. WARN (exit 0, 1 under
  `--strict`) a `doX(..., Context)` beside a no-Context `doX(...)` (the Context one never runs, ledger G1) and
  `this::doX` or `new Thread(() -> doX(` of a declared action, which skips the action pipeline (course F3).
- New `toolbelt/lint-async-blocking.sh [--strict] <src-root>...` (Go rule `asyncblock`, ledger G28), WARN only. An
  `ASYNC` action of a class with no `post(Action...)` override whose `do<Name>` calls `Thread.sleep(`,
  `openConnection(`, `getOutputStream(`, `sendSync(`, names `HttpClient` or calls a BQL `.get(` blocks the shared
  engine thread; a `post(Action...)` override that runs `this::doX` or `-> doX(` itself is WARNed too. The two
  course acceptance cases (Lab 9 `BEngineThreadExecutor`, the exam `BHistoryExportService`) WARN and exit 0.
- Both wrappers take `-h`/`--help` (the header, never a code line).
