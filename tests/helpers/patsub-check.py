#!/usr/bin/env python3
"""patsub-check.py: flag an unquoted `$` or `&` in a bash pattern-substitution replacement.

Usage:  patsub-check.py <file>...
Row:    <file>:<line>: unquoted replacement in ${...} -- quote it: ${v//pat/"$rep"}
        <file>:<line>: unterminated ${.../...} expansion -- cannot classify it (fail closed)
Exit:   0 clean · 1 any row · 3 usage or an unreadable file

Under bash 5.2 `patsub_replacement` an unquoted `&` in the replacement of ${v/pat/rep} or
${v//pat/rep} (including one that arrives through an unquoted `$rep`) expands to the matched
text. A replacement with an expansion or `&` must therefore sit inside double quotes.
Structural scan: inside `${NAME/` the pattern runs to the first unquoted, unescaped `/` at
nesting depth 0; the replacement runs to the matching `}`. Quote states: '...' is literal (a
backslash in it is a plain character), $'...' honors backslash escapes, "..." honors them and
nests ${...}. An expansion that does not close on its line is followed onto the next lines (up to
MAX_SPAN); one that never closes is reported as unterminated, never passed as clean.
Comment-only lines are skipped as starting lines.
[ev: retro polish-2026-10-02-close Δ3]
"""
import re
import sys

START = re.compile(r'\$\{[A-Za-z_][A-Za-z0-9_]*(\[[^]]*\])?/')
# How far an expansion is followed past its first line. Real kit expansions close on their line; 20 lines is
# far beyond any legitimate one, and 'unterminated' therefore also means 'did not close within MAX_SPAN lines'.
MAX_SPAN = 20


def scan_part(s, i, stop_slash):
    """Walk from i to the end of a pattern (stop at a depth-0 '/') or a replacement (stop at the
    depth-0 '}'). Return (end_index, unquoted_hit, closed)."""
    depth = 0
    quote = None          # None, "'" (literal), "$'" (ANSI-C) or '"'
    hit = False
    n = len(s)
    while i < n:
        c = s[i]
        if quote == "'":
            if c == "'":
                quote = None
            i += 1
            continue
        if c == '\\':
            i += 2
            continue
        if quote == "$'":
            if c == "'":
                quote = None
        elif quote == '"':
            if c == '"':
                quote = None
            elif s.startswith('${', i):
                depth += 1
                i += 2
                continue
            elif c == '}' and depth > 0:
                depth -= 1
        else:
            if s.startswith("$'", i):
                quote = "$'"
                i += 2
                continue
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
                    return i, hit, True
                depth -= 1
            elif depth == 0 and stop_slash and c == '/':
                return i, hit, True
            elif depth == 0 and c in '$&':
                hit = True
        i += 1
    return n, hit, False


def check_expansion(text, i):
    """i = index just past `${NAME/`. Return 'hit', 'clean' or 'open' (not closed in text)."""
    if i < len(text) and text[i] in '/#%':
        i += 1
    end, _, closed = scan_part(text, i, True)
    if not closed:
        return 'open'
    if text[end] != '/':
        return 'clean'          # ${v/pat} -- no replacement part
    _, hit, closed = scan_part(text, end + 1, False)
    if not closed:
        return 'open'
    return 'hit' if hit else 'clean'


def check_lines(lines):
    """Yield (line_no, kind) for the first offending expansion of a line; kind is 'hit' or 'open'."""
    for no, line in enumerate(lines, 1):
        if line.lstrip().startswith('#'):
            continue
        for m in START.finditer(line):
            verdict = 'open'
            text = line
            for extra in range(MAX_SPAN + 1):
                if extra:
                    if no - 1 + extra >= len(lines):
                        break
                    text += '\n' + lines[no - 1 + extra]
                verdict = check_expansion(text, m.end())
                if verdict != 'open':
                    break
            if verdict != 'clean':
                yield no, verdict
                break


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
        for no, kind in check_lines(lines):
            if kind == 'hit':
                print('%s:%d: unquoted replacement in ${...} -- quote it: ${v//pat/"$rep"}' % (path, no))
            else:
                print('%s:%d: unterminated ${.../...} expansion -- cannot classify it (fail closed)' % (path, no))
            rows += 1
    return 1 if rows else 0


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
