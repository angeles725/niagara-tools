### Changed — ci-local reads the bats pin in Go

- `scripts/ci-local.sh` reads the bats pin through `ci-steps bats-pin`; an "Install bats" step that sets `BATS_VER=` or `BATS_SHA256=` twice, or malformed, now exits 3 instead of taking the first value.
