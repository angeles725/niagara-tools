### Fixed — pre-push reads a retro header flip with rename detection off

- The promotion exit's header-flip check now diffs the range with `--no-renames`. Before, a range that
  deleted a pending retro and added a similar retro born folded was a rename in git's default diff, so both
  header lines sat in one file's diff and counted as a flip. Now only a flip inside the same path counts. Retro
  `2026-11-14-go-pre-push-port` Δ2.
