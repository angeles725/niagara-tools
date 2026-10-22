#!/usr/bin/env bash
# lint-agent-on-shape.sh — every module-include.xml <on type="..."/> must be module:Type shaped.
#
# An agent-on binding whose target omits the module prefix (e.g. <on type="ReflowService"/>
# instead of <on type="nmodsreflow:ReflowService"/>), or is otherwise malformed, makes the
# agent silently fail to appear in Workbench — no build error. The value must be exactly
# `module:Type` (one colon, non-empty parts). See types/module-wiring.md §3.
# [ev: retro 2026-09-19-module-wiring]
#
# Usage:  lint-agent-on-shape.sh <src-root>     (finds module-include.xml under it)
#   Row:  FAIL  lint-agent-on-shape  <file>:<line>  malformed agent-on target (need module:Type): <value>
#   Exit: 0  clean · 1  any FAIL · 3  usage/env or a sub-directory find cannot enter
# VCS-free by design.
# Mutation: AOS2 -- relax the module:Type regex and AOS2 stops failing on a missing module prefix
# Mutation: AOS-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
# Mutation: AOS-unreadable -- ignoring the per-file scan failure reports an unreadable file clean (exit 0)
set -u
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib/awk-lint.sh"
awk_lint_parse lint-agent-on-shape raw 'usage: lint-agent-on-shape.sh <src-root>' "$@"
awk_lint_list -name 'module-include.xml'

# One awk per file (was three printf|grep/sed pipelines per line). The value is the one GNU sed's
# `s/.*<on[[:space:]][^>]*type="([^"]*)".*/\1/` captured: the LAST `<on<space>` that holds a
# quoted type= (greedy .*), and inside it the LAST `type="` with no `>` before it (greedy [^>]*).
# shellcheck disable=SC2016  # a literal awk program: $0, $n are awk fields
awk_lint_each '
    {
        val = ""; found = 0
        for (p = length($0) - 3; p >= 1 && !found; p--) {
            if (substr($0, p, 3) != "<on" || substr($0, p + 3, 1) !~ /[[:space:]]/) continue
            tag = substr($0, p + 4); gt = index(tag, ">"); if (!gt) gt = length(tag) + 1
            for (q = gt - 1; q >= 1; q--) {
                if (substr(tag, q, 6) != "type=\"") continue
                v = substr(tag, q + 6); e = index(v, "\"")
                if (e) { val = substr(v, 1, e - 1); found = 1; break }
            }
        }
        # well-formed = exactly one colon, non-empty module and type parts
        if (val != "" && val !~ /^[A-Za-z0-9_.-]+:[A-Za-z0-9_$]+$/)
            printf "FAIL  lint-agent-on-shape  %s:%d  malformed agent-on target (need module:Type): %s\n", ENVIRON["AL_FILE"], FNR, val
    }'

awk_lint_exit "$AL_ROWS"
