### Changed

- `n4_invoke_action` (`spyderDownload`, `compile`): the read-back of a failed or canceled job says where the
  compiler's own message is. `has_errors` is the compile job's `hasErrors` slot, and `cause` is one sentence chosen
  from it and the job log (`go/internal/mcpn4/writeops/spyder_failure.go`): a download that logged only "Download
  failed due to compilation error" kept none of the compiler's message (`BSpyderDownloadJob` drops the exception),
  so the cause says to run `compile`, whose CompilationJob logs it; `hasErrors` true says the validation lines are in
  the job log; `hasErrors` false with the `IllegalArgumentException` that `JobLog.failed` throws for a null message
  says the cause is only in the station log (logger `honeywellSpyderTool`). The read-back of a job whose log shows
  none of this is unchanged (`device_state`, `log_error`, `hint`). `TestAFailedDownloadThatDroppedTheCompilerMessageNamesCompile`,
  `TestAFailedCompileWithValidationErrorsSaysSo`, `TestAFailedCompileWithANullMessageNamesTheStationLog`,
  `TestAFailedDownloadWithAnOrdinaryLogCarriesNoCause`. Issue #818.
- `n4_commissioning_report`: new argument `spyder_hardware` (boolean, default false). It adds the `model_mismatch`
  check to the `spyder` section (two more queries: the Spyders' `controllerModel` and the numeric points with their
  `proxyExt.objectId` and `out.value`): the configured model against the hardware id of the point that proxies
  `analogValue:0`, which only the controller publishes (`BBacnetSpyder.readControllerModelFromDevice`). A Spyder
  with no such point is counted, never judged (`with_hardware_point`, `hardware_unreadable`); a refused query is
  `not_checked`. Without the argument the check is absent and no query is sent; alone it adds the `spyder` section.
  The two selects are not yet run on a live station. `TestCommissioningSpyderModel*`. Issue #820.
