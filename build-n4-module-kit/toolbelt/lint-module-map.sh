#!/usr/bin/env bash
# lint-module-map.sh — a module root's MODULE-MAP.md must name every source file's concern and each concern's tests.
#
# The map (types/structure.md § Module map) says which concern owns which files, which tests cover it and when
# a change belongs there; this lint checks it against the module tree. `modularity: strict` (a scaffolded, new
# module) turns MM4 unlisted files and MM5 untested concerns into FAIL rows; `modularity: legacy` (an existing
# module, with a legacy-reason: line) keeps them WARN. Java files are listed one per entry; src/rc front-end
# files may be listed by a glob (`**` any depth, `*` inside one segment); a glob never lists a .java file.
#   MM1  no MODULE-MAP.md at the root (WARN; nothing else runs)
#   MM2  no valid modularity: header, or no Concern | Part | Files | Tests | Change here when table (FAIL; nothing else runs)
#   MM3  a listed path does not exist, is a directory, is absolute or leaves the root, or a glob matches no file (FAIL)
#   MM4  a part's src/**/*.java or src/rc/**/*.{js,html,css} (minus vendor/, ext/, *.min.*) in no row's Files
#        (FAIL strict, WARN legacy) or in two rows' Files (FAIL)
#   MM5  a non-plumbing concern names no existing srcTest/ path in Tests (FAIL strict, WARN legacy)
#   MM6  a part's srcTest/**/*.java listed in no row (WARN)
#   MM7  a concern folder (the directory right under a role folder: rt model/ or adapter/; ux route/, service/
#        or model/; wb and se model/) whose files two rows list, or a non-plumbing row listing files in concern
#        folders of two names (rt model/battery and adapter/battery are one concern) (FAIL strict, WARN legacy)
# [ev: retro modularity-module-map Δ3] [ev: retro modularity-structure Δ2]
#
# Usage:  lint-module-map.sh <module-root>   (the directory holding the <MOD>-rt/-ux/-wb/-se parts)
#   Row:  FAIL|WARN  lint-module-map  <path>[:<line>]  MM<n>: <reason>
#   Exit: 0  clean or WARN-only · 1  any FAIL · 3  usage, a root with no -rt/-ux/-wb/-se part (ERROR row), an
#         unreadable directory under a part, or an unreadable map file
# Implementation: rule package go/internal/lint/rules/modulemap on the shared engine go/internal/lint (walker,
# parser, row formatter; go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds
# go/bin/n4lint when it is missing or older than a Go source, then this script execs `n4lint lint-module-map <args>`.
# Needs `go` on PATH to build (exit 3 otherwise).
# VCS-free by design.
# Mutation: MMB1 -- reporting an unlisted file at WARN in a strict module lets a scaffolded module grow unmapped files
# Mutation: MMB2 -- dropping the MM3 existence check passes a map that names a deleted file
# Mutation: MMB3 -- letting a glob list a .java file hides a Java file inside a directory entry
# Mutation: MMB4 -- dropping the MM7 concern-folder check lets one strict row span two concern folders
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-module-map: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-module-map "$@"
