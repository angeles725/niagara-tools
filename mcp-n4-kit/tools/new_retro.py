#!/usr/bin/env python3
"""Write the session retro for an mcp-n4 session and register it in retros/INDEX.md.

Reads the server's `audit.jsonl` + `journal.jsonl` (the same derivation as the
`n4_session_retro_draft` tool), writes `YYYY-MM-DD-<station>-session.md` and appends
the INDEX row as `pending`. Proposes only: it never edits the kit and never stages
GitHub issues (do that by hand). Idempotent: an existing retro is never overwritten
without `--force`. `--dry-run` prints the retro and writes nothing.

Exit codes: 0 ok, 2 usage, 3 refused (exists without --force, bad input, no INDEX).
Importing this module only prepends the kit directory to `sys.path` (so `mcp_n4` resolves);
it reads and writes no files. Both outputs are written atomically (temp file + rename).
"""
import argparse
import datetime
import os
import re
import stat
import sys
import tempfile

KIT_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, KIT_DIR)

from mcp_n4 import retro  # noqa: E402

DEFAULT_OUT_DIR = os.path.join(KIT_DIR, "retros")
_STATION = re.compile(r"[A-Za-z0-9][A-Za-z0-9._-]{0,63}")
_DATE = re.compile(r"\d{4}-\d{2}-\d{2}")


class Refused(Exception):
    pass


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
        umask = os.umask(0)
        os.umask(umask)
        return 0o644 & ~umask


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
    try:
        result = retro.draft(args.state_dir, station=args.station, date=date, since=args.since)
    except ValueError as exc:
        raise Refused(str(exc))
    deltas = retro.count_deltas(result["markdown"])
    name = "%s-%s-session.md" % (date, args.station)
    row = "| %s | %s | %s | pending | %d |" % (name, args.station, date, deltas)
    if args.dry_run:
        out.write(result["markdown"] + "\nINDEX row: %s\n" % row)
        return
    path, index = os.path.join(args.out_dir, name), os.path.join(args.out_dir, "INDEX.md")
    if not os.path.isfile(index):
        raise Refused("%s not found: create the retros directory with its INDEX.md first" % index)
    if os.path.exists(path) and not args.force:
        raise Refused("%s already exists: pass --force to rewrite it" % path)
    with open(index, encoding="utf-8") as fh:
        new_index = _index_rows(fh.read(), name, row)
    _atomic_write(path, result["markdown"])
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
    return 0


if __name__ == "__main__":
    sys.exit(main())
