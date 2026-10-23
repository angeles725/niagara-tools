#!/usr/bin/env bash
# lint-structure.sh — module-structure lint for Niagara N4 source trees (Campaign 8 PR18, B817).
#
# Discovers profiles as every direct child dir of <module-root> that is named
# *-(rt|ux|wb|se|doc), or directly contains build.gradle.kts / build.gradle /
# module-include.xml.  Dot-directories are excluded (D9b).  A profile without
# module-include.xml gets L6 (missing) and still runs L9 and all checks that do
# not need the include file.  Source-tree-only; L8 (signed-jar check) stays in
# verify-module.sh (D15).
#
# Checks:
#   L1  FAIL  package naming — every .java package is com.<vendor>.*, never javax.baja.*
#   L2  FAIL  one @NiagaraType per .java file (max one public type per file)
#   L3  WARN  pure-model package (src/.../model/) has javax.baja.* imports (advisory)
#   L4  FAIL  module.lexicon non-empty (>=1 key=value) when >=1 type is declared
#   L5  FAIL  module.palette non-empty for -rt profile
#   L6  FAIL  module-include.xml present; no hand-authored META-INF/module.xml in source
#   L7  FAIL  dependency version is a 3-part floor (4.14.0), not 2-part (4.14)
#   L9  FAIL  no empty skeleton artifact: -wb/-ux with 0 Java classes AND empty palette
#   L10 FAIL  no absolute host paths in gradle.properties (module root + upward to .git root)
#   L11 FAIL  mixed srcTest (BTest+JUnit) without both :test-wb AND junit gradle declarations
#   L12 FAIL  org.gradle.java.installations.paths/.auto-detect absent or commented in gradle.properties
#   L13 FAIL  >=2 sibling gradle.properties under group dir with divergent niagara_home=
#   L14 FAIL  settings.gradle.kts (module root, else nearest ancestor up to the .git sentinel) hardcodes
#             gradlePluginVersion as a string literal instead of
#             providers.gradleProperty("niagaraPluginVersion").getOrElse(...) — -PniagaraPluginVersion
#             (build.sh --plugin-version) is then silently ignored [ev: retro panccadia-restart-seq-comp-lockout-hours Δ2]
#   L15 WARN  the ROOT build.gradle.kts beside that settings.gradle.kts applies com.tridium.niagara-signing
#             without the signing block (uncommented allowDefaultProfile.set(false) or = false, niagaraSigning {,
#             aliases, signingProfileFile) — a missing profile then signs with the self-signed default key (#266)
#   (L8 signed-jar check is in verify-module.sh)
#
# Usage:  lint-structure.sh <module-root>
#
#   Row format:  FAIL|WARN  lint-structure  <path>  L<n>: <reason>
#   Exits:       0  no FAIL (WARN-only is still 0) · 1  any FAIL · 3  usage/env (K20) or an unreadable file/dir
#
# A file or directory it cannot read prints `lint-structure: cannot scan|list ...` on stderr (the rows still
# print); an unreadable root build.gradle.kts stays the visible L15 WARN. Sub-directories named .* or
# *-worktrees are never entered. A root without profiles gets the module-level checks only.
#
# VCS-free by design; version control is never invoked. The .git directory is used
# only as a filesystem sentinel (stop-marker) for the L10 and L14 upward walks; no VCS
# commands are ever executed. kit-links.bats L2 enforces this on all toolbelt scripts.
#
# Implementation: rule package go/internal/lint/rules/structure (a lint.RootRule) on the shared engine
# go/internal/lint (go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin`
# builds go/bin/n4lint when it is missing or older than a Go source, then this script execs
# `n4lint lint-structure <args>`. Needs `go` on PATH to build (exit 3 otherwise).
# [ev: retro campaign8-structure]
# Mutation: LS7 -- removes 3-part version floor check, allowing :baja:4.14 (2-part) to pass instead of FAIL
# Mutation: LS14 -- drops the L14 hardcoded-gradlePluginVersion check, letting a literal pin pass instead of FAIL
# Mutation: LS15 -- drops the L15 root signing-block check, so the wizard root (LS15) no longer WARNs
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-structure: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-structure "$@"
