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
