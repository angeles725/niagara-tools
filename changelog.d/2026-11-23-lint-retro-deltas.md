### Changed — lint retro deltas (R7 ΔK5, ΔK9, ΔK10)

- `lint-spa-poll-no-recovery`: a call to a probe-then-reload helper (a name holding `probe` and `reload`, e.g.
  `probeThenReload()`) is the recovery, in place of a bare `location.reload()` in the poll file; the `lastOk*` key is
  still required, and a helper only declared or named in a comment is no call (issue #895). Doctrine in
  `types/frontend-standard.md`: reload only when the station still answers HTTP, keep polling on network loss.
- `lint-module-map` MM5: a row whose `Files` are under `src/rc/` also counts an existing `webtest/**/*.test.mjs` path
  in `Tests`; an rt row, or a webtest file that is not `*.test.mjs`, does not (issue #899).
- `lint-write-path`: a matrix row marked `(no facade slot)` (any case) is no STALE for a control that is a sibling
  device point with no facade slot (issue #900).
