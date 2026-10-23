"""The Python surface the Go port serves verbatim (docs/go-port-plan.md, slice S2).

`go/internal/mcpn4/registry/registry.json` holds, generated from the Python server:

- `server`: SERVER_NAME, SERVER_VERSION and SUPPORTED_PROTOCOL_VERSIONS;
- `cli`: the argparse usage and help text, formatted at 80 columns without color;
- `read_tools` / `write_tools`: every tool's name, description, inputSchema, annotations and
  needs_session, in the order of `tools_read.TOOLS` / `tools_write.TOOLS`.

`test_go_harness` fails when the file and the Python tables disagree. Regenerate with
`python3 mcp-n4-kit/tests/go_registry.py --write` after changing a tool, a flag or the version.
"""
import contextlib
import json
import os
import sys

TESTS_DIR = os.path.dirname(os.path.abspath(__file__))
KIT_DIR = os.path.dirname(TESTS_DIR)
REGISTRY_FILE = os.path.join(os.path.dirname(KIT_DIR), "go", "internal", "mcpn4", "registry",
                             "registry.json")
if KIT_DIR not in sys.path:
    sys.path.insert(0, KIT_DIR)

from mcp_n4 import server, tools_read, tools_write  # noqa: E402


@contextlib.contextmanager
def _plain_80_columns():
    """argparse wraps at $COLUMNS and colors on request: pin both while formatting."""
    saved = {k: os.environ.get(k) for k in ("COLUMNS", "NO_COLOR", "PYTHON_COLORS")}
    os.environ.update(COLUMNS="80", NO_COLOR="1", PYTHON_COLORS="0")
    try:
        yield
    finally:
        for key, value in saved.items():
            if value is None:
                os.environ.pop(key, None)
            else:
                os.environ[key] = value


def _tool(tool):
    return {"name": tool.name, "description": tool.description,
            "inputSchema": tool.input_schema, "annotations": tool.annotations,
            "needs_session": tool.needs_session}


def registry():
    with _plain_80_columns():
        parser = server.build_parser()
        usage, help_text = parser.format_usage(), parser.format_help()
    return {
        "server": {"name": server.SERVER_NAME, "version": server.SERVER_VERSION,
                   "protocol_versions": list(server.SUPPORTED_PROTOCOL_VERSIONS)},
        "cli": {"usage": usage, "help": help_text},
        "read_tools": [_tool(t) for t in tools_read.TOOLS],
        "write_tools": [_tool(t) for t in tools_write.TOOLS],
    }


def registry_text():
    """The exact bytes registry.json must hold."""
    return json.dumps(registry(), indent=1) + "\n"


if __name__ == "__main__":
    if sys.argv[1:] == ["--write"]:
        with open(REGISTRY_FILE, "w", encoding="ascii", newline="\n") as out:
            out.write(registry_text())
    elif sys.argv[1:]:
        sys.exit("usage: go_registry.py [--write]")
    else:
        sys.stdout.write(registry_text())
