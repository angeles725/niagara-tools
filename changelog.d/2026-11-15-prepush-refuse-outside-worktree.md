### Changed — pre-push refuses a push it cannot place in a work tree

- When `git rev-parse --show-toplevel` fails, the pre-push gate now refuses the push with git's own error
  (`pre-push: git rev-parse --show-toplevel failed: <git error>; refusing to push unchecked`) instead of
  passing it silently with exit 0. A bare repository is the one exception: the push goes on and a one-line
  notice names it (`pre-push: bare repository <dir>: no work tree, nothing to gate`). Retro
  `2026-11-14-go-pre-push-port` Δ1.
