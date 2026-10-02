#!/usr/bin/env bats
# Pins for toolbelt/hmi-sweep.js — the reusable HMI no-scroll + target-visibility sweep.
# [ev: retro comppan-fase2-amps-alarms Δ3] [ev: retro dashboard-rc-file-split Δ5]
#
# Pure pins (node only): argument parsing, per-view verdict rows, summary/exit.
# Degrade pin: puppeteer-core forced missing via KIT_PUPPETEER -> exit 4 with an "unavailable" line.
# Browser pins (HS6/HS7): run only when puppeteer-core AND a Chrome binary resolve; they sweep the
# file:// fixtures in tests/fixtures/hmi-sweep/ (no server).

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  HS="$KIT/toolbelt/hmi-sweep.js"
  FX="$BATS_TEST_DIRNAME/fixtures/hmi-sweep"
}

_need_node() { command -v node >/dev/null 2>&1 || skip "node not installed"; }
_need_browser() {
  _need_node
  local chrome="${KIT_CHROME:-${PUPPETEER_EXECUTABLE_PATH:-}}"
  if [ -z "$chrome" ] || [ ! -x "$chrome" ]; then skip "no Chrome (set KIT_CHROME or PUPPETEER_EXECUTABLE_PATH)"; fi
  if [ -n "${KIT_PUPPETEER:-}" ]; then return 0; fi
  node -e 'require.resolve("puppeteer-core")' >/dev/null 2>&1 || skip "puppeteer-core not installed"
}

@test "HS1: no --url -> usage exit 3" {
  _need_node
  run node "$HS"
  [ "$status" -eq 3 ]
  [[ "$output" == *"usage"* ]]
}

@test "HS2: parseArgs -> viewport default 1280x800, repeatable --scenario name=query, default nav .nav-item" {
  _need_node
  run node -e '
    const h = require(process.argv[1]);
    const a = h.parseArgs(["--url","http://x/","--scenario","worst=worst=1","--scenario","default"]);
    console.log(JSON.stringify([a.width,a.height,a.nav,a.scenarios]));
  ' "$HS"
  [ "$status" -eq 0 ]
  [ "$output" = '[1280,800,".nav-item",[{"name":"worst","query":"worst=1"},{"name":"default","query":""}]]' ]
}

@test "HS3: rowsForView -> document scroll FAIL, occluded target FAIL, unnamed inner scroller WARN" {
  _need_node
  run node -e '
    const h = require(process.argv[1]);
    const rows = h.rowsForView("worst", "rooms", {
      docScrollY: true, docScrollX: false,
      target: { present: true, visible: true, unoccluded: false },
      scrollers: [ { sel: "div.list", allowed: false }, { sel: "div.log", allowed: true } ] });
    rows.forEach(r => console.log(r));
  ' "$HS"
  [ "$status" -eq 0 ]
  [[ "$output" == *"FAIL  hmi-sweep  worst/rooms  no-scroll:"* ]]
  [[ "$output" == *"FAIL  hmi-sweep  worst/rooms  target:"* ]]
  [[ "$output" == *"WARN  hmi-sweep  worst/rooms  inner-scroller: div.list"* ]]
  [[ "$output" != *"div.log"* ]]
}

@test "HS4: rowsForView on a fitting view with a visible target -> PASS rows only" {
  _need_node
  run node -e '
    const h = require(process.argv[1]);
    h.rowsForView("default", "home", { docScrollY: false, docScrollX: false,
      target: { present: true, visible: true, unoccluded: true }, scrollers: [] })
      .forEach(r => console.log(r));
  ' "$HS"
  [ "$status" -eq 0 ]
  [[ "$output" == *"PASS  hmi-sweep  default/home  no-scroll:"* ]]
  [[ "$output" == *"PASS  hmi-sweep  default/home  target:"* ]]
  [[ "$output" != *"FAIL"* ]]
}

@test "HS5: puppeteer-core missing -> exit 4 with an unavailable line (never a silent pass)" {
  _need_node
  KIT_PUPPETEER=/nonexistent/puppeteer-core run node "$HS" --url "file://$FX/fits.html"
  [ "$status" -eq 4 ]
  [[ "$output" == *"unavailable"* ]] && [[ "$output" == *"puppeteer-core"* ]]
}

@test "HS6: real sweep of a fitting page (nav + sub-tabs) -> CLEAN, exit 0" {
  _need_browser
  run node "$HS" --url "file://$FX/fits.html" --subtab .sub-tab --target '#alarmBanner'
  [ "$status" -eq 0 ]
  [[ "$output" == *"default/rooms#B"* ]]
  [[ "$output" == *"CLEAN"* ]]
}

@test "HS7: real sweep of an overflowing page -> no-scroll FAIL on long, target FAIL on hidden, exit 1" {
  _need_browser
  run node "$HS" --url "file://$FX/overflow.html" --target '#alarmBanner'
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  hmi-sweep  default/long  no-scroll:"* ]]
  [[ "$output" == *"FAIL  hmi-sweep  default/hidden  target:"* ]]
  [[ "$output" == *"PASS  hmi-sweep  default/ok  no-scroll:"* ]]
}
