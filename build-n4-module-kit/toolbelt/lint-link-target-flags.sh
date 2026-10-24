#!/usr/bin/env bash
# lint-link-target-flags.sh — flags on linked slots: READONLY on a link-in target (LTF1),
# TRANSIENT on an operator mode/HOA slot (LTF2).
#
#   LTF1 FAIL: a @NiagaraProperty that is a link-in TARGET carries Flags.READONLY. Workbench
#              LinkCheck refuses a READONLY target (linkcheck.propReadonly), so the mirror can never
#              be linked and the dashboard shows the type default forever. A slot is a known link-in
#              target when (a) the comment block heading its run of consecutive annotations says
#              "link-in", "linked from", "commissioning link" or "written by (a) BLink"
#              (case-insensitive; the block ends at a blank line, code or another annotation),
#              or (b) it is a backticked first-column slot of Table 2 (control -> facade) in the
#              --wiring-map file (generate-wiring-map.sh layout) whose Source RT slot names a source
#              (an unfilled `_(fill)_`, empty, self, n/a, none or dash source = the slot is self-set).
#              READONLY stays legal for a slot the owning component/reader sets itself and no
#              external Link targets (e.g. a timer anchor). [ev: retro panccadia-commissioning-lessons Δ1]
#              [ev: retro panccadia-persistent-config-hoa Δ3]
#   LTF2 FAIL: an OPERATOR slot named *Mode / *Hoa (optionally digit-suffixed) or hoa* carries
#              Flags.TRANSIENT. A TRANSIENT operator choice is never encoded into config.bog, reverts
#              to its default (AUTO) on restart, and a facade->control link then re-pushes that
#              default onto the persisted control slot. Lives here (not lint-config-sanity) because the
#              failure is a flag on a link endpoint, the same class of defect as LTF1.
#              A non-OPERATOR computed status mode may stay TRANSIENT. [ev: retro panccadia-persistent-config-hoa Δ1]
#
# Heuristic limits (documented, grep-level): a link-in target with neither a comment nor a wiring-map
# row is invisible to LTF1 — keep the module comment convention or pass --wiring-map. Flags are
# matched as the READONLY / TRANSIENT / OPERATOR tokens anywhere in the annotation.
#
# Mutation: LTF-comment -- dropping the link-comment detection lets a commented READONLY link target pass
# Mutation: LTF-map -- dropping the --wiring-map Table 2 harvest lets a mapped READONLY target pass
# Mutation: LTF-selfset -- harvesting every Table 2 slot regardless of its source FAILs a self-set READONLY row
# Mutation: LTF-srcname -- matching the sentinels as substrings drops a real fillLevel / selfTestOk source
# Mutation: LTF-backtick -- matching the sentinels on the raw cell FAILs a backticked `self` row
# Mutation: LTF-header -- detecting the header on any row drops a data row whose display name starts with Source
# Mutation: LTF-transient -- dropping the TRANSIENT operator-mode check lets a non-persisted HOA pass
# Mutation: LTF-mapread -- going on with no Table 2 target passes a mapped READONLY slot clean on an unreadable map
#
# Implementation: rule package go/internal/lint/rules/linktargetflags on the shared engine go/internal/lint
# (walker, Mode Valued parser, row formatter; go/README.md); the --wiring-map Table 2 harvest keeps the
# toolbelt/generate-wiring-map.sh layout above. This wrapper is the stable entry point: go/go-tool.sh
# `go_tool_bin` builds go/bin/n4lint when it is missing or older than a Go source, then this script execs
# `n4lint lint-link-target-flags <args>`. Needs `go` on PATH to build (exit 3 otherwise).
#
# Row:    FAIL  lint-link-target-flags  <file>:<line>  LTF<n>: <reason>
# Usage:  lint-link-target-flags.sh [--wiring-map <docs/wiring-map.md>] <java-src-dir>
# Exits:  0 clean · 1 any FAIL · 3 usage/env (K20; no Java sources is exit 3 + ERROR row)
# Dot-directories pruned (D9b). VCS-free by design (kit-links L2). ASCII matching (the bash ran LC_ALL=C).
# Declared difference from the bash lint (port 2026-10-24): a --wiring-map file that exists but cannot be read
# is an env error (exit 3, `cannot read --wiring-map <file>: <reason>` on stderr); the bash awk printed its
# error and went on with no Table 2 target.
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-link-target-flags: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-link-target-flags "$@"
