#!/usr/bin/env bats
# RED-FIRST pins for rc-scan.sh (campaign 8 PR6). Scans a module's browser resources
# (**/rc/** — html/js/css ONLY) for defects the Java-side lints never see, from the real
# DashboardPan-ux/src/rc/index.html [CERT-live]:
#   ord   FAIL  hardcoded `station:|` / `slot:/` ORD literal (index.html-class :701 shape)
#   host  FAIL  a hardcoded network host: `http(s)://` with an IPv4 literal, or a non-namespace
#               host. W3C SVG/xlink namespace URIs (http://www.w3.org/...) are NOT hosts.
#   bare-catch  WARN  a write fetch swallowed by `.catch(() => {})` (index.html :1298)
#   null-branch WARN  a `? null :` display branch on a process field (index.html :852-853)
#
# SURFACE: rc-scan.sh <module-root>
#   Row: `<check>  PASS|FAIL|WARN|SKIP  <subject-without-whitespace>  <detail>`
#   Exit: 0 no FAIL (WARN-only still 0) · 1 any FAIL · 3 usage. Consistent with lint-delays/triage.
#
# RED today: build-n4-module-kit/toolbelt/rc-scan.sh does not exist -> every pin fails for the
# right reason (tool absent). Green once PR6 lands the scanner.
#
# NAMED MUTATIONS (post-green):
#   - remove the ORD rule            -> RC1's ord FAIL vanishes; ord fixture is ord-only so exit 1 -> 0.
#   - drop the bare-catch WARN check -> RC3's WARN row vanishes.

load lib/client-root   # C11 T2: one blessed client read root; env override wins [ev: design.md D3b]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  RS="$KIT/toolbelt/rc-scan.sh"
  FX="$BATS_TEST_DIRNAME/fixtures/rc-scan"
}

@test "RC1: a hardcoded ORD literal under rc/ FAILs (ord), exit 1 (ord-only -> the clean mutation target)" {
  run "$RS" "$FX/ord"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]] && [[ "$output" == *"ord"* ]]
  [[ "$output" == *"app.js"* ]]
}

@test "RC2: an IPv4 host literal FAILs (host) but a W3C SVG namespace does NOT (false-positive control)" {
  run "$RS" "$FX/host"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]] && [[ "$output" == *"host"* ]]
  [[ "$output" == *"127.0.0.1"* ]] || [[ "$output" == *"config.js"* ]]
  # the SVG/xlink namespace URIs on the same file must NOT be reported as hosts:
  [[ "$output" != *"www.w3.org"* ]]
}

@test "RC3: a write fetch with a bare .catch(() => {}) WARNs (bare-catch), exit 0 (WARN does not fail)" {
  run "$RS" "$FX/warns"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]] && [[ "$output" == *"bare-catch"* ]]
}

@test "RC4: a ? null : display branch on a process field WARNs (null-branch)" {
  run "$RS" "$FX/warns"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]] && [[ "$output" == *"null-branch"* ]]
}

@test "RC5: a clean rc tree -> exit 0, no FAIL and no WARN" {
  run "$RS" "$FX/clean"
  [ "$status" -eq 0 ]
  [[ "$output" != *"FAIL"* ]] && [[ "$output" != *"WARN"* ]]
}

@test "RC6: an ORD literal in src/ (outside rc/) is NOT scanned (scope: rc/ only)" {
  run "$RS" "$FX/scope"
  [ "$status" -eq 0 ]
  [[ "$output" != *"FAIL"* ]]
  [[ "$output" != *"Foo.java"* ]]   # src/ is never read
}

@test "RC7: no module-root argument -> exit 3 (usage)" {
  run "$RS"
  [ "$status" -eq 3 ]
}

@test "RC8: real smoke — DashboardPan-ux rc FAILs on host literal (SKIP if not present)" {
  UX="$C9_CLIENT_ROOT/Dashboard/DashboardPan/DashboardPan-ux"   # via client-root.bash [ev: design.md D3c site 10]
  [ -d "$UX/src/rc" ] || skip "DashboardPan-ux rc not on this machine (local-only real smoke)"
  run "$RS" "$UX"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]] && [[ "$output" == *"host"* ]]
}

# ---- RC9: D9b dot-dir prune -----------------------------------------------
# Design addendum D9b: kit source scanners prune dot-directories. report-module keeps a previous-deploy
# snapshot at <artifact>/.deploy-baseline, which can contain a stale rc/ copy; rc-scan must NOT descend
# into it (nor into .git). The live rc/ here is clean, so a clean exit proves the dot-dir was pruned.
# RED today: rc-scan.sh absent (tool-absent, like the rest of PR6). NAMED MUTATION (post-green): remove
# the dot-dir prune -> the .deploy-baseline host literal is flagged -> RC9 flips.
@test "RC9: a host literal in a .deploy-baseline/ copy is NOT flagged (dot-dir pruned; live rc/ is clean)" {
  run "$RS" "$FX/dotdir"
  [ "$status" -eq 0 ]
  [[ "$output" != *"FAIL"* ]]
  [[ "$output" != *"config.js"* ]]   # the stale baseline copy is never read
}

# ---- RC10: h: handle ORD anchored at segment start, hex payload ---------------
# Lead correction: h: is the Niagara HANDLE scheme (h:<hex>, e.g. h:45ef7), not h:/path.
# The old h:/ pattern was a false NEGATIVE for every handle literal.
# Fix: match h: only when preceded by |, ", ', or ` (ORD segment boundary) and followed
# by 1+ hex digits: (^|[|"'`])h:[0-9a-fA-F]+
# This also avoids CSS false positives: width:100px has h: preceded by 't', not a segment char.
@test "RC10: a handle ORD h:<hex> FAILs (ord-literal, segment-start anchored), CSS width:/depth: does NOT" {
  run "$RS" "$FX/handle"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]] && [[ "$output" == *"ord"* ]]
  [[ "$output" == *"handle.js"* ]]
  # CSS and HTML width:/height: must NOT be flagged as handle ORDs:
  [[ "$output" != *"style.css"* ]]
  [[ "$output" != *"page.html"* ]]
}

# ===========================================================================
# WU3 (fold-2026-10-02-pending-retros) — new checks. Each check id is pinned by a positive and a
# negative fixture; the rc-scan.sh header names one mutation per check.
#   browser-floor   [ev: retro dashboard-frontend-reliability-rules Δ5] [ev: retro panccadia-commissioning-lessons Δ6]
#                   [ev: retro panccadia-persistent-config-hoa Δ4]
#   disabled-gate   [ev: retro dashboard-frontend-reliability-rules Δ7] [ev: retro panccadia-persistent-config-hoa Δ4]
#   fetch-no-signal [ev: retro dashboard-frontend-reliability-rules Δ1]
#   setinterval-async [ev: retro dashboard-frontend-reliability-rules Δ2]
#   innerhtml-server / datauri-budget / orphan-page [ev: retro dashboard-frontend-standard Δ2/Δ3/Δ5]
#   inline-block-size [ev: retro dashboard-rc-file-split Δ6]
# ===========================================================================

@test "RC11: browser-floor WARNs every CSS/JS feature above the Chromium 83 panel floor, exit 0 by default" {
  run "$RS" "$FX/floor"
  [ "$status" -eq 0 ]
  [[ "$output" != *"FAIL"* ]]
  [[ "$output" == *"WARN  rc-scan  rc/style.css:1  browser-floor"*"inset"* ]]
  [[ "$output" == *"rc/style.css:2  browser-floor"*"gap"* ]]
  [[ "$output" == *"rc/style.css:3  browser-floor"*"aspect-ratio"* ]]
  [[ "$output" == *"rc/style.css:4  browser-floor"*"min("* ]]
  [[ "$output" == *"rc/style.css:5  browser-floor"*"backdrop-filter"* ]]
  [[ "$output" == *"rc/app.js:1  browser-floor"*"??="* ]]
  [[ "$output" == *"rc/app.js:2  browser-floor"*"replaceAll"* ]]
  # <style> block inside HTML and an inline style="" attribute are CSS context too:
  [[ "$output" == *"rc/index.html:5  browser-floor"*"inset"* ]]
  [[ "$output" == *"rc/index.html:9  browser-floor"*"gap"* ]]
}

@test "RC12: browser-floor --strict promotes every row to FAIL, exit 1" {
  run "$RS" "$FX/floor" --strict
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  rc-scan  rc/style.css:1  browser-floor"* ]]
  [[ "$output" != *"WARN"* ]]
}

@test "RC13: browser-floor FAILs under --profile hmi (and both), stays WARN under --profile lan" {
  run "$RS" "$FX/floor" --profile hmi
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  rc-scan  rc/style.css:1  browser-floor"* ]]
  run "$RS" "$FX/floor" --profile both
  [ "$status" -eq 1 ]
  run "$RS" "$FX/floor" --profile lan
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"*"browser-floor"* ]] && [[ "$output" != *"FAIL"* ]]
}

@test "RC14: browser-floor negatives — grid gap, minmax(), min-width, @media (min-width), ?? and ?. and the allow marker stay clean" {
  run "$RS" "$FX/floor-ok" --strict
  [ "$status" -eq 0 ]
  [[ "$output" != *"browser-floor"* ]]
}

@test "RC15: an unknown --profile value or a missing --profile argument exits 3 (usage)" {
  run "$RS" "$FX/clean" --profile kiosk
  [ "$status" -eq 3 ]
  run "$RS" "$FX/clean" --profile
  [ "$status" -eq 3 ]
}

@test "RC16: disabled-gate WARNs a .disabled = true and a disabled attribute in a login-gated file" {
  run "$RS" "$FX/gate"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  rc-scan  rc/config.js:3  disabled-gate"* ]]
  [[ "$output" == *"rc/index.html:2  disabled-gate"* ]]
}

@test "RC17: disabled-gate negatives — class/aria-disabled marking, .disabled = false, and an ungated file stay clean" {
  run "$RS" "$FX/gate-ok"
  [ "$status" -eq 0 ]
  [[ "$output" != *"disabled-gate"* ]]
}

@test "RC18: fetch-no-signal WARNs a multi-line fetch( with no signal and a window.fetch( probe" {
  run "$RS" "$FX/fetch"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  rc-scan  rc/api.js:2  fetch-no-signal"* ]]
  [[ "$output" == *"rc/api.js:5  fetch-no-signal"* ]]
}

@test "RC19: fetch-no-signal negatives — the fetchT helper (signal on a later line), fetchT/apiFetch callers and the allow marker" {
  run "$RS" "$FX/fetch-ok"
  [ "$status" -eq 0 ]
  [[ "$output" != *"fetch-no-signal"* ]]
}

@test "RC20: setinterval-async WARNs setInterval over a named async function, an inline async fn and an async arrow const" {
  run "$RS" "$FX/poll"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  rc-scan  rc/main.js:2  setinterval-async"* ]]
  [[ "$output" == *"rc/main.js:3  setinterval-async"* ]]
  [[ "$output" == *"rc/main.js:5  setinterval-async"* ]]
}

@test "RC21: setinterval-async negatives — self-scheduled setTimeout poll and setInterval over a sync function" {
  run "$RS" "$FX/poll-ok"
  [ "$status" -eq 0 ]
  [[ "$output" != *"setinterval-async"* ]]
}

@test "RC22: innerhtml-server WARNs innerHTML / insertAdjacentHTML concatenating a member without esc(" {
  run "$RS" "$FX/inner"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  rc-scan  rc/alarms.js:2  innerhtml-server"* ]]
  [[ "$output" == *"rc/alarms.js:3  innerhtml-server"* ]]
}

@test "RC23: innerhtml-server negatives — esc(), textContent and literal-only innerHTML stay clean" {
  run "$RS" "$FX/inner-ok"
  [ "$status" -eq 0 ]
  [[ "$output" != *"innerhtml-server"* ]]
}

_datauri_tree() {   # $1 = dir, $2 = payload length
  mkdir -p "$1/rc"
  { printf '.logo { background: url("data:image/png;base64,'
    head -c "$2" /dev/zero | tr '\0' 'A'
    printf '"); }\n'; } > "$1/rc/style.css"
}

@test "RC24: datauri-budget FAILs a data: URI over 20 KB (exit 1); --legacy downgrades it to WARN (exit 0)" {
  T="$(mktemp -d)"
  _datauri_tree "$T/big" 21000
  run "$RS" "$T/big"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  rc-scan  rc/style.css:1  datauri-budget"* ]]
  run "$RS" "$T/big" --legacy
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  rc-scan  rc/style.css:1  datauri-budget"* ]]
  rm -rf "$T"
}

@test "RC25: datauri-budget negative — a small data: URI is clean" {
  T="$(mktemp -d)"
  _datauri_tree "$T/small" 200
  run "$RS" "$T/small"
  [ "$status" -eq 0 ]
  [[ "$output" != *"datauri-budget"* ]]
  rm -rf "$T"
}

@test "RC26: orphan-page WARNs a section#page-<id> with no nav entry and a data-page with no section" {
  run "$RS" "$FX/orphan"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  rc-scan  rc/index.html:4  orphan-page"*"graficas"* ]]
  [[ "$output" == *"rc/index.html:2  orphan-page"*"ghost"* ]]
  [[ "$output" != *"rooms"* ]]
}

@test "RC27: orphan-page negative — every page section has its nav entry and vice versa" {
  run "$RS" "$FX/orphan-ok"
  [ "$status" -eq 0 ]
  [[ "$output" != *"orphan-page"* ]]
}

_inline_tree() {   # $1 = dir, $2 = body lines inside one inline <script>
  mkdir -p "$1/rc"
  { printf '<!doctype html><html><head>\n<script src="js/main.js?v=1"></script>\n<script>\n'
    i=0; while [ "$i" -lt "$2" ]; do printf 'var x%d = %d;\n' "$i" "$i"; i=$((i + 1)); done
    printf '</script>\n</head><body></body></html>\n'; } > "$1/rc/index.html"
}

@test "RC28: inline-block-size WARNs an inline <script> block over 300 lines (at its opening line)" {
  T="$(mktemp -d)"
  _inline_tree "$T/big" 320
  run "$RS" "$T/big"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  rc-scan  rc/index.html:3  inline-block-size"* ]]
  rm -rf "$T"
}

@test "RC29: inline-block-size negative — a short inline block and a <script src> stay clean" {
  T="$(mktemp -d)"
  _inline_tree "$T/small" 40
  run "$RS" "$T/small"
  [ "$status" -eq 0 ]
  [[ "$output" != *"inline-block-size"* ]]
  rm -rf "$T"
}

# polish-2026-10-02 P5 (#199 WU3): the login-gate mention matches auth as a word / camelCase start
# (auth, Auth, authToken, isAuthenticated, AUTH_URL), not inside author / authority.
@test "RC30: disabled-gate — author/authority do not make a file login-gated; isAuthenticated / authToken do" {
  T="$(mktemp -d)"
  mkdir -p "$T/a/rc" "$T/b/rc" "$T/c/rc"
  printf '<!doctype html><html><head><meta name="author" content="Plant team"></head>\n<body><button id="b" disabled>Export</button><p>Data authority: plant</p></body></html>\n' > "$T/a/rc/index.html"
  printf 'if (!session.isAuthenticated) { showGate(); }\nsaveBtn.disabled = true;\n' > "$T/b/rc/app.js"
  printf 'const authToken = readToken();\nsaveBtn.disabled = true;\n' > "$T/c/rc/app.js"
  run "$RS" "$T/a"
  [ "$status" -eq 0 ]
  if [[ "$output" == *"disabled-gate"* ]]; then return 1; fi
  run "$RS" "$T/b"
  [[ "$output" == *"rc/app.js:2  disabled-gate"* ]]
  run "$RS" "$T/c"
  [[ "$output" == *"rc/app.js:2  disabled-gate"* ]]
  rm -rf "$T"
}

@test "RC31: --strict promotes the cross-file orphan-page WARN to FAIL (one severity source for both awk passes)" {
  run "$RS" "$FX/orphan" --strict
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  rc-scan  "*"orphan-page"* ]]
}

# polish-2026-10-02 P5b (P5 review R4-001 / R3-001): the P5 word-start rule dropped OAuth / oauth2 /
# reauth / unauthorized gate mentions (the pre-P5 substring matched them) — a fail-open. They gate again.
@test "RC32: disabled-gate — an OAuth / oauth2 / reauth / unauthorized mention still makes a file login-gated" {
  # BATS_TEST_TMPDIR is removed by bats even when the test fails, and every mention is checked before the verdict.
  local T="$BATS_TEST_TMPDIR/rc32" i=0 m miss=""
  for m in 'startOAuthFlow();' 'const client = oauth2Client();' 'function reauth() { openGate(); }' 'if (res.status === 401) onUnauthorized();'; do
    i=$((i+1)); mkdir -p "$T/m$i/rc"
    printf '%s\nsaveBtn.disabled = true;\n' "$m" > "$T/m$i/rc/app.js"
    run "$RS" "$T/m$i"
    if [[ "$output" != *"rc/app.js:2  disabled-gate"* ]]; then miss="$miss [$m]"; fi
  done
  if [ -n "$miss" ]; then echo "not gated:$miss"; return 1; fi
}

@test "RS-locked: an unreadable sub-directory is an env error (exit 3), never a clean pass" {
  # Mutation: RS-locked -- a raw find walk skips the locked directory and passes clean.
  mkdir -p "$BATS_TEST_TMPDIR/art/rc"
  printf "<p>x</p>\n" > "$BATS_TEST_TMPDIR/art/rc/index.html"
  mkdir -p "$BATS_TEST_TMPDIR/art/locked"
  chmod 000 "$BATS_TEST_TMPDIR/art/locked"
  if [ -r "$BATS_TEST_TMPDIR/art/locked" ]; then chmod 755 "$BATS_TEST_TMPDIR/art/locked"; skip "running as root"; fi
  run "$RS" "$BATS_TEST_TMPDIR/art"
  chmod 755 "$BATS_TEST_TMPDIR/art/locked"
  [ "$status" -eq 3 ]
  [[ "$output" == *"rc-scan: cannot list every file under $BATS_TEST_TMPDIR/art: "* ]]
}

@test "RS-unreadable: an unreadable rc file is an env error (exit 3), never a skipped file" {
  # Mutation: RS-unreadable -- skipping a file os.ReadFile cannot read (the awk's fatal-and-continue) passes clean
  mkdir -p "$BATS_TEST_TMPDIR/art/rc"
  printf 'var ok = 1;\n' > "$BATS_TEST_TMPDIR/art/rc/a.js"
  printf 'var o = "station:|slot:/x";\n' > "$BATS_TEST_TMPDIR/art/rc/u.js"
  chmod 000 "$BATS_TEST_TMPDIR/art/rc/u.js"
  if [ -r "$BATS_TEST_TMPDIR/art/rc/u.js" ]; then chmod 644 "$BATS_TEST_TMPDIR/art/rc/u.js"; skip "running as root"; fi
  run "$RS" "$BATS_TEST_TMPDIR/art"
  chmod 644 "$BATS_TEST_TMPDIR/art/rc/u.js"
  [ "$status" -eq 3 ]
  [[ "$output" == *"rc-scan: cannot scan $BATS_TEST_TMPDIR/art/rc/u.js: Permission denied"* ]]
}

# Issue #459 (d): vendored libraries under rc/vendor/**/*.js belong to lint-vendor-floor.sh (acorn at the
# Chromium 83 floor); rc-scan's line heuristics flagged three.js RenderPass.js / SSAOPass.js there.
@test "RC33: rc/vendor/**/*.js is not scanned (lint-vendor-floor owns it); vendored CSS and the module's own JS are" {
  # Mutation: RC33 -- scanning rc/vendor/**/*.js again flags the vendored null-branch and ??= rows
  local T="$BATS_TEST_TMPDIR/rc33"
  mkdir -p "$T/rc/vendor/three/addons" "$T/rc/js"
  printf 'this.c = c !== undefined ? null : x;\ny ??= 1;\n' > "$T/rc/vendor/three/addons/RenderPass.js"
  printf '.a { inset: 0 }\n' > "$T/rc/vendor/three/lib.css"
  printf 'x = a ? null : b;\n' > "$T/rc/js/app.js"
  run "$RS" "$T"
  [ "$status" -eq 0 ]
  [[ "$output" == *"rc/js/app.js:1  null-branch"* ]]
  [[ "$output" == *"rc/vendor/three/lib.css:1  browser-floor"* ]]
  if [[ "$output" == *"RenderPass.js"* ]]; then return 1; fi
}

# Issue #459 (b): a control disabled with a stated non-gate reason on the same element and line (PANCCADIA
# index.html:1482 `b.disabled = true; b.title = "Sin salida cableada";`) is meant to be inert, not gated.
@test "RC34: disabled-gate — a literal non-gate title on the disabled element is a stated reason; a gate title or none still WARNs" {
  # Mutation: RC34 -- ignoring the stated title reason WARNs the wiring-disabled control again
  local T="$BATS_TEST_TMPDIR/rc34"
  mkdir -p "$T/rc"
  printf '%s\n' 'if (!session.loggedIn) showLogin();' \
    'if (unwired) { b.disabled = true; b.title = "Sin salida cableada"; }' \
    'saveBtn.disabled = true;' \
    'b.disabled = true; b.title = "Inicia sesion (login) para escribir";' > "$T/rc/app.js"
  run "$RS" "$T"
  [ "$status" -eq 0 ]
  [[ "$output" == *"rc/app.js:3  disabled-gate"* ]]
  [[ "$output" == *"rc/app.js:4  disabled-gate"* ]]
  if [[ "$output" == *"rc/app.js:2  disabled-gate"* ]]; then return 1; fi
}

# Issue #459 (a), part 1: innerhtml-server reads the sink's own value — a member elsewhere on a minified line,
# a .length, a .toFixed( number or a Math.* value is not server text (Umbrella index.html:2995, :3023).
@test "RC35: innerhtml-server — only a raw member in the sink's own value WARNs; .toFixed(, .length, Math.* and a member beside it do not" {
  # Mutation: RC35 -- matching the member anywhere on the line again WARNs the spec.unit line (app.js:2)
  local T="$BATS_TEST_TMPDIR/rc35"
  mkdir -p "$T/rc"
  printf '%s\n' 'el.innerHTML = "<b>" + a.message;' \
    "const h = '<i>' + spec.unit; el.innerHTML = '<p>static</p>';" \
    "el.innerHTML = '<b>' + val.toFixed(1) + Math.round(x) + rows.length;" > "$T/rc/app.js"
  run "$RS" "$T"
  [ "$status" -eq 0 ]
  [[ "$output" == *"rc/app.js:1  innerhtml-server"* ]]
  if [[ "$output" == *"rc/app.js:2  innerhtml-server"* || "$output" == *"rc/app.js:3  innerhtml-server"* ]]; then return 1; fi
}

# Issue #459 (a), part 2: a member of a const literal the page declares (Umbrella index.html:2987
# units.map(u => ... + u.name)) is not server data; data from elsewhere, a let and a literal the page writes
# into still WARN.
@test "RC36: innerhtml-server — a member of a same-file const literal is no row; a non-literal, a let or a written literal still WARNs" {
  # Mutation: RC36 -- dropping literal provenance WARNs the units.map(u => u.name) line (app.js:2)
  local T="$BATS_TEST_TMPDIR/rc36"
  mkdir -p "$T/rc"
  printf '%s\n' 'const units = [{ id: "UP-01", name: "RH A" }];' \
    "el.innerHTML = units.map(u => '<b>' + u.name).join('');" \
    "el.innerHTML = data.map(u => '<b>' + u.name).join('');" \
    "const live = [{ name: 'a' }]; live[0].name = srv; el.innerHTML = live.map(m => '<b>' + m.name).join('');" > "$T/rc/app.js"
  run "$RS" "$T"
  [ "$status" -eq 0 ]
  [[ "$output" == *"rc/app.js:3  innerhtml-server"* ]]
  [[ "$output" == *"rc/app.js:4  innerhtml-server"* ]]
  if [[ "$output" == *"rc/app.js:2  innerhtml-server"* ]]; then return 1; fi
}

# Walk contract (lib/scan-files.sh, kept by the Go port's lint.Walk): a sibling *-worktrees checkout is pruned,
# a symlinked artifact root is followed (find -H) and a dot-named root is walked (find -mindepth 1).
@test "RS-walk: a *-worktrees copy gives no row; a symlinked and a dot-named root are scanned" {
  # Mutation: RS-walk -- dropping the *-worktrees prune reports the x-worktrees copy's ORD literal
  local T="$BATS_TEST_TMPDIR/rswalk"
  mkdir -p "$T/art/rc" "$T/art/x-worktrees/m/rc" "$T/.dotroot/rc"
  printf 'var o = "station:|x";\n' > "$T/art/rc/b.js"
  printf 'var o = "station:|x";\n' > "$T/art/x-worktrees/m/rc/a.js"
  cp "$T/art/rc/b.js" "$T/.dotroot/rc/b.js"
  ln -s "$T/art" "$T/link"
  run "$RS" "$T/art"
  [ "$status" -eq 1 ]
  [[ "$output" == *"rc/b.js:1  ord-literal"* ]]
  if [[ "$output" == *"x-worktrees"* ]]; then return 1; fi
  run "$RS" "$T/link"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  rc-scan  rc/b.js:1  ord-literal"* ]]
  run "$RS" "$T/.dotroot"
  [ "$status" -eq 1 ]
  [[ "$output" == *"rc/b.js:1  ord-literal"* ]]
}
