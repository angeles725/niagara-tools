#!/usr/bin/env bash
# lint-status-parity.sh — Per-instance config/status slot ratio gate (Wave 3, LR2, B861).
#
# Detects facade classes where the count of per-instance config slots
# (*Interval*, *Duration*, *Setpoint*; N>1) exceeds the count of scalar status
# slots (*Status, *Active, *Start, *Since; M). When N>1 and M<N this is an
# asymmetric facade: operators cannot tell which of the N configured instances
# is currently active/running.
#
# Default:     WARN (exit 0) — name-heuristic false positives are real for
#              legitimately aggregated rollup facades; accept WARN as advisory.
# With --strict: FAIL (exit 1).
#
# Usage:  lint-status-parity.sh [--strict] <java-src-dir>
#
#   Scans all *.java recursively under <java-src-dir> (only the first argument is read).
#   Emits WARN rows (or FAIL under --strict); exits 0 WARN-only, 1 under --strict, 3 usage/env.
#
# Row format:  WARN|FAIL  lint-status-parity  <class>  N=<n> per-instance config slots but <m> status slots
#   <class> is the file's basename without .java: one row per file, every class in it counted together.
# Exits:       0 clean or WARN-only · 1 any WARN under --strict · 3 usage/env, an unscannable file, an unreadable sub-directory or no *.java (K20)
# Empty root:  `ERROR  lint-status-parity  <root>  no files to scan (K20: wrong path or empty scaffold)` on stdout, exit 3.
#
# Dot-directories excluded (D9b). VCS-free by design. kit-links.bats L2 enforces
# the no-version-control rule on all toolbelt scripts.
# [ev: retro live-commissioning-verification-gaps]
#
# Documented limitation (name-heuristic):
#   Legitimately aggregated rollup facades may trigger a WARN when they intentionally
#   have more config slots than status slots (e.g. a master facade aggregating N rooms
#   with one global active flag). Use --strict only after auditing the WARN list.
#   Source of slot names: the first name="..." of each @NiagaraProperty annotation, read
#   to its balanced ")" across lines; module-include.xml facets are NOT consulted for this check.
#
# Implementation: rule package go/internal/lint/rules/statusparity on the shared engine go/internal/lint
# (walker, parser, row formatter; go/README.md). This wrapper is the stable entry point: go/go-tool.sh
# `go_tool_bin` builds go/bin/n4lint when it is missing or older than a Go source, then this script execs
# `n4lint lint-status-parity <args>`. Needs `go` on PATH to build (exit 3 otherwise).
# Declared differences from the bash lint (port 2026-10-24): a listed file the scan cannot read is
# `lint-status-parity: cannot scan <path>: ...` on stderr and exit 3 (gawk printed its fatal and the lint
# exited 0), and a file name holding a blank keeps its whole class name (the bash `read` split it into the
# N and M columns).
# Truth fix T1 (2026-10-24): a one-line @NiagaraProperty closes on its own line (javasrc.NiagaraProperties,
# shared with lint-config-sanity); the bash awk always took the next line too, so every other annotation of
# a run of one-line ones went uncounted.
#
# Mutation: LSP-asymmetric -- removes N>M asymmetry detection so all facades pass regardless of parity
# Mutation: LSP-oneline -- closing an annotation only from its second line drops every other one-line slot
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-status-parity: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-status-parity "$@"
