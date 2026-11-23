### Added

- `toolbelt/lab-station.sh` (`go/internal/labstation`): start, stop, restart and read a local lab station from a script, with no interactive console (issue #848). The station reads its console from a named pipe; `stop` sends `quit` and waits for `*** Station shutdown ***`; `console` sends one read-only verb (`gc`, `heap`, `help`, `threads`, `version`); `--dry-run` prints the plan.
