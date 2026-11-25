### Changed

- `scripts/install-skill.sh --check` (Go: `go/internal/skillinstall`, `check.go`): a read-only drift check of every
  skill of the `SKILLS` table in every harness. It prints the `check-skill-drift.sh` rows
  (`PASS|FAIL|SKIP  skill-drift  <skill>[<harness>]  …`, table order then harness order; the same code as
  `check-skill-drift.sh --all`, which still checks one skill per run) and writes nothing. Exit 0 when every checked
  copy is current, 1 on any `FAIL` row (diverged or not installed), 2 when `--skill`, `--harness`, `--dry-run` or
  `--force` is given with it (`--check` takes only `--home`), 3 when a tracked launcher is missing (before any row)
  or an installed copy cannot be read. The usage line and the script header name it. A stale installed launcher (e.g. one still naming the retired Python server) is a `FAIL` row for any skill in one run. Retro deltas: build-n4-module `client-e-dashboard-live-check` Δ35 (#986) and mcp-n4
  `dashboard-live-check-4-15` Δ21 (#985), the drift-detection part; no session-start hook runs it.
- Not changed: the `serverInfo.version` >= launcher `metadata.version` floor stays a rule of the mcp-n4 `SKILL.md`
  resolve step, not an install-time check (`install-skill` never talks to a server).
- Go tests (`check_test.go`) and bats `CK1`–`CK7` pin the rows, the exit codes, the usage errors and the
  read-only guarantee; every case uses a temp home.
