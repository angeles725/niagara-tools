### Fixed — `ci-local.sh --only "Go"` no longer fails the vendorfloor tests on a host without acorn

- `--only` of the Go step skipped CI's npm step but kept `export KIT_REQUIRE_JS_TOOLS=1`, so a host without node
  or acorn failed `go/internal/vendorfloor` on the missing tool. When the npm step is not part of the run and
  acorn cannot be resolved, the replay now runs the step with the variable unset and prints a NOTE; a host with
  acorn, or a full replay, keeps CI's strict setting (`scripts/lib/ci-local-js.sh`).
