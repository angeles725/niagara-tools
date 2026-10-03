#!/usr/bin/env python3
"""patsub-check.py: flag an unquoted `$` or `&` in a bash pattern-substitution replacement.

Usage:  patsub-check.py <file>...
Row:    <file>:<line>: unquoted replacement in ${...} -- quote it: ${v//pat/"$rep"}
Exit:   0 clean · 1 any row · 3 usage or an unreadable file

Under bash 5.2 `patsub_replacement` an unquoted `&` in the replacement of ${v/pat/rep} or
${v//pat/rep} (including one that arrives through an unquoted `$rep`) expands to the matched
text. A replacement with an expansion or `&` must therefore sit inside double quotes.
Structural scan: inside `${NAME/` the pattern runs to the first unquoted, unescaped `/` at
nesting depth 0; the replacement runs to the matching `}`. Comment-only lines are skipped.
[ev: retro polish-2026-10-02-close Δ3]
"""
import re
import sys

START = re.compile(r'\$\{[A-Za-z_][A-Za-z0-9_]*(\[[^]]*\])?/')


def scan_part(s, i, stop_slash):
    """Walk from i to the end of a pattern (stop at depth-0 '/') or a replacement (stop at
    depth-0 '}'). Return (end_index, unquoted_hit)."""
    depth = 0
    quote = None
    hit = False
    n = len(s)
    while i < n:
        c = s[i]
        if c == '\\':
            i += 2
            continue
        if quote == "'":
            if c == "'":
                quote = None
        elif quote == '"':
            if c == '"':
                quote = None
            elif c == '$' and s.startswith('${', i):
                depth += 1
                i += 2
                continue
            elif c == '}' and depth > 0:
                depth -= 1
        else:
            if c == "'":
                quote = "'"
            elif c == '"':
                quote = '"'
            elif s.startswith('${', i):
                if depth == 0:
                    hit = True
                depth += 1
                i += 2
                continue
            elif c == '}':
                if depth == 0:
                    return i, hit
                depth -= 1
            elif depth == 0 and stop_slash and c == '/':
                return i, hit
            elif depth == 0 and c in '$&':
                hit = True
        i += 1
    return n, hit


def check_line(line):
    if line.lstrip().startswith('#'):
        return False
    for m in START.finditer(line):
        i = m.end()
        if i < len(line) and line[i] in '/#%':
            i += 1
        end, _ = scan_part(line, i, True)
        if end >= len(line) or line[end] != '/':
            continue
        _, hit = scan_part(line, end + 1, False)
        if hit:
            return True
    return False


def main(argv):
    if not argv:
        sys.stderr.write('usage: patsub-check.py <file>...\n')
        return 3
    rows = 0
    for path in argv:
        try:
            with open(path, encoding='utf-8', errors='replace') as fh:
                lines = fh.read().split('\n')
        except OSError as exc:
            sys.stderr.write('patsub-check: cannot read %s: %s\n' % (path, exc))
            return 3
        for no, line in enumerate(lines, 1):
            if check_line(line):
                print('%s:%d: unquoted replacement in ${...} -- quote it: ${v//pat/"$rep"}' % (path, no))
                rows += 1
    return 1 if rows else 0


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
