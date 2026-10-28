#!/usr/bin/env bash
# verify-module.sh — THE gate for built N4 module jars. A jar that has not passed it does not go to a station.
#
# Checks (each guards a failure seen on a real build — build-n4-module-kit/retros/):
#   bytecode    every .class has major 52 (Java 8). 65 = built with the default JDK 21.          [default]
#               META-INF/versions/** (multi-release) is skipped and counted: Java 8 never loads it.
#   signed      a META-INF/<ALIAS>.SF with its <ALIAS>.RSA|.DSA|.EC block; the alias is reported. [default]
#               NOTE: `signed` checks PRESENCE only — the signer's cert may not be in the
#               station's trust store. A self-signed/untrusted-CA jar PASSES this gate yet a
#               station enforcing cert-chain validation loads it UNSIGNED (`Could not validate
#               cert chain`). Signed = jar has a *.SF block; trusted = station-side policy.
#               `[ev: corpus B800 §800.8]`
#               folded as code: verify-module.sh signed-check (cert-chain trust caveat, D8, R7.6).
#   types       every <type class=...> in META-INF/module.xml exists as a .class in the jar.     [default]
#   baja        module.xml baja vendorVersion <= --target-version (a 4.15 jar is rejected by 4.14). [--target-version]
#   stored      zero Deflated entries — required only when the jar must be re-signed in Workbench. [--stored]
#   typecount   packaged <type> count == <module-dir>/<jar-basename>/module-include.xml count.    [--src]
#   facets      no BFacets.make(BFacets.MIN|MAX, <raw number>) under <module-dir>/<jar-basename>/src. [--src]
#   facets-req  OPERATOR numeric slot without a facets key (WARN); setpoint/count-like slot without UNITS/PRECISION (WARN). [--src]
#               BFacets.makeNumeric(<unit>, ...) satisfies UNITS and any makeNumeric( satisfies PRECISION;
#               a precision-only makeNumeric(<int>) still WARNs missing UNITS, also when the precision is a
#               named constant (makeNumeric(PRECISION)). [ev: retro comppan-fase2-amps-alarms Δ1]
#               Mutation: VMN1 -- drop the makeNumeric(<unit>, ...) branch from the UNITS test -> VMN1 WARNs again
#               Mutation: VMN3 -- accept ANY makeNumeric( as a unit -> the precision-only makeNumeric(1) stops WARNing
#               Mutation: VMN4 -- accept a lone identifier argument as a unit -> makeNumeric(PRECISION) stops WARNing
#   ord-literal Java string literal matching station:|local:|slot:/ under src (WARN); exempt: *OrdConstants*+comment, defaultValue=, srcTest/**. [--src]
#   rcbackup    no editor/backup files (*~ *.orig *.bak*) packaged under rc/ — WARN, or FAIL under --strict. [default]
#   palette     a module that declares types must not ship an EMPTY module.palette (nothing to drag in
#               Workbench) — WARN, or FAIL under --strict; SKIP when the jar has no module.palette.     [default]
#               palette-type FAILs a t= that is not "<symbol>:<Type>" with an m=-declared symbol; palette-root
#               WARNs every baja Folder (root or nested, any alias); palette-parse FAILs an unterminated
#               comment. Entries: palette_scan below.
#   subscription-leak a TypeSubscriber subclass with no stopped() override — unsubscribeAll() is never
#               called; typeSubscriptionMap holds strong refs, leaking until station restart — WARN.    [--src]
#   slot-wall   more than 15 @NiagaraProperty on one class (flat slot wall) — WARN; FAIL in a strict  [--src]
#               module (modularity: strict in the nearest MODULE-MAP.md, lint.ModuleMode).
#
# Usage: verify-module.sh [--target-version X.Y] [--stored] [--src <module-dir>] [--strict] <jar>...
#   <module-dir> = the dir holding the profile dirs (e.g. .../Dashboard/DashboardPan); the profile is
#   derived from each jar's basename (DashboardPan-rt.jar -> DashboardPan-rt/).
# Report: one row per check per jar: PASS|FAIL|SKIP  <check>  <jar>  <detail>; a summary line closes it.
# Exit: 0 every executed check passed · 1 at least one check FAILED · 2 usage · 3 environment (tool missing, jar unreadable,
#   or a --src sub-directory the walk could not enter: the rows still print, the verdict is INCOMPLETE).
# Needs no JDK.
#
# Implementation: the Go module at <repo>/go (go/internal/verifymodule, over the shared lint.Walk and the
# jarversion zip reader; the usage text is lines 2-20 of the bash script it replaced, embedded as
# go/internal/verifymodule/usage.txt). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin`
# builds go/bin/verify-module when the binary is missing or older than a Go source, then this script execs
# it. Needs `go` on PATH to build; no Go toolchain or a failed build is exit 3 (its message starts with
# "verify-module:", never with a row status, since report-module.sh parses 2>&1 as rows).
# This script is VCS-free by design.
set -u

here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'verify-module: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" verify-module)" || exit 3
exec "$BIN" "$@"
