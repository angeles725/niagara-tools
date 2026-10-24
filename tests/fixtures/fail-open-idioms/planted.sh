#!/usr/bin/env bash
# Seeded fail-open idioms (tests/fail-open-idioms.bats FOI1): every line below must be reported.
fail() { echo FAIL; exit 1; }
(cd go && go build ./... && go vet ./... && go test ./... 2>&1 | grep -E '^(FAIL|---)') && { echo FAIL; exit 1; }
(make test | grep -q "FAIL") && fail
[ -z "$(gofmt -l go)" ] || fail
test -z "$(gofmt -l .)" || exit 1
