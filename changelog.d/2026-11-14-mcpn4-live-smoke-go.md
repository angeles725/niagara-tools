### Added — live-smoke replaces live_smoke.py
- `go/bin/live-smoke` (`go/cmd/mcp-n4/live-smoke`, logic in `go/internal/mcpn4/livesmoke`) is the live smoke runner: plan only by default; `--apply` starts `mcp-n4 --allow-writes` over stdio and runs the kitControl thermostat scenario, every write as dry run -> token -> execute -> verdict, with credential values scrubbed from all output. Same flags, plan, usage errors, report and exit codes as `tools/live_smoke.py` (differential: identical stdout, stderr and exit codes once the program name `live_smoke` reads `live-smoke`). The default server is the `mcp-n4` next to the `live-smoke` binary (`go/bin/mcp-n4`).
- `go/internal/mcpn4/readtools/readtoolstest/boxstation`: an in-memory fake BOX station over HTTP (the Go port of `tests/fake_station.py`'s FakeStation) that the Go tests drive the real server code against.

### Removed — mcp-n4-kit/tools/live_smoke.py and tests/test_live_smoke.py
- Use `go/bin/live-smoke`; its 37 tests are Go tests in `go/internal/mcpn4/livesmoke`.
