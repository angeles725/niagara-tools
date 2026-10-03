#!/usr/bin/env python3
"""Run one mutation-check command without stale bytecode.

A mutation restored with a plain copy within the same second, with the same file size,
can leave a `__pycache__/*.pyc` whose recorded mtime and size still match the source:
Python then keeps running the mutant (retro 2026-10-03 D1). This helper removes every
`__pycache__` directory under `--root` (default: the kit directory) and runs the command
with `PYTHONDONTWRITEBYTECODE=1`, so neither the mutated nor the restored run can reuse
or leave bytecode. Run it once with the mutation applied (expect RED) and once after
the restore (expect GREEN).

Usage, from the repo root:
    python3 mcp-n4-kit/tools/mutation_check.py -- python3 -m unittest discover -s mcp-n4-kit/tests

Exit code: the command's own exit code; 2 on usage errors.
"""
import argparse
import os
import shutil
import subprocess
import sys

KIT_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))


def clear_pycache(root):
    """Remove every `__pycache__` directory under `root`; return how many were removed."""
    removed = 0
    for dirpath, dirnames, _ in os.walk(root):
        if "__pycache__" in dirnames:
            shutil.rmtree(os.path.join(dirpath, "__pycache__"))
            dirnames.remove("__pycache__")
            removed += 1
    return removed


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--root", default=KIT_DIR,
                        help="directory whose __pycache__ trees are removed (default: the kit)")
    parser.add_argument("command", nargs=argparse.REMAINDER,
                        help="the command to run, after '--'")
    args = parser.parse_args(argv)
    command = args.command[1:] if args.command[:1] == ["--"] else args.command
    if not command:
        parser.error("no command given (put it after '--')")
    if not os.path.isdir(args.root):
        parser.error("--root is not a directory: %s" % args.root)
    removed = clear_pycache(args.root)
    print("mutation_check: removed %d __pycache__ dir(s) under %s" % (removed, args.root),
          file=sys.stderr)
    env = dict(os.environ, PYTHONDONTWRITEBYTECODE="1")
    return subprocess.call(command, env=env)


if __name__ == "__main__":
    sys.exit(main())
