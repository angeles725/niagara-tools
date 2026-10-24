#!/usr/bin/env bash
# commissioning-verify.sh — §6.b commissioning punch-list auditor (issue #114).
#
# Orchestrates the Wave 3 lints and bog-audit checks into a single §6.b commissioning
# punch-list. Emits rows STATUS  commissioning  <check>  <detail>; exits 0 clean (no FAIL)
# or 1 any FAIL, 3 usage/env.
#
# Usage:
#   commissioning-verify.sh <module-root> [--bog <config.bog>] [--module <MOD>]
#                           [--values-owed <file>] [--wiring-map <file>] [--strict]
#
# Source checks (static, VCS-free, no station required):
#   config-sanity:<rt>  lint-config-sanity.sh per -rt/src dir (CS1/CS2 FAIL, CS3 WARN)
#   status-parity:<rt>  lint-status-parity.sh per -rt/src dir (WARN, or FAIL under --strict)
#   recovery-path:<rt>  lint-recovery-path.sh per -rt/src dir (FAIL)
#   link-target-flags:<rt>  lint-link-target-flags.sh per -rt/src dir (LTF1/LTF2 FAIL), with
#                       --wiring-map <file> (default <module-root>/docs/wiring-map.md when present) so the
#                       Table 2 link-in targets are known. [ev: retro panccadia-commissioning-lessons Δ1]
#                       The default differs from report-module.sh (which also tries <module-root>/../docs)
#                       because this tool takes the repo root while build.sh passes <repo>/<MOD> to
#                       report-module; both resolve <repo>/docs/wiring-map.md. The chosen map is named in
#                       the row detail. Exit 3 is SKIP here (this script's env convention for every source
#                       lint; report-module maps it to ERROR + exit 3). A non-zero exit with no FAIL row is
#                       a FAIL row, never PASS -- for every source lint above, not only this one (the bash tool
#                       PASSed config-sanity, status-parity and recovery-path on exit 126/127 or a crash).
#                       [ev: retro go-commissioning-verify]
#
# Station checks (require --bog <config.bog> and --module <MOD>):
#   proxy-link-safety        bog-audit.sh CHECK11
#   station-logic:*          bog-audit.sh CHECK13-19
#   numeric-to-bool-direct   bog-audit.sh CHECK20
#   (SKIP rows emitted when --bog is absent)
#   Each folded check relays its FAIL, WARN and PASS rows (the bash fold dropped WARN for CHECK13/15-20, so the
#   WARN-only CHECK15 sensor-crossed and CHECK19 link-direction never surfaced). [ev: retro go-commissioning-verify]
#
# Facade slot inventory note (PASS informational row):
#   wiring-map  generate-wiring-map.sh scaffolds docs/wiring-map.md
#
# Values owed by the field (BUILD-LOOP §6.b table; default <module-root>/docs/values-owed.md):
#   values-owed  one MANUAL row per table row whose Status cell does not start with
#                filled/provided (columns: Slot | Owed by | Unit | Safe default | Status); a row
#                with fewer cells is a MANUAL "malformed" row (fails closed); alignment rows skip;
#                one PASS row when every value is provided; one MANUAL row when no table exists.
#                An unreadable table is exit 3 before any row (the bash tool PASSed it). [ev: retro go-commissioning-verify]
#   [ev: retro panccadia-commissioning-lessons Δ11] [ev: retro panccadia-version-defect-ledger Δ4]
#
# Manual-only footer (MANUAL rows — informational, never FAIL):
#   hot-reload-console       triage-console.sh clean after station restart
#   plant-control            station actually controlling the plant
#   per-instance-values      per-instance runtime values match physical setpoints
#   persisted-state-restart  persisted operator state seeded/saved before the first restart, re-read after
#   alarm-routing            alarm class + recipient per source; a test alarm reaches the console, acked
#   consumer-impact          downstream consumers notified of every changed point meaning
#   link-source-audit        obix-link-audit.sh: each declared link-in comes from its declared source
#
# Row format: STATUS  commissioning  <check>  <detail>
# Summary:    commissioning-verify: P PASS · F FAIL · W WARN · S SKIP · M MANUAL  ->  CLEAN|ISSUES
# Exit:       0 clean (zero FAIL) · 1 any FAIL · 3 usage/env
# VCS-free by design (kit-links.bats L2). Named in BUILD-LOOP.md (kit-links.bats L5).
# [ev: retro live-commissioning-verification-gaps]
#
# Implementation: go/internal/commissioningverify (command go/cmd/commissioning-verify; go/README.md). The source
# lints run through their wrappers in the toolbelt this script was invoked from (COMMISSIONING_VERIFY_TOOLBELT,
# unresolved, so a stubbed toolbelt copy runs its stubs); bog-audit runs in process (go/internal/bogaudit). This
# wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/commissioning-verify when it is
# missing or older than a Go source, then this script execs it. Needs `go` on PATH to build (exit 3 otherwise).
# [ev: retro go-commissioning-verify]
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'commissioning-verify: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" commissioning-verify)" || exit 3
COMMISSIONING_VERIFY_TOOLBELT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export COMMISSIONING_VERIFY_TOOLBELT
exec "$BIN" "$@"
