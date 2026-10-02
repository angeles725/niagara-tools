#!/usr/bin/env bats
# Pins for triage-console.sh --site (site-issues list).
# Repeating NON-own console errors (device offline, duplicate device id, comm timeout, history flood)
# are classified into a separate SITE list with counts and first/last seen, so a known site fault is
# not re-diagnosed every session. SITE rows are informational: they never change the exit code
# (exit stays 1 only when an OWN row exists). Unclassified foreign warnings stay out of both lists
# (an unknown cause stays unknown). Without --site the output is unchanged.
# [ev: retro site-fault-triage-and-incident-journal Δ1]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  TC="$KIT/toolbelt/triage-console.sh"
  FX="$BATS_TEST_DIRNAME/fixtures/triage-console"
}

@test "TCS1: --site classifies offline/duplicate-id/timeout/history-flood into SITE rows with counts and first/last seen" {
  run "$TC" --site --package com.vendor --tag plant "$FX/console-site.txt"
  [ "$status" -eq 1 ]
  [[ "$output" == *"SITE  triage-console  console-site.txt  device-offline  3x 06:01:10 01-Oct-26 CST -> 07:21:10 01-Oct-26 CST"* ]]
  [[ "$output" == *"SITE  triage-console  console-site.txt  duplicate-device-id  2x 06:02:00 01-Oct-26 CST -> 06:30:00 01-Oct-26 CST"* ]]
  [[ "$output" == *"SITE  triage-console  console-site.txt  comm-timeout  2x"* ]]
  [[ "$output" == *"SITE  triage-console  console-site.txt  history-flood  1x"* ]]
}

@test "TCS2: an unclassified foreign warning is in neither list (unknown stays unknown)" {
  run "$TC" --site --package com.vendor --tag plant "$FX/console-site.txt"
  [[ "$output" != *"longer than expected"* ]]
}

@test "TCS3: an own exception whose text says 'timed out' is an OWN row, never a SITE row" {
  run "$TC" --site --package com.vendor --tag plant "$FX/console-site.txt"
  [[ "$output" == *"FAIL  triage-console  console-site.txt  1x"*"TimeoutException: write timed out"* ]]
  [[ "$output" != *"SITE"*"write timed out"* ]]
}

@test "TCS4: a site-only console under --site -> SITE rows but exit 0 (site rows are informational)" {
  run "$TC" --site --package com.vendor --tag plant "$FX/console-site-only.txt"
  [ "$status" -eq 0 ]
  [[ "$output" == *"SITE  triage-console  console-site-only.txt  device-offline  2x"* ]]
  [[ "$output" != *"FAIL  triage-console"* ]]
}

@test "TCS5: without --site the same console emits no SITE row (backward compatible)" {
  run "$TC" --package com.vendor --tag plant "$FX/console-site.txt"
  [ "$status" -eq 1 ]
  [[ "$output" != *"SITE  "* ]]
}

@test "TCS6: SITE rows print after the own rows, ordered by count descending" {
  run "$TC" --site --package com.vendor --tag plant "$FX/console-site.txt"
  first_site=$(printf '%s\n' "$output" | grep -n '^SITE' | head -1 | cut -d: -f1)
  last_own=$(printf '%s\n' "$output" | grep -n '^FAIL' | tail -1 | cut -d: -f1)
  [ "$last_own" -lt "$first_site" ]
  [[ "$(printf '%s\n' "$output" | grep '^SITE' | head -1)" == *"device-offline  3x"* ]]
}

# polish-2026-10-02 P3 (#199 WU9): the duplicate-device-id class needs a whole-word device / instance /
# address / id token; "valid", "invalid" and "idle" contain "id" but are not an id.
# Named mutation TCS-dupid (unanchored token again) -> TCS-dupid flips.
@test "TCS-dupid: 'duplicate ... invalid / valid / idle' is not a duplicate-device-id; 'duplicate device id 7' is" {
  local c="$BATS_TEST_TMPDIR/console.txt"
  cat > "$c" <<'TXT'
INFO [06:00:00 01-Oct-26 CST][sys] Station starting
WARNING [06:01:00 01-Oct-26 CST][web] Duplicate request rejected: token invalid
WARNING [06:02:00 01-Oct-26 CST][web] Duplicate session marked valid
WARNING [06:03:00 01-Oct-26 CST][sched] Duplicate worker idle
TXT
  run "$TC" --site --package com.vendor --tag plant "$c"
  [ "$status" -eq 0 ]
  [[ "$output" != *"duplicate-device-id"* ]] || { echo "$output"; return 1; }
  printf 'SEVERE [06:04:00 01-Oct-26 CST][bacnet] Duplicate device id 7 on the network\n' >> "$c"
  run "$TC" --site --package com.vendor --tag plant "$c"
  [[ "$output" == *"SITE  triage-console  console.txt  duplicate-device-id  1x"* ]]
}
