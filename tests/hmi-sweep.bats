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

# polish-2026-10-02 P5 (#199 WU4): input checks — a --target that matches nothing is a FAIL row
# (was silent), a sub-tab that vanished after the previous click is a WARN row (was a TypeError),
# and --settle-ms must be a non-negative integer (was Number(), NaN accepted).
@test "HS8: --settle-ms must be a non-negative integer -> usage exit 3 otherwise" {
  _need_node
  for v in abc -5 1.5 ''; do
    run node "$HS" --url http://x/ --settle-ms "$v"
    [ "$status" -eq 3 ]
    [[ "$output" == *"--settle-ms"* ]]
  done
  run node -e 'console.log(require(process.argv[1]).parseArgs(["--url","http://x/","--settle-ms","0"]).settleMs)' "$HS"
  [ "$output" = "0" ]
}

@test "HS9: a --target selector that matches nothing in a view -> FAIL target row naming the selector" {
  _need_node
  run node -e '
    const h = require(process.argv[1]);
    h.rowsForView("default", "home", { docScrollY: false, docScrollX: false,
      target: { present: false, sel: "#alarm-banner" }, scrollers: [] }).forEach(r => console.log(r));
  ' "$HS"
  [ "$status" -eq 0 ]
  [[ "$output" == *"FAIL  hmi-sweep  default/home  target: #alarm-banner not found"* ]]
}

@test "HS10: clickVisibleNth returns null (no click, no throw) when the i-th visible element is gone" {
  _need_node
  run node -e '
    const h = require(process.argv[1]);
    let clicks = 0;
    const mk = (vis, tab) => ({ getClientRects: () => (vis ? [1] : []), click: () => { clicks++; },
      dataset: { tab }, textContent: "" });
    const els = [mk(true, "a"), mk(false, "b")];
    console.log(JSON.stringify([h.clickVisibleNth(els, 0), h.clickVisibleNth(els, 1), clicks]));
    h.rowsForView("default", "home#.sub[1]", { gone: true }).forEach(r => console.log(r));
  ' "$HS"
  [ "$status" -eq 0 ]
  [[ "$output" == *'["a",null,1]'* ]]
  [[ "$output" == *"WARN  hmi-sweep  default/home#.sub[1]  subtab: no longer visible"* ]]
}

@test "HS10b: clickNavNth labels by data-page, returns null (no click, no throw) once the nav set shrank (#226)" {
  # Mutation: HS10b -- the nav click's own inline guard was untested; dropping it throws on a vanished nav item.
  _need_node
  run node -e '
    const h = require(process.argv[1]);
    let clicks = 0;
    const mk = (page, text) => ({ click: () => { clicks++; }, dataset: { page }, textContent: text });
    const els = [mk("home", "Home"), mk(undefined, " Alarms ")];
    console.log(JSON.stringify([h.clickNavNth(els, 0), h.clickNavNth(els, 1), h.clickNavNth(els, 2), clicks]));
  ' "$HS"
  [ "$status" -eq 0 ]
  [[ "$output" == *'["home","Alarms",null,2]'* ]]
}

# #1087 R2: responsive widths. --widths / --profile repeat the sweep per width (viewport W x height of
# --viewport); every row of a multi-width run names its width; the default run is byte-identical.
@test "HS11: parseArgs -> default is one plain 1280x800 run; --profile hmi is the same; --viewport gives the height" {
  _need_node
  run node -e '
    const h = require(process.argv[1]);
    const runs = (a) => JSON.stringify(h.parseArgs(["--url","http://x/"].concat(a)).runs);
    console.log(runs([]));
    console.log(runs(["--profile","hmi"]));
    console.log(runs(["--viewport","1024x600","--profile","hmi"]));
    console.log(runs(["--widths","360,1280","--viewport","1024x600"]));
    console.log(runs(["--widths","400","--profile","lan"]));
  ' "$HS"
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = '[{"width":1280,"height":800,"label":null}]' ]
  [ "${lines[1]}" = "${lines[0]}" ]
  [ "${lines[2]}" = '[{"width":1024,"height":600,"label":null}]' ]
  [ "${lines[3]}" = '[{"width":360,"height":600,"label":"360"},{"width":1280,"height":600,"label":"1280"}]' ]
  [ "${lines[4]}" = '[{"width":400,"height":800,"label":"400"}]' ]
}

@test "HS12: parseArgs -> --profile lan = 360,768,1280; both = those plus the 1280x800 hmi run" {
  _need_node
  run node -e '
    const h = require(process.argv[1]);
    const runs = (a) => h.parseArgs(["--url","http://x/"].concat(a)).runs.map((r) => r.width + "x" + r.height + ":" + r.label);
    console.log(runs(["--profile","lan"]).join(" "));
    console.log(runs(["--profile","both"]).join(" "));
  ' "$HS"
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "360x800:360 768x800:768 1280x800:1280" ]
  [ "${lines[1]}" = "360x800:360 768x800:768 1280x800:1280 1280x800:1280x800" ]
}

@test "HS13: an invalid --widths list or --profile is a usage error, exit 3, nothing swept" {
  _need_node
  for v in '' ',' '360,' ',360' '360,,768' abc '360,abc' 1.5 -360 '360, 768' 199 4001 0; do
    run node "$HS" --url http://x/ --widths "$v"
    [ "$status" -eq 3 ]
    [[ "$output" == *"--widths"* ]]
    [[ "$output" == *"usage"* ]]
  done
  run node "$HS" --url http://x/ --widths
  [ "$status" -eq 3 ]
  [[ "$output" == *"missing value for --widths"* ]]
  run node "$HS" --url http://x/ --profile tablet
  [ "$status" -eq 3 ]
  [[ "$output" == *"--profile"* ]]
  [[ "$output" == *"usage"* ]]
  run node -e 'console.log(JSON.stringify(require(process.argv[1]).parseArgs(["--url","http://x/","--widths","200,4000"]).runs.map((r) => r.width)))' "$HS"
  [ "$output" = "[200,4000]" ]
}

@test "HS14: rowsForView names the width on every row of a multi-width run, and not on a plain run" {
  _need_node
  run node -e '
    const h = require(process.argv[1]);
    const m = { docScrollY: false, docScrollX: true, target: null, scrollers: [{ sel: "div.l", allowed: false }] };
    h.rowsForView("default", "home", m, "360").forEach((r) => console.log(r));
    h.rowsForView("default", "home", { gone: true }, "360").forEach((r) => console.log(r));
    console.log("--");
    h.rowsForView("default", "home", m).forEach((r) => console.log(r));
  ' "$HS"
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "FAIL  hmi-sweep  default/home  width=360  no-scroll: document scrolls (horizontal) at the panel viewport" ]
  [ "${lines[1]}" = "WARN  hmi-sweep  default/home  width=360  inner-scroller: div.l scrolls and is not named by --allow-scroller" ]
  [[ "${lines[2]}" == "WARN  hmi-sweep  default/home  width=360  subtab:"* ]]
  [ "${lines[4]}" = "FAIL  hmi-sweep  default/home  no-scroll: document scrolls (horizontal) at the panel viewport" ]
  [[ "${lines[5]}" != *"width="* ]]
}

@test "HS15: --widths 360,1280 on a page that breaks only when narrow -> FAIL names 360, 1280 is clean, exit 1" {
  _need_browser
  run node "$HS" --url "file://$FX/narrow-breaks.html" --widths 360,1280
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  hmi-sweep  default/main  width=360  no-scroll: document scrolls (horizontal)"* ]]
  [[ "$output" == *"PASS  hmi-sweep  default/main  width=1280  no-scroll: document fits"* ]]
  [[ "$output" != *"FAIL  hmi-sweep  default/main  width=1280"* ]]
  [[ "$output" == *"2 views · 1 PASS · 1 FAIL"* ]]
}

@test "HS16: --profile lan sweeps three widths (360, 768, 1280), --profile both adds the 1280x800 run" {
  _need_browser
  run node "$HS" --url "file://$FX/fits.html" --profile lan
  [ "$status" -eq 0 ]
  for w in 360 768 1280; do
    [[ "$output" == *"PASS  hmi-sweep  default/home  width=$w  no-scroll:"* ]]
    [[ "$output" == *"PASS  hmi-sweep  default/rooms  width=$w  no-scroll:"* ]]
  done
  [[ "$output" == *"6 views"* ]]
  [[ "$output" == *"CLEAN"* ]]
  run node "$HS" --url "file://$FX/narrow-breaks.html" --profile both
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  hmi-sweep  default/main  width=360  no-scroll:"* ]]
  [[ "$output" == *"PASS  hmi-sweep  default/main  width=768  no-scroll:"* ]]
  [[ "$output" == *"PASS  hmi-sweep  default/main  width=1280  no-scroll:"* ]]
  [[ "$output" == *"PASS  hmi-sweep  default/main  width=1280x800  no-scroll:"* ]]
  [[ "$output" == *"4 views"* ]]
}

@test "HS17: default run (no --widths/--profile) is unchanged: plain rows, no width field, plain screenshot names" {
  _need_browser
  local shots="$BATS_TEST_TMPDIR/shots"
  run node "$HS" --url "file://$FX/fits.html" --screenshot-dir "$shots"
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "PASS  hmi-sweep  default/home  no-scroll: document fits" ]
  [ "${lines[1]}" = "PASS  hmi-sweep  default/rooms  no-scroll: document fits" ]
  [ "${lines[2]}" = "hmi-sweep: 2 views · 2 PASS · 0 FAIL · 0 WARN  ->  CLEAN" ]
  [ "${#lines[@]}" -eq 3 ]
  [ -f "$shots/default_home.png" ]
  [ -f "$shots/default_rooms.png" ]
  run node "$HS" --url "file://$FX/fits.html" --profile hmi
  [ "${#lines[@]}" -eq 3 ]
  [[ "$output" != *"width="* ]]
}

@test "HS18: --widths writes one screenshot per width, the width in the file name" {
  _need_browser
  local shots="$BATS_TEST_TMPDIR/shots-w"
  run node "$HS" --url "file://$FX/narrow-breaks.html" --widths 360,1280 --screenshot-dir "$shots"
  [ "$status" -eq 1 ]
  [ -s "$shots/default_main_w360.png" ]
  [ -s "$shots/default_main_w1280.png" ]
  [ "$(find "$shots" -name '*.png' | wc -l)" -eq 2 ]
}
