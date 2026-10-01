#!/usr/bin/env python3
"""Live smoke runner for mcp-n4-kit: drives the REAL MCP server over stdio.

Plan only by default (prints the scenario, contacts nothing). With `--apply` it
starts `python3 -m mcp_n4.server --allow-writes ...` as a subprocess and runs the
kitControl thermostat scenario (niagara-research B1199) through the public tool
interface only: every write is dry run -> confirmation token -> execute -> verdict.

Credentials reach the server only through the environment variables it reads
(`<PREFIX>_USER` / `<PREFIX>_PASSWORD`); the runner scrubs their values from
everything it prints or writes. Exit 0 only when every required step is verified.
Importing this module has no side effects.
"""
import argparse
import json
import os
import queue
import subprocess
import sys
import threading
import time

KIT_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

SCRATCH = "McpSmoke"
ROOT_ORD = "station:|slot:/"
FOLDER_ORD = ROOT_ORD + SCRATCH
READ_TOOLS = frozenset({"n4_connect", "n4_describe_session", "n4_navigate", "n4_read_slots",
                        "n4_list_links", "n4_find_dangling_outputs"})
NUMERIC, BOOLEAN, COMPARE = "control:NumericWritable", "control:BooleanWritable", \
    "kitControl:GreaterThan"
COMPONENTS = [("Temp", NUMERIC, 1, 1), ("Setpoint", NUMERIC, 1, 6),
              ("Compare", COMPARE, 16, 3), ("Cooling", BOOLEAN, 32, 3)]
LINKS = [("Temp", "out", "Compare", "inA"), ("Setpoint", "out", "Compare", "inB"),
         ("Compare", "out", "Cooling", "in10")]
PROBE_ONLY = ["n4_rollback of the folder removal re-creates a nested subtree: the T4 claim "
              "not yet proven live; its verdict is recorded but does not gate the exit code"]


def write_scopes():
    """The scratch folder plus the root (the folder create/remove targets the root)."""
    return [FOLDER_ORD, ROOT_ORD]


def scrub(text, secrets):
    for secret in sorted((s for s in secrets if s), key=len, reverse=True):
        text = text.replace(secret, "***")
    return text


class SmokeError(Exception):
    """The run cannot continue (server gone, protocol error, tool error)."""


class McpClient:
    """Minimal newline-delimited JSON-RPC client over a server subprocess."""

    def __init__(self, argv, env, timeout):
        self.timeout, self._id, self.stderr_lines = timeout, 0, []
        self.proc = subprocess.Popen(argv, cwd=KIT_DIR, env=env, text=True, bufsize=1,
                                     stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                     stderr=subprocess.PIPE)
        self._lines = queue.Queue()
        for stream, sink in ((self.proc.stdout, self._lines.put),
                             (self.proc.stderr, self.stderr_lines.append)):
            threading.Thread(target=self._pump, args=(stream, sink), daemon=True).start()

    def _pump(self, stream, sink):
        for line in stream:
            sink(line.rstrip("\n"))
        if stream is self.proc.stdout:
            self._lines.put(None)

    def request(self, method, params=None):
        self._id += 1
        msg = {"jsonrpc": "2.0", "id": self._id, "method": method, "params": params or {}}
        self._send(msg)
        deadline = time.monotonic() + self.timeout
        while True:
            try:
                line = self._lines.get(timeout=max(0.01, deadline - time.monotonic()))
            except queue.Empty:
                raise SmokeError("timeout waiting for %s" % method)
            if line is None:
                raise SmokeError("server exited (code %s)" % self.proc.wait(5))
            try:
                reply = json.loads(line)
            except ValueError:
                continue
            if reply.get("id") == self._id:
                if "error" in reply:
                    raise SmokeError("protocol error on %s: %s" % (method, reply["error"]))
                return reply["result"]

    def _send(self, msg):
        try:
            self.proc.stdin.write(json.dumps(msg) + "\n")
            self.proc.stdin.flush()
        except (BrokenPipeError, ValueError, OSError):
            raise SmokeError("server exited (code %s)" % self.proc.poll())

    def start(self):
        self.request("initialize", {"protocolVersion": "2025-06-18", "capabilities": {},
                                    "clientInfo": {"name": "live_smoke", "version": "1"}})
        self._send({"jsonrpc": "2.0", "method": "notifications/initialized"})

    def call(self, tool, **args):
        """Return (is_error, payload): the structured result, or the error text."""
        res = self.request("tools/call", {"name": tool, "arguments": args})
        if res.get("isError"):
            return True, res["content"][0]["text"]
        return False, res["structuredContent"]

    def close(self):
        try:
            self.proc.stdin.close()
        except OSError:
            pass
        try:
            self.proc.wait(5)
        except subprocess.TimeoutExpired:
            self.proc.kill()  # only the server this runner started
            self.proc.wait()
        for stream in (self.proc.stdout, self.proc.stderr):
            stream.close()


class Scenario:
    def __init__(self, client, station, settle, log):
        self.client, self.station, self.settle, self.log = client, station, settle, log
        self.steps, self.aborted = [], False
        self.created = self.saved_with_folder = False
        self.removal_batch = None

    # ---- step plumbing ---------------------------------------------------
    def step(self, name, tool, fn, required=True, probe=False):
        entry = {"name": name, "tool": tool, "required": required, "probe": probe,
                 "dry_run_done": False, "batch_id": None, "verdict": "skipped", "detail": None}
        self.steps.append(entry)
        if self.aborted:
            entry["detail"] = "skipped: an earlier required step did not verify"
        else:
            try:
                entry["verdict"], entry["detail"] = fn(entry)
            except SmokeError as exc:
                entry["verdict"], entry["detail"] = "failed", str(exc)
            if required and entry["verdict"] != "verified":
                self.aborted = True
        self.log("%-9s %s" % (entry["verdict"], name))
        return entry

    def read(self, entry, tool, **args):
        err, out = self.client.call(tool, **args)
        if err:
            raise SmokeError("%s: %s" % (tool, out))
        return out

    def write(self, entry, tool, **args):
        """dry run -> token -> execute; returns the executed reply."""
        err, plan = self.client.call(tool, **args)
        if err:
            raise SmokeError("dry run refused: %s" % plan)
        entry["dry_run_done"], entry["plan_hash"] = True, plan.get("plan_hash")
        err, out = self.client.call(tool, dry_run=False,
                                    confirmation_token=plan["confirmation_token"], **args)
        if err:
            raise SmokeError("execute refused: %s" % out)
        entry["batch_id"] = out.get("batch_id")
        return out

    @staticmethod
    def verdict_of(out, **extra):
        return out.get("verdict", "unverified"), dict(extra, observed=out.get("observed"))

    def children(self, entry):
        out = self.read(entry, "n4_navigate", ord=ROOT_ORD, depth=1)
        return [c["name"] for c in out["children"]]

    def slot_value(self, entry, comp, slot="out"):
        out = self.read(entry, "n4_read_slots", ord="%s/%s" % (FOLDER_ORD, comp))
        found = [s for s in out["slots"] if s["name"] == slot]
        return found[0]["value"] if found else None

    # ---- steps -----------------------------------------------------------
    def connect(self, entry):
        out = self.read(entry, "n4_connect", station=self.station, expected_station=self.station)
        ok = out.get("station_name") == self.station
        return ("verified" if ok else "mismatch"), {"station_name": out.get("station_name")}

    def absent_before(self, entry):
        present = SCRATCH in self.children(entry)
        return ("failed" if present else "verified"), \
            {"detail": "%s already exists: refusing to touch it" % SCRATCH if present else "absent"}

    def create(self, name, type_, ws, parent):
        def run(entry):
            args = dict(parent_ord=parent, name=name, type=type_)
            if ws:
                args["wire_sheet"] = {"x": ws[0], "y": ws[1], "w": 8}
            out = self.write(entry, "n4_create_component", **args)
            if name == SCRATCH:
                self.created = True
            return self.verdict_of(out)
        return run

    def link(self, src, sslot, dst, dslot):
        def run(entry):
            out = self.write(entry, "n4_create_link", source_ord="%s/%s" % (FOLDER_ORD, src),
                             source_slot=sslot, target_ord="%s/%s" % (FOLDER_ORD, dst),
                             target_slot=dslot)
            return self.verdict_of(out)
        return run

    def dangling(self, entry):
        out = self.read(entry, "n4_find_dangling_outputs", ord=FOLDER_ORD, depth=1)
        names = sorted(d["path"].rsplit("/", 1)[-1] for d in out["dangling"])
        return ("verified" if names == ["Cooling"] else "mismatch"), {"dangling": names}

    def set_value(self, comp, value):
        def run(entry):
            out = self.write(entry, "n4_invoke_action", ord="%s/%s" % (FOLDER_ORD, comp),
                             action="set", arg=value, arg_type="baja:Double")
            return self.verdict_of(out)
        return run

    def expect_outputs(self, want):
        def run(entry):
            deadline, seen = time.monotonic() + self.settle, {}
            while True:
                seen = {c: self.slot_value(entry, c) in (True, "true") for c in ("Compare", "Cooling")}
                if all(v is want for v in seen.values()) or time.monotonic() >= deadline:
                    break
                time.sleep(0.25)
            return ("verified" if all(v is want for v in seen.values()) else "mismatch"), \
                {"expected": want, "observed": seen}
        return run

    def save(self, label):
        def run(entry):
            out = self.write(entry, "n4_save_station")
            if out.get("persisted") is True:
                verdict = "verified"
                self.saved_with_folder = label == "first"
            else:
                verdict = "mismatch" if out.get("persisted") is False else "unverified"
            return verdict, {"persisted": out.get("persisted"), "evidence": out.get("evidence")}
        return run

    def remove(self, remember=False):
        def run(entry):
            out = self.write(entry, "n4_remove_component", parent_ord=ROOT_ORD, name=SCRATCH)
            if remember:
                self.removal_batch = entry["batch_id"]
            return self.verdict_of(out)
        return run

    def rollback(self, entry):
        out = self.write(entry, "n4_rollback", batch_id=self.removal_batch)
        return self.verdict_of(out, relinks=out.get("relinks"))

    def remove_again(self):
        present = SCRATCH in self.children({})
        if not present:
            self.steps.append({"name": "remove folder (again)", "tool": "n4_remove_component",
                               "required": False, "probe": False, "dry_run_done": False,
                               "batch_id": None, "verdict": "skipped",
                               "detail": "folder absent after the rollback"})
            self.log("skipped   remove folder (again)")
            return
        self.step("remove folder (again)", "n4_remove_component", self.remove())

    def folder_absent(self, entry):
        present = SCRATCH in self.children(entry)
        return ("failed" if present else "verified"), {"present": present}

    def cleanup(self):
        """Best effort after an abort: remove the folder this run created."""
        self.aborted = False
        try:
            present = SCRATCH in self.children({})
        except SmokeError:
            present = True
        if present:
            self.step("cleanup", "n4_remove_component", self.remove(), required=False)
        if self.saved_with_folder:
            self.step("cleanup save", "n4_save_station", self.save("cleanup"), required=False)

    # ---- the scenario ------------------------------------------------------
    def run(self):
        self.step("connect", "n4_connect", self.connect)
        self.step("folder absent before", "n4_navigate", self.absent_before)
        self.step("create Folder %s" % SCRATCH, "n4_create_component",
                  self.create(SCRATCH, "baja:Folder", None, ROOT_ORD))
        for name, type_, x, y in COMPONENTS:
            self.step("create %s" % name, "n4_create_component",
                      self.create(name, type_, (x, y), FOLDER_ORD))
        for src, sslot, dst, dslot in LINKS:
            self.step("link %s.%s->%s.%s" % (src, sslot, dst, dslot), "n4_create_link",
                      self.link(src, sslot, dst, dslot))
        self.step("find_dangling_outputs == [Cooling]", "n4_find_dangling_outputs", self.dangling)
        self.step("set Temp=30", "n4_invoke_action", self.set_value("Temp", 30))
        self.step("set Setpoint=25", "n4_invoke_action", self.set_value("Setpoint", 25))
        self.step("read outputs == true", "n4_read_slots", self.expect_outputs(True))
        self.step("set Temp=20", "n4_invoke_action", self.set_value("Temp", 20))
        self.step("read outputs == false", "n4_read_slots", self.expect_outputs(False))
        self.step("save (folder present)", "n4_save_station", self.save("first"))
        self.step("remove folder", "n4_remove_component", self.remove(remember=True))
        if self.removal_batch:
            self.step("rollback folder removal", "n4_rollback", self.rollback,
                      required=False, probe=True)
        if not self.aborted:
            self.remove_again()
        self.step("save (folder removed)", "n4_save_station", self.save("second"))
        self.step("folder absent", "n4_navigate", self.folder_absent)
        if self.aborted and self.created:
            self.cleanup()


def build_report(station, steps, error=None, server_stderr=()):
    required_ok = all(s["verdict"] == "verified" for s in steps if s["required"])
    ok = required_ok and bool(steps) and error is None
    return {"ok": ok, "station": station, "mode": "apply", "error": error,
            "probe_only": PROBE_ONLY, "steps": steps, "server_stderr": list(server_stderr)}


def plan_text():
    lines = ["live_smoke: plan only (nothing is sent; pass --apply to execute)",
             "scratch folder: %s, write scopes: %s" % (FOLDER_ORD, ", ".join(write_scopes())),
             "every write: dry run -> confirmation token -> execute -> verdict", ""]
    for i, (name, tool) in enumerate(plan_rows(), 1):
        lines.append("%2d. %-42s %s" % (i, name, tool))
    lines += ["", "probe-only (not required): " + "; ".join(PROBE_ONLY)]
    return "\n".join(lines) + "\n"


def plan_rows():
    rows = [("connect (expected_station = NAME)", "n4_connect"),
            ("folder absent before", "n4_navigate"),
            ("create Folder %s" % SCRATCH, "n4_create_component")]
    rows += [("create %s (%s, wire sheet)" % (n, t), "n4_create_component")
             for n, t, _, _ in COMPONENTS]
    rows += [("link %s.%s->%s.%s" % l, "n4_create_link") for l in LINKS]
    rows += [("find_dangling_outputs == [Cooling]", "n4_find_dangling_outputs"),
             ("set Temp=30, Setpoint=25", "n4_invoke_action"),
             ("read Compare.out and Cooling.out == true", "n4_read_slots"),
             ("set Temp=20, read outputs == false", "n4_invoke_action"),
             ("save, persisted must be true", "n4_save_station"),
             ("remove folder", "n4_remove_component"),
             ("rollback the removal (probe-only)", "n4_rollback"),
             ("remove folder (again)", "n4_remove_component"),
             ("save, persisted must be true", "n4_save_station"),
             ("folder absent", "n4_navigate")]
    return rows


def parse_args(argv):
    p = argparse.ArgumentParser(prog="live_smoke", description=__doc__.split("\n\n")[0])
    p.add_argument("--apply", action="store_true", help="execute (default: plan only)")
    p.add_argument("--station", action="append", default=[], metavar="NAME=URL")
    p.add_argument("--station-home", action="append", default=[], metavar="NAME=PATH")
    p.add_argument("--credential-env", default="MCP_N4", metavar="PREFIX",
                   help="env var prefix holding <PREFIX>_USER/<PREFIX>_PASSWORD")
    p.add_argument("--insecure-tls", action="append", default=[], metavar="NAME")
    p.add_argument("--state-dir", default=None)
    p.add_argument("--report", default=None, metavar="PATH", help="also write the JSON report")
    p.add_argument("--settle", type=float, default=5.0, metavar="SECONDS",
                   help="how long to wait for the logic outputs to settle (default 5)")
    p.add_argument("--call-timeout", type=float, default=120.0, metavar="SECONDS")
    p.add_argument("--allow-http-for-tests", action="store_true")
    return p.parse_args(argv)


def _pairs(items):
    return dict(item.partition("=")[::2] for item in items)


def server_argv(args):
    argv = [sys.executable, "-m", "mcp_n4.server", "--allow-writes",
            "--credential-env", args.credential_env]
    for scope in write_scopes():
        argv += ["--write-scope", scope]
    for flag, items in (("--station", args.station), ("--station-home", args.station_home),
                        ("--insecure-tls", args.insecure_tls)):
        for item in items:
            argv += [flag, item]
    if args.state_dir:
        argv += ["--state-dir", args.state_dir]
    if args.allow_http_for_tests:
        argv.append("--allow-http-for-tests")
    return argv


def main(argv=None, env=None, stdout=None, stderr=None):
    stdout, stderr = stdout or sys.stdout, stderr or sys.stderr
    env = dict(os.environ if env is None else env)
    args = parse_args(argv)
    if not args.apply:
        stdout.write(plan_text())
        return 0
    stations, homes = _pairs(args.station), _pairs(args.station_home)
    if len(stations) != 1 or set(stations) != set(homes) or not all(stations.values()):
        stderr.write("live_smoke: --apply needs exactly one --station NAME=URL and the matching "
                     "--station-home NAME=PATH\n")
        return 2
    name = next(iter(stations))
    secrets = [env.get(args.credential_env + "_" + k, "") for k in ("USER", "PASSWORD")]

    def log(text):
        stdout.write(scrub(text, secrets) + "\n")

    error, client, scenario = None, None, None
    try:
        client = McpClient(server_argv(args), env, args.call_timeout)
        scenario = Scenario(client, name, args.settle, log)
        client.start()
        scenario.run()
    except SmokeError as exc:
        error = str(exc)
    finally:
        if client:
            client.close()
    report = build_report(name, scenario.steps if scenario else [], error,
                          client.stderr_lines if client else ())
    text = scrub(json.dumps(report, indent=2), secrets)
    if args.report:
        with open(args.report, "w") as fh:
            fh.write(text + "\n")
    stdout.write(text + "\n")
    return 0 if report["ok"] else 1


if __name__ == "__main__":
    sys.exit(main())
