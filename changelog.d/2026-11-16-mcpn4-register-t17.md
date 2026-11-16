### Fixed — mcp-n4 register: kept launcher resolved and nothing written on a duplicate name (retro row T17 of the 2026-11-15 coordination stretch)

- A launcher kept without server flags whose exec target is the installed binary path (`~/.local/bin/mcp-n4` by
  default) is now refused when that path is a symlink into a git checkout, with the same message as the server-flags
  case; `--force` still keeps it.
- A duplicate entry name (or any other unreadable JSON on the entry path) in the harness config is refused with
  "; nothing written", and every file stays byte-identical.
- When writing the harness entry fails after the launcher was written, the launcher is put back (or removed when it
  is new) and the error ends with "; nothing written"; the "written launcher" line is printed only once both writes
  succeeded.
