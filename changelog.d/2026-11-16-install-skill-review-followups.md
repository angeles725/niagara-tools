### Fixed — install-skill --harness review follow-ups

- An explicit `--home` now wins over `PI_CODING_AGENT_DIR` in `install-skill.sh` and `check-skill-drift.sh`, so
  `--home <scratch>` keeps every pi write under the scratch directory. Retro 2026-11-15 T3 R1-001, T4 (1).
- `install-skill.sh` checks every file of every selected harness before the first write: a diverged copy without
  `--force` (exit 1) or a symbolic link on the way (exit 3) stops the run with nothing written, and the refusal comes
  before any `--force` line. A symlinked `skills/` or `prompts/` directory is refused like a symlinked target.
  T3 R3-symlink-scope, T4 (2) and (3).
- A diverged prompt file names its template as `template: <path> (rendered for this harness)` with the
  `rendered sha256`. T3 R1-002.
- The prompt file always carries the absolute path of the installed `SKILL.md`, also under a relative `--home`.
  T3 R3-relative-home-prompt.
- Printed paths keep the `--home` spelling (a trailing slash is not cleaned), as before `--harness`.
  T3 R3-path-join-output.
- A run whose every selected harness is absent no longer materializes the kit state. T3
  R3-materialize-when-nothing-installed.
- `--home ""` prints the usage line in `install-skill.sh` too. T4 (4).
