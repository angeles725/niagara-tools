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
#   L3  WARN  pure-model package (src/.../model/) has javax.baja.* imports (advisory). In a strict module
#             (modularity: strict in <module-root>/MODULE-MAP.md) per file of every -rt/-ux/-wb/-se part
#             instead: FAIL a Baja-free class outside its role package (rt/wb/se model/; ux route/, service/
#             or model/), a model/ import of javax.baja/com.tridium, a model/ class with no <C>Test.java in
#             srcTest/ [ev: retro modularity-strict-new Δ3]
#   L3b FAIL  strict only: a model/ class imports a type of the module's own non-model package
#             [ev: retro modularity-gaps Δ1]
#   L4  FAIL  module.lexicon non-empty (>=1 key=value) when >=1 type is declared
#   L5  FAIL  module.palette non-empty for -rt profile
#   L6  FAIL  module-include.xml present; no hand-authored META-INF/module.xml in source
#   L7  FAIL  dependency version is a 3-part floor (4.14.0), not 2-part (4.14)
#   L9  FAIL  no empty skeleton artifact: -wb/-ux with 0 Java classes AND empty palette
#   L10 FAIL  no absolute host paths in gradle.properties (module root + upward to .git root). R10-D8: one of
#             the four host build-location keys (niagara_home, niagara_user_home, nodeHome,
#             org.gradle.java.installations.paths; props.BuildLocationKeys) outside build-n4-module-kit/fixtures/
#             and templates/ is a WARN instead: migrate it to the user-level gradle.properties (BUILD-LOOP.md
#             bring-up step 2). user_home, and those keys in a kit fixture or template (K16), stay the FAIL
#   L11 FAIL  mixed srcTest (BTest+JUnit) without both :test-wb AND junit gradle declarations
#   L12 FAIL  org.gradle.java.installations.auto-detect=false absent or commented in the module root's
#             gradle.properties (R10-D8: the paths key may live in the user-level gradle.properties)
#   L13 FAIL  >=2 sibling gradle.properties under group dir with divergent niagara_home=
#   L14 FAIL  settings.gradle.kts (module root, else nearest ancestor up to the .git sentinel) hardcodes
#             gradlePluginVersion as a string literal instead of
#             providers.gradleProperty("niagaraPluginVersion").getOrElse(...) — -PniagaraPluginVersion
#             (build.sh --plugin-version) is then silently ignored [ev: retro panccadia-restart-seq-comp-lockout-hours Δ2]
#   L15 WARN  the ROOT build.gradle.kts beside that settings.gradle.kts applies com.tridium.niagara-signing
#             without the signing block (uncommented allowDefaultProfile.set(false) or = false, niagaraSigning {,
#             aliases, signingProfileFile) — a missing profile then signs with the self-signed default key (#266)
#   L16 FAIL  the gradle.properties beside that settings.gradle.kts names (niagaraSigningProfile) a
#             LocalSigningProfile whose niagara.signing.validity - niagara.signing.refresh is below 32 days: the
#             signing plugin re-mints its key at every signing and the jars carry a signature block no JVM verifies.
#             One predicate with run-niagara-test (go/internal/signingprofile). A named profile that does not exist
#             WARNs (margin unchecked); one that is not a properties XML file or holds a non-integer day count is
#             exit 3 [ev: retro r10-signing-margin-lint Δ1]
#   L17 FAIL  module.palette (read as XML, comments out) t= that is a dotted Java class name ("Invalid typespec"), or
#             <symbol>:B<Type> where the module's module-include.xml declares <Type> and not B<Type> ("Type not
#             found"); another module's B-prefixed name is not judged (vendor types may carry the B). WARN an m= that maps
#             a symbol to a module other than its first m= (the decoder keeps the last mapping) and a palette holding
#             <p that is not well-formed XML (its specs unchecked). An identical repeated m= is clean
#             [ev: retro course-coverage Δ20]
#   (L8 signed-jar check is in verify-module.sh)
#
# Mutation: LS17a -- dropping the declared-type lookup turns the B-prefixed own type into a WARN, not the FAIL
# Mutation: LS17b -- comparing every repeated m= instead of a different module WARNs the identical repeat
#
# Usage:  lint-structure.sh <module-root>
#
#   Row format:  FAIL|WARN  lint-structure  <path>  L<n>: <reason>
#   Exits:       0  no FAIL (WARN-only is still 0) · 1  any FAIL · 3  usage/env (K20) or an unreadable file/dir
#
# A file or directory it cannot read prints `lint-structure: cannot scan|list ...` on stderr (the rows still
# print); an unreadable root build.gradle.kts stays the visible L15 WARN. Sub-directories named .* or
# *-worktrees are never entered. A root without profiles prints only
# `ERROR  lint-structure  <root>  no module profiles found (...)` and exits 3 (a wrong path, never a clean pass).
#
# VCS-free by design; version control is never invoked. The .git directory is used
# only as a filesystem sentinel (stop-marker) for the L10 and L14 upward walks; no VCS
# commands are ever executed. kit-links.bats L2 enforces this on all toolbelt scripts.
#
# Implementation: rule package go/internal/lint/rules/structure (a lint.RootRule) on the shared engine
# go/internal/lint (go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin`
# builds go/bin/n4lint when it is missing or older than a Go source, then this script execs
# `n4lint lint-structure <args>`. Needs `go` on PATH to build (exit 3 otherwise).
# [ev: retro campaign8-structure] [ev: retro go-lint-structure Δ1]
# Truth fix 2026-10-24: L15 reads the root build.gradle.kts as Kotlin (go/internal/ktssrc), not with the Java
# scanner: a block in a nested /* */ comment or in a string inside a ${...} template is not code, a raw """ string
# takes no \ escape, an id("com.tridium.niagara-signing") split over lines applies the plugin, and a script the
# scanner cannot read is the visible L15 WARN. Real trees unchanged. [ev: retro r10-ktssrc-gradle-lints Δ1]
# Truth fix 2026-10-24: L14 matches settings.gradle.kts on its ktssrc code lines, not the raw text: a pin inside a
# /* */ comment or a raw string is not flagged, one after a leading comment is, and a script the scanner cannot read
# is `cannot scan`, exit 3. Real trees unchanged. [ev: retro r10-ktssrc-gradle-lints Δ2]
# Mutation: LS10-keys -- dropping the kit-template check WARNs a fixture's build-location key instead of FAILing it
# Mutation: LS10-userhome -- adding user_home to the build-location keys turns its host-path FAIL into a WARN
# Mutation: LS10-shapes -- matching only key=X:\ (the bash regex) passes spaced, colon, indented, POSIX and UNC host paths
# Mutation: LS-noprofile -- running the module-level checks on a profile-less root passes a wrong path (exit 0)
# Mutation: LS7 -- removes 3-part version floor check, allowing :baja:4.14 (2-part) to pass instead of FAIL
# Mutation: LS14 -- drops the L14 hardcoded-gradlePluginVersion check, letting a literal pin pass instead of FAIL
# Mutation: LS15 -- drops the L15 root signing-block check, so the wizard root (LS15) no longer WARNs
# Mutation: LS16-margin -- lowering signingprofile.MinMargin to 31 passes a 214/183 LocalSigningProfile
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-structure: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-structure "$@"
