### Fixed — mcp-n4: `n4_read_history` ok / not-ok status filters

- The `ok` and `not-ok` filters returned no records on a live station (`status.ok` is no BQL field: zero rows, no error); they now filter on `status.isOk`, live-checked on 4.14.0.162.
