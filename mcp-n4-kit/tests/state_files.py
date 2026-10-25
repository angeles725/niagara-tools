"""The server's state files as the black-box tests write and read them (stdlib only).

The tests arrange `journal.jsonl` and `audit.jsonl` (crafted or pre-existing records) and read
back the journal's merged view of a batch. The Go server owns the real format; these helpers
write what it reads and merge what it writes, so the black-box tests need no server code:

- `append` creates the state dir 0700 and a file 0600, as the server does, and appends one JSON
  line (an existing file is appended to as it is);
- `Journal.read` merges one batch the way the journal does: the first intent, the last result,
  the relink and component write-ahead ops, and `state` `in-doubt` when the result is missing
  or flagged `in_doubt`.
"""
import json
import os

REDACTED = "***"
_SECRET_MARKERS = ("pass", "secret", "token", "credential")


def redact_args(value):
    """Copy of `value` with the content of secret-looking keys replaced by `***`."""
    if isinstance(value, dict):
        return {k: REDACTED if any(m in str(k).lower() for m in _SECRET_MARKERS)
                else redact_args(v) for k, v in value.items()}
    if isinstance(value, list):
        return [redact_args(v) for v in value]
    return value


def canonical(obj):
    """Deterministic JSON text: sorted keys, no whitespace (what plan hashes cover)."""
    return json.dumps(obj, sort_keys=True, separators=(",", ":"))


class _JsonlFile:
    def __init__(self, state_dir, filename):
        self.state_dir, self.path = state_dir, os.path.join(state_dir, filename)

    def append(self, entry):
        if not os.path.isdir(self.state_dir):
            os.makedirs(self.state_dir, mode=0o700, exist_ok=True)
            os.chmod(self.state_dir, 0o700)
        fd = os.open(self.path, os.O_WRONLY | os.O_APPEND | os.O_CREAT, 0o600)
        try:
            os.write(fd, (json.dumps(entry) + "\n").encode())
        finally:
            os.close(fd)

    def entries(self):
        """Every JSON object line; a missing file, an undecodable or torn line is skipped."""
        try:
            with open(self.path, "rb") as fh:
                data = fh.read()
        except OSError:
            return []
        out = []
        for raw in data.splitlines():
            try:
                entry = json.loads(raw.decode("utf-8"))
            except ValueError:
                continue
            if isinstance(entry, dict):
                out.append(entry)
        return out


class AuditLog(_JsonlFile):
    def __init__(self, state_dir):
        super().__init__(state_dir, "audit.jsonl")


class Journal(_JsonlFile):
    def __init__(self, state_dir):
        super().__init__(state_dir, "journal.jsonl")

    def _views(self):
        intents, results, relinks, comps = {}, {}, {}, {}
        for entry in self.entries():
            bid, phase = entry.get("batch_id"), entry.get("phase")
            if phase == "intent":
                intents.setdefault(bid, entry)
            elif phase == "relink-intent":
                relinks.setdefault(bid, entry.get("ops") or [])
            elif phase == "component-intent":
                comps.setdefault(bid, []).extend(entry.get("ops") or [])
            elif phase == "result":
                results[bid] = entry
        views = {}
        for bid, intent in intents.items():
            view = {k: v for k, v in intent.items() if k != "phase"}
            view.update(inverse=None, accepted=None, verdict=None)
            if bid in relinks:
                view["relink_ops"] = relinks[bid]
            if bid in comps:
                view["component_ops"] = comps[bid]
            result = results.get(bid)
            if result is not None:
                view.update({k: v for k, v in result.items() if k not in ("phase", "ts")})
                view["result_ts"] = result.get("ts")
            view["state"] = ("in-doubt" if result is None or result.get("in_doubt")
                             else "completed")
            views[bid] = view
        return views

    def read(self, batch_id):
        """Merged intent+result view of one batch, or None when it was never journaled."""
        return self._views().get(batch_id)

    def rollbacks_of(self, batch_id):
        """Merged views of every batch recorded as rolling back `batch_id`."""
        return [v for v in self._views().values() if v.get("rollback_of") == batch_id]
