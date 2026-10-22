#!/usr/bin/env python3
"""ci_steps.py <workflow.yml> — print every `steps:` item of a GitHub workflow for scripts/ci-local.sh.

Dependency-free (stdlib only, no PyYAML): it reads the subset of YAML that ci.yml uses and refuses the rest.
Output: one record per step, three NUL-terminated fields: <kind> <name> <body>.
  kind run  — body is the `run:` script (single line, or a `|` / `|-` block, dedented)
  kind uses — body is the action ref
  kind bad  — body names why the local replay cannot reproduce the step (env:, shell:, if:, ...)
A nameless step is named by its first `run:` line or its action ref.
Exit 2 with `<file>:<line>: <reason>` on stderr for a form it cannot read (folded `>`, an unterminated
quote, a `run:`/`uses:` outside a step item, a workflow-level env:/defaults:), so no step can vanish.
"""
import re
import sys

KEY = re.compile(r'^( *)(- +)?([A-Za-z_][\w-]*):(?: (.*))?$')
STEP_KEYS = {'name', 'run', 'uses', 'with', 'id'}  # anything else changes step semantics


def unquote(val, where):
    val = val.strip()
    if val[:1] in ('"', "'"):
        if len(val) < 2 or val[-1] != val[0]:
            raise ValueError(f'{where}: unterminated quoted scalar')
        return val[1:-1]
    return val


def block(lines, i, key_indent):
    """Collect the `|` block after line i; return (text, next index)."""
    body = []
    j = i + 1
    while j < len(lines) and (not lines[j].strip() or len(lines[j]) - len(lines[j].lstrip(' ')) > key_indent):
        body.append(lines[j])
        j += 1
    while body and not body[-1].strip():
        body.pop()
    cut = min((len(b) - len(b.lstrip(' ')) for b in body if b.strip()), default=0)
    return '\n'.join(b[cut:] for b in body), j


def parse(path):
    lines = open(path, encoding='utf-8').read().split('\n')
    steps, cur, steps_indent, item_keys = [], None, None, None
    i = 0
    while i < len(lines):
        line, where = lines[i], f'{path}:{i + 1}'
        m = KEY.match(line)
        if not m or line.lstrip().startswith('#'):
            i += 1
            continue
        indent, dash, key, val = len(m.group(1)), m.group(2), m.group(3), m.group(4) or ''
        if steps_indent is not None and indent <= steps_indent and not dash:
            steps_indent, cur = None, None  # left the steps: list
        if dash and steps_indent is not None and indent >= steps_indent:
            cur = {'line': i + 1, 'bad': []}
            steps.append(cur)
            item_keys = indent + len(dash)
        elif cur is not None and indent != item_keys:
            i += 1  # nested mapping (with:, env: values)
            continue
        if cur is None:
            if key in ('run', 'uses'):
                raise ValueError(f'{where}: {key}: outside a steps: item')
            if key in ('env', 'defaults'):
                raise ValueError(f'{where}: workflow/job-level {key}: changes every step; not replayed')
            if key == 'steps':
                steps_indent = indent
            i += 1
            continue
        if key not in STEP_KEYS:
            cur['bad'].append(f'{key}:')
        if key == 'run' and val.strip() in ('|', '|-'):
            cur['run'], i = block(lines, i, item_keys)
            continue
        if key == 'run' and val.strip()[:1] in ('|', '>'):
            raise ValueError(f'{where}: run: {val.strip()} block style not supported (use | or |-)')
        if key in ('name', 'run', 'uses'):
            cur[key] = unquote(val, where)
        i += 1
    return steps


def main(argv):
    if len(argv) != 2:
        print('usage: ci_steps.py <workflow.yml>', file=sys.stderr)
        return 2
    try:
        steps = parse(argv[1])
    except (OSError, ValueError) as err:
        print(err, file=sys.stderr)
        return 2
    out = sys.stdout
    for s in steps:
        run, uses = s.get('run'), s.get('uses')
        name = s.get('name') or (run.split('\n')[0] if run else uses) or f'step at line {s["line"]}'
        if s['bad'] or (run is None) == (uses is None):
            why = ' '.join(s['bad']) or 'needs exactly one of run:/uses:'
            out.write(f'bad\0{name}\0step-level {why} not replayed\0')
        elif run is not None:
            out.write(f'run\0{name}\0{run}\0')
        else:
            out.write(f'uses\0{name}\0{uses}\0')
    return 0


if __name__ == '__main__':
    sys.exit(main(sys.argv))
