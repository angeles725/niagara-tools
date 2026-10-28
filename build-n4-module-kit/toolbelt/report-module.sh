#!/usr/bin/env bash
# report-module.sh — aggregated conformance report for a Niagara N4 module (Campaign 8 PR8).
#
# Composes the campaign-6/8 toolbelt over every profile artifact under <module-root>.
# Invents no new checks — reuses each tool's own row text, prefixed by the artifact name.
# Per artifact (in order): verify-module.sh --src (SKIP if no built jar), slot-coverage.sh
# parse + dup-keys, lint-timers.sh <artifact>/src, --plano only when src/rc/index.html exists,
# lint-delays.sh <artifact>/src (SKIP if no src/), schema-risk.sh (SKIP if no .deploy-baseline/).
# Once per run: triage-console.sh --console-dir <dir> (SKIP row when --console-dir is absent).
# *-ux frontend members (WU4): rc-scan.sh receives --profile <ui_profile> and --legacy from this
# script's own flags (the caller reads the module's BUILD-STATE envelope; no envelope parsing here);
# ESLint with toolbelt/eslint.config.mjs on src/rc js (SKIP "no rc js" when only vendor/ext/min);
# lint-vendor-floor.sh on src/rc/vendor. A missing node/eslint/acorn is one SKIP row naming the
# tool (never a silent pass, never an env fault for the whole report).
# Tool resolution: KIT_ESLINT, else toolbelt/eslint/node_modules/.bin/eslint, else eslint on PATH
# (only at major >= 9, the flat-config major — an older PATH eslint is a SKIP row naming its version);
# node/acorn as in lint-vendor-floor.sh (KIT_NODE, KIT_ACORN). Install once:
# npm install --prefix build-n4-module-kit/toolbelt/eslint   (exact-pinned direct deps; no lockfile
# is committed — see toolbelt/eslint/.gitignore). A frontend member with WARN rows and no FAIL reports
# PASS "no failing row (N WARN row(s) above)", never "clean".
# [ev: retro dashboard-frontend-standard Δ10] [ev: retro dashboard-frontend-standard Δ16]
# Link endpoints (polish-2026-10-02 P1): lint-link-target-flags.sh <artifact>/src per artifact, with
# --wiring-map <file> when given, else <module-root>/docs/wiring-map.md, else <module-root>/../docs/
# wiring-map.md (the build.sh layout <repo>/<MOD>); a named --wiring-map file that is missing is exit 3.
# The default differs from commissioning-verify.sh (<module-root>/docs only) because the two tools take
# different roots: build.sh passes <repo>/<MOD> here, commissioning-verify takes <repo>; both resolve
# <repo>/docs/wiring-map.md. In a multi-module repo the parent fallback may be another module's map, so
# the chosen map is named in the row detail. Exit 3 from the lint is ERROR + exit 3 here (this script's
# env convention); commissioning-verify maps it to SKIP (its convention). A non-zero exit with no FAIL
# row is a FAIL row, never PASS.
# [ev: retro panccadia-commissioning-lessons Δ1] slot-coverage FAIL lines other than dup-keys (facade
# type without lexicon, empty lexicon) are FAIL rows, not a WARN percentage.
# [ev: retro panccadia-commissioning-lessons Δ9] ADVISORY rows (lint-silent-protection console-only)
# keep their own severity: counted apart, never PASS, never FAIL, never change the verdict.
# [ev: retro alarm-console-design Δ3]
# Advisory members (deferred-lints-2026-10-03): lint-size.sh, lint-license-isoperational-gate.sh,
# lint-set-null-ord.sh and lint-ws-placement.sh [ev: retro ws-placement] on <artifact>/src — their WARN rows are relayed with a PASS verdict row; an
# exit 1 with a FAIL row (lint-size in a `modularity: strict` module) relays its rows as FAIL with no PASS verdict
# [ev: retro modularity-strict-new Δ4]; any other non-zero exit of an advisory member is an ERROR row carrying the
# member's first non-row output line (exit 3), never PASS.
# lint-agent-on-include-sync.sh <module-root> once per run (FAIL, step 8b): a @NiagaraType @AgentOn type whose
# module-include.xml <type> line lacks one of its <on type> entries (types/module-wiring.md § 2).
# lint-slotomatic-markers.sh <artifact>/src (FAIL; a WARN unclosed-region row relayed): a generated region
# whose stamp count is not 1 is skipped by Slot-o-Matic while the build stays green (METHODOLOGY.md K15).
# Go-ported members (issue #459): lint-persist-hot-write, lint-changed-hot-write, lint-inert-coordination,
# lint-session-store-lazy-evict, lint-bformat-call (WARN), lint-service-types and lint-driver-scaffold (FAIL/WARN)
# on <artifact>/src; lint-wb-external-ord-value on a -wb src/; lint-spa-poll-no-recovery on a -ux src/rc/; once
# per run on <module-root>: lint-duplicate-helper, lint-machine-default-path, lint-inline-site-literal and the
# driver lints lint-private-api-import, lint-ndriver-static-collection, lint-ndriver-tagless-correlation (a module
# with no source they read is a SKIP row). lint-manual-labels is not a member: it needs an operator manual
# path, which a module root does not name; lint-bacnet-status-flags is not one either: it reads a station bog.
# Module map (modularity campaign M5): lint-module-map.sh <module-root> once per run after lint-structure (FAIL
# rows block, WARN rows such as MM1 "no MODULE-MAP.md" do not), then one `(module)  INFO  modularity  mode=<m>
# concerns=<n> unlisted=<n> largest=<path> (<n> hand-written lines)` row: the map mode (none with no valid map,
# `-` counts), its concern rows, the part sources in no concern and the largest part class by hand-written lines.
# INFO is printed and not counted: the row orients, it never changes the verdict.
# [ev: retro modularity-module-map Δ3] [ev: retro modularity-gaps Δ8]
#
# Usage: report-module.sh <module-root> [--target-version x.y] [--console-dir <dir>]
#                         [--profile hmi|lan|both|unknown] [--legacy] [--wiring-map <file>] [--first-deploy] [--lint-only]
# Row:     <artifact>  PASS|FAIL|WARN|SKIP|ADVISORY|ERROR|INFO  <check>  <detail>   (ERROR = member env fault, not counted; exit 3; INFO = the modularity row, not counted)
# Summary: report-module: N artifacts · p PASS · f FAIL · w WARN · s SKIP · a ADVISORY  ->  CLEAN|ISSUES|INCOMPLETE|ISSUES (+INCOMPLETE)
# Floor (BUILD-LOOP §5): verify-module (needs a built jar) and schema-risk (needs <artifact>/.deploy-baseline) never
# read CLEAN when they SKIP — the verdict is INCOMPLETE, exit 4. --first-deploy: the module was never deployed, so
# schema-risk has no baseline to diff (N/A, not a floor gap). --lint-only: a punch-list run that is not the floor
# (no jar expected); floor SKIPs stay rows and are named on stderr. [ev: retro change-tier-time-budgets Δ3]
# Mutation: RM-floor -- dropping the floor-SKIP verdict reads a run that checked neither floor member as CLEAN
# Mutation: RM-floor-schema -- --first-deploy ignored keeps a never-deployed module INCOMPLETE forever
# Mutation: RM-n4lint -- dropping a Go-ported member's invocation loses its rows from the report
# Mutation: RM-wp-env -- mapping every exit 3 to SKIP no write-path-matrix.md hides a broken write-path scan
# Exit: 0 clean (zero FAIL) · 1 any FAIL · 2 usage · 3 env (member env fault) · 4 INCOMPLETE (a floor check — verify-module, schema-risk — SKIPped; --first-deploy / --lint-only, see below)
# This script is VCS-free by design. version control is never invoked.
# kit-links.bats L2 enforces the no-version-control rule on all toolbelt scripts.
# [ev: retro campaign7-report-module]  [ev: retro campaign8-report-integration]
#
# Implementation: go/internal/reportmodule (command go/cmd/report-module; go/README.md). Every member runs through
# its wrapper in the toolbelt this script was invoked from (REPORT_MODULE_TOOLBELT, unresolved, so a stubbed
# toolbelt copy runs its stubs); the member table is go/internal/reportmodule/members.go, which gen-lint-index.sh
# reads for the INDEX.md Auto column. This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds
# go/bin/report-module when it is missing or older than a Go source, then this script execs it. Needs `go` on PATH
# to build (exit 3 otherwise). [ev: retro r10-report-module-go]
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'report-module: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" report-module)" || exit 3
REPORT_MODULE_TOOLBELT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export REPORT_MODULE_TOOLBELT
exec "$BIN" "$@"
