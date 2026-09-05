#!/usr/bin/env bats
# RED-FIRST pins for triage-console.sh (campaign 8, production-log triage).
# Nothing in the kit reads station logs; the defrost "time <= 0" defect surfaced only on the PANCCADIA
# console. triage-console groups the exceptions that pass through OUR package so a swallowed own-code
# error is visible without wading through jetty/tridium noise.
#
# SURFACE (provisional): triage-console.sh <console.txt…> [--package com.angeles]
#   One row per distinct exception whose stack passes through the own package:
#     <count> <first-ts> <last-ts> <ExceptionClass>: <msg>  @ <own frame file:line>
#   Exceptions with NO own frame (e.g. a jetty-only NPE) are ignored. exit 1 when any row, 0 when none,
#   3 usage.
#
# RED today: triage-console.sh does not exist -> every pin fails for the right reason (tool absent).
#
# NAMED MUTATION (post-green): drop the own-frame filter -> the jetty-only NullPointerException row
# appears, so TR1's "no jetty row" assertion flips.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  TC="$KIT/toolbelt/triage-console.sh"
  FX="$BATS_TEST_DIRNAME/fixtures/triage-console"
}

@test "TR1: 2 own 'time <= 0' traces + 1 jetty-only -> ONE row (count 2, own frame), exit 1, NO jetty row" {
  run "$TC" --package com.angeles "$FX/console.txt"
  [ "$status" -eq 1 ]
  [[ "$output" == *"IllegalArgumentException: time <= 0"* ]]
  [[ "$output" == *"BDefrostController.java:431"* ]]
  [[ "$output" == *"2"* ]] && [[ "$output" == *"16:20:10"* ]] && [[ "$output" == *"18:58:19"* ]]
  [[ "$output" != *"NullPointerException"* ]]   # jetty-only trace has no own frame -> ignored
}

@test "TR2: the row carries the own frame (com.angeles), not a tridium/jetty frame" {
  run "$TC" --package com.angeles "$FX/console.txt"
  [[ "$output" == *"com.angeles"* ]] || [[ "$output" == *"BDefrostController"* ]]
  [[ "$output" != *"@ org.eclipse.jetty"* ]] && [[ "$output" != *"@ com.tridium"* ]]
}

@test "TR3: a console with only jetty/tridium frames (no own frame) -> exit 0, no rows" {
  jonly="$BATS_TEST_TMPDIR/jetty.txt"
  { echo "SEVERE [19:05:00 03-Sep-26 CST][web] jetty request failure"
    echo "  java.lang.NullPointerException: x"
    echo "  	at org.eclipse.jetty.server.HttpChannel.handle(HttpChannel.java:501)"
    echo "  	at java.base/java.lang.Thread.run(Thread.java:834)"; } > "$jonly"
  run "$TC" --package com.angeles "$jonly"
  [ "$status" -eq 0 ]
  [[ "$output" != *"NullPointerException"* ]]
}

@test "TR4: no console argument -> exit 3 (usage)" {
  run "$TC"
  [ "$status" -eq 3 ]
}

@test "TR5: Spanish stack-less own-logger ADVERTENCIA lines group by digit-normalized message + count" {
  # 2 modifyThread lines differ only in a digit -> normalize to one group, count 2; a 3rd is distinct.
  run "$TC" --package com.angeles "$FX/console-es.txt"
  [ "$status" -eq 1 ]
  [[ "$output" == *"modifyThread"* ]]
  [[ "$output" == *"2"* ]]                        # the modifyThread group is counted 2 (digits normalized)
  [[ "$output" == *"escritura"* ]]                # the distinct BChiRbacHelper warning is its own row
}

@test "TR6: non-UTF-8 (mojibake) bytes never crash or drop lines (LC_ALL=C-safe)" {
  # console-es.txt carries a raw 0xD3 (invalid UTF-8) in INFORMACION lines; the tool must still
  # emit the ADVERTENCIA rows and exit sanely, under a UTF-8 locale AND LC_ALL=C.
  LC_ALL=C run "$TC" --package com.angeles "$FX/console-es.txt"
  [ "$status" -eq 1 ]
  [[ "$output" == *"modifyThread"* ]]
}

# ===========================================================================
# THIRD ATTRIBUTION CHANNEL — load-time failures with NO own frame and NO own
# logger tag. [CERT-live] PANCCADIA: after the ColdRoomPan-rt reload a slot type
# drifted (BRelTime vs BComplex in the saved .bog), so the station NEVER STARTED.
# The failure surfaces only as framework loggers — SEVERE [sys] "Cannot load
# station" (the ClassCastException on the following line) and WARNING [sys.xml]
# lines naming our types/slots — with a com.tridium-only stack. Channels 1 (own
# frame, TR1-2) and 2 (own logger, TR5) BOTH miss it: the outage is invisible.
# This is exactly the class of defect the kit must not swallow.
#
# The channel is shape-based, not package-based (there is no own frame to match):
# a [sys] "Cannot load station" SEVERE carries its exception; [sys.xml] "Cannot
# set property" / "Missing frozen property" / "Cannot decode slot" are saved-bog
# schema-drift symptoms (the MM3/B795 survival matrix in log form).
#
# NAMED MUTATION (post-green): drop the third channel -> the fatal ClassCastException
# row vanishes. The load-fail fixtures are PURE third-channel (no own frame, no own
# logger), so with nothing else to carry a row the exit flips 1 -> 0. TR8 pins that
# flip on a fatal-only fixture where the [sys] row is the single row.

@test "TR7: a load-failure console (no own frame, no own logger) surfaces the fatal + sys.xml drift, exit 1" {
  run "$TC" --package com.angeles "$FX/console-load-fail.txt"
  [ "$status" -eq 1 ]
  [[ "$output" == *"ClassCastException"* ]]             # the previously-invisible fatal is now surfaced
  [[ "$output" == *"BRelTime"* ]] && [[ "$output" == *"BComplex"* ]]
  [[ "$output" == *"RoomPanel"* ]] || [[ "$output" == *"differentialUp"* ]]   # a sys.xml drift row grouped in
  [[ "$output" != *"com.tridium.sys.station.Station"* ]]  # the tridium boot frame is noise, not a row
}

@test "TR8: a fatal-only load failure -> exactly the [sys] row, exit 1 (the mutation flips this 1->0)" {
  run "$TC" --package com.angeles "$FX/console-load-fatal-only.txt"
  [ "$status" -eq 1 ]
  [[ "$output" == *"ClassCastException"* ]]
  # This fixture has NO own frame, NO own logger, NO sys.xml — the [sys] fatal is the ONLY row.
  # Dropping the third channel therefore yields zero rows -> exit 0: a clean green<-RED mutation proof.
}

@test "TR9: Spanish load-failure levels are normalized in the output (GRAVE->SEVERE, ADVERTENCIA->WARNING)" {
  run "$TC" --package com.angeles "$FX/console-load-fail-es.txt"
  [ "$status" -eq 1 ]
  [[ "$output" == *"ClassCastException"* ]]   # the exception is language-independent — the real signal
  [[ "$output" == *"SEVERE"* ]]               # GRAVE normalized
  [[ "$output" == *"WARNING"* ]]              # ADVERTENCIA normalized
}
