#!/usr/bin/env bash
# lint-size.sh — advisory size smell: a Java class whose hand-written code (slotomatic regions excluded) exceeds
# ~800 lines, or a method of a PURE class (no Baja type) that spans more than ~80 lines.
#
# A BComponent that grows past ~800 hand-written lines, or a pure control method past ~80, is where mixed concerns
# hide: the decision-logic decomposition retro traced a live defect to a monolithic step() whose counts used
# different membership rules. Size is a REVIEW QUESTION for new modules, never a hard gate, so every row is WARN.
# Doctrine: types/logic.md § Composition & organization, METHODOLOGY.md § Conformance rules (size smell); deployed
# modules are not restructured to meet it (METHODOLOGY.md § Schema / upgrade safety).
# [ev: retro decision-logic-decomposition Δ2]
#
# Structural rules (decided up front; METHODOLOGY.md § Conformance rules, heuristic parsers):
#   - Hand-written lines = non-blank lines after comments are blanked (go/internal/javasrc Strip), outside
#     every slotomatic region. A region opens on a line containing `BEGIN BAJA AUTO GENERATED CODE` and closes on
#     the next line containing `END BAJA AUTO GENERATED CODE` (the marker text slot-o-matic emits; matched on the
#     RAW line because the marker sits inside a comment; a marker quoted in a string literal, which is still on the
#     comment-blanked line, is not one — truth fix T1 2026-10-24, the bash lint opened a region there).
#     [ev: corpus B711] [ev: code docDeveloper slot-o-matic.html:421,459]
#   - A region that opens and never closes cannot be classified: WARN `unclosed-region`, and its lines count as
#     hand-written (fail closed: never shrink the count on an unparsed shape). A BEGIN inside an open region is a
#     WARN `nested-region` (the earlier region counts as hand-written); an END with no open region is a WARN
#     `stray-end`.
#   - Pure class = a file with no `@NiagaraType` annotation and no `extends B<Upper>` declaration, both read in code
#     with string literal contents blanked (truth fix T2 2026-10-24: a quoted "@NiagaraType" made the bash lint skip
#     the method rule; the class name is read the same way). Only pure classes
#     get the method rule (the BComponent shell is measured by the class rule). Methods come from javasrc.Methods (a
#     signature whose `(...)` closes on the `{` line, or a `{` alone after it; a signature wrapped over several
#     lines before `{` is not seen — a false negative, never a false WARN).
#   - Module map modes (lint.ModuleMode: the nearest <module-root>/MODULE-MAP.md above the file): with NO map the
#     rules above apply unchanged. With a map the method rule covers every method and constructor of every class
#     (Baja classes too), measured from the line of its name to its closing brace by javasrc.Decls, so a wrapped
#     signature counts from its first line; the class and method rows WARN under `modularity: legacy` and FAIL under
#     `modularity: strict` (exit 1 without --strict). [ev: retro modularity-strict-new Δ4]
#   - adapter-branches (map modes only, always WARN): a Baja class that imports a model/ type (its pure peer) and
#     keeps more than 15 hand-written branch keywords (if, case, for, while, catch; slotomatic regions excluded).
#     [ev: retro modularity-gaps Δ5]
#   - One row per file for the class rule, named after the first `class`, `interface` or `enum` declaration (truth
#     fix T3 2026-10-24: the bash lint matched `class` only, so an interface printed `class ?` and an enum took its
#     nested class's name).
#
# Usage:  lint-size.sh [--strict] [--max-class <n>] [--max-method <n>] <src-root>
#   Defaults: --max-class 800, --max-method 80 (strictly greater than the limit WARNs).
#   Row:  WARN  lint-size  <file>:<line>  <detail>
#   Exit: 0  no WARN (or WARN without --strict) · 1  any FAIL (a strict module) or any WARN under --strict · 3  usage/env, an unscannable source file or a sub-directory find cannot enter
# Empty root: a root with no *.java is `ERROR  lint-size  <root>  no files to scan (K20: wrong path or empty
# scaffold)` on stdout, exit 3. A bad --max-class/--max-method value prints `lint-size: <flag> needs a
# non-negative integer` and the usage line, exit 3.
#
# Implementation: rule package go/internal/lint/rules/size on the shared engine go/internal/lint (walker,
# parser, row formatter, Spec.IntFlags for the two limits; go/README.md). This wrapper is the stable entry
# point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when it is missing or older than a Go source, then
# this script execs `n4lint lint-size <args>`. Needs `go` on PATH to build (exit 3 otherwise).
# VCS-free by design (kit-links L2).
# Mutation: LSZ2 -- counting the slotomatic region lines as hand-written WARNs a small class with a large generated block
# Mutation: LSZ4 -- applying the method rule to a Baja (BComponent) class WARNs its long callback
# Mutation: LSZ5 -- dropping the unclosed-region row lets an unterminated BEGIN marker pass silently
# Mutation: LSZ8 -- ignoring a nested BEGIN lets a stray END close an unterminated region and hide its lines
# Mutation: LSZ10 -- matching the markers on the raw line lets a string literal hide the lines after it
# Mutation: LSZ11 -- reading the Baja test in literal text skips the method rule of a pure class quoting @NiagaraType
# Mutation: LSZ12 -- matching only `class` names an enum after its nested class
# Mutation: LSZ-awkfail -- ignoring the awk exit status reports an unreadable source file as clean
# Mutation: LSZ-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-size: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-size "$@"
