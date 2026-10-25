#!/usr/bin/env python3
"""Write the session retro for an mcp-n4 session and register it in retros/INDEX.md.

Reads the server's `audit.jsonl` + `journal.jsonl` (the same derivation as the
`n4_session_retro_draft` tool), writes `YYYY-MM-DD-<station>-session.md` and appends
the INDEX row as `pending`. Proposes only: it never edits the kit and never stages
GitHub issues (do that by hand). Idempotent: an existing retro is never overwritten
without `--force`. `--dry-run` prints the retro and writes nothing.

The draft comes from the Go binary `retro-draft` (go/cmd/mcp-n4/retro-draft, the Go server's
own drafting code): `$MCP_N4_RETRO_DRAFT_BIN`, else `go/bin/retro-draft` at the repository root.

Exit codes: 0 ok, 2 usage or no retro-draft binary, 3 refused (exists without --force, bad
input, no INDEX). Importing this module reads and writes no files. Both outputs are written
atomically (temp file + rename).
"""
import argparse
import datetime
import json
import os
import re
import stat
import subprocess
import sys
import tempfile

KIT_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
REPO_DIR = os.path.dirname(KIT_DIR)
DEFAULT_OUT_DIR = os.path.join(KIT_DIR, "retros")
DEFAULT_BINARY = os.path.join(REPO_DIR, "go", "bin", "retro-draft")
BINARY_ENV = "MCP_N4_RETRO_DRAFT_BIN"
_STATION = re.compile(r"[A-Za-z0-9][A-Za-z0-9._-]{0,63}")
_DATE = re.compile(r"\d{4}-\d{2}-\d{2}")


class Refused(Exception):
    pass


class NoBinary(Exception):
    pass


def resolve_binary(env=os.environ):
    """The retro-draft binary: $MCP_N4_RETRO_DRAFT_BIN, else go/bin/retro-draft."""
    path = env.get(BINARY_ENV) or DEFAULT_BINARY
    if not (os.path.isfile(path) and os.access(path, os.X_OK)):
        raise NoBinary("no retro-draft executable at %s: build it (cd go && go build -o bin/ "
                       "./cmd/mcp-n4/retro-draft) or set %s" % (path, BINARY_ENV))
    return path


def draft(state_dir, station, date, since):
    """`(markdown, deltas)` from the Go drafter; its refusal (a bad --since) is `Refused`."""
    argv = [resolve_binary(), "--state-dir", state_dir, "--station", station, "--date", date]
    if since is not None:
        argv += ["--since", since]
    proc = subprocess.run(argv, capture_output=True, text=True, encoding="utf-8")
    if proc.returncode == 3:
        raise Refused(proc.stderr.strip().removeprefix("retro-draft: "))
    if proc.returncode != 0:
        raise RuntimeError("retro-draft exited %d: %s" % (proc.returncode, proc.stderr.strip()))
    out = json.loads(proc.stdout)
    return out["markdown"], out["deltas"]


def parse_args(argv=None):
    p = argparse.ArgumentParser(prog="new_retro.py", description=__doc__.split("\n\n")[0])
    p.add_argument("--station", required=True, help="station name (used in the file name)")
    p.add_argument("--state-dir", required=True, help="the server's journal/audit directory")
    p.add_argument("--since", default=None, help="ISO-8601 timestamp; only newer entries count")
    p.add_argument("--out-dir", default=DEFAULT_OUT_DIR, help="retros directory (with INDEX.md)")
    p.add_argument("--date", default=None, help="YYYY-MM-DD (default: today, UTC)")
    p.add_argument("--force", action="store_true", help="overwrite an existing retro")
    p.add_argument("--dry-run", action="store_true", help="print only; write nothing")
    return p.parse_args(argv)


def _index_rows(index_text, name, row):
    """INDEX text with `row` appended, or replacing the row already naming `name`."""
    lines = index_text.splitlines()
    mine = [i for i, ln in enumerate(lines) if ln.startswith("| %s |" % name)]
    if mine:
        lines[mine[0]] = row
        for i in reversed(mine[1:]):
            del lines[i]
    else:
        lines.append(row)
    return "\n".join(lines) + "\n"


def _target_mode(path):
    """Mode for `path`: the existing file's, else 0644 under the umask (not mkstemp's 0600)."""
    try:
        return stat.S_IMODE(os.stat(path).st_mode)
    except FileNotFoundError:
        return 0o644 & ~_current_umask()


def _current_umask():
    """The process umask, read without changing it where the OS allows (issue #179 R3-002).

    Linux exposes it in /proc/self/status (`Umask:`); elsewhere the os.umask set-and-restore
    pair is the only API, and it briefly changes the process-global umask, which is safe for
    this single-threaded CLI.
    """
    try:
        with open("/proc/self/status", encoding="ascii") as fh:
            for line in fh:
                if line.startswith("Umask:"):
                    return int(line.split()[1], 8)
    except (OSError, ValueError, IndexError):
        pass
    umask = os.umask(0)
    os.umask(umask)
    return umask


def _atomic_write(path, text):
    """Write `text` to `path` through a temp file in the same directory and a rename."""
    mode = _target_mode(path)
    fd, tmp = tempfile.mkstemp(prefix=".new_retro-", suffix=".tmp", dir=os.path.dirname(path))
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as fh:
            fh.write(text)
        os.chmod(tmp, mode)
        os.replace(tmp, path)
    except BaseException:
        if os.path.exists(tmp):
            os.remove(tmp)
        raise


def run(args, out):
    if not _STATION.fullmatch(args.station):
        raise Refused("--station must match %s" % _STATION.pattern)
    date = args.date or datetime.datetime.now(datetime.timezone.utc).date().isoformat()
    try:
        if not _DATE.fullmatch(date):
            raise ValueError(date)
        datetime.date.fromisoformat(date)
    except ValueError:
        raise Refused("--date must be a real calendar date, YYYY-MM-DD") from None
    markdown, deltas = draft(args.state_dir, args.station, date, args.since)
    name = "%s-%s-session.md" % (date, args.station)
    row = "| %s | %s | %s | pending | %d |" % (name, args.station, date, deltas)
    if args.dry_run:
        out.write(markdown + "\nINDEX row: %s\n" % row)
        return
    path, index = os.path.join(args.out_dir, name), os.path.join(args.out_dir, "INDEX.md")
    if not os.path.isfile(index):
        raise Refused("%s not found: create the retros directory with its INDEX.md first" % index)
    if os.path.exists(path) and not args.force:
        raise Refused("%s already exists: pass --force to rewrite it" % path)
    with open(index, encoding="utf-8") as fh:
        new_index = _index_rows(fh.read(), name, row)
    _atomic_write(path, markdown)
    _atomic_write(index, new_index)
    out.write("retro  %s  written (%d delta(s), pending)\n" % (path, deltas))


def main(argv=None, out=None):
    out = sys.stdout if out is None else out
    args = parse_args(argv)
    try:
        run(args, out)
    except Refused as exc:
        print("new_retro: %s" % exc, file=sys.stderr)
        return 3
    except NoBinary as exc:
        print("new_retro: %s" % exc, file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    sys.exit(main())
