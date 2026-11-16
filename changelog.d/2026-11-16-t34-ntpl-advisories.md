### Fixed — the ntpl lint run is bounded by entries read too (retro row T34 of the 2026-11-15 coordination stretch)

- `n4lint lint-ntpl-manifest-deps` stops a run after 100,000 entries read, as well as after 128 MiB inflated:
  empty or tiny entries drew nothing from the byte budget, so many of them could keep a run going for minutes. The
  stop is one FAIL row, "100000 entries read in all, the bound of one run -- <entry> and everything after it not
  read" (exit 1).
