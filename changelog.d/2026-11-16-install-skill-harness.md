### Added — install-skill and check-skill-drift for pi and gentle-shell

- `install-skill.sh --harness <claude|pi|gentle-shell|all>` (default `claude`, unchanged) installs a skill launcher
  into `<agent-dir>/skills/<skill>/SKILL.md` for pi (`$PI_CODING_AGENT_DIR`, else `~/.pi/agent`) and gentle-shell
  (`~/.gentle-shell/agent`), plus the prompt file `<agent-dir>/prompts/<skill>.md` that gives those harnesses a bare
  `/<skill>`. The prompt file is rendered from the kit's `skill/prompt.md` beside the tracked launcher and follows the
  launcher's current / diverged / `--force` rules. An agent directory that does not exist is an `absent` row (exit 0),
  never created. Retro 2026-11-15 H1, H2.
- `check-skill-drift.sh --harness <h>` / `--all` checks the launcher and the prompt file of each harness; the skill
  column reads `<skill>[<harness>]` and an absent harness is a `SKIP` row. Without the option the row is the one it
  always was. Retro H3 (the session card line is a separate change).
- `--home ""` is a usage error (exit 2) in both scripts instead of falling back to `$HOME`. Retro S3.
- `install-skill.sh` refuses (exit 3) to write through a symlinked `SKILL.md`, skill directory or prompt file; before,
  `--force` overwrote the link's target. Retro S4.
- The close gates' `CLOSE-install-skill` cases install both launchers into a throwaway home in every harness, compare
  them byte for byte with the kit and run the drift check; before, `--dry-run` exit 0 passed for any state. Retro S5.
- The mcp-n4 launcher no longer assumes `~/.claude/skills/` or `.mcp.json`; the kit README gains "Registering the
  server per harness" (pi tool names, `"exposure": "direct"`). Retro H4.
