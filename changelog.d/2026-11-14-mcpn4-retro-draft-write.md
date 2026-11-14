### Added — retro-draft --write replaces new_retro.py
- `go/bin/retro-draft --write --station NAME --state-dir DIR [--date D] [--since TS] [--out-dir DIR] [--force] [--dry-run]` writes the session retro and its `pending` row in `retros/INDEX.md`, with the same refusals, output and exit codes as `tools/new_retro.py` (differential: identical stdout, files, modes and exit codes; stderr differs only in the program prefix). `--date` defaults to today (UTC) in write mode only; draft mode still requires it.
- `go/internal/atomicfile`: the one atomic-write helper (temporary file in the same directory, O_EXCL name, rename; an existing target keeps its exact mode, a new file is 0644 under the umask; the temporary file is removed on any failure).

### Changed — gen writes through atomicfile
- Generated files (BUILD-STATE, lint index) now keep an existing target's exact mode: a group-writable 0664 file stays 0664 instead of being masked by the umask. The temporary file is a hidden O_EXCL name next to the target instead of `<path>.tmp.<pid>`.

### Removed — mcp-n4-kit/tools/new_retro.py
- Use `retro-draft --write`; the skill, README, METHODOLOGY and retros/INDEX.md name it.
