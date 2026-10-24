"""Safety primitives for the write tools: tokens, scope, journal and audit log.

Pure logic plus append-only file I/O. Importing this module has no side effects
and constructing the classes touches no file: the state directory is created on
the first append.
"""
import hashlib
import hmac
import json
import os
import re
import secrets
import stat
import time

REDACTED = "***"
_SECRET_MARKERS = ("pass", "secret", "token", "credential")
#: Args that are not part of what a confirmation token vouches for.
_UNBOUND_ARGS = ("dry_run", "confirmation_token")
#: ASCII digits only: str.isdigit() also accepts superscripts and other scripts.
_EXPIRY = re.compile(r"[0-9]{1,15}")


#: Where the journal and audit live unless the operator passes --state-dir (one owner).
DEFAULT_STATE_DIR = "~/.local/state/mcp-n4"

#: Leading text of each refusal. The raise sites build their messages from these and the
#: session retro classifies audit reasons with them, so the two cannot drift apart.
REASON_TOKEN_MISSING = "confirmation_token is missing"
REASON_TOKEN_MISMATCH = "confirmation_token does not match this tool"
REASON_TOKEN_EXPIRED = "confirmation_token expired"
REASON_TOKEN_REUSED = "confirmation_token already used"
REASON_IDENTITY = "station identity not verified"
REASON_SCOPE_NONE = "no --write-scope configured"
REASON_SCOPE_PLAIN = "is not a plain station ORD"
REASON_SCOPE_OUTSIDE = "is outside the write scope"
REASON_BUDGET = "write budget exhausted"
REASON_BUDGET_SMALL = "write budget too small"
REASON_NOT_CONNECTED = "not connected"
REASON_PARTIAL_STATUS = "Status slots are written whole"
REASON_ACTION = "is not allowed in this version"
REASON_VALUE_TYPE = "value_type must be one of"
REASON_TYPE_SPEC = "type must look like"
REASON_TIER = "version tier refuses writes"
REASON_JOURNAL_UNREADABLE = "journal line unreadable"


class SafetyError(Exception):
    """A refusal whose message is safe to show to the model (never holds a secret)."""


def canonical(obj):
    """Deterministic JSON text: sorted keys, no whitespace."""
    return json.dumps(obj, sort_keys=True, separators=(",", ":"))


def bound_args(args):
    """The args a token is bound to: everything except `dry_run` and the token itself."""
    return {k: v for k, v in args.items() if k not in _UNBOUND_ARGS}


def redact_args(value):
    """Copy of `value` with the content of secret-looking keys replaced by `***`."""
    if isinstance(value, dict):
        return {k: REDACTED if any(m in str(k).lower() for m in _SECRET_MARKERS)
                else redact_args(v) for k, v in value.items()}
    if isinstance(value, list):
        return [redact_args(v) for v in value]
    return value


class ConfirmationTokens:
    """Single-use HMAC-SHA256 tokens under a per-process random key.

    A token is `<expiry>.<nonce>.<mac>`; the mac covers the tool name, the canonical
    args (minus `dry_run`/`confirmation_token`), the plan hash, the expiry and the nonce.
    """

    def __init__(self, ttl=300, clock=time.time):
        self.ttl, self.clock = ttl, clock
        self._key = secrets.token_bytes(32)
        self._pending = {}  # nonce -> expiry, for issued and not yet spent tokens

    def _mac(self, tool, args, plan_hash, expiry, nonce):
        msg = canonical([tool, canonical(bound_args(args)), plan_hash, expiry, nonce])
        return hmac.new(self._key, msg.encode(), hashlib.sha256).hexdigest()

    def issue(self, tool, args, plan_hash):
        """Return `(token, expires_at)` for this exact tool, args and plan."""
        now = self.clock()
        self._pending = {n: e for n, e in self._pending.items() if e > now}
        expiry, nonce = int(now) + self.ttl, secrets.token_hex(8)
        self._pending[nonce] = expiry
        return "%d.%s.%s" % (expiry, nonce, self._mac(tool, args, plan_hash, expiry, nonce)), expiry

    def consume(self, tool, args, plan_hash, token):
        """Spend `token`; raise SafetyError unless it is authentic, bound, live and unused."""
        parts = token.split(".") if isinstance(token, str) else []
        if len(parts) != 3 or not _EXPIRY.fullmatch(parts[0]) \
                or not all(p.isascii() for p in parts):
            raise SafetyError(REASON_TOKEN_MISSING + " or malformed: run a dry run "
                              "first and pass the token it returns")
        expiry, nonce, mac = int(parts[0]), parts[1], parts[2]
        if not hmac.compare_digest(mac.encode(),
                                   self._mac(tool, args, plan_hash, expiry, nonce).encode()):
            raise SafetyError(REASON_TOKEN_MISMATCH + ", these arguments "
                              "and this plan: run the dry run again")
        if self.clock() >= expiry:
            self._pending.pop(nonce, None)
            raise SafetyError(REASON_TOKEN_EXPIRED + ": run the dry run again")
        if self._pending.pop(nonce, None) is None:
            raise SafetyError(REASON_TOKEN_REUSED + ": run the dry run again")


#: The only ORD form a write may target: one station slot path, nothing chained after it.
SLOT_ORD_PREFIX = "station:|slot:/"


def _plain_slot_ord(ord_str):
    """True for `station:|slot:/<path>` with no further ORD hop (`|h:`, `|slot:`, `|bql:`).

    A chained ORD resolves past its slot path, so a prefix match on its text says nothing
    about the component it reaches (audit 2026-10-03 F2).
    """
    return isinstance(ord_str, str) and ord_str.startswith(SLOT_ORD_PREFIX) \
        and "|" not in ord_str[len(SLOT_ORD_PREFIX):] \
        and ".." not in ord_str.split("/")


class WriteScope:
    """Allowlist of ORD prefixes a write may touch (boundary-aware, plain slot ORDs only)."""

    def __init__(self, prefixes):
        for prefix in prefixes:
            if not _plain_slot_ord(prefix):
                raise SafetyError("--write-scope %r must be a plain slot ORD such as "
                                  "'station:|slot:/Drivers' (no '|' after the slot path)"
                                  % (prefix,))
        self.prefixes = [p.rstrip("/") for p in prefixes]

    def require_any(self):
        """For writes that target the whole station rather than one ORD."""
        if not self.prefixes:
            raise SafetyError(REASON_SCOPE_NONE + ": every write is refused")

    def check(self, ord_str):
        if not self.prefixes:
            raise SafetyError(REASON_SCOPE_NONE + ": every write is refused")
        if not _plain_slot_ord(ord_str):
            raise SafetyError("ord %r %s (station:|slot:/<path>, nothing chained after the "
                              "slot path)" % (ord_str, REASON_SCOPE_PLAIN))
        target = ord_str.rstrip("/")
        if not any(target == p or target.startswith(p + "/") for p in self.prefixes):
            raise SafetyError("ord %r %s" % (ord_str, REASON_SCOPE_OUTSIDE))


STATE_FILES = ("journal.jsonl", "audit.jsonl")


def loose_file_reason(st):
    """Why an existing journal/audit file (an `os.stat_result`) is not private, else None."""
    if st.st_uid != os.getuid():
        return "is owned by another user"
    if st.st_mode & 0o077:
        return "is readable or writable by group/others (mode %o)" % stat.S_IMODE(st.st_mode)
    return None


def check_state_dir(path):
    """Refuse a pre-existing state dir the server cannot trust; never chmod it.

    A missing dir is fine (the first append creates it as 0700). An existing one must
    be a directory owned by the current user and not group/world-writable.
    """
    try:
        st = os.stat(path)
    except FileNotFoundError:
        return
    except OSError as exc:
        raise SafetyError("--state-dir %s cannot be inspected (%s)" % (path, exc.strerror))
    if not stat.S_ISDIR(st.st_mode):
        raise SafetyError("--state-dir %s exists and is not a directory" % path)
    if st.st_uid != os.getuid():
        raise SafetyError("--state-dir %s is owned by another user: choose a directory you "
                          "own (the server never changes its permissions)" % path)
    if st.st_mode & 0o022:
        raise SafetyError("--state-dir %s is group/world-writable (mode %o): run `chmod 700 "
                          "%s` yourself or choose another directory (the server never changes "
                          "the permissions of a directory it did not create)"
                          % (path, stat.S_IMODE(st.st_mode), path))
    for name in STATE_FILES:  # files we create are 0600; a loose one was changed by someone
        file_path = os.path.join(path, name)
        try:
            reason = loose_file_reason(os.stat(file_path))
        except OSError:
            continue
        if reason:
            raise SafetyError("%s %s: run `chmod 600 %s` yourself or choose another "
                              "--state-dir (the server never changes the permissions of an "
                              "existing file)" % (file_path, reason, file_path))


class _JsonlFile:
    def __init__(self, state_dir, filename):
        self.state_dir, self.path = state_dir, os.path.join(state_dir, filename)

    def check_private(self):
        """Raise `SafetyError` when the file exists and is loose (never chmods it)."""
        try:
            reason = loose_file_reason(os.stat(self.path))
        except OSError:
            return  # missing: append will create it private
        if reason:
            raise SafetyError("%s %s: run `chmod 600 %s` yourself (the server never changes "
                              "the permissions of an existing file)"
                              % (self.path, reason, self.path))

    def append(self, entry):
        if not os.path.isdir(self.state_dir):  # only a directory we create is chmodded
            os.makedirs(self.state_dir, mode=0o700, exist_ok=True)
            os.chmod(self.state_dir, 0o700)
        try:  # a file we create is private from the start
            fd = os.open(self.path, os.O_WRONLY | os.O_APPEND | os.O_CREAT | os.O_EXCL, 0o600)
        except FileExistsError:  # an existing one is never chmodded: it must already be private
            fd = os.open(self.path, os.O_WRONLY | os.O_APPEND)
            reason = loose_file_reason(os.fstat(fd))
            if reason:
                os.close(fd)
                raise PermissionError("%s %s: run `chmod 600 %s` yourself (the server never "
                                      "changes the permissions of an existing file)"
                                      % (self.path, reason, self.path))
        try:
            os.write(fd, (json.dumps(entry) + "\n").encode())
        finally:
            os.close(fd)

    def entries(self):
        """Every parseable JSON object line; a missing file or a torn line is skipped."""
        try:
            with open(self.path) as fh:
                lines = fh.readlines()
        except OSError:
            return []
        out = []
        for line in lines:
            try:
                entry = json.loads(line)
            except ValueError:
                continue
            if isinstance(entry, dict):
                out.append(entry)
        return out

    def _lines(self):
        """`(n, entry)` for every JSON object line, `(n, None)` for a line that is not UTF-8.

        Lines are numbered as readlines() splits them (LF, CRLF and CR) and decoded
        one by one, so one bad byte costs one line, not the file (issues #448, #466). A
        missing file or a torn line is skipped, as `entries()` skips them.
        """
        # Copy of #448's Journal._lines (PR #467): dedupe onto it once #467 merges.
        try:
            with open(self.path, "rb") as fh:
                data = fh.read()
        except OSError:
            return []
        out = []
        for n, raw in enumerate(data.splitlines(), 1):
            try:
                line = raw.decode("utf-8")
            except UnicodeDecodeError:
                out.append((n, None))
                continue
            try:
                entry = json.loads(line)
            except ValueError:
                continue
            if isinstance(entry, dict):
                out.append((n, entry))
        return out


class Journal(_JsonlFile):
    """Append-only write-ahead record: an `intent` before each station op, a `result` after.

    Entries of one batch share `batch_id`. An intent without a result, or a result
    flagged `in_doubt`, means the station state is unknown (state `in-doubt`).
    """

    def __init__(self, state_dir):
        super().__init__(state_dir, "journal.jsonl")

    @staticmethod
    def _merge(intent, result, relink_ops=None, component_ops=None):
        view = {k: v for k, v in intent.items() if k != "phase"}
        view.update(inverse=None, accepted=None, verdict=None)
        # Both are present exactly when their write-ahead record exists (even if empty).
        if relink_ops is not None:  # the second write-ahead record of a rollback's relinks
            view["relink_ops"] = relink_ops
        if component_ops is not None:  # write-ahead records of a rollback's nested adds
            view["component_ops"] = component_ops
        if result is not None:
            view.update({k: v for k, v in result.items() if k not in ("phase", "ts")})
            view["result_ts"] = result.get("ts")
        view["state"] = "in-doubt" if result is None or result.get("in_doubt") else "completed"
        return view

    def _lines(self):
        """`(n, entry)` for every JSON object line, `(n, None)` for a line that is not UTF-8.

        Lines are numbered as readlines() splits them (LF, CRLF and CR) and decoded
        one by one, so one bad byte costs one line, not the file (issue #448). A missing file
        or a torn line is skipped, as `entries()` skips them.
        """
        try:
            with open(self.path, "rb") as fh:
                data = fh.read()
        except OSError:
            return []
        out = []
        for n, raw in enumerate(data.splitlines(), 1):
            try:
                line = raw.decode("utf-8")
            except UnicodeDecodeError:
                out.append((n, None))
                continue
            try:
                entry = json.loads(line)
            except ValueError:
                continue
            if isinstance(entry, dict):
                out.append((n, entry))
        return out

    def _scan(self):
        """`({batch_id: view}, unreadable)`: the merged views and the lines they could not use.

        An unreadable line is left out of every view: one that is not UTF-8 (its batch is
        unknown), or one of a batch phase whose `batch_id` or `rollback_of` cannot key a dict
        or whose component ops cannot extend a list (`TypeError`; its batch is known).
        """
        intents, results, relinks, comps, bad = {}, {}, {}, {}, []
        for n, entry in self._lines():
            if entry is None:
                bad.append({"line": n, "reason": "UnicodeDecodeError", "known": False})
                continue
            bid, phase = entry.get("batch_id"), entry.get("phase")
            if phase not in ("intent", "relink-intent", "component-intent", "result"):
                continue
            try:
                hash(bid)
                if phase in ("intent", "result") and entry.get("rollback_of"):
                    hash(entry["rollback_of"])  # n4_list_batches keys rolled_back_by by it
                ops = list(entry.get("ops") or []) if phase == "component-intent" else None
            except TypeError:
                bad.append({"line": n, "reason": "TypeError", "known": True, "batch_id": bid,
                            "rollback_of": entry.get("rollback_of")})
                continue
            if phase == "intent":
                intents.setdefault(bid, entry)
            elif phase == "relink-intent":
                relinks.setdefault(bid, entry.get("ops") or [])
            elif phase == "component-intent":
                comps.setdefault(bid, []).extend(ops)
            else:
                results[bid] = entry
        views = {bid: self._merge(i, results.get(bid), relinks.get(bid), comps.get(bid))
                 for bid, i in intents.items()}
        return views, bad

    def _views(self):
        return self._scan()[0]

    def scan(self):
        """`(views, unreadable)`: `views()` and the unreadable lines, from one read."""
        views, bad = self._scan()
        return list(views.values()), bad

    def views(self):
        """Merged views of every journaled batch, in journal order."""
        return list(self._views().values())

    def unreadable_for(self, batch_id):
        """The unreadable lines that may belong to `batch_id`: the rollback check (#448).

        A line may belong to it when its batch is unknown (not UTF-8), when its `batch_id` is
        `batch_id`, or when its `rollback_of` is: a lost rollback of a batch could let it be
        rolled back twice. A line of another known batch blocks only that batch.
        """
        return [u for u in self._scan()[1] if not u["known"] or u["batch_id"] == batch_id
                or u["rollback_of"] == batch_id]

    def read(self, batch_id):
        """Merged intent+result view of one batch, or None when it was never journaled."""
        return self._views().get(batch_id)

    def rollbacks_of(self, batch_id):
        """Merged views of every batch recorded as rolling back `batch_id`."""
        return [v for v in self._views().values() if v.get("rollback_of") == batch_id]


class AuditLog(_JsonlFile):
    """Append-only record of every write call: planned, executed or refused."""

    def __init__(self, state_dir):
        super().__init__(state_dir, "audit.jsonl")

    def scan(self):
        """`(entries, unreadable)`: every JSON object line, and `{"line": n, "reason":
        "UnicodeDecodeError"}` for each line that is not UTF-8 (left out; issue #466)."""
        entries, bad = [], []
        for n, entry in self._lines():
            if entry is None:
                bad.append({"line": n, "reason": "UnicodeDecodeError"})
            else:
                entries.append(entry)
        return entries, bad
