#!/usr/bin/env bash
# slot-coverage.sh — MM2 exposed-set coverage (palette / lexicon / type-registration).
#
# Operationalizes three silent-deploy footguns as one set-difference metric before deploy:
#   - empty module.palette passes the gate yet nothing drags in Workbench (B5)
#   - missing module.lexicon key renders raw camelCase in operator views (T8)
#   - dangling module-include.xml <type> surfaces as "Missing class" live (B12)
#
# Usage (pure subcommand — QA-pinned via exact MM2 vectors):
#   slot-coverage.sh set-coverage <declared-csv> <required-csv>
#   ("" = empty set; each set deduped + lexicographically sorted)
#   stdout (exactly 3 lines, sets comma-joined, empty string after = when set is empty):
#       pct=<n.n|N/A>
#       missing=<sorted,csv>
#       extra=<sorted,csv>
#   exit 0   always for set-coverage
#   exit 2   argc != 2  +  "usage: slot-coverage.sh set-coverage ..." on stderr
#
# Usage (per-slot subcommand — Campaign 8 PR5, D6):
#   slot-coverage.sh per-slot <module-include.xml> <module.lexicon> <src-dir>
#   required   <- OPERATOR-flagged @NiagaraProperty slots from *.java under <src-dir>
#                 (dot-dirs pruned — D9b; non-OPERATOR slots are NOT required to have a key)
#   declared   <- BARE lexicon keys only: the framework reads a slot's display name from the bare slot key in the
#                 declaring type's module (NSlot.java:42); "FanMode.fan" is never read for slot fan.
#   MISSING    <- OPERATOR slot with no bare lexicon key (renders raw camelCase in operator views)
#   DOTTED     <- a dotted key "X.<slot>" naming an annotated slot that has no bare key: dead for that slot.
#                 Other dotted keys (help.guide.base, Type.displayName, error keys) are not judged.
#                 [ev: retro devguide415-lexicons-ux-i18n Δ1] [ev: corpus B1305 §1305.2 C1]
#   STALE      <- bare lexicon key matching no "known key" — a truly dead translation.
#                 known keys = @NiagaraProperty slots (any flags) ∪ declared type names
#                 (module-include.xml <type name="X">) ∪ @Range enum-tag values.
#                 A key for a READONLY slot or a frozen-enum tag is live, NOT stale. A bare type-name key is
#                 not reported here although it is ineffective (the type reads <Type>.displayName,
#                 NTypeInfo.java:178): the parse mode WARNs it, so it is not reported twice.
#   stdout:    pct=<n.n> (per-slot)   followed by MISSING <slot>, STALE <key> and DOTTED <key> lines
#   exit 0   no MISSING slots
#   exit 1   any MISSING slot
#   exit 2   bad arity + "usage: slot-coverage.sh per-slot ..." on stderr
#   exit 3   file/dir not found, a <src-dir> sub-directory the walk could not enter, or an input (the xml, the
#            lexicon or a *.java) it cannot read: "slot-coverage: cannot read <path>: <reason>", nothing on stdout
#
# Usage (parse subcommand — reads module files, calls the same pure function):
#   slot-coverage.sh [--strict] [--facade] <module-include.xml> <module.lexicon>
#   required   <- the name of every <type> element of module-include.xml, read through go/internal/xmlsrc (a tag
#                 over several lines counts, a commented-out one does not; a malformed file is exit 3 with
#                 "slot-coverage: <path>:<line>:<col>: <reason>")
#   declared   <- "<Type>.displayName" keys in module.lexicon (comments and blank lines ignored): the
#                 framework reads a type's display name from that key only (NTypeInfo.java:178). A bare
#                 "<Type>=" key is ineffective -> "slot-coverage: WARN ineffective type key: <Type>=" and the
#                 type stays missing; a "Type.slot" key does not declare the type.
#                 [ev: retro n4-course-lexicon-logging-testing Δ1] [ev: corpus B1236 §1236.3]
#   |required|==0   -> N/A
#   empty lexicon + |required|>=1 -> pct=0.0 + "slot-coverage: FAIL empty lexicon ..." on stdout + exit 1
#   duplicate keys in lexicon -> "slot-coverage: FAIL dup-keys: <key>" on stdout + exit 1 (A1/B792)
#   facade FAIL (default, no --strict needed): a MISSING type whose name ends in Panel or Facade is
#              operator-facing — with no lexicon key its slots render raw camelCase in Workbench and the
#              operator cannot find them — so "slot-coverage: FAIL facade type without lexicon: <T>" is
#              printed per such type and the exit is 1. --facade declares the whole module operator-facing
#              (every missing type FAILs). Internal control types stay advisory (missing=, exit 0). A facade
#              type whose only key is a bare "<Type>=" gets "WARN facade type with only a bare key" instead:
#              Workbench shows the raw type name, nothing breaks (NTypeInfo.java:178); --strict promotes it.
#              [ev: retro panccadia-commissioning-lessons Δ9]
#   --strict   -> exit 1 when missing is non-empty or WARN was emitted
#   exit 0   clean (no FAIL, no WARN in strict); exit 1 dup-keys FAIL, empty-lexicon FAIL, facade FAIL or strict WARN
#   exit 2   wrong argc
#   exit 3   env (file missing, the xml or lexicon unreadable: "slot-coverage: cannot read <path>: <reason>",
#            or a malformed module-include.xml)
#
# Integer-tenths rounding: t=(1000*present+R/2)/R; print "$((t/10)).$((t%10))"
# (verbatim from verify-module.sh coverage — no awk/printf %.1f; no locale dependency)
# This script is VCS-free by design. version control is never invoked.
# kit-links.bats L2 enforces the no-version-control rule on all toolbelt scripts.
#
# Implementation: the Go module at <repo>/go (go/internal/slotcoverage, over the shared javasrc
# PropertyScan and lint.Walk). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds
# go/bin/slot-coverage when the binary is missing or older than a Go source, then this script execs it.
# Needs `go` on PATH to build; no Go toolchain or a failed build is exit 3.
set -u

here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'slot-coverage: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" slot-coverage)" || exit 3
exec "$BIN" "$@"
