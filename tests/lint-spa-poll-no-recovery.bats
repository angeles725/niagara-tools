#!/usr/bin/env bats
# Tests for build-n4-module-kit/toolbelt/lint-spa-poll-no-recovery.sh
# A setInterval/setTimeout poll loop whose catch path only marks data stale and never reloads
# or re-authenticates leaves a kiosk/HMI blank forever after a station restart.
# [ev: retro panccadia-defrost-sequencing-hmi-reload-deltas Δ1]
# WU3 rework: the recovery must be a watchdog keyed on time since the last SUCCESS (a lastOk*
# timestamp) plus a reload; a failure-count-only watchdog or a bare reload-on-error still WARNs.
# [ev: retro dashboard-frontend-reliability-rules Δ3]

setup() {
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  SPR="$KIT/toolbelt/lint-spa-poll-no-recovery.sh"
  mkdir -p "$TMPDIR_T/Mod/src/rc"
}
teardown() { rm -rf "$TMPDIR_T"; }

@test "SPR-usage: no arg exits 3" { run "$SPR"; [ "$status" -eq 3 ]; }

@test "SPR-nondir: a non-directory arg exits 3" { run "$SPR" "$TMPDIR_T/nope"; [ "$status" -eq 3 ]; }

_write_bad() {
  # setInterval(poll,...) -> poll() has a catch, no location.reload/href anywhere in the file.
  cat > "$TMPDIR_T/Mod/src/rc/index.html" << 'EOF'
<script>
async function poll() {
  try {
    await readJson();
    paint("live");
  } catch (err) {
    data.forEach(d => { d.st = "stale"; });
    paint("error");
  }
}
poll();
setInterval(poll, N4.pollMs);
</script>
EOF
}

@test "SPR1: poll loop with a catch and no reload/href recovery anywhere -> WARN" {
  _write_bad
  run "$SPR" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]]
  [[ "$output" == *"lint-spa-poll-no-recovery"* ]]
  [[ "$output" == *"poll()"* ]]
}

@test "SPR1-strict: --strict promotes the WARN to exit 1" {
  _write_bad
  run "$SPR" --strict "$TMPDIR_T/Mod"
  [ "$status" -eq 1 ]
}

@test "SPR2: a failure-count-only watchdog (reload after N failures, no lastOk timestamp) still WARNs" {
  cat > "$TMPDIR_T/Mod/src/rc/index.html" << 'EOF'
<script>
let _wdFailCount = 0;
function _wdProbe() {
  fetch(location.href, { method: "GET" }).then(function (r) {
    if (r.ok) { location.reload(); }
  });
}
async function poll() {
  try {
    await readJson();
    _wdFailCount = 0;
    paint("live");
  } catch (err) {
    _wdFailCount++;
    if (_wdFailCount >= 6) { _wdProbe(); }
    paint("error");
  }
}
poll();
setInterval(poll, N4.pollMs);
</script>
EOF
  run "$SPR" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]]
  [[ "$output" == *"time since the last success"* ]]
  # Named mutation: drop the lastOk success-time requirement -> SPR2 passes clean (no WARN).
}

@test "SPR3: a poll function with no catch at all is clean (not the flagged shape)" {
  cat > "$TMPDIR_T/Mod/src/rc/index.html" << 'EOF'
<script>
function poll() {
  readJson().then(function () { paint("live"); });
}
setInterval(poll, 5000);
</script>
EOF
  run "$SPR" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}

@test "SPR4: no setInterval/setTimeout at all is clean" {
  cat > "$TMPDIR_T/Mod/src/rc/index.html" << 'EOF'
<script>
function once() {
  try { doThing(); } catch (e) { console.error(e); }
}
once();
</script>
EOF
  run "$SPR" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}

@test "SPR5: a bare location.href reassignment on error (no lastOk watchdog) still WARNs" {
  cat > "$TMPDIR_T/Mod/src/rc/index.html" << 'EOF'
<script>
async function poll() {
  try {
    await readJson();
    paint("live");
  } catch (err) {
    location.href = location.href;
  }
}
setInterval(poll, 5000);
</script>
EOF
  run "$SPR" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]]
  [[ "$output" == *"time since the last success"* ]]
}

_write_success_watchdog() {   # $1 = recovery statement
  cat > "$TMPDIR_T/Mod/src/rc/index.html" << EOF
<script>
const T = { pollMs: 5000, fetchTimeoutMs: 8000, recoverMs: 30000, ceilingMs: 600000 };
let lastOkAt = Date.now();
function watchdog() {
  const since = Date.now() - lastOkAt;
  if (since > T.ceilingMs) { $1 }
  if (since > T.recoverMs) {
    fetchT("/api/data", {}, T.fetchTimeoutMs).then(function (r) { return r.json(); })
      .then(function () { $1 }).catch(function () {});
  }
}
async function poll() {
  try {
    await readJson();
    lastOkAt = Date.now();
    paint("live");
  } catch (err) {
    paint("error");
  } finally {
    watchdog();
    setTimeout(poll, T.pollMs);
  }
}
setTimeout(poll, T.pollMs);
</script>
EOF
}

@test "SPR6: a success-time watchdog (lastOkAt = Date.now() + location.reload()) is clean" {
  _write_success_watchdog 'location.reload();'
  run "$SPR" --strict "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}

@test "SPR7: a success-time watchdog recovering through location.href = is clean" {
  _write_success_watchdog 'location.href = location.pathname;'
  run "$SPR" --strict "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}

@test "SPR8: a lastOkAt timestamp with no reload/href recovery anywhere still WARNs (no recovery)" {
  _write_success_watchdog 'paint("stale");'
  run "$SPR" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]]
  [[ "$output" == *"no location.reload("* ]]
  # Named mutation: drop the reload/href recovery check -> SPR8 passes clean.
}

# polish-2026-10-02 P5 (#199 WU3): a lastOk* timestamp set only at load (never on a poll success)
# is not a success-time key — the watchdog fires on the load time, not on the last success.
@test "SPR9: a lastOkAt set only at load (never in the poll's success path) still WARNs (not keyed)" {
  cat > "$TMPDIR_T/Mod/src/rc/index.html" << 'EOF2'
<script>
let lastOkAt = Date.now();
function watchdog() {
  if (Date.now() - lastOkAt > 30000) { location.reload(); }
}
async function poll() {
  try {
    await readJson();
    paint("live");
  } catch (err) {
    paint("error");
  } finally {
    watchdog();
    setTimeout(poll, 5000);
  }
}
setTimeout(poll, 5000);
</script>
EOF2
  run "$SPR" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]]
  [[ "$output" == *"time since the last success"* ]]
  # Named mutation SPR9: count a clock assignment anywhere in the file -> SPR9 passes clean.
}

@test "SPR13: a location.reload() quoted in a comment or a string is not a recovery" {
  # Mutation: SPR13 -- reading block comments as code takes /* location.reload(); */ for the recovery
  # Go golden: go/internal/lint/rules/spapoll/testdata/truth-code-view-1.txt
  printf '%s\n' 'function poll() { a().catch(e => 0); }' 'setInterval(poll, 1);' '/* location.reload(); */' > "$TMPDIR_T/Mod/src/rc/a.js"
  run "$SPR" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"a.js:2  setInterval/setTimeout poll loop 'poll()' has a catch path with no location.reload("* ]]
}

@test "SPR14: a minified library line does not stand for the page's poll loop" {
  # Mutation: SPR14 -- searching minified lines takes a bundled library's setTimeout(n,0) for the poll call
  # Go golden: go/internal/lint/rules/spapoll/testdata/truth-minified-1.txt
  { printf '<script>!function(t){function n(){try{t()}catch(e){}}setTimeout(n,0);'
    for _ in $(seq 150); do printf 'var a=1;'; done
    printf '}(this);</script>\n'; } > "$TMPDIR_T/Mod/src/rc/lib.html"
  run "$SPR" --strict "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "SPR15: a location.href comparison is not a recovery" {
  # Mutation: SPR15 -- matching location.href = also inside location.href == takes a comparison for the recovery
  # Go golden: go/internal/lint/rules/spapoll/testdata/truth-recovery-shapes-1.txt
  printf '%s\n' 'function poll() { a().catch(e => 0); }' 'setInterval(poll, 1);' 'if (location.href == "/x") go();' > "$TMPDIR_T/Mod/src/rc/a.js"
  run "$SPR" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"a.js:2  setInterval/setTimeout poll loop 'poll()' has a catch path with no location.reload("* ]]
}

@test "SPR-awkfail: an unreadable source file is an env error (exit 3, named on stderr), never a clean pass" {
  # Mutation: SPR-awkfail -- ignoring the awk exit status reports the unreadable file as clean (exit 0).
  mkdir -p "$(dirname "$TMPDIR_T/Mod/src/rc/u.js")"
  printf 'x\n' > "$TMPDIR_T/Mod/src/rc/u.js"
  chmod 000 "$TMPDIR_T/Mod/src/rc/u.js"
  if [ -r "$TMPDIR_T/Mod/src/rc/u.js" ]; then chmod 644 "$TMPDIR_T/Mod/src/rc/u.js"; skip "running as root: chmod 000 does not block reads"; fi
  run "$SPR" "$TMPDIR_T/Mod"
  chmod 644 "$TMPDIR_T/Mod/src/rc/u.js"
  [ "$status" -eq 3 ]
  [[ "$output" == *"Mod/src/rc/u.js"* ]]
}

@test "SPR-finderr: an unreadable sub-directory is an env error (exit 3, named on stderr), never a clean pass" {
  # Mutation: SPR-finderr -- ignoring the find exit status skips the unreadable directory's files and exits 0.
  # [ev: issue #226 R3-find-error-still-fail-open]
  mkdir -p "$TMPDIR_T/Mod/src/rc/locked"
  printf 'var a = 1;\n' > "$TMPDIR_T/Mod/src/rc/a.js"
  printf 'var b = 1;\n' > "$TMPDIR_T/Mod/src/rc/locked/b.js"
  chmod 000 "$TMPDIR_T/Mod/src/rc/locked"
  if [ -r "$TMPDIR_T/Mod/src/rc/locked" ]; then chmod 755 "$TMPDIR_T/Mod/src/rc/locked"; skip "running as root: chmod 000 does not block reads"; fi
  run "$SPR" "$TMPDIR_T/Mod"
  chmod 755 "$TMPDIR_T/Mod/src/rc/locked"
  [ "$status" -eq 3 ]
  [[ "$output" == *"Mod/src/rc/locked"* ]]
}

@test "SPR10: a poll declared as a const arrow function is found (no recovery -> WARN)" {
  # Mutation: SPR10 -- matching only `function <name>(` loses an arrow-function poll (silent pass).
  cat > "$TMPDIR_T/Mod/src/rc/arrow.js" << 'JEOF'
const poll = async () => {
  try {
    const r = await fetch('/api/state');
    render(await r.json());
  } catch (e) {
    markStale();
  }
};
setInterval(poll, 5000);
JEOF
  run "$SPR" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"arrow.js:9"* ]]
  [[ "$output" == *"no location.reload("* ]]
}

@test "SPR11: a concise-body arrow poll ends at its statement -- a lastOk set in a later function is not its success path (#226)" {
  # Mutation: SPR11 -- with no braces to extract, the poll body ran to end of file, so a lastOkAt assignment in
  # an unrelated later function read as the poll's success time and the WARN vanished.
  cat > "$TMPDIR_T/Mod/src/rc/concise.js" << 'JEOF'
const poll = () => fetch('/api/state').then((r) => r.json()).then(render).catch(markStale);
setInterval(poll, 5000);
function onLoginOk() { lastOkAt = Date.now(); }
function watchdog() { if (Date.now() - lastOkAt > 30000) location.reload(); }
JEOF
  run "$SPR" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"concise.js:2"* ]]
  [[ "$output" == *"not keyed on time since the last success"* ]]
}

@test "SPR12: a concise-body arrow poll that spans lines up to its ';' keeps its own .catch( (#226)" {
  cat > "$TMPDIR_T/Mod/src/rc/concise2.js" << 'JEOF'
const poll = () =>
  fetch('/api/state')
    .then((r) => r.json())
    .catch(markStale);
setInterval(poll, 5000);
JEOF
  run "$SPR" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"concise2.js:5"* ]]
  [[ "$output" == *"no location.reload("* ]]
}

@test "SPR16: a call to a probe-then-reload helper is the recovery (issue #895); only a declaration or a comment is not" {
  # Mutation: SPR16 -- reading a `function probeThenReload(` declaration as a call takes it for the recovery
  d="$TMPDIR_T/Mod/src/rc"
  printf 'function poll() { lastOkAt = Date.now(); a().catch(e => 0); }\nsetInterval(poll, 1);\nif (Date.now() - lastOkAt > 6e4) probeThenReload();\n' > "$d/a.js"
  run "$SPR" "$d"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  printf 'function poll() { lastOkAt = Date.now(); a().catch(e => 0); }\nsetInterval(poll, 1);\nfunction probeThenReload() { return 1; }\n// probeThenReload();\n' > "$d/a.js"
  run "$SPR" "$d"
  [[ "$output" == *"a.js:2"*"no location.reload("* ]]
}

# Issue #895 (FAIL half): a poll whose catch path swallows the failure and then stamps lastOk* defeats the
# success-time watchdog -- the watchdog never sees the failure -- so it is a FAIL, not a WARN.
@test "SPR17: a poll catch that falls through to a lastOk* stamp FAILs (exit 1, no --strict needed); rethrow, return or a stamp inside the try is clean (issue #895)" {
  # Mutation: SPR17 -- not reading the catch fall-through takes a swallowed failure for a success and passes clean
  d="$TMPDIR_T/Mod/src/rc"
  printf 'let lastOkAt = 0;\nasync function poll() {\n  try { await load(); } catch (e) { paint("error"); }\n  lastOkAt = Date.now();\n  if (Date.now() - lastOkAt > 3e4) location.reload();\n}\nsetTimeout(poll, 5000);\n' > "$d/a.js"
  run "$SPR" "$d"
  [ "$status" -eq 1 ]
  [[ "$output" == "FAIL  lint-spa-poll-no-recovery  $d/a.js:7  "*"swallows a failure"* ]]
  printf 'let lastOkAt = 0;\nasync function poll() {\n  await load().catch(e => paint("error"));\n  lastOkAt = Date.now();\n  if (Date.now() - lastOkAt > 3e4) location.reload();\n}\nsetTimeout(poll, 5000);\n' > "$d/a.js"
  run "$SPR" "$d"
  [ "$status" -eq 1 ]
  [[ "$output" == *"a.js:7"*"swallows a failure"* ]]
  printf 'let lastOkAt = 0;\nasync function poll() {\n  try { await load(); } catch (e) { paint("error"); throw e; }\n  lastOkAt = Date.now();\n  if (Date.now() - lastOkAt > 3e4) location.reload();\n}\nsetTimeout(poll, 5000);\n' > "$d/a.js"
  run "$SPR" --strict "$d"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  printf 'let lastOkAt = 0;\nasync function poll() {\n  try { await load(); lastOkAt = Date.now(); } catch (e) { paint("error"); }\n  if (Date.now() - lastOkAt > 3e4) location.reload();\n}\nsetTimeout(poll, 5000);\n' > "$d/a.js"
  run "$SPR" --strict "$d"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

# Issue #966 item 1: a stamp the catch does not make unconditional is not the swallowed-failure shape. Each stamp sits
# behind if / else / ?: / && / ||, or at another brace depth than the catch (a bare block, a function declared later).
@test "SPR18: a lastOk* stamp behind if/else/?:/&&/|| or at another brace depth is clean (issue #966)" {
  # Mutation: SPR18 -- dropping the `conditional` regex or the brace-depth filter FAILs a stamp that is not the shape
  d="$TMPDIR_T/Mod/src/rc"
  stamps=(
    'if (ready) lastOkAt = Date.now();'
    'if (stale) go(); else lastOkAt = Date.now();'
    'ok ? (lastOkAt = Date.now()) : 0;'
    'ready && (lastOkAt = Date.now());'
    'cached || (lastOkAt = Date.now());'
    '{ lastOkAt = Date.now(); }'
    'function later() { lastOkAt = Date.now(); }'
  )
  for st in "${stamps[@]}"; do
    printf 'let lastOkAt = 0;\nasync function poll() {\n  try { await load(); } catch (e) { paint("error"); }\n  %s\n  if (Date.now() - lastOkAt > 3e4) location.reload();\n}\nsetTimeout(poll, 5000);\n' "$st" > "$d/a.js"
    run "$SPR" "$d"
    [ "$status" -eq 0 ] || { echo "FAILed on: $st: $output"; false; }
    [ -z "$output" ]
  done
  printf 'let lastOkAt = 0;\nasync function poll() {\n  try { await load(); } catch (e) { paint("error"); }\n  lastOkAt = Date.now();\n  if (Date.now() - lastOkAt > 3e4) location.reload();\n}\nsetTimeout(poll, 5000);\n' > "$d/a.js"
  run "$SPR" "$d"
  [ "$status" -eq 1 ]
}

# Issue #966 item 3: the catch exits when its OWN handler throws, returns or hands back Promise.reject; an exit inside
# a callback the handler passes on is the callback's, and a comment or a string is no code.
@test "SPR19: .catch(e => Promise.reject(e)) rethrows; a return in a nested callback of the handler does not (issue #966)" {
  # Mutation: SPR19 -- reading the whole handler text takes a nested callback's return for the catch exiting (missed swallow)
  d="$TMPDIR_T/Mod/src/rc"
  tail='  lastOkAt = Date.now();\n  if (Date.now() - lastOkAt > 3e4) location.reload();\n}\nsetTimeout(poll, 5000);\n'
  clean=(
    'await load().catch(e => Promise.reject(e));'
    'await load().catch(e => { return Promise.reject(e); });'
    'await load().catch(function (e) { throw e; });'
  )
  for h in "${clean[@]}"; do
    printf 'let lastOkAt = 0;\nasync function poll() {\n  %s\n%b' "$h" "$tail" > "$d/a.js"
    run "$SPR" "$d"
    [ "$status" -eq 0 ] || { echo "FAILed on: $h: $output"; false; }
    [ -z "$output" ]
  done
  swallow=(
    'try { await load(); } catch (e) { items.forEach(i => { return; }); }'
    'await load().catch(e => { setTimeout(function () { throw e; }); });'
    'await load().catch(e => later(() => Promise.reject(e)));'
    'await load().catch(e => { Promise.reject(e); });'
    'try { await load(); } catch (e) { /* return; */ paint("throw e"); }'
  )
  for h in "${swallow[@]}"; do
    printf 'let lastOkAt = 0;\nasync function poll() {\n  %s\n%b' "$h" "$tail" > "$d/a.js"
    run "$SPR" "$d"
    [ "$status" -eq 1 ] || { echo "clean on: $h: $output"; false; }
    [[ "$output" == *"a.js:7"*"swallows a failure"* ]]
  done
}
