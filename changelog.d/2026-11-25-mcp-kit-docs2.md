### Changed

- `mcp-n4-kit/README.md`: `--progress-file` documents the `n4_histories_audit` progress lines (#1059); `n4_rollback`
  says a verified `chain: true` rollback no longer makes a later rollback of an ancestor create refuse on the batches
  it removed (#1059); `n4_commissioning_report` documents `spyder_hardware` and the `model_mismatch` check with its
  two queries, not yet run on a live station (#1060); `n4_invoke_action` documents `device_state`, `has_errors` and
  `cause` on a failed `spyderDownload` or `compile` job (#1060).
