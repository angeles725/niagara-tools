"""Stdio MCP server for Niagara N4 stations (newline-delimited JSON-RPC 2.0).

Run from `mcp-n4-kit/` with `python3 -m mcp_n4.server`. Importing this module
has no side effects; `main()` is only called under `__main__`.

MCP protocol versions: this server speaks 2025-06-18, 2025-03-26 and 2024-11-05
(`SUPPORTED_PROTOCOL_VERSIONS`). A client asking for a newer revision is answered
with 2025-06-18. Only add a newer revision to that tuple once its semantics (new
message shapes, capabilities, required fields) are implemented and tested here.
"""
import argparse
import json
import os
import sys

from . import __version__, box, safety, tools_read, tools_write

SERVER_NAME = "mcp-n4"
SERVER_VERSION = __version__
#: Newest first. The reference server (niagara_help_mcp.py) simply echoes the
#: client's value; here an unknown value is answered with the newest we know.
SUPPORTED_PROTOCOL_VERSIONS = ("2025-06-18", "2025-03-26", "2024-11-05")

PARSE_ERROR, INVALID_REQUEST, METHOD_NOT_FOUND, INVALID_PARAMS = -32700, -32600, -32601, -32602

_JSON_TYPES = {"string": str, "boolean": bool, "integer": int, "number": (int, float),
               "array": list, "object": dict}


class InvalidParams(Exception):
    pass


class MethodNotFound(Exception):
    pass


def _validate(schema, args):
    """Check required keys, JSON types and integer ranges against `schema`."""
    props = schema.get("properties", {})
    for key in schema.get("required", []):
        if key not in args:
            raise InvalidParams("missing required argument: %s" % key)
    for key, value in args.items():
        spec = props.get(key)
        if spec is None:
            continue
        kind = spec.get("type")
        if kind is None:  # an untyped property accepts any JSON value
            continue
        if kind not in _JSON_TYPES:  # a server bug, never a client error
            raise ValueError("schema for %s uses unsupported type %r" % (key, kind))
        want = _JSON_TYPES[kind]
        # bool is an int subclass in Python; a numeric argument must not be a bool.
        if not isinstance(value, want) or (kind in ("integer", "number")
                                           and isinstance(value, bool)):
            raise InvalidParams("argument %s must be %s" % (key, spec["type"]))
        if "minimum" in spec and value < spec["minimum"]:
            raise InvalidParams("argument %s must be >= %s" % (key, spec["minimum"]))
        if "maximum" in spec and value > spec["maximum"]:
            raise InvalidParams("argument %s must be <= %s" % (key, spec["maximum"]))


class Server:
    def __init__(self, allow_writes=False, allow_http=False, env=None,
                 client_factory=None, tools=None, write_scopes=(), state_dir=None,
                 token_ttl=300, max_writes=200, stations=None, credential_env="MCP_N4",
                 insecure_tls=(), station_homes=None, progress_file=None,
                 allow_tier_b=(), allow_tier_c=()):
        self.ctx = tools_read.Context(allow_writes=allow_writes, allow_http=allow_http,
                                      env=env, client_factory=client_factory,
                                      stations=stations, credential_env=credential_env,
                                      insecure_tls=insecure_tls, allow_tier_b=allow_tier_b,
                                      allow_tier_c=allow_tier_c)
        self.ctx.state_dir = os.path.expanduser(state_dir or safety.DEFAULT_STATE_DIR)
        self.ctx.progress_path = os.path.expanduser(progress_file) if progress_file else None
        if tools is None:
            tools = tools_read.TOOLS + (tools_write.TOOLS if allow_writes else [])
        if allow_writes:
            self.ctx.write = tools_write.WriteState(write_scopes, state_dir, token_ttl,
                                                    max_writes,
                                                    station_homes=station_homes)
        self.tools = {t.name: t for t in tools}

    # ---- framing ---------------------------------------------------------
    def handle_line(self, line):
        """Process one input line; return the reply line or None."""
        line = line.strip()
        if not line:
            return None
        try:
            msg = json.loads(line)
        except ValueError:
            return json.dumps(self._error(None, PARSE_ERROR, "parse error"))
        reply = self.dispatch(msg)
        return None if reply is None else json.dumps(reply)

    def serve(self, instream, outstream):
        try:
            for line in instream:
                reply = self.handle_line(line)
                if reply is not None:
                    outstream.write(reply + "\n")
                    outstream.flush()
        except KeyboardInterrupt:
            pass
        finally:
            self.ctx.close()

    # ---- dispatch --------------------------------------------------------
    @staticmethod
    def _error(id_, code, message):
        return {"jsonrpc": "2.0", "id": id_, "error": {"code": code, "message": message}}

    def dispatch(self, msg):
        if not isinstance(msg, dict):
            return self._error(None, INVALID_REQUEST, "request must be a JSON object")
        method, id_ = msg.get("method"), msg.get("id")
        params = msg.get("params")
        if "id" not in msg:  # notification: never answered ("id": null is a request)
            return None
        try:
            if method == "initialize":
                result = self._initialize(params)
            elif method == "ping":
                result = {}
            elif method == "tools/list":
                result = {"tools": [self._describe(t) for t in self.tools.values()]}
            elif method == "tools/call":
                result = self._call(params)
            else:
                return self._error(id_, METHOD_NOT_FOUND, "method not found: %s" % method)
        except InvalidParams as exc:
            return self._error(id_, INVALID_PARAMS, str(exc))
        except MethodNotFound as exc:
            return self._error(id_, METHOD_NOT_FOUND, str(exc))
        return {"jsonrpc": "2.0", "id": id_, "result": result}

    def _initialize(self, params):
        asked = params.get("protocolVersion") if isinstance(params, dict) else None
        version = asked if asked in SUPPORTED_PROTOCOL_VERSIONS else SUPPORTED_PROTOCOL_VERSIONS[0]
        return {"protocolVersion": version,
                "capabilities": {"tools": {"listChanged": False}},
                "serverInfo": {"name": SERVER_NAME, "version": SERVER_VERSION}}

    @staticmethod
    def _describe(tool):
        return {"name": tool.name, "description": tool.description,
                "inputSchema": tool.input_schema, "annotations": tool.annotations}

    def _call(self, params):
        if not isinstance(params, dict) or not isinstance(params.get("name"), str):
            raise InvalidParams("tools/call needs a string 'name'")
        tool = self.tools.get(params["name"])
        if tool is None and params["name"] in tools_write.NAMES:
            raise MethodNotFound("write tools are disabled: start the server with --allow-writes")
        if tool is None:
            raise InvalidParams("unknown tool: %s" % params["name"])
        args = params.get("arguments", {})
        if args is None:
            args = {}
        if not isinstance(args, dict):
            raise InvalidParams("'arguments' must be an object")
        _validate(tool.input_schema, args)
        try:
            if tool.needs_session and self.ctx.session is None:
                raise tools_read.ToolError(safety.REASON_NOT_CONNECTED + ": call n4_connect first")
            obj = tool.handler(self.ctx, args)
        except (tools_read.ToolError, box.BoxError, ValueError) as exc:
            return self._tool_error(str(exc))
        except Exception as exc:  # never leak a message (could hold secrets) or a trace
            return self._tool_error("internal error (%s)" % type(exc).__name__)
        return {"content": [{"type": "text", "text": json.dumps(obj)}],
                "structuredContent": obj}

    def _tool_error(self, text):
        return {"content": [{"type": "text", "text": self.ctx.scrub(text)}], "isError": True}


def _positive_int(text):
    """argparse type: an integer >= 1 (a zero or negative TTL/budget is refused at startup)."""
    try:
        value = int(text)
    except ValueError:
        raise argparse.ArgumentTypeError("%r is not an integer" % text) from None
    if value < 1:
        raise argparse.ArgumentTypeError("must be >= 1, got %d" % value)
    return value


def parse_args(argv=None):
    parser = argparse.ArgumentParser(prog="mcp_n4.server", description=__doc__)
    parser.add_argument("--allow-writes", action="store_true",
                        help="register the write tools (dry run + confirmation token each)")
    parser.add_argument("--write-scope", action="append", default=[], metavar="ORD_PREFIX",
                        help="ORD prefix writes may touch (repeatable); none = no writes")
    parser.add_argument("--state-dir", default=None,
                        help="journal/audit directory (default ~/.local/state/mcp-n4)")
    parser.add_argument("--token-ttl", type=_positive_int, default=300, metavar="SECONDS",
                        help="confirmation token lifetime (default 300)")
    parser.add_argument("--max-writes", type=_positive_int, default=200, metavar="N",
                        help="executed writes allowed per session (default 200)")
    parser.add_argument("--station", action="append", default=[], metavar="NAME=URL",
                        help="station n4_connect may use (repeatable); NAME should equal the "
                             "station's stationName; URL must be https://")
    parser.add_argument("--station-home", action="append", default=[], metavar="NAME=PATH",
                        help="directory holding the config.bog of configured station NAME; lets "
                             "n4_save_station prove persistence (repeatable)")
    parser.add_argument("--credential-env", default="MCP_N4", metavar="PREFIX",
                        help="env var prefix holding <PREFIX>_USER/<PREFIX>_PASSWORD "
                             "(default MCP_N4)")
    parser.add_argument("--insecure-tls", action="append", default=[], metavar="NAME",
                        help="skip TLS certificate verification for this configured station "
                             "(self-signed); repeatable")
    parser.add_argument("--allow-tier-b", action="append", default=[], metavar="NAME",
                        help="let writes run on configured station NAME although its version "
                             "is tier B (4.15/4.3); only after a PoC matched that build "
                             "(METHODOLOGY section 5); repeatable")
    parser.add_argument("--allow-tier-c", action="append", default=[], metavar="NAME",
                        help="let writes run on configured station NAME although its version "
                             "is tier C (other or unknown); only after reg.loadContract, "
                             "loadRoot and a harmless scratch write succeeded; repeatable")
    parser.add_argument("--progress-file", default=None, metavar="PATH",
                        help="append JSON progress lines of long reads (n4_inventory) to PATH")
    parser.add_argument("--allow-http-for-tests", action="store_true",
                        help="permit http:// base URLs (fake station in tests only)")
    return parser.parse_args(argv)


def main(argv=None):
    args = parse_args(argv)
    try:
        stations = {}
        for item in args.station:
            name, sep, url = item.partition("=")
            if not sep or not name or not url:
                raise ValueError("--station expects NAME=URL, got %r" % item)
            stations[name] = url
        homes = {}
        for item in args.station_home:
            name, sep, path = item.partition("=")
            if not sep or not name or not path:
                raise ValueError("--station-home expects NAME=PATH, got %r" % item)
            homes[name] = path
        srv = Server(allow_writes=args.allow_writes, allow_http=args.allow_http_for_tests,
                     write_scopes=args.write_scope, state_dir=args.state_dir,
                     token_ttl=args.token_ttl, max_writes=args.max_writes, stations=stations,
                     credential_env=args.credential_env, insecure_tls=args.insecure_tls,
                     station_homes=homes, progress_file=args.progress_file,
                     allow_tier_b=args.allow_tier_b, allow_tier_c=args.allow_tier_c)
    except (ValueError, safety.SafetyError) as exc:
        print("mcp_n4.server: %s" % exc, file=sys.stderr)
        return 2
    srv.serve(sys.stdin, sys.stdout)
    return 0


if __name__ == "__main__":
    sys.exit(main())
