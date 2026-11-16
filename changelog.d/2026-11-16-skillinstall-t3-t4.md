### Fixed — install-skill: a file that cannot be placed is refused before the first write (retro rows T3 and T4)

- `install-skill.sh` now also checks, before the first write, that every file of every selected harness can be
  placed: a non-directory where its `skills/`, skill or `prompts/` directory must be, or a target that exists and is
  not a regular file, refuses the run with `refusing to install: not a directory: <path>` or `not a regular file:
  <path>` (exit 3, "move it away, then re-run"). Before, the launcher of a harness was installed and its prompt then
  failed, leaving `SKILL.md` without its prompt. `--dry-run` output is unchanged. Retro 2026-11-15 T4 (3) residual;
  the other T3 and T4 findings were already answered by #694 and T10.
