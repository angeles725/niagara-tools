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
# lint-vendor-floor.sh on src/rc/vendor, then lint-vendored.sh on src/rc (issue #1145): WARN rows for a vendored
# file at a version the kit's known-defects table lists, never a verdict change; an rc dir without vendor/ gets
# no row. A missing node/eslint/acorn is one SKIP row naming the
# tool (never a silent pass, never an env fault for the whole report).
# Tool resolution: KIT_ESLINT, else toolbelt/eslint/node_modules/.bin/eslint, else eslint on PATH
# (only at major >= 9, the flat-config major — an older PATH eslint is a SKIP row naming its version);
# node/acorn as in lint-vendor-floor.sh (KIT_NODE, KIT_ACORN). Install once:
# npm install --prefix build-n4-module-kit/toolbelt/eslint   (exact-pinned direct deps; no lockfile
# is committed — see toolbelt/eslint/.gitignore). A frontend member with WARN rows and no FAIL reports
# PASS "no failing row (N WARN row(s) above)", never "clean".
# [ev: retro dashboard-frontend-standard Δ10] [ev: retro dashboard-frontend-standard Δ16]
# JS tests (issue #1124): per *-ux artifact with a srcTest/, row `js-tests` runs `node --test <file>...` in the
# artifact directory over its srcTest/**/*.test.js and *.test.cjs files (listed by the tool in byte order, none under
# node_modules/ or a dot directory; `node --test <dir>` fails on Node 22). PASS `node --test: N files passed`; FAIL
# `node --test exited <n> (N files): <last lines of its output>`, and FAIL when the run passes KIT_JS_TESTS_TIMEOUT
# seconds (default 600); SKIP when node is not on PATH (KIT_NODE names another) or srcTest/ holds no such file. An
# artifact with no srcTest/ gets no row. Native, no toolbelt script (types/frontend-standard.md § 11).
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
# exit 1 with a FAIL row from the mode-aware member (lint-size, in a `modularity: strict` module; no other advisory
# member) relays its rows as FAIL with no PASS verdict
# [ev: retro modularity-strict-new Δ4]; any other non-zero exit of an advisory member is an ERROR row carrying the
# member's first non-row output line (exit 3), never PASS.
# lint-agent-on-include-sync.sh <module-root> once per run (FAIL, step 8b): a @NiagaraType @AgentOn type whose
# module-include.xml <type> line lacks one of its <on type> entries (types/module-wiring.md § 2).
# lint-slotomatic-markers.sh <artifact>/src (FAIL; a WARN unclosed-region row relayed): a generated region
# whose stamp count is not 1 is skipped by Slot-o-Matic while the build stays green (METHODOLOGY.md K15).
# lint-instance-type-order.sh <artifact>/src (FAIL): a @NiagaraType class that declares a static INSTANCE after its
# static TYPE is refused by the station ("INSTANCE field is null; insure getType() comes last"), issue #847.
# Go-ported members (issue #459): lint-persist-hot-write, lint-changed-hot-write, lint-inert-coordination,
# lint-session-store-lazy-evict, lint-bformat-call (WARN), lint-service-types and lint-driver-scaffold (FAIL/WARN)
# on <artifact>/src; lint-wb-external-ord-value on a -wb src/; lint-spa-poll-no-recovery on a -ux src/rc/; once
# per run on <module-root>: lint-duplicate-helper, lint-machine-default-path, lint-inline-site-literal and the
# driver lints lint-private-api-import, lint-ndriver-static-collection, lint-ndriver-tagless-correlation, then
# lint-vacuous-tests (FAIL rows block; kit backlog B6) (a module with no source they read, for lint-vacuous-tests no
# srcTest/ *.java, is a SKIP row). lint-manual-labels is not a member: it needs an operator manual
# path, which a module root does not name; lint-bacnet-status-flags is not one either: it reads a station bog.
# Module map (modularity campaign M5): lint-module-map.sh <module-root> once per run after lint-structure (FAIL
# rows block, WARN rows such as MM1 "no MODULE-MAP.md" do not), then one `(module)  INFO  modularity  mode=<m>
# concerns=<n> unlisted=<n> largest=<path> (<n> hand-written lines)` row: the mode the per-file rules apply
# (lint.ModuleMode: the nearest map's header, so a strict header over a broken table still reads strict; none with
# no map or an invalid header), the root map's concern rows and part sources in no concern (`-` with no valid map
# at the root) and the largest part class by hand-written lines over the same files lint-module-map walks. With a
# map at the root, every value but the mode is `?` when the root has no part or the map or a part could not be
# read; with none, an unreadable part makes `largest` `?` (lint-module-map's own row reports the fault, exit 3). INFO is printed and not counted: the row orients, it never changes the verdict.
# [ev: retro modularity-module-map Δ3] [ev: retro modularity-gaps Δ8]
#
# Usage: report-module.sh <module-root> [--target-version x.y] [--console-dir <dir>]
#                         [--profile hmi|lan|both|unknown] [--legacy] [--wiring-map <file>] [--niagara-home <dir>]
#                         [--first-deploy] [--lint-only] [--write-baseline] [--no-baseline] [--feature-doc <file>]
#        report-module.sh <module-root> --ready [--niagara-home <dir>] [--expect-niagara <ver>] [--expect-signer <alias>]
#                         [--target-version x.y] [--first-deploy] [--json]
# <module-root>: the MODULE directory, <gradle root>/<MOD>, the one holding <MOD>-rt/ -ux/ -wb/ (build.sh's <module-root> is the
# gradle root above it and passes <gradle root>/<MOD> here). Given a gradle root it exits 3 naming the directory to pass.
# [ev: retro build-kit-bench-session Δ10]
# --niagara-home <dir>: the installation lint-ntpl-manifest-deps reads the stock modules from (build.sh passes its
# niagara_home); without it that member reads NIAGARA_HOME from the environment, and with neither its ntpl-deps
# rows are WARN. [ev: retro ntpl-lint-walk-k-rows Δ4]
# Row:     <artifact>  PASS|FAIL|WARN|SKIP|ADVISORY|ERROR|INFO  <check>  <detail>   (ERROR = member env fault, not counted; exit 3; INFO = the modularity row, not counted)
# Summary: report-module: N artifacts · p PASS · f FAIL · w WARN · s SKIP · a ADVISORY  ->  <verdict>
# Verdict: CLEAN | ISSUES | INCOMPLETE (<cause>) | ISSUES (+INCOMPLETE (<cause>)). <cause> says what left the run
# incomplete, so a skipped module check never hides among host faults: `environment` = a member that could not run
# (an ERROR row, exit 3: a missing tool, node, acorn, a JDK); `module` = a skipped floor check (below); both are
# listed when both hold, `INCOMPLETE (environment, module)`. The cause is text only: the exit codes do not change.
# [ev: issue 1116]
# Floor (BUILD-LOOP §5): verify-module (needs a built jar) and schema-risk (needs <artifact>/.deploy-baseline) never
# read CLEAN when they SKIP — the verdict is INCOMPLETE, exit 4. --first-deploy: the module was never deployed, so
# schema-risk has no baseline to diff (N/A, not a floor gap). --lint-only: a punch-list run that is not the floor
# (no jar expected); floor SKIPs stay rows and are named on stderr. [ev: retro change-tier-time-budgets Δ3]
# A member env fault (an ERROR row, exit 3) with no FAIL reads INCOMPLETE (environment), never CLEAN (kit-backlog
# run-3 fix); the
# row carries the member's own first reason line (the missing tool), and the other members' rows stay a usable result.
# FAIL baseline: <module-root>/.report-baseline (committed with the module) lists the FAIL rows the module already
# had, one key per line: `<artifact>  <check>  <detail>`, the row without its status; in <detail> a path is cut to
# its file name and a number standing alone (a line number, a count) reads N, so a key survives unrelated edits
# (rows that differ only there share a key); "#" and blank lines are ignored. With the file, the FAIL rows are
# printed after the other rows in two groups, `report-module: NEW FAIL rows (...)` and `report-module: BASE FAIL
# rows (...)`, the summary reads `f FAIL (n new, b base in <file>)`, and the verdict and the exit code read the NEW
# rows only, so a change that adds no FAIL exits 0 and build.sh hands off. A line no FAIL row matches is a
# `(module)  WARN  report-baseline  stale baseline line ...` row (no exit change). Never baselined, always NEW:
# the verify-module gate rows and schema-risk rows (the floor), the row of a member that "exited N with no FAIL
# row" (a crash, floor member or not) -- a listed one of either is a WARN row -- and triage-console lines; a SKIPped floor check still reads INCOMPLETE. --write-baseline writes the file from the run's FAIL rows
# (sorted, de-duplicated, atomic; floor and crash rows left out and named on stderr) and exits as the run does;
# --no-baseline ignores the file for one run. A file that exists and cannot be read is exit 3. Without the file
# and without these options the report is unchanged. (source: delta B7 of an external session retro, 2026-11-13)
# --ready (retro row P4): the ready-to-load report of each built jar instead of the punch-list: module, niagara-target,
# niagara-home, signer, verify-module, schema-risk, sha256 and mirror rows, then READY TO LOAD with the post-load
# read-back, or NOT READY with the failing rows (--expect-niagara, --expect-signer, --json; build-verify.md
# § Ready to load). Exit 0 ready, 1 not ready, 3 env. [ev: retro ready-to-load-p4 Δ1]
# --feature-doc <file> (issue #1127): reads the feature document of the change and prints one `(module)  ADVISORY
# feature-doc` row per skill step it lacks, in this order: a tier line (the word tier then P0-P3, or a line that
# starts with the tier, before the first `## ` heading; BUILD-LOOP §0.c), a `## Behavior decisions` heading with
# content (a table row with a cell besides its id filled, or "none"; §1) and a `## Checks skipped` heading (its text
# may be "none"; §5); a file that cannot be read is one row saying so. The rows are counted as ADVISORY and never
# change the exit code or the verdict. Without the option nothing is read: no candidate document is guessed, and
# --ready refuses it. skill/SKILL.md § Steps no tool checks lists what the rows do not cover.
# Mutation: RM-baseline-floor -- dropping `floor: true` of a floor member lets a baseline hide a verify-module or schema-risk FAIL
# Mutation: RM-baseline-crash -- dropping the crashRow case lets a baseline hide a member that crashed without a FAIL row
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
