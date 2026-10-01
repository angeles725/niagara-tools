"""Stdio MCP server for Niagara N4 stations (newline-delimited JSON-RPC 2.0).

Run from `mcp-n4-kit/` with `python3 -m mcp_n4.server`. Importing this module
has no side effects; `main()` is only called under `__main__`.
"""
import argparse
import json
import sys

from . import __version__, box, tools_read

SERVER_NAME = "mcp-n4"
SERVER_VERSION = __version__
#: Newest first. The reference server (niagara_help_mcp.py) simply echoes the
#: client's value; here an unknown value is answered with the newest we know.
SUPPORTED_PROTOCOL_VERSIONS = ("2025-06-18", "2025-03-26", "2024-11-05")

PARSE_ERROR, INVALID_REQUEST, METHOD_NOT_FOUND, INVALID_PARAMS = -32700, -32600, -32601, -32602

_JSON_TYPES = {"string": str, "boolean": bool, "integer": int, "object": dict}


class InvalidParams(Exception):
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
        want = _JSON_TYPES[spec["type"]]
        # bool is an int subclass in Python; an integer argument must not be a bool.
        if not isinstance(value, want) or (want is int and isinstance(value, bool)):
            raise InvalidParams("argument %s must be %s" % (key, spec["type"]))
        if "minimum" in spec and value < spec["minimum"]:
            raise InvalidParams("argument %s must be >= %s" % (key, spec["minimum"]))
        if "maximum" in spec and value > spec["maximum"]:
            raise InvalidParams("argument %s must be <= %s" % (key, spec["maximum"]))


class Server:
    def __init__(self, allow_writes=False, allow_http=False, env=None,
                 client_factory=None, tools=None):
        self.ctx = tools_read.Context(allow_writes=allow_writes, allow_http=allow_http,
                                      env=env, client_factory=client_factory)
        self.tools = {t.name: t for t in (tools_read.TOOLS if tools is None else tools)}

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
        for line in instream:
            reply = self.handle_line(line)
            if reply is not None:
                outstream.write(reply + "\n")
                outstream.flush()
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
        if id_ is None:  # notification: never answered
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
                raise tools_read.ToolError("not connected: call n4_connect first")
            obj = tool.handler(self.ctx, args)
        except (tools_read.ToolError, box.BoxError, ValueError) as exc:
            return self._tool_error(str(exc))
        except Exception as exc:  # never leak a message (could hold secrets) or a trace
            return self._tool_error("internal error (%s)" % type(exc).__name__)
        return {"content": [{"type": "text", "text": json.dumps(obj)}],
                "structuredContent": obj}

    def _tool_error(self, text):
        return {"content": [{"type": "text", "text": self.ctx.scrub(text)}], "isError": True}


def parse_args(argv=None):
    parser = argparse.ArgumentParser(prog="mcp_n4.server", description=__doc__)
    parser.add_argument("--allow-writes", action="store_true",
                        help="record writes-allowed mode (write tools arrive in a later task)")
    parser.add_argument("--allow-http-for-tests", action="store_true",
                        help="permit http:// base URLs (fake station in tests only)")
    return parser.parse_args(argv)


def main(argv=None):
    args = parse_args(argv)
    Server(allow_writes=args.allow_writes, allow_http=args.allow_http_for_tests).serve(
        sys.stdin, sys.stdout)
    return 0


if __name__ == "__main__":
    sys.exit(main())
