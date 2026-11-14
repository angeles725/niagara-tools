### Changed — mcp-n4-kit mutation checks follow the Go rule
- A mutation check runs both the mutated and the restored test with `go test -count=1`, so a cached result is never read as RED or GREEN, and confirms the mutation is gone with `git diff --exit-code` on the mutated file before reading GREEN (`mcp-n4-kit/METHODOLOGY.md` section 8). The Python bytecode rule and `tools/mutation_check.py` are retired.
- `mcp-n4-kit/README.md` "Run the tests" is Go only: `go test -count=1 ./internal/mcpn4/... ./cmd/mcp-n4/...`; shuffled runs use `go test -shuffle`. The untested-branch audit uses `go test -coverprofile`.
- `blackbox.TestMethodologyCitesNoPythonTest` replaces the Python-citation check: METHODOLOGY.md may cite no `test_` Python name.
