#!/usr/bin/env bats
# Tests for build-n4-module-kit/toolbelt/lint-spa-poll-no-recovery.sh
# A setInterval/setTimeout poll loop whose catch path only marks data stale and never reloads
# or re-authenticates leaves a kiosk/HMI blank forever after a station restart.
# [ev: retro panccadia-defrost-sequencing-hmi-reload-deltas Δ1]

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

@test "SPR2: a failure-count watchdog that eventually calls location.reload() suppresses the WARN" {
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
  [[ "$output" != *"WARN"* ]]
  # Named mutation: drop the location.reload(/location.href check -> SPR2 false-WARNs on this fixture.
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

@test "SPR5: a plain location.href reassignment in the catch path also counts as recovery" {
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
  [[ "$output" != *"WARN"* ]]
}
