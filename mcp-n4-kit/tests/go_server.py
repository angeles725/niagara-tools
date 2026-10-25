"""Black-box harness for the Go port (docs/go-port-plan.md, slice S0).

`MCP_N4_IMPL` selects the server the tool tests drive:

- unset or `go` (the default since S16): `GoServer`, a long-lived `mcp-n4` subprocess spoken to
  over stdio, one JSON line per request. The binary is `MCP_N4_GO_BIN`, else `go/bin/mcp-n4` at
  the repository root. A missing binary raises `GoBinaryMissing`: a Go run never passes
  silently on Python;
- `python` (the fallback, kept one release): the in-process `mcp_n4.server.Server`.

Tests that cannot run black-box carry `@go_twin(package, category)`: they are skipped under
Go and name the Go test that replaces them. The per-file inventory lives in the plan doc and
`test_go_harness` keeps the two in step.

A black-box test that reaches a part the binary does not port yet is skipped, not failed: a
tool answering the `UNPORTED` tool error, or a start refused with it (`--allow-writes` before
S10). Only that exact text skips, so the skip disappears as each slice lands.

Any stdio server can stand in for the binary: pointing `MCP_N4_GO_BIN` at a wrapper that runs
`python3 -m mcp_n4.server "$@"` checks that every unmarked test really is black-box.
"""
import itertools
import json
import os
import queue
import re
import subprocess
import tempfile
import threading
import unittest

TESTS_DIR = os.path.dirname(os.path.abspath(__file__))
KIT_DIR = os.path.dirname(TESTS_DIR)
REPO_DIR = os.path.dirname(KIT_DIR)
DEFAULT_BINARY = os.path.join(REPO_DIR, "go", "bin", "mcp-n4")
IMPLS = ("python", "go")


def resolve_impl(environ=os.environ):
    """The server `MCP_N4_IMPL` selects: Go unless it says otherwise (S16)."""
    return environ.get("MCP_N4_IMPL") or "go"


IMPL = resolve_impl()
GO = IMPL == "go"
REPLY_TIMEOUT = 60.0
#: The tool error / start refusal text of a part the Go binary does not port yet
#: (jsonrpc.NotImplemented in go/internal/mcpn4).
UNPORTED = "is not implemented in this version of the Go server"

#: Why a test cannot run against the binary. Each skipped test names its future Go twin.
CATEGORIES = {
    "unit": "calls a Python module function or class directly; the twin is a Go unit test",
    "internals": "reads or writes in-process server state (srv.ctx: session, env, journal)",
    "injection": "replaces behaviour in-process (mock/patch, clock, client_factory, opener, "
                 "custom tools)",
    "in-process": "uses an entry point with no wire form (parse_args, main, serve, _validate, "
                  "docstrings, the Python TOOLS table, an in-process Server)",
}

#: (test module, class name, test name, Go twin, category) of every @go_twin test.
REGISTRY = set()
#: `module.Class.test` -> reason of every @go_twin(..., python_only=...) test: no Go twin.
PYTHON_ONLY = {}


class GoBinaryMissing(RuntimeError):
    """The Go server (the default, or MCP_N4_IMPL=go) was asked for but there is no binary."""


class GoServerError(RuntimeError):
    """The subprocess broke the one-line-per-reply contract, timed out or exited."""


class GoServerStartupError(ValueError):
    """The server refused its options at start (exit 2), like `Server(...)` raising ValueError."""


def skip_if_unported(reply):
    """Raise SkipTest when `reply` (a decoded reply object) is the unported-tool error."""
    result = reply.get("result") if isinstance(reply, dict) else None
    if isinstance(result, dict) and result.get("isError") is True:
        content = result.get("content")
        text = content[0].get("text") if isinstance(content, list) and content \
            and isinstance(content[0], dict) else None
        if isinstance(text, str) and text.endswith(" " + UNPORTED):
            raise unittest.SkipTest("not ported to Go yet: " + text)


# Server(...) keyword -> command-line flag (server.parse_args). Lists repeat the flag.
_FLAGS = {
    "allow_writes": "--allow-writes", "allow_http": "--allow-http-for-tests",
    "write_scopes": "--write-scope", "state_dir": "--state-dir", "token_ttl": "--token-ttl",
    "max_writes": "--max-writes", "credential_env": "--credential-env",
    "insecure_tls": "--insecure-tls", "allow_tier_b": "--allow-tier-b",
    "allow_tier_c": "--allow-tier-c", "progress_file": "--progress-file",
    "auth_cooldown": "--auth-cooldown", "http_timeout": "--http-timeout",
    "load_wait": "--load-wait",
}
_BOOLS = {"allow_writes", "allow_http"}
_LISTS = {"write_scopes", "insecure_tls", "allow_tier_b", "allow_tier_c"}
# Kept in the child environment when a test passes its own credential `env`.
_BASE_ENV = ("PATH", "HOME", "TMPDIR", "LANG", "SYSTEMROOT")


def server_args(stations=None, station_homes=None, **kw):
    """The command-line form of `Server(**kw)`; TypeError for an option it cannot express."""
    args = []
    for key, value in kw.items():
        if key not in _FLAGS:
            raise TypeError("GoServer: %s has no command-line form; mark the test "
                            "@go_twin(..., 'injection')" % key)
        if value is None:
            continue
        flag = _FLAGS[key]
        if key in _BOOLS:
            args += [flag] if value else []
        elif key in _LISTS:
            for item in value:
                args += [flag, str(item)]
        else:
            args += [flag, str(value)]
    for name, url in (stations or {}).items():
        args += ["--station", "%s=%s" % (name, url)]
    for name, path in (station_homes or {}).items():
        args += ["--station-home", "%s=%s" % (name, path)]
    return args


def resolve_binary(binary=None):
    path = binary or os.environ.get("MCP_N4_GO_BIN") or DEFAULT_BINARY
    if not (os.path.isfile(path) and os.access(path, os.X_OK)):
        raise GoBinaryMissing(
            "the Go server (MCP_N4_IMPL=go, the default) has no executable at %s; build it "
            "(cd go && go build -o bin/ ./cmd/mcp-n4), set MCP_N4_GO_BIN, or select the Python "
            "fallback with MCP_N4_IMPL=python" % path)
    return path


class GoServer:
    """`Server`-shaped adapter over a long-lived stdio server process.

    `dispatch(msg)` and `handle_line(line)` write one line, then a ping sentinel, and read
    replies up to the sentinel's: zero lines before it means "no reply" (a notification),
    one line is the reply, more is a protocol error.
    """

    def __init__(self, binary=None, command=None, cwd=None, env=None, timeout=REPLY_TIMEOUT,
                 **server_kw):
        argv = (list(command) if command else [resolve_binary(binary)]) + server_args(**server_kw)
        if env is None:
            child_env = dict(os.environ)
        else:
            child_env = {k: os.environ[k] for k in _BASE_ENV if k in os.environ}
            child_env.update(env)
        self.timeout = timeout
        self._sync = itertools.count(1)
        self._lines = queue.Queue()
        self._stderr = tempfile.TemporaryFile()
        self._proc = subprocess.Popen(argv, cwd=cwd, env=child_env, stdin=subprocess.PIPE,
                                      stdout=subprocess.PIPE, stderr=self._stderr)
        threading.Thread(target=self._read_stdout, daemon=True).start()
        try:
            self._exchange(None)
        except GoServerError as exc:
            self.close(keep_stderr=True)
            text = self.stderr()
            self._stderr.close()
            if self._proc.returncode == 2:
                if text.endswith(" " + UNPORTED):
                    raise unittest.SkipTest("not ported to Go yet: " + text)
                raise GoServerStartupError(text or str(exc)) from None
            raise

    def _read_stdout(self):
        for raw in self._proc.stdout:
            self._lines.put(raw.decode("utf-8"))
        self._lines.put(None)  # EOF

    def stderr(self):
        try:
            self._stderr.seek(0)
            return self._stderr.read().decode("utf-8", "replace").strip()
        except ValueError:  # already closed
            return ""

    def _exchange(self, line):
        if self._proc.stdin is None or self._proc.stdin.closed:
            raise GoServerError("GoServer is closed")
        sync_id = "__go_server_sync_%d" % next(self._sync)
        sentinel = json.dumps({"jsonrpc": "2.0", "id": sync_id, "method": "ping"})
        payload = (sentinel + "\n") if line is None else (line + "\n" + sentinel + "\n")
        try:
            self._proc.stdin.write(payload.encode("utf-8"))
            self._proc.stdin.flush()
        except (BrokenPipeError, OSError) as exc:
            raise GoServerError("server stdin closed: %s; stderr: %s"
                                % (exc, self.stderr())) from None
        replies = []
        while True:
            try:
                out = self._lines.get(timeout=self.timeout)
            except queue.Empty:
                raise GoServerError("no reply within %.0f s" % self.timeout) from None
            if out is None:
                raise GoServerError("server exited (code %s); stderr: %s"
                                    % (self._proc.poll(), self.stderr()))
            try:
                obj = json.loads(out)
            except ValueError:
                raise GoServerError("stdout line is not JSON: %r" % out) from None
            if isinstance(obj, dict) and obj.get("id") == sync_id:
                break
            replies.append(out.rstrip("\n"))
        if len(replies) > 1:
            raise GoServerError("one request produced %d reply lines: %r"
                                % (len(replies), replies))
        return replies[0] if replies else None

    def handle_line(self, line):
        out = self._exchange(line.rstrip("\n"))
        if out is not None:
            skip_if_unported(json.loads(out))
        return out

    def dispatch(self, msg):
        out = self._exchange(json.dumps(msg))
        if out is None:
            return None
        reply = json.loads(out)
        skip_if_unported(reply)
        return reply

    def close(self, keep_stderr=False):
        proc = self._proc
        if proc.stdin is not None and not proc.stdin.closed:
            try:
                proc.stdin.close()
            except OSError:
                pass
        if proc.poll() is None:
            try:
                proc.wait(timeout=10)
            except subprocess.TimeoutExpired:
                proc.kill()
                proc.wait()
        if proc.stdout is not None:
            proc.stdout.close()
        if not keep_stderr:
            self._stderr.close()

    def __del__(self):
        try:
            if self._proc.poll() is None:
                self._proc.kill()
        except Exception:  # interpreter shutdown: best effort only
            pass


def make_server(impl=None, binary=None, **kw):
    """The server under test: the Go binary (default) or in-process Python (MCP_N4_IMPL=python)."""
    impl = impl or IMPL
    if impl == "python":
        from mcp_n4 import server
        return server.Server(**kw)
    if impl == "go":
        return GoServer(binary=binary, **kw)
    raise ValueError("MCP_N4_IMPL=%r: expected one of %s" % (impl, ", ".join(IMPLS)))


def close_server(srv):
    if isinstance(srv, GoServer):
        srv.close()
    else:
        srv.ctx.close()


def twin_name(package, test_name):
    words = test_name[len("test_"):].split("_") if test_name.startswith("test_") else [test_name]
    return "%s.Test%s" % (package, "".join(w[:1].upper() + w[1:] for w in words))


def go_twin(package, category, twin=None, python_only=None):
    """Skip a test (or every test of a class) under Go, naming the Go twin that replaces it.

    The twin is `package.Test<CamelCase test name>` unless `twin` names it explicitly as
    `package.TestX` (a Go name that capitalises differently or lives in another package);
    `twin` is for one test only, never a class. `python_only` instead says, in words, why the
    test pins Python-only behaviour no Go test can have (docs/go-port-plan.md section 21).
    """
    if category not in CATEGORIES:
        raise ValueError("unknown go_twin category %r" % category)
    if twin is not None and not re.match(r"^[a-z]+\.Test[A-Za-z0-9]+$", twin):
        raise ValueError("go_twin twin %r is not package.TestName" % twin)
    if python_only is not None and (twin is not None or not isinstance(python_only, str)
                                    or not python_only.strip()):
        raise ValueError("go_twin python_only needs a reason and no twin")

    def mark(obj):
        if isinstance(obj, type):
            if twin is not None:
                raise ValueError("go_twin(twin=...) names one test, not class %s" % obj.__name__)
            names = sorted(n for n in vars(obj) if n.startswith("test"))
            reason = "Go twins %s.Test* (%s)" % (package, category)
            for name in names:
                REGISTRY.add((obj.__module__, obj.__name__, name, twin_name(package, name),
                              category))
                if python_only:
                    PYTHON_ONLY["%s.%s.%s" % (obj.__module__, obj.__name__, name)] = python_only
        else:
            cls_name = obj.__qualname__.rpartition(".")[0]
            name = twin or twin_name(package, obj.__name__)
            reason = "Go twin %s (%s)" % (name, category)
            REGISTRY.add((obj.__module__, cls_name, obj.__name__, name, category))
            if python_only:
                PYTHON_ONLY["%s.%s.%s" % (obj.__module__, cls_name, obj.__name__)] = python_only
        if python_only:
            reason = "Python only (%s): %s" % (category, python_only)
        return unittest.skipIf(GO, reason)(obj)
    return mark


GO_TESTS_DIR = os.path.join(REPO_DIR, "go", "internal", "mcpn4")
_GO_TEST_FUNC = re.compile(r"^func (Test\w+)\(t \*testing\.T\)", re.M)


def go_tests(root=GO_TESTS_DIR):
    """{package: {TestX, ...}}: every `func TestX(t *testing.T)` of each package's *_test.go."""
    out = {}
    for name in sorted(os.listdir(root)):
        pkg = os.path.join(root, name)
        if not os.path.isdir(pkg):
            continue
        for fname in sorted(os.listdir(pkg)):
            if fname.endswith("_test.go"):
                with open(os.path.join(pkg, fname), encoding="utf-8") as fh:
                    out.setdefault(name, set()).update(_GO_TEST_FUNC.findall(fh.read()))
    return out


def twin_resolves(entry, tests, box_ports):
    """True when a REGISTRY entry's twin is a real Go test of its package, or (box only) a
    replay scenario (go_box.SCENARIOS) whose `ports` names the Python test."""
    _module, _cls, name, twin, _category = entry
    package, _, func = twin.partition(".")
    return func in tests.get(package, ()) or (package == "box" and name in box_ports)


def unresolved_twins():
    """`module.Class.test` of every @go_twin marker whose twin resolves to nothing (a
    python_only marker names no twin, so it never counts)."""
    import go_box
    ports = {s.get("ports", "test_" + s["name"]) for s in go_box.SCENARIOS}
    tests = go_tests()
    return sorted("%s.%s.%s" % e[:3] for e in REGISTRY
                  if "%s.%s.%s" % e[:3] not in PYTHON_ONLY and not twin_resolves(e, tests, ports))


#: Markers whose twin resolves to nothing (docs/go-port-plan.md sections 19 and 21). Frozen and
#: checked both ways: an entry that starts to resolve or disappears must be deleted, and the
#: list never grows past TWIN_DEBT_CEILING.
TWIN_DEBT_CEILING = 2
TWIN_DEBT = frozenset([
    "test_box.TestLockoutSafeOpen.test_interrupt_after_make_still_closes_the_server_session",
    "test_server.TestRobustness.test_serve_exits_cleanly_on_keyboard_interrupt_and_closes",
])


INVENTORY_BEGIN = "<!-- go-skip-inventory:begin -->"
INVENTORY_END = "<!-- go-skip-inventory:end -->"
PLAN_DOC = os.path.join(KIT_DIR, "docs", "go-port-plan.md")


def discovered_ids():
    """`module.Class.test` of every test in the suite (imports every module: fills REGISTRY)."""
    def flat(suite):
        for item in suite:
            if isinstance(item, unittest.TestSuite):
                yield from flat(item)
            else:
                yield item.id()
    return sorted(flat(unittest.defaultTestLoader.discover(TESTS_DIR)))


def inventory_table(ids):
    """The skip inventory as the Markdown table the plan doc publishes."""
    cats = list(CATEGORIES)
    totals, skipped = {}, {}
    for test_id in ids:
        module = test_id.split(".")[0]
        totals[module] = totals.get(module, 0) + 1
    for module, _cls, _name, _twin, category in REGISTRY:
        skipped[(module, category)] = skipped.get((module, category), 0) + 1
    head = ["Test file", "Tests"] + cats + ["Skipped under Go", "Runs under Go"]
    lines = ["| " + " | ".join(head) + " |", "|" + "---|" * len(head)]
    sums = [0] * (len(head) - 1)
    for module in sorted(totals):
        per = [skipped.get((module, c), 0) for c in cats]
        row = [totals[module]] + per + [sum(per), totals[module] - sum(per)]
        sums = [a + b for a, b in zip(sums, row)]
        lines.append("| `%s.py` | %s |" % (module, " | ".join(str(n) for n in row)))
    lines.append("| **Total** | %s |" % " | ".join("**%d**" % n for n in sums))
    return "\n".join(lines)


if __name__ == "__main__":  # print the inventory table for docs/go-port-plan.md
    import go_server  # the module the tests import, which holds their REGISTRY
    print(go_server.inventory_table(go_server.discovered_ids()))
