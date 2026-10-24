#!/usr/bin/env bash
# lint-machine-default-path.sh — flags a default value in Java, rc JavaScript or .properties that is an absolute machine path (G43).
#
# A default that points at a developer home, a temporary directory, a WSL/Windows user directory or a
# scratchpad only works on the machine (or in the session) it was written on; on a JACE or another PC the
# module starts with a dead path. The value comes from a slot, a station-relative ORD (file:^...), an
# environment variable or the site table instead. Contract: the deferred G43 spec in
# ledger.d/2026-10-22-modular-line-efficient-quality.md. Doctrine: METHODOLOGY.md § rt "No machine- or
# session-specific default paths in source", types/issues-and-gotchas.md I3.
# [ev: retro modular-line-efficient-quality Δ5] [ev: retro go-lint-machine-default-path Δ1]
# Advisory WARN: exit 0, 1 under --strict.
#
# Files: *.java, *.js under an rc/ directory and *.properties under each module root; srcTest/ (test sources),
# build/ (Gradle output, unpacked jars), node_modules/, dot and *-worktrees directories are not entered.
# Default value (narrow on purpose, to keep false positives low):
#   *.java, a "..." literal on the shared scanner's blanked code (comment and Javadoc text never counts) that is
#     (a) an @NiagaraProperty defaultValue: the literals inside its expression (BString.make("...")),
#     (b) a field initializer: right after `<type> <name> =` on a line outside every method body,
#     (c) a fallback argument: after the `,` of getProperty(k, / getOrDefault(k, / requireNonNullElse(x, or orElse(;
#     the text before the literal may start up to three lines up. A local variable, a call argument, a comparison,
#     another annotation element and a text block are not defaults. Only the first literal of a concatenation is
#     read ("/home/" + user is a row; "/ho" + "me/x" is not).
#   rc *.js (go/internal/jssrc tokens, comments dropped), a string or a template with no ${...} that is a
#     `const|let|var X = ...` initializer (any depth: rc code lives in IIFE/AMD wrappers) or an object-literal
#     property default (`{ key: ... }`); an assignment, a call argument or a ternary branch is not.
#   *.properties (go/internal/props, java.util.Properties.load rules: continuations, # ! comments, escapes): every value.
#     Prescribed (R10-D5): in a gradle.properties, a machine path under niagara_home, niagara_user_home, nodeHome or
#     org.gradle.java.installations.paths is an INFO row (exit 0, also under --strict): the kit tells a module to copy these
#     from a sibling (BUILD-LOOP.md bring-up step 2, types/structure.md point 7). In a kit template fixture
#     (absolute path under build-n4-module-kit/fixtures/) the same line stays WARN (METHODOLOGY.md K16).
# Machine path: the value (\\ \" \' \` read, after an optional file: scheme), case-insensitive, starts with /home/,
# /Users/, /tmp (/ or end), /var/tmp (/ or end), /mnt/<drive>/Users/ or <drive>:\Users\ / <drive>:/Users/, or contains
# "scratchpad". Not rows: install roots (/opt/Niagara, C:\Niagara\Niagara-4.x), relative paths, and /home/niagara/
# (declared exclusion: the JACE platform's own account home, the same on every unit).
# Out of scope (retro go-lint-machine-default-path): station-name literals (need the G42 site table), a ternary
# getenv fallback, a path built from several literals.
#
# Implementation: rule package go/internal/lint/rules/machinedefaultpath on the shared engine go/internal/lint
# (walker, parser, row formatter; go/README.md). go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when it is
# missing or older than a Go source, then this script execs `n4lint lint-machine-default-path <args>`. Needs
# `go` on PATH to build (exit 3 otherwise). VCS-free by design.
#
# Usage:  lint-machine-default-path.sh [--strict] <module-root>...
#   Row:  WARN  lint-machine-default-path  <file>:<line>  machine-specific default "<path>" in <where> -- take it from a slot, a station-relative ORD, an environment variable or the site table (METHODOLOGY.md § rt)
#   Row:  INFO  lint-machine-default-path  <file>:<line>  <key> is a prescribed build-location key (BUILD-LOOP.md bring-up step 2; types/structure.md point 7)
#   Exit: 0  no row, or WARN rows without --strict · 1  a WARN row under --strict · 3  usage/env, an empty root, an unscannable source file or a sub-directory find cannot enter
# Empty root: a root with no *.java, *.js or *.properties under it is an env error (exit 3), never a clean pass;
# stdout gets `ERROR  lint-machine-default-path  <root>  no *.java or *.js or *.properties sources found (wrong path or empty source tree)`.
# Mutation: MDP1 -- dropping the field-initializer context passes static final String DIR = "/home/me/x" clean
# Mutation: MDP2 -- reading comment text flags a /home/ path quoted in a Javadoc
# Mutation: MDP3 -- dropping the method-body check flags a local variable in a method
# Mutation: MDP4 -- dropping the rc JavaScript pass passes const DIR = "/home/me/x" in src/rc clean
# Mutation: MDP5 -- dropping the .properties pass passes export_dir=C:\\Users\\... clean
# Mutation: MDP6 -- dropping the prescribed-key check WARNs niagara_user_home=C:\\Users\\... and exits 1 under --strict
# Mutation: MDP7 -- dropping the kit-fixture check reports a fixture's niagara_user_home as INFO
# Mutation: MDP-strict -- giving the rule a non-WARN severity exits 0 under --strict with a row
# Mutation: MDP-K20 -- dropping the engine empty-root guard reports an empty root as a clean exit 0
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-machine-default-path: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-machine-default-path "$@"
