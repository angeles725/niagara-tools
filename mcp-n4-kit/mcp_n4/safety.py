"""Safety primitives for the write tools: tokens, scope, journal and audit log.

Pure logic plus append-only file I/O. Importing this module has no side effects
and constructing the classes touches no file: the state directory is created on
the first append.
"""
import hashlib
import hmac
import json
import os
import secrets
import time

REDACTED = "***"
_SECRET_MARKERS = ("pass", "secret", "token", "credential")
#: Args that are not part of what a confirmation token vouches for.
_UNBOUND_ARGS = ("dry_run", "confirmation_token")


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
        if len(parts) != 3 or not parts[0].isdigit():
            raise SafetyError("confirmation_token is missing or malformed: run a dry run "
                              "first and pass the token it returns")
        expiry, nonce, mac = int(parts[0]), parts[1], parts[2]
        if not hmac.compare_digest(mac, self._mac(tool, args, plan_hash, expiry, nonce)):
            raise SafetyError("confirmation_token does not match this tool, these arguments "
                              "and this plan: run the dry run again")
        if self.clock() >= expiry:
            self._pending.pop(nonce, None)
            raise SafetyError("confirmation_token expired: run the dry run again")
        if self._pending.pop(nonce, None) is None:
            raise SafetyError("confirmation_token already used: run the dry run again")


class WriteScope:
    """Allowlist of ORD prefixes a write may touch (boundary-aware)."""

    def __init__(self, prefixes):
        self.prefixes = [p.rstrip("/") for p in prefixes]

    def check(self, ord_str):
        if not self.prefixes:
            raise SafetyError("no --write-scope configured: every write is refused")
        if not isinstance(ord_str, str) or not ord_str.startswith("station:") \
                or ".." in ord_str.split("/"):
            raise SafetyError("ord %r is not a plain station ORD" % (ord_str,))
        target = ord_str.rstrip("/")
        if not any(target == p or target.startswith(p + "/") for p in self.prefixes):
            raise SafetyError("ord %r is outside the write scope" % ord_str)


class _JsonlFile:
    def __init__(self, state_dir, filename):
        self.state_dir, self.path = state_dir, os.path.join(state_dir, filename)

    def append(self, entry):
        os.makedirs(self.state_dir, mode=0o700, exist_ok=True)
        os.chmod(self.state_dir, 0o700)
        fd = os.open(self.path, os.O_WRONLY | os.O_APPEND | os.O_CREAT, 0o600)
        try:
            os.fchmod(fd, 0o600)
            os.write(fd, (json.dumps(entry) + "\n").encode())
        finally:
            os.close(fd)


class Journal(_JsonlFile):
    """Append-only record of executed writes with the ops that undo them."""

    def __init__(self, state_dir):
        super().__init__(state_dir, "journal.jsonl")


class AuditLog(_JsonlFile):
    """Append-only record of every write call: planned, executed or refused."""

    def __init__(self, state_dir):
        super().__init__(state_dir, "audit.jsonl")
