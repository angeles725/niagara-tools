#!/usr/bin/env bats
# lint-mqtt-tls-port.bats — a Generic MQTT authenticator over TLS that leaves the broker port at 1883 WARNs (G87).
# The rule's golden cases live in go/internal/lint/rules/mqtttlsport. [ev: retro doc-coverage-audit Δ16]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  L="$KIT/toolbelt/lint-mqtt-tls-port.sh"
  S="$BATS_TEST_TMPDIR/station"
  mkdir -p "$S"
}

# bog <authenticator element>...: a station file whose Services container holds the given elements.
bog() {
  printf '%s\n' "<bajaObjectGraph version='4.0'>" " <p m='b=baja' t='b:Station'>" \
    "  <p n='Services' m='mq=abstractMqttDriver' t='b:ServiceContainer'>" "$@" "  </p>" " </p>" "</bajaObjectGraph>" > "$S/config.bog"
}

@test "MT-usage: no argument prints the usage and exits 3" {
  run "$L"
  [ "$status" -eq 3 ]
  [ "$output" = "usage: lint-mqtt-tls-port.sh [--strict] <station-dir>..." ]
}

@test "MT1: a Generic authenticator on the default port WARNs; --strict exits 1; brokerPort 8883 is clean" {
  bog "   <p n='Broker' t='mq:GenericMqttAuthenticator'/>"
  run "$L" "$S"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-mqtt-tls-port  $S/config.bog:4  /Services/Broker TLS connection on port 1883: set brokerPort to the broker's TLS port, 8883 (types/cloud-connector.md § 9)" ]
  run "$L" --strict "$S"
  [ "$status" -eq 1 ]
  bog "   <p n='Broker' t='mq:GenericMqttAuthenticator'><p n='brokerPort' v='8883'/></p>"
  run "$L" --strict "$S"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "MT2: only the Generic authenticator is judged; an AWS JITP authenticator (port preset 8883) is clean" {
  bog "   <p n='Aws' t='mq:AwsJitpMqttAuthenticator'/>"
  run "$L" --strict "$S"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "MT-K20: a station directory with no *.bog is an ERROR row and exit 3, never a clean pass" {
  printf 'notes\n' > "$S/README.txt"
  run "$L" "$S"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-mqtt-tls-port  $S  no *.bog under the root (wrong path: pass a station directory)" ]
}
