#!/usr/bin/env bash
# lint-mqtt-tls-port.sh — flags a Generic MQTT authenticator in a station file that connects over TLS and leaves the broker port at 1883 (G87).
#
# The Generic authenticator defaults to connectionType=UserLoginOverSSL, but brokerPort defaults to 1883
# (javap of abstractMqttDriver-rt 4.15.6.20, BAbstractMqttAuthenticator static init: sipush 1883; guide
# abstractMqttDriver-GenericMqttAuthenticator.txt:19). A TLS broker listens on 8883, so the connection fails
# against a TLS listener. Evidence and the fix: types/cloud-connector.md § 9. Contract: the G87 spec in
# ledger.d/2026-10-23-r8-u9-c68-c71.md. [ev: retro devguide415-r2-drivers-protocols Δ2] [ev: retro doc-coverage-audit Δ16]
# Advisory WARN: exit 0, 1 under --strict.
#
# Files: every *.bog under each station directory (a saved config.bog is a zip: its file.xml member is read).
# An authenticator is a `<p>` whose t is GenericMqttAuthenticator of the abstractMqttDriver module (the prefix is
# resolved through the m="prefix=module" declarations). It WARNs when its connectionType slot is absent (the
# default UserLoginOverSSL) or UserLoginOverSSL or AnonymousOverSSL, AND its brokerPort slot is absent (the
# default 1883) or 1883. brokerPort 8883, connectionType Anonymous and every other authenticator type (the AWS
# JITP one presets 8883) are clean. The enum is read as its tag as written, a "@range" suffix dropped; an enum
# stored in another form is not recognised (not checked against a saved station: an [INFER] reading).
# Declared difference from the spec: the check column is the lint ID (kit lint-<name> convention), the row names
# the slot path and the fix. Not in n4lint --all (station directories, Spec.ModuleRoot) and not run by report-module.
#
# Implementation: rule package go/internal/lint/rules/mqtttlsport on the shared engine go/internal/lint (walker,
# row formatter; go/README.md) and go/internal/xmlsrc (the bog reader). go/go-tool.sh `go_tool_bin` builds
# go/bin/n4lint when it is missing or older than a Go source, then this script execs `n4lint lint-mqtt-tls-port
# <args>`. Needs `go` on PATH to build (exit 3 otherwise). VCS-free by design.
#
# Usage:  lint-mqtt-tls-port.sh [--strict] <station-dir>...
#   Row:  WARN  lint-mqtt-tls-port  <bog>:<line>  <authenticator slot path> TLS connection on port 1883: set brokerPort to the broker's TLS port, 8883 (types/cloud-connector.md § 9)
#   Exit: 0  no row, or WARN rows without --strict · 1  a WARN row under --strict · 3  usage/env, no *.bog under a root, or a malformed or unreadable bog
# Empty root: a root with no *.bog is an env error (exit 3), never a clean pass; stdout gets
# `ERROR  lint-mqtt-tls-port  <root>  no *.bog under the root (wrong path: pass a station directory)`.
# Mutation: MT1 -- dropping the absent-brokerPort default passes a Generic authenticator that sets neither slot
# Mutation: MT2 -- reading every authenticator type, not only GenericMqttAuthenticator, WARNs an AWS JITP authenticator
# Mutation: MT-K20 -- dropping the Spec.EmptyError message replaces the ERROR row with the engine default
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-mqtt-tls-port: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-mqtt-tls-port "$@"
