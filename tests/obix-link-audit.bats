#!/usr/bin/env bats
# Pins for obix-link-audit.sh — declared wiring map vs the links actually present on a live
# component (read-only oBIX). A link from the WRONG-but-valid source is structurally fine, so
# bog-audit cannot see it; only a diff against the declared source closes that gap.
# Rows: MATCH | MISMATCH | MISSING | SKIP  obix-link-audit  <slot>  <detail>
# Exit: 0 no MISMATCH/MISSING · 1 any · 3 usage/env.
# Live-mode tests talk only to a python stand-in station on 127.0.0.1 (_station), never a real host.
# [ev: retro panccadia-commissioning-lessons Δ4]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  OLA="$KIT/toolbelt/obix-link-audit.sh"
  FX="$BATS_TEST_DIRNAME/fixtures/obix-link-audit"
  STATION_PID=""
}

teardown() { [ -z "$STATION_PID" ] || kill "$STATION_PID" 2>/dev/null || true; }

# _station [status] [tls]: start a local oBIX stand-in on 127.0.0.1 (never a real host) that answers every
# GET with status (default 200) and facade.xml, logging "METHOD path user:pass" per request to
# $BATS_TEST_TMPDIR/requests; with tls it serves https from a throwaway self-signed certificate. Sets PORT;
# skips without python3 (or openssl for tls).
_station() {
  local status="${1:-200}" tls="${2:-}" pf="$BATS_TEST_TMPDIR/port" cert="" _
  command -v python3 >/dev/null 2>&1 || skip "python3 required"
  if [ -n "$tls" ]; then
    command -v openssl >/dev/null 2>&1 || skip "openssl required"
    cert="$BATS_TEST_TMPDIR/station.pem"
    openssl req -x509 -newkey rsa:2048 -nodes -days 1 -subj /CN=localhost \
      -keyout "$cert" -out "$cert.crt" >/dev/null 2>&1 || skip "openssl could not make a certificate"
    cat "$cert.crt" >> "$cert"
  fi
  : > "$BATS_TEST_TMPDIR/requests"
  python3 - "$pf" "$BATS_TEST_TMPDIR/requests" "$FX/facade.xml" "$status" "$cert" >/dev/null 2>&1 3>&- <<'PY' &
import base64, http.server, os, ssl, sys
pf, log, body, status, cert = sys.argv[1:6]
class H(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        auth = self.headers.get("Authorization", "").split()
        cred = base64.b64decode(auth[1]).decode() if len(auth) == 2 else "-"
        with open(log, "a") as f:
            f.write(f"{self.command} {self.path} {cred}\n")
        data = open(body, "rb").read() if status == "200" else b""
        self.send_response(int(status))
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)
    def log_message(self, *a):
        pass
s = http.server.HTTPServer(("127.0.0.1", 0), H)
if cert:
    ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
    ctx.load_cert_chain(cert)
    s.socket = ctx.wrap_socket(s.socket, server_side=True)
with open(pf + ".tmp", "w") as f:
    f.write(str(s.server_port))
os.rename(pf + ".tmp", pf)
s.serve_forever()
PY
  STATION_PID=$!
  for _ in $(seq 1 100); do [ -s "$pf" ] && break; sleep 0.1; done
  [ -s "$pf" ] || skip "local station did not start"
  PORT=$(cat "$pf")
}

# _live <scheme> [args...]: run the tool in live mode against the local station, proxies off.
_live() {
  local scheme="$1"; shift
  run env -u http_proxy -u HTTP_PROXY -u https_proxy -u HTTPS_PROXY -u all_proxy -u ALL_PROXY NO_PROXY='*' \
    "$OLA" --map "$FX/wiring-map.md" --obix "$scheme://127.0.0.1:$PORT/obix" --component /Plant/Facade "$@"
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

@test "OLA5: live mode -> ONE read-only GET of <base>/config/<component>/, credentials only in the auth header" {
  _station
  OBIX_USER=reader OBIX_PASS=secret _live http
  [ "$status" -eq 1 ]
  [[ "$output" == *"MISMATCH  obix-link-audit  comp1State"* ]]
  [ "$(cat "$BATS_TEST_TMPDIR/requests")" = "GET /obix/config/Plant/Facade/ reader:secret" ]
  [[ "$output" != *"secret"* ]]
}

@test "OLA6: live mode without OBIX_USER/OBIX_PASS -> exit 3, no request made" {
  _station
  OBIX_USER='' OBIX_PASS='' _live http
  [ "$status" -eq 3 ]
  [[ "$output" == *"live mode needs OBIX_USER and OBIX_PASS"* ]]
  [ ! -s "$BATS_TEST_TMPDIR/requests" ]
}

@test "OLA7: a component with NO links -> every declared row MISSING (empty link set never reads the map as links)" {
  printf '<obj href="https://station.example/obix/config/Plant/Facade/"/>\n' > "$BATS_TEST_TMPDIR/empty.xml"
  run "$OLA" --map "$FX/wiring-map.md" --xml "$BATS_TEST_TMPDIR/empty.xml"
  [ "$status" -eq 1 ]
  [[ "$output" == *"0 MATCH · 0 MISMATCH · 4 MISSING · 1 SKIP  ->  ISSUES"* ]]
}

# The credentials reach the station intact: a password with `"`, `\`, a space and `#` is the one the
# station receives (the bash tool needed curl config quoting for this; the Go port sets the basic-auth
# header itself). Fake credentials only.
@test "OLA-esc: a password with a quote, backslash, space and # reaches the station intact" {
  _station
  OBIX_USER=reader OBIX_PASS='p"a\ss #1 x' _live http
  [ "$status" -eq 1 ] || { echo "$output"; return 1; }
  [[ "$output" == *"MISMATCH  obix-link-audit  comp1State"* ]]
  [ "$(cat "$BATS_TEST_TMPDIR/requests")" = 'GET /obix/config/Plant/Facade/ reader:p"a\ss #1 x' ]
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
  _station
  OBIX_USER=reader OBIX_PASS=fake _live http
  [ "$status" -eq 1 ]
  [[ "$output" == *"obix-link-audit: WARNING: http:// base sends the oBIX credentials in clear text"* ]]
  # https:// to the plain-http station: the handshake fails (exit 3) and nothing warns
  OBIX_USER=reader OBIX_PASS=fake _live https
  [ "$status" -eq 3 ]
  [[ "$output" != *"WARNING"* ]]
}

@test "OLA-getfail: the station answers 404 -> GET failed, exit 3, no audit rows" {
  _station 404
  OBIX_USER=reader OBIX_PASS=fake _live http
  [ "$status" -eq 3 ]
  [[ "$output" == *"obix-link-audit: GET failed: http://127.0.0.1:$PORT/obix/config/Plant/Facade/"* ]]
  [[ "$output" != *"MATCH"* ]]
}

# Named mutation OLA-insecure (verify the certificate always, or never) -> OLA-insecure flips.
@test "OLA-insecure: a self-signed station is refused without --insecure and audited with it" {
  _station 200 tls
  OBIX_USER=reader OBIX_PASS=fake _live https
  [ "$status" -eq 3 ]
  [[ "$output" == *"certificate"*"GET failed"* ]]
  [ ! -s "$BATS_TEST_TMPDIR/requests" ]
  OBIX_USER=reader OBIX_PASS=fake _live https --insecure
  [ "$status" -eq 1 ]
  [[ "$output" == *"MISMATCH  obix-link-audit  comp1State"* ]]
}

# A malformed response is exit 3 with its position, never a partial link set: the bash tool grepped the
# raw text and audited what it found (a truncated dump gave MISSING rows, a mis-nested one even a MATCH).
# Named mutation OLA-malformed (read a parse error as no links) -> OLA-malformed flips.
@test "OLA-malformed: a truncated or mis-nested component dump -> exit 3 path:line:col, no rows" {
  run "$OLA" --map "$FX/wiring-map.md" --xml "$FX/truncated.xml"
  [ "$status" -eq 3 ]
  [ "$output" = "obix-link-audit: $FX/truncated.xml:5:51: unexpected EOF" ]
  run "$OLA" --map "$FX/wiring-map.md" --xml "$FX/bad-close.xml"
  [ "$status" -eq 3 ]
  [ "$output" = "obix-link-audit: $FX/bad-close.xml:1:84: element <obj> opened at 1:1 closed by </obk>" ]
}

# The links are read from the XML: &#8594;, -&gt;, single quotes and spaces around = spell the same
# links (the bash grep read each of them as no link, so every row was a false MISSING).
@test "OLA-entities: link displays in any well-formed XML spelling are links" {
  run "$OLA" --map "$FX/wiring-map.md" --xml "$FX/entities.xml"
  [ "$status" -eq 0 ]
  [[ "$output" == *"4 MATCH · 0 MISMATCH · 0 MISSING · 1 SKIP  ->  CLEAN"* ]]
}

@test "OLA-notable: a map with no rows in the selected table -> exit 3" {
  local m="$BATS_TEST_TMPDIR/t2only.md"
  sed '/^## Table 1/,/^## Table 2/{/^## Table 2/!d}' "$FX/wiring-map.md" > "$m"
  if grep -q '^## Table 1' "$m"; then echo "Table 1 still in $m"; return 1; fi
  run "$OLA" --map "$m" --xml "$FX/control.xml" --table 1
  [ "$status" -eq 3 ]
  [[ "$output" == *"obix-link-audit: no Table 1 rows in $m"* ]]
}
