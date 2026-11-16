# tests/lib/scaffold-setup.bash — the setup every tests/scaffold-module*.bats suite shares: the Go binary
# built once per file, and the kit, fixture, golden and out-dir paths each case reads.
# Load at file scope with:   load lib/scaffold-setup
# shellcheck shell=bash

# Build the Go binary once with the caller's environment: TC-K8 runs the launcher under HOME=/nonexistent,
# where an on-demand `go build` has no build cache and fails.
setup_file() {
  local root
  root="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  # shellcheck disable=SC1091  # go/ helper, resolved at runtime via root
  . "$root/go/go-tool.sh"
  go_tool_bin "$root/go" scaffold-module >/dev/null
}

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  SC="$KIT/toolbelt/scaffold-module.sh"
  FIXTURE="$KIT/fixtures/MinimalPan"
  FIXTURE_DASH="$KIT/fixtures/MinimalDash"
  FIXTURE_DRV="$KIT/fixtures/MinimalDriver"
  # logic + dashboard also get the shared station-test overlay (templates/station-test), which the
  # fixture trees do not carry; TC-ST* pin it against tests/fixtures/scaffold-module/station-test.
  ST_GOLDEN="$BATS_TEST_DIRNAME/fixtures/scaffold-module/station-test"
  # the client version ledger (odd/VERSION-LEDGER.md, from templates/VERSION-LEDGER.md; pinned by
  # tests/ledger-check.bats LCB3) is carried by every fixture, so the diffs below compare it byte-for-byte
  ST_EXCLUDE="--exclude=moduleTest-include.xml --exclude=B*StationTestBase.java --exclude=B*LinkTest.java"
  OUT="$BATS_TEST_TMPDIR/out"
  mkdir -p "$OUT"
}
