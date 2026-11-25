#!/usr/bin/env bash
# nre-tool.sh — run a Niagara install's build-time nre tools by name (issue #851), no station and no Workbench needed.
#
# Usage: nre-tool.sh list
#        nre-tool.sh deps [--raw] <niagara_home> <module>
#        nre-tool.sh palette <niagara_home> <moduleDir>
#        nre-tool.sh recompile <niagara_home> <dir|bog> [-alias:<a>] [-tsaUrl:<u>]
#   list       the supported tools, one per line with a purpose; needs no install.
#   deps       the module parts <module> depends on, one per line, from the install's registry (read-only):
#              nre devkit:com.tridium.devkit.util.DependencyChecker <module>. --raw prints nre's whole output instead.
#   palette    nre devkit:com.tridium.devkit.util.PaletteGenerator <moduleDir>: writes <moduleDir>/module.palette from
#              the gifs under src/, REPLACING the file. Runs only on a module part directory (module-include.xml and
#              src/), prints the file it writes first (so the change shows in the working-tree diff), and needs
#              bin/nre.exe.
#   recompile  nre program:com.tridium.program.ui.RecompileTool <dir|bog>: recompiles and re-signs Program objects.
#              Never passes -password: (a `-password...` argument is refused); nre prompts on the terminal, so run it
#              from one. -alias: and -tsaUrl: pass through. On a terminal nre gets the real stdout and stderr (a
#              piped stdout may break or echo the prompt), so its output is not read: exit 1 when no *.bog under
#              the target was rewritten. Piped or redirected, the output is read (ERROR / "--- Processed" lines).
#   A Windows install is run from WSL through its bin/nre.exe; the runtime boots from the install (about 35 s), and
#   a path handed to nre.exe goes through wslpath -w.
# exit:  0 ok · 1 deps: the install holds no such module · palette/recompile: nre exited 0 but printed a failure
#        (PaletteGenerator "Failed to generate palette", RecompileTool an ERROR line or no "--- Processed" line;
#        on a terminal: no bog file rewritten)
#        · 3 usage/env: an unknown tool (the supported list is printed), no bin/nre.exe or bin/nre, a bad module name,
#        a refused directory, flag or password · anything else: nre's own exit code, passed through
#
# Implementation: the Go module at <repo>/go (go/internal/nretool); go/go-tool.sh builds go/bin/nre-tool on demand.
# This script is VCS-free by design — version control is never invoked (kit-links L2).
set -u

here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'nre-tool: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" nre-tool)" || exit 3
exec "$BIN" "$@"
