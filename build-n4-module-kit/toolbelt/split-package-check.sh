#!/usr/bin/env bash
# split-package-check.sh — project-level check: one Java package declared in two N4 modules of a project.
#
# N4 is not OSGi. ModuleClassLoader.nfind() asks the parent loader, then the module's own jar, then its
# dependencies — a sticky last-hit cache, the deps that declare the package (in module.xml dependency order),
# then every other dep — and the FIRST hit wins. A same-named class in the same package of a second module is
# silently shadowed: no error, and behavior that changes with the dependency order. Rule: one package belongs to
# exactly one module; shared code goes in its own module listed as a dependency. See
# types/issues-and-gotchas.md §D4c. [ev: retro module-hardening-failure-modes-deltas Δ12] [ev: corpus B1125]
# [ev: code com/tridium/sys/module/ModuleClassLoader.java:186-294] [ev: code com/tridium/sys/module/NModule.java:333-335]
#
# Cross-module by nature, so it is not a per-module lint and report-module.sh (one module) does not run it. It runs
# as the client repository's gate (issue #142) — the push-time hook templates/client-pre-push (installed with
# scripts/install-client-hooks.sh) or the CI template templates/split-package-check.yml — and by hand before a
# release. Advisory by default (WARN, exit 0); --strict makes a split package exit 1 for a gate.
# <project-root> is ONE checkout: a directory that holds several checkouts or worktrees of the same project
# reports every copied package as a split.
#
# Structural rules:
#   Module  — a directory holding `module-include.xml`, `build.gradle.kts` or `build.gradle` AND a `src/` directory
#             (each -rt / -ux / -wb / -se artifact is its own N4 module). Dot-dirs, `build/`, `node_modules/`,
#             `src/` and `srcTest/` are pruned from the module search (sources are listed per module below).
#   Package — the `package <name>;` declaration of every *.java under the module's `src/` (srcTest/ is not a
#             module's runtime classpath). A file with no declaration is the default package, `(default)`.
#   Split   — the same package in two or more modules: one WARN row naming every module and one file in each.
#   Fail closed — a directory find cannot enter or a source file awk cannot read is exit 3, never a clean pass.
#
# Usage:  split-package-check.sh [--strict] <project-root>
#   Row:  WARN  split-package-check  <package>  <detail>
#   Exit: 0  no split package (or WARN without --strict) · 1  a split package under --strict · 3  usage/env, an unreadable file or directory, or no module found
# Implementation: RootRule go/internal/lint/rules/splitpackage on the shared engine go/internal/lint (Mode One
# parser, the fail-closed walk of each module's src/, BarePathRow rows; go/README.md). The build markers are found by
# name only: no XML is read. This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint
# when it is missing or older than a Go source, then this script execs `n4lint split-package-check <args>`. Needs
# `go` on PATH to build (exit 3 otherwise).
# Declared differences from the bash (port 2026-10-24): a missing root given with a trailing slash is named with it;
# the aggregation runs in process, so there is no external sort whose failure SPC-aggfail pinned.
# Truth fix 2026-10-24: `*-worktrees` directories are pruned from the module search (a sibling checkout's copies of
# every package were reported as splits; the root itself may be a worktree). Cliente 8 -> 4 rows.
# Truth fix 2026-10-24: the package declaration is read on code (comments blanked by the shared scanner): a
# `package` line inside a /* */ comment no longer wins over the real declaration. Real trees unchanged.
# VCS-free by design (kit-links L2).
# Mutation: SPC2 -- counting files instead of distinct modules WARNs a package spread over two files of ONE module
# Mutation: SPC4 -- scanning srcTest/ as well WARNs a test fixture that mirrors a runtime package
# Mutation: SPC-finderr -- ignoring the per-module listing status skips the unreadable directory and reports clean
# Mutation: SPC-findtree -- ignoring the module-search status skips a module under an unreadable directory
# Mutation: SPC-block -- reading a package line inside a /* */ comment takes a moved-from note over the real declaration
# Mutation: SPC-worktrees -- searching *-worktrees directories reports every package of a sibling checkout as a split
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'split-package-check: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" split-package-check "$@"
