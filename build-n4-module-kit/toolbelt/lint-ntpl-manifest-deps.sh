#!/usr/bin/env bash
# lint-ntpl-manifest-deps.sh — FAILs a template archive (.ntpl, or .napl application template) a module ships
# without template.bog or template-manifest.xml, with a manifest that is no well-formed template manifest, with
# a <dependency> that neither the module's build declares nor the Niagara installation ships, or whose bog or
# Px files use a module no <dependency> lists and the installation does not ship.
#
# A .ntpl is a zip holding template.bog and template-manifest.xml; the manifest's <dependencies> name the
# modules the template needs, and an application-template install hard-fails on a missing one
# (types/logic-authoring.md § Shipping a template module). A .napl application template is the same zip. For
# every *.ntpl and *.napl under the module root (the extension in any letter case; a part's build/ and test/ outputs, dot directories and *-worktrees checkouts
# are not read):
#   ntpl-entries   the archive lacks template.bog or template-manifest.xml (matched by their whole name, at
#                  the top of the archive), holds one of them twice, or is refused whole: not a zip, over
#                  32 MiB on disk, over 10000 entries, or an unsafe entry name (an absolute path, a ".."
#                  element, a backslash, a drive letter, a control character). The archive is untrusted
#                  input: nothing is unpacked, and only the manifest is inflated, up to 4 MiB.
#   ntpl-manifest  template-manifest.xml is not well-formed XML (line:col inside the manifest), cannot be
#                  inflated, has a root element other than <template>, or holds a <dependency> with no name
#                  or outside <dependencies>.
#   ntpl-deps      a <dependencies><dependency name> that no *.gradle.kts under the root declares with
#                  api(":X"), nre(":X") or implementation(":X") (or project(":X")) and that the installation
#                  does not ship. settings.gradle.kts is no build file, and each part (the directory of a
#                  build file, and <part>.gradle.kts) counts as declared for its own module's templates.
#                  Names are listed once, sorted.
#   ntpl-station   a template under a newStations/ directory (the station templates Workbench lists,
#                  types/distribution.md § 8) whose manifest root lacks isStation="true".
#   ntpl-uses      a module that template.bog (each m="<alias>=<module>", each module://<module> ORD) or a
#                  *.px entry (each <import><module name>, each module:// ORD) uses, that no <dependency>
#                  names (a part counts: control-rt lists control) and the installation does not ship; baja
#                  and the module's own parts are exempt. One row per entry, names sorted; a bog or Px entry
#                  that is not well-formed XML, or inflates past 32 MiB, is a FAIL row naming it.
#                  [ev: retro ntpl-alias-napl-k4 Δ1]
# Run bounds: one run reads at most 1000 templates (in path order, or entry-name order in a jar); past them one
# FAIL ntpl-entries row names the first template not read. Every template of a jar and every manifest, bog and
# Px entry of every archive inflate from one budget of 128 MiB for the whole run; the entry it leaves no room for
# is one FAIL row (ntpl-manifest, ntpl-uses, or ntpl-entries for a template of a jar) and nothing more is read.
# A small jar of many templates, each inflating its own bound, cannot run for minutes. [ev: retro ntpl-budget-t26 Δ1]
#
# Stock modules: a station template lists every module its station uses, most of them shipped by every
# Niagara installation and declared by no build file. --niagara-home <dir> (or NIAGARA_HOME in the
# environment; the option wins) names the installation: each <dir>/modules/<name>.jar that is a regular file
# the lint can open (a link to one counts; a dangling link, a directory or an unreadable file ships nothing) is
# a module it ships (part suffix kept: fox-rt), and a dependency it ships is not a finding. An empty
# --niagara-home value names no installation: it is a usage error (exit 3), never "no installation". Without an installation the lint
# cannot tell a stock module from a missing one: it prints one WARN row saying so and reports the undeclared
# dependencies as WARN ntpl-deps rows, exit 0; it never blocks on what it cannot know. An installation path
# that does not exist, has no modules/ directory or whose modules/ cannot be listed is exit 3.
# [ev: retro gate-followups-and-template-lints Δ12]
#
# Symlinks: a link is followed only when its real target is inside the module root. A .ntpl that is such a
# link is checked through it, and such a symlinked directory is entered unless its target is pruned (out ->
# build stays a build output); both are reported under the link's own path, each directory and file once (a
# link back to an ancestor ends). A link whose target is outside the module root (a sibling module, "/") is
# not followed: one WARN ntpl-link row per link. A .ntpl link that leads nowhere, and a .ntpl that is not a
# regular file (a FIFO), are exit 3 (they could not be read). "Nothing to check" is never printed over
# something that was skipped. [ev: retro ntpl-lint-walk-k-rows Δ1] [ev: retro ntpl-lint-walk-k-rows Δ2] A path holding a control character, a double quote or a
# backslash is printed in double quotes with C escapes, so a row is always one line.
# A module root that ships no .ntpl is one PASS row and exit 0.
# A template module jar (the one .jar of .ntpl entries Workbench's Make Module writes) given as the root, a
# regular file named *.jar: each *.ntpl and *.napl entry is checked as a shipped template, in entry-name order,
# each row naming <jar>!/<entry>. The jar is opened within the bounds of a .ntpl (32 MiB on disk, checked before
# the jar is opened, 10000 entries, no unsafe name); each template inflates to at most 32 MiB, within the run
# bounds above. A jar declares no
# build dependencies: a dependency is judged against the installation alone. A jar with no template entry is one
# PASS row. [ev: retro ntpl-jar-manifest-k4b Δ2]
# The manifest: an archive without template-manifest.xml is read through template.manifest.xml, the spelling of
# the Templates guide ("Example template.manifest.xml file"); with both, the hyphenated one alone is read. A
# manifest row names the entry it read.
# [ev: retro ntpl-jar-manifest-k4b Δ1] [ev: retro ntpl-jar-manifest-k4b Δ4]
# [ev: retro gate-followups-and-template-lints Δ7] [ev: retro gate-followups-and-template-lints Δ8]
#
# Implementation: rule package go/internal/lint/rules/ntplmanifest on the shared engine go/internal/lint
# (go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when
# it is missing or older than a Go source, then this script execs `n4lint lint-ntpl-manifest-deps <args>`.
# Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-ntpl-manifest-deps.sh [--niagara-home <dir>] <module-root | template-module.jar>
#   Row:  FAIL  ntpl-entries|ntpl-manifest|ntpl-deps|ntpl-station|ntpl-uses  <file.ntpl>: <reason>
#   Row:  WARN  ntpl-uses  <file.ntpl>: <entry> uses <modules>: ...   (no installation given)
#   Row:  WARN  ntpl-deps  <file.ntpl>: <reason>            (no installation given)
#   Row:  WARN  ntpl-link  <link>: a symlink to <target>, outside the module root -- not followed, ...
#   Row:  WARN  lint-ntpl-manifest-deps  <module-root>: the installation's module list is not known, ...
#   Row:  PASS  lint-ntpl-manifest-deps  <module-root>: no *.ntpl under the root (nothing to check)
#   Row:  PASS  lint-ntpl-manifest-deps  <file.jar>: no *.ntpl or *.napl entry in the jar (nothing to check)
#   Exit: 0  no FAIL (WARN rows, or no .ntpl under the module root, are still 0) · 1  any FAIL · 3  usage/env (an empty --niagara-home included), a root with neither a *.gradle.kts nor a *.ntpl, an installation path that gives no module list, a .ntpl link that leads nowhere or a .ntpl that is not a regular file, an archive or a build file that cannot be opened, or a directory that cannot be entered
# Mutation: NTPL1 -- dropping the lacks-entry row passes a .ntpl with no template.bog or no manifest
# Mutation: NTPL2 -- reading a manifest that does not parse as one with no dependencies passes it
# Mutation: NTPL3 -- dropping the undeclared-dependency row passes a template that needs a module nobody declares or ships
# Mutation: NTPL4 -- dropping the station row ships a newStations/ template whose manifest is not a station manifest
# Mutation: NTPL6 -- walking without following links reads a symlinked template as "nothing to check"
# Mutation: NTPL7 -- comparing against the build files alone fails a station template on the stock modules it lists
# Mutation: NTPL8 -- failing without an installation blocks a module on stock modules the lint cannot know about
# Mutation: NTPL15 -- refusing a file root, or skipping the templates inside a jar, passes a template module that needs an unshipped module
# VCS-free by design.
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-ntpl-manifest-deps: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-ntpl-manifest-deps "$@"
