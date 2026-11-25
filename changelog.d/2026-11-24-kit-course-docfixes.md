### Fixed — course and API findings: fixture lexicons and one-line card corrections

- `fixtures/MinimalPan` and `fixtures/MinimalDash` `module.lexicon`: the slot keys are the bare slot names
  (`setpoint=`, `interval=`, `status=`); the dotted `<Type>.<slot>` keys were never read (`NSlot.java:42`), so every
  scaffolded module showed raw camelCase slot names.
- `templates/dashboard-login/fragments/module.lexicon`: `applyLogin=` replaces `ModuleAService.applyLogin=`, a key
  nothing reads, so the scaffolded service now shows "Apply Login" in Workbench; its tests follow
  (`tests/scaffold-module-login.bats`, `go/internal/scaffoldmodule/login_test.go`). The `logic.md` line baseline
  in `go/internal/cardwidth/baseline.txt` goes from 201 to 199.
- `types/service.md` § 3: prefer `Sys.findService(Type)` (`Optional<BIService>`, no throw) when the service may be
  absent.
- `types/logic.md`: "no `BLoggingService`" now says there is no public `javax.baja` API (the station's
  `BLoggingService` is private, `BLoggingService.java:67`); the thread-probe line cites the course `BThreadLogger`
  source; a new bullet names `BThreadPoolWorker` and the deck's `ITodo` (the transcript's "I2Do").
- `types/observability.md`: LogHistory keeps `INFO` and above by default (`minimumSeverity` is `info`), not only
  `WARNING`.
- `types/diagnostics.md` § 4.1: the course's Ctrl+Break thread dump next to the `threads` verb.
- `types/driver-authoring.md` § 1.10: `NComm.registerCommListener` with an `ICommFilter`, and `noListener` as the
  counter of unsolicited frames that nobody consumed.
