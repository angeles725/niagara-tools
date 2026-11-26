#!/usr/bin/env bash
# signing-inventory.sh — read-only inventory of the signing material you have, so a developer with several Niagara
# installs and several signing profiles stops losing track of which certificates exist, where each lives, when each
# expires and which one a module must be signed with. [ev: issue #1160]
#
# Per signing profile: its keystore, then per alias the certificate subject and issuer, the days left (WARN under 60,
# FAIL expired), whether it is a code-signing certificate and not a CA (the signing-doctor.sh rules), and the file it
# lives in. A store that needs a password the profile does not hold is listed as locked, with its path only. It never
# prints a password or a key. With --module <root> it also prints the module's recorded expected_signer (BUILD-STATE),
# the signer and the compiled-for Niagara version of each built jar, and FAILs when the recorded alias is in no
# listed store (or only expired) or a built jar is signed by another.
#
# Implementation: the Go module at <repo>/go (go/internal/signinginventory); go/go-tool.sh builds go/bin/signing-inventory
# on demand, then this script execs it. Keystores are listed with the keytool of $JAVA_HOME, else the one on PATH.
#
# Usage:  signing-inventory.sh [--home <niagara_home>]... [--profile <profile.xml>]...
#                              [--module <module-root> [--module-name <MOD>] [--build-state <file>]]
# Output: one row per fact: PASS|WARN|FAIL|INFO, the check id (install, profile, store, store-locked, store-unreadable,
#         alias, validity, usage, ca, expected-signer, expected-signer-known, jar, jar-signer) and what it found.
# Exit:   0 (WARN rows allowed) · 1 a FAIL row · 2 usage or an unreadable input · 3 no Go toolchain
set -u
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'signing-inventory: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" signing-inventory)" || exit 3
SIGNING_INVENTORY_TOOLBELT="$here" exec "$BIN" "$@"
