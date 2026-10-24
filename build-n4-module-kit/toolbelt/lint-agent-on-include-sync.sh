#!/usr/bin/env bash
# lint-agent-on-include-sync.sh — FAILs a @NiagaraType @AgentOn type whose module-include.xml line lacks its <on type>.
#
# javac's NiagaraTypeAnnotationProcessor never rewrites a module-include.xml <type> line whose existing <agent>
# elements all match, and a line with no <agent> matches any annotation (ModuleInclude.entryMatches, nre 4.14.0.162
# vineflower javax/baja/nre/annotations/processors/ModuleInclude.java:376-415): an @AgentOn added to a type that
# already has a line is silently dropped and the agent never shows in Workbench (seen live on 4.15.6.20 with the
# course Bonus Lab BTrafficLightDriverPointManager). For every *.java under a part's src/ with
# @NiagaraType(agent = @AgentOn(types = ...)) (one string or an array, one @AgentOn or several), the
# <type class="<fqcn>"> line of the part's module-include.xml must carry an <agent><on type="..."/> for each
# declared type. Fix: delete the type's line (or the file) and rebuild (types/module-wiring.md § 2).
# Java is read as tokens (go/internal/javasrc): an annotation in a comment or a string is not one; a types value
# that is not a string literal is not checked. module-include.xml is read as XML (go/internal/xmlsrc). The part is
# the deepest directory holding a module-include.xml whose src/ holds the file; srcTest/ and build/ are not read.
# lint-agent-on-shape.sh checks the module:Type shape of each <on type>; this lint checks source against include.
# [ev: retro n4-course-handson-slotomatic-authoring Δ5]
#
# Implementation: rule package go/internal/lint/rules/agentonincludesync on the shared engine go/internal/lint
# (go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when
# it is missing or older than a Go source, then this script execs `n4lint lint-agent-on-include-sync <args>`.
# Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-agent-on-include-sync.sh [--] <module-root>...
#   Row:  FAIL  lint-agent-on-include-sync  <file>:<line>  <fqcn>: module-include.xml <type> line lacks <on type> for: <types>
#   Row:  FAIL  lint-agent-on-include-sync  <file>:<line>  <fqcn>: no <type class="<fqcn>"> line in <part>/module-include.xml (needs ...)
#   Row:  FAIL  lint-agent-on-include-sync  <file>:<line>  <fqcn>: no module-include.xml beside its src/ (needs ...)
#   Exit: 0  clean · 1  any FAIL · 3  usage/env, a root with no module-include.xml, a malformed one or a sub-directory find cannot enter
# Mutation: AOIS1 -- skipping a <type> line with no <agent> passes the Bonus Lab tree with its agent block removed
# Mutation: AOIS2 -- reading @AgentOn on raw lines (not tokens) FAILs an @AgentOn inside a comment or a string
# VCS-free by design.
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-agent-on-include-sync: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-agent-on-include-sync "$@"
