### Added — two lints on the repository's own tests and doctrine

- `go/internal/cardwidth`: every live doctrine card keeps its lines to 120 characters where a line can be wrapped;
  `BUILD-LOOP.md` and `skill/SKILL.md` are wrapped (no text change), and the other cards are held at the overlong-line
  count in `go/internal/cardwidth/baseline.txt`, which may only go down. Runs with `go test ./...`. Retro
  `2026-11-14-go-toolbelt-small-port` Δ5.
- `go/internal/batshome`: a bats suite that runs a Go launcher under a changed or unset `HOME` must build the binary in
  `setup_file` (or in a helper it loads); `build-sh`, `check-skill-drift` and `kit-resolve` gained that prebuild.
  Retro `2026-11-14-go-toolbelt-small-port` Δ4.
