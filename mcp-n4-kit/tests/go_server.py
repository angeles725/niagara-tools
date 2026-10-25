"""Black-box harness: the tool tests drive the Go `mcp-n4` binary over stdio.

`GoServer` is a long-lived `mcp-n4` subprocess spoken to one JSON line per request. The binary is
`MCP_N4_GO_BIN`, else `go/bin/mcp-n4` at the repository root; a missing binary raises
`GoBinaryMissing`, so a run without the server never passes silently. `make_server(**kw)` takes
the options the server takes on its command line (`server_args`) and returns a `GoServer`.

Since S17 the Go binary is the only mcp-n4 server (docs/go-port-plan.md section 23).
"""
import itertools
import json
import os
import queue
import subprocess
import tempfile
import threading

TESTS_DIR = os.path.dirname(os.path.abspath(__file__))
KIT_DIR = os.path.dirname(TESTS_DIR)
REPO_DIR = os.path.dirname(KIT_DIR)
DEFAULT_BINARY = os.path.join(REPO_DIR, "go", "bin", "mcp-n4")
REPLY_TIMEOUT = 60.0


class GoBinaryMissing(RuntimeError):
    """There is no Go server binary to run."""


class GoServerError(RuntimeError):
    """The subprocess broke the one-line-per-reply contract, timed out or exited."""


class GoServerStartupError(ValueError):
    """The server refused its options at start (exit 2)."""


# Option keyword -> command-line flag of the server. Lists repeat the flag.
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
    """The command-line form of the options `kw`; TypeError for an option it cannot express."""
    args = []
    for key, value in kw.items():
        if key not in _FLAGS:
            raise TypeError("GoServer: %s has no command-line form" % key)
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
            "the Go server has no executable at %s; build it (cd go && go build -o bin/ "
            "./cmd/mcp-n4) or set MCP_N4_GO_BIN" % path)
    return path


class GoServer:
    """Adapter over a long-lived stdio server process.

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
        return self._exchange(line.rstrip("\n"))

    def dispatch(self, msg):
        out = self._exchange(json.dumps(msg))
        return None if out is None else json.loads(out)

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


def make_server(binary=None, **kw):
    """The server under test: the Go binary started with the options `kw`."""
    return GoServer(binary=binary, **kw)


def close_server(srv):
    srv.close()
