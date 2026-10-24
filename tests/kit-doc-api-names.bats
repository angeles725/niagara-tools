#!/usr/bin/env bats
# shellcheck disable=SC2016  # the backticks in the fixture text are literal Markdown, not command substitution
# Guards the kit's core docs against N4 API names that do not exist (quality round 8 T9b, register C02/C03/C06).
# Each banned form was taught by a kit card and fails to compile against 4.15.6.20 source:
#   BWorker.post / post()/postAsync()  BWorker declares no post or postAsync (S/util/BWorker.java:19-139); a
#                                      subclass may declare its own postAsync(Runnable), so only the BWorker-qualified
#                                      form and the old "post()/postAsync()" API phrase are banned, not the bare name.
#   BJobs.                             no such class in javax.baja.job
#   BJobService.submit(                submit is an instance method; use job.submit(cx) (S/job/BJob.java:420)
#   makeDouble(                        BFacets has no makeDouble; use makeNumeric(BUnit, int) (S/sys/BFacets.java:310)
# Scope: every *.md under build-n4-module-kit/ except retros/, odd/, ledger.d/, BUILD-STATE.src.md and the
# generated BUILD-STATE.md (the ledger, its source, its fragments and the retros quote the wrong text as evidence).
# [ev: retro devguide415-runtime-api-corrections Δ1] [ev: retro devguide415-runtime-api-corrections Δ2]

setup() { KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"; }

BANNED='BWorker\.post|post\(\)/postAsync\(\)|BJobs\.|BJobService\.submit\(|makeDouble\('

# doc_api_hits <root>: every banned-name line in the core docs under <root>, as path:line:text.
doc_api_hits() {
  find "$1" -name '*.md' -not -path '*/retros/*' -not -path '*/odd/*' -not -path '*/ledger.d/*' -not -name BUILD-STATE.md -not -name BUILD-STATE.src.md -print0 \
    | xargs -0 grep -nE "$BANNED" || true
}

@test "DOC-API1: no core kit doc names a non-existent worker, job or facets API" {
  run doc_api_hits "$KIT"
  [ "$status" -eq 0 ]
  [ -z "$output" ] || { printf '%s\n' "$output"; false; }
}

@test "DOC-API2: the scan bites every banned form, and skips retros/, odd/, ledger.d/ and BUILD-STATE{,.src}.md" {
  D="$BATS_TEST_TMPDIR/kit"; mkdir -p "$D/types" "$D/retros" "$D/odd/tasks" "$D/ledger.d"
  printf '%s\n' 'Post via `BWorker.post(Runnable)`.' 'deliver back via `post()/postAsync()`' \
    '`BJobs.submit(job, cx)`' '`BJobService.submit(job,cx)`' '`BFacets.makeDouble(unit, 2)`' > "$D/types/bad.md"
  printf '%s\n' 'A subclass may declare its own `postAsync(Runnable)`; use `job.submit(cx)`.' > "$D/types/good.md"
  cp "$D/types/bad.md" "$D/retros/quoted.md"; cp "$D/types/bad.md" "$D/odd/tasks/quoted.md"; cp "$D/types/bad.md" "$D/BUILD-STATE.md"; cp "$D/types/bad.md" "$D/BUILD-STATE.src.md"; cp "$D/types/bad.md" "$D/ledger.d/quoted.md"
  run doc_api_hits "$D"
  [ "$(printf '%s\n' "$output" | grep -c .)" -eq 5 ]
  [[ "$output" == *"types/bad.md"* ]]
  [[ "$output" != *"good.md"* && "$output" != *"retros/"* && "$output" != *"odd/"* && "$output" != *"BUILD-STATE"* && "$output" != *"ledger.d/"* ]]
}
