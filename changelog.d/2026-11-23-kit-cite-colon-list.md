### Fixed — kit-cite-resolve reads a backticked file with a colon-repeating line list

- A prose citation such as `` `toolbelt/build.sh:15,:82-88` `` (the colon repeated before each range) is a pointer now,
  checked like `:15,82-88`; before it was not read at all, so a stale line went unreported. Retro
  `2026-11-14-go-build-module-port` Δ4.
