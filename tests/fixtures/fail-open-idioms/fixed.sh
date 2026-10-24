#!/usr/bin/env bash
# The fixed idioms (tests/fail-open-idioms.bats FOI1): nothing here may be reported.
fail() { echo FAIL; exit 1; }
log=out.log; x=a
(cd go && go build ./... && go vet ./... && go test ./... >"$log" 2>&1) || fail
fmt=$(gofmt -l go 2>&1) && [ -z "$fmt" ] || fail
[ -z "$(gofmt -l go 2>&1)" ] || fail
printf '%s\n' "$x" | grep -q pat && echo "the producer is printf: no gate here"
echo "( a | grep b ) && { exit 1; }"   # quoted text is not code
# ( a | grep b ) && { exit 1; }        # a comment is not code
