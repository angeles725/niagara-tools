### Fixed — report-module retries a member that is "text file busy"

- `report-module` ran a member script once. When another process had just forked while the script was still open for
  writing, exec failed with ETXTBSY and the member was reported as `unexpected exit 126` without having run. A busy
  executable is now retried for up to about a second. This fixes the intermittent `TestReadyVerifyModule` failure in CI.
