### Fixed — kit rare-case follow-ups (lint-logger-name, L17, preflight version-drift)

- `lint-logger-name` reads `moduleName` with gradle comments blanked: a commented-out old name is no declaration.
- `lint-structure` L17 WARNs when a gradle file ends inside an unclosed `/* */` comment or text block (no
  `moduleName` after it is read), and pins that a Groovy `'/*'` string is not a comment.
- `preflight` version-drift skips `<repo>-worktrees/` checkouts and the gradle root's own ancestors (the repository
  root), which are not sibling modules.
