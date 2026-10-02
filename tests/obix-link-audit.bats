#!/usr/bin/env bats
# Pins for obix-link-audit.sh — declared wiring map vs the links actually present on a live
# component (read-only oBIX). A link from the WRONG-but-valid source is structurally fine, so
# bog-audit cannot see it; only a diff against the declared source closes that gap.
# Rows: MATCH | MISMATCH | MISSING | SKIP  obix-link-audit  <slot>  <detail>
# Exit: 0 no MISMATCH/MISSING · 1 any · 3 usage/env.
# [ev: retro panccadia-commissioning-lessons Δ4]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  OLA="$KIT/toolbelt/obix-link-audit.sh"
  FX="$BATS_TEST_DIRNAME/fixtures/obix-link-audit"
}

@test "OLA1: Table 2 vs a saved component dump -> MATCH / MISMATCH (actual source named) / MISSING / SKIP, exit 1" {
  run "$OLA" --map "$FX/wiring-map.md" --xml "$FX/facade.xml"
  [ "$status" -eq 1 ]
  [[ "$output" == *"MISMATCH  obix-link-audit  comp1State  declared comp1Running in Rack, linked from relayOut1 (slot:/Plant/IO)"* ]]
  [[ "$output" == *"MATCH  obix-link-audit  comp2State  "* ]]
  [[ "$output" == *"MATCH  obix-link-audit  comp3State  "* ]]
  [[ "$output" == *"MISSING  obix-link-audit  inDrip  declared dripActive, no link into inDrip"* ]]
  [[ "$output" == *"SKIP  obix-link-audit  freezeActive  "* ]]
  [[ "$output" == *"obix-link-audit: 2 MATCH · 1 MISMATCH · 1 MISSING · 1 SKIP  ->  ISSUES"* ]]
}

@test "OLA2: every declared link-in present from its declared source (arrow forms + Direct) -> exit 0, CLEAN" {
  run "$OLA" --map "$FX/wiring-map.md" --xml "$FX/facade-all-match.xml"
  [ "$status" -eq 0 ]
  [[ "$output" == *"4 MATCH · 0 MISMATCH · 0 MISSING · 1 SKIP  ->  CLEAN"* ]]
}

@test "OLA3: --table 1 audits facade->control rows against the control component" {
  run "$OLA" --map "$FX/wiring-map.md" --xml "$FX/control.xml" --table 1
  [ "$status" -eq 1 ]
  [[ "$output" == *"MATCH  obix-link-audit  suctionSetpoint  "* ]]
  [[ "$output" == *"MISSING  obix-link-audit  suctionDiffUp  declared diffUp, no link into suctionDiffUp"* ]]
  [[ "$output" == *"SKIP  obix-link-audit  diffDown  "* ]]
}

@test "OLA4: usage faults -> exit 3 (no args, missing map, both --xml and --obix, bad --table)" {
  run "$OLA"
  [ "$status" -eq 3 ]
  run "$OLA" --map "$BATS_TEST_TMPDIR/absent.md" --xml "$FX/facade.xml"
  [ "$status" -eq 3 ]
  run "$OLA" --map "$FX/wiring-map.md" --xml "$FX/facade.xml" --obix https://station.example/obix --component /Plant/Facade
  [ "$status" -eq 3 ]
  run "$OLA" --map "$FX/wiring-map.md" --xml "$FX/facade.xml" --table 3
  [ "$status" -eq 3 ]
}

@test "OLA5: live mode -> ONE read-only GET of <base>/config/<component>/, credentials via stdin config (never argv)" {
  bin="$BATS_TEST_TMPDIR/bin"; mkdir -p "$bin"
  cat > "$bin/curl" <<STUB
#!/usr/bin/env bash
printf '%s\n' "\$@" > "$BATS_TEST_TMPDIR/curl.args"
cat > "$BATS_TEST_TMPDIR/curl.stdin"
out=""
while [ \$# -gt 0 ]; do [ "\$1" = "-o" ] && out="\$2"; shift; done
cp "$FX/facade.xml" "\$out"
STUB
  chmod +x "$bin/curl"
  OBIX_USER=reader OBIX_PASS=secret PATH="$bin:$PATH" run "$OLA" --map "$FX/wiring-map.md" \
    --obix https://station.example/obix --component /Plant/Facade
  [ "$status" -eq 1 ]
  [[ "$output" == *"MISMATCH  obix-link-audit  comp1State"* ]]
  grep -qx 'https://station.example/obix/config/Plant/Facade/' "$BATS_TEST_TMPDIR/curl.args"
  [ "$(grep -cE '^(-X|--request|-d|--data.*|-T|--upload-file)$' "$BATS_TEST_TMPDIR/curl.args")" -eq 0 ]
  [ "$(grep -c 'secret' "$BATS_TEST_TMPDIR/curl.args")" -eq 0 ]
  grep -q 'reader:secret' "$BATS_TEST_TMPDIR/curl.stdin"
}

@test "OLA6: live mode without OBIX_USER/OBIX_PASS -> exit 3, no request made" {
  bin="$BATS_TEST_TMPDIR/bin"; mkdir -p "$bin"
  printf '#!/usr/bin/env bash\ntouch "%s/called"\n' "$BATS_TEST_TMPDIR" > "$bin/curl"; chmod +x "$bin/curl"
  OBIX_USER='' OBIX_PASS='' PATH="$bin:$PATH" run "$OLA" --map "$FX/wiring-map.md" \
    --obix https://station.example/obix --component /Plant/Facade
  [ "$status" -eq 3 ]
  [ ! -e "$BATS_TEST_TMPDIR/called" ]
}

@test "OLA7: a component with NO links -> every declared row MISSING (empty link set never reads the map as links)" {
  printf '<obj href="https://station.example/obix/config/Plant/Facade/"/>\n' > "$BATS_TEST_TMPDIR/empty.xml"
  run "$OLA" --map "$FX/wiring-map.md" --xml "$BATS_TEST_TMPDIR/empty.xml"
  [ "$status" -eq 1 ]
  [[ "$output" == *"0 MATCH · 0 MISMATCH · 4 MISSING · 1 SKIP  ->  ISSUES"* ]]
}

# polish-2026-10-02 P3 (#199 WU9): stub curl that records argv and stdin, copies a fixture to -o, and
# exits with $CURL_RC (default 0).
_stub_curl() {
  bin="$BATS_TEST_TMPDIR/bin"; mkdir -p "$bin"
  cat > "$bin/curl" <<STUB
#!/usr/bin/env bash
printf '%s\n' "\$@" > "$BATS_TEST_TMPDIR/curl.args"
cat > "$BATS_TEST_TMPDIR/curl.stdin"
[ "\${CURL_RC:-0}" -eq 0 ] || exit "\$CURL_RC"
out=""
while [ \$# -gt 0 ]; do [ "\$1" = "-o" ] && out="\$2"; shift; done
cp "$FX/facade.xml" "\$out"
STUB
  chmod +x "$bin/curl"
}

# The credentials reach curl's config parser intact: `"` and `\` are backslash-escaped inside the
# quoted value (curl config quoting rules), so a password with `"`, `\`, a space and `#` is the one the
# station receives. Proven end to end against a one-shot local HTTP listener with the REAL curl (the
# Authorization header is decoded). Fake credentials only.
# Named mutation OLA-esc (interpolate the raw values again) -> OLA-esc flips.
@test "OLA-esc: a password with a quote, backslash, space and # reaches the station intact (real curl)" {
  command -v python3 >/dev/null 2>&1 && command -v curl >/dev/null 2>&1 || skip "python3 and curl required"
  local pf="$BATS_TEST_TMPDIR/port" af="$BATS_TEST_TMPDIR/auth" port="" _
  python3 - "$pf" "$af" "$FX/facade.xml" >/dev/null 2>&1 3>&- <<'PY' &
import http.server, os, sys
pf, af, body = sys.argv[1:4]
class H(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        with open(af, "w") as f:
            f.write(self.headers.get("Authorization", ""))
        data = open(body, "rb").read()
        self.send_response(200)
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)
    def log_message(self, *a):
        pass
s = http.server.HTTPServer(("127.0.0.1", 0), H)
s.timeout = 20
with open(pf + ".tmp", "w") as f:
    f.write(str(s.server_port))
os.rename(pf + ".tmp", pf)
s.handle_request()
PY
  for _ in $(seq 1 100); do [ -s "$pf" ] && break; sleep 0.1; done
  [ -s "$pf" ] || skip "local listener did not start"
  port=$(cat "$pf")
  run env -u http_proxy -u HTTP_PROXY -u all_proxy -u ALL_PROXY NO_PROXY='*' \
    OBIX_USER=reader OBIX_PASS='p"a\ss #1 x' "$OLA" --map "$FX/wiring-map.md" \
    --obix "http://127.0.0.1:$port/obix" --component /Plant/Facade
  [ "$status" -eq 1 ] || { echo "$output"; return 1; }
  [[ "$output" == *"MISMATCH  obix-link-audit  comp1State"* ]]
  local got
  got=$(python3 -c 'import base64,sys; print(base64.b64decode(open(sys.argv[1]).read().split()[1]).decode())' "$af")
  [ "$got" = 'reader:p"a\ss #1 x' ] || { echo "station received: $got"; return 1; }
}

# MATCH compares the source COMPONENT too when the map declares one: on a per-instance facade a link
# from the right slot of the WRONG instance (Rack1 instead of Rack2) is a MISMATCH.
# Named mutation OLA-comp (compare the slot name only) -> OLA-comp flips.
# shellcheck disable=SC2016  # literal markdown backticks in the map, not expansions
@test "OLA-comp: right slot from the wrong component instance -> MISMATCH; the declared instance -> MATCH" {
  local m="$BATS_TEST_TMPDIR/map.md" x="$BATS_TEST_TMPDIR/f.xml"
  printf '## Table 2 — SUMMARY display slots (control → facade)\n\n| Facade slot | Source RT slot | Notes |\n|---|---|---|\n| `comp1State` | `Rack2/comp1Running` | |\n| `comp2State` | `Plant/Rack2.comp2Running` | |\n' > "$m"
  printf '<obj href="x/">\n<obj name="Link" is="baja:Link" display="Indirect: slot:/Plant/Rack1.comp1Running &#x2192; slot:/Plant/Facade2.comp1State"/>\n<obj name="Link1" is="baja:Link" display="Indirect: slot:/Plant/Rack2.comp2Running &#x2192; slot:/Plant/Facade2.comp2State"/>\n</obj>\n' > "$x"
  run "$OLA" --map "$m" --xml "$x"
  [ "$status" -eq 1 ]
  [[ "$output" == *"MISMATCH  obix-link-audit  comp1State  declared comp1Running in Rack2, linked from comp1Running (slot:/Plant/Rack1)"* ]]
  [[ "$output" == *"MATCH  obix-link-audit  comp2State  linked from declared comp2Running in Plant/Rack2"* ]]
  # a suffix that is not a whole path segment is not the declared instance (Rack2 vs Rack12)
  sed -i 's#Rack2.comp2Running &#Rack12.comp2Running \&#' "$x"
  run "$OLA" --map "$m" --xml "$x"
  [[ "$output" == *"MISMATCH  obix-link-audit  comp2State  "* ]]
}

# An http:// base sends the oBIX credentials in clear text: warn on stderr (the GET still runs).
# Named mutation OLA-http (drop the warning) -> OLA-http flips.
@test "OLA-http: an http:// base -> clear-text credential warning; https:// -> none" {
  _stub_curl
  OBIX_USER=reader OBIX_PASS=fake PATH="$bin:$PATH" run "$OLA" --map "$FX/wiring-map.md" \
    --obix http://station.example/obix --component /Plant/Facade
  [ "$status" -eq 1 ]
  [[ "$output" == *"obix-link-audit: WARNING: http:// base sends the oBIX credentials in clear text"* ]]
  OBIX_USER=reader OBIX_PASS=fake PATH="$bin:$PATH" run "$OLA" --map "$FX/wiring-map.md" \
    --obix https://station.example/obix --component /Plant/Facade
  [[ "$output" != *"WARNING"* ]]
}

@test "OLA-curlfail: curl failure (non-zero exit) -> GET failed, exit 3, no audit rows" {
  _stub_curl
  CURL_RC=22 OBIX_USER=reader OBIX_PASS=fake PATH="$bin:$PATH" run "$OLA" --map "$FX/wiring-map.md" \
    --obix https://station.example/obix --component /Plant/Facade
  [ "$status" -eq 3 ]
  [[ "$output" == *"obix-link-audit: GET failed: https://station.example/obix/config/Plant/Facade/"* ]]
  [[ "$output" != *"MATCH"* ]]
}

@test "OLA-insecure: --insecure forwards -k to curl; without it no -k" {
  _stub_curl
  OBIX_USER=reader OBIX_PASS=fake PATH="$bin:$PATH" run "$OLA" --map "$FX/wiring-map.md" \
    --obix https://station.example/obix --component /Plant/Facade --insecure
  grep -qx -- '-k' "$BATS_TEST_TMPDIR/curl.args"
  OBIX_USER=reader OBIX_PASS=fake PATH="$bin:$PATH" run "$OLA" --map "$FX/wiring-map.md" \
    --obix https://station.example/obix --component /Plant/Facade
  if grep -qx -- '-k' "$BATS_TEST_TMPDIR/curl.args"; then echo "-k sent without --insecure"; return 1; fi
}

@test "OLA-notable: a map with no rows in the selected table -> exit 3" {
  local m="$BATS_TEST_TMPDIR/t2only.md"
  sed '/^## Table 1/,/^## Table 2/{/^## Table 2/!d}' "$FX/wiring-map.md" > "$m"
  if grep -q '^## Table 1' "$m"; then echo "Table 1 still in $m"; return 1; fi
  run "$OLA" --map "$m" --xml "$FX/control.xml" --table 1
  [ "$status" -eq 3 ]
  [[ "$output" == *"obix-link-audit: no Table 1 rows in $m"* ]]
}
