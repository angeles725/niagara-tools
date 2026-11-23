#!/usr/bin/env bash
# signing-doctor.sh — checks the signing material a module is re-signed with, BEFORE the build: the exported
# certificate (PEM or DER) must be a code-signing one, not the CA that issued it (CA:TRUE is a FAIL), carry the
# code-signing usage and be in date; the private key export must be in a form OpenSSL 3 can read (the Workbench
# export, PBES2 with AES-GCM, is not: a FAIL that sends the conversion to Java). [ev: retro mcpn4-history-batch-tools Δ5]
#
# Implementation: the Go module at <repo>/go (go/internal/signingdoctor); go/go-tool.sh builds go/bin/signing-doctor
# on demand, then this script execs it.
#
# Usage:  signing-doctor.sh [--cert <certificate file>] [--key <private key export file>]   (at least one)
# Output: one row per check: PASS|WARN|FAIL, the check id (cert-parse, cert-ca, cert-usage, cert-validity,
#         key-format) and what it found.
# Exit:   0 pass (WARN rows allowed) · 1 a FAIL row · 2 usage or an unreadable file · 3 no Go toolchain
set -u
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'signing-doctor: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" signing-doctor)" || exit 3
exec "$BIN" "$@"
