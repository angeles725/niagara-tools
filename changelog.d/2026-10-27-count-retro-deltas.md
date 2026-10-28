### Added — count-retro-deltas

- New `toolbelt/count-retro-deltas.sh <retro.md>...` (G39): counts each retro's delta-table rows and compares them
  with its `**Delta count**` line, its `<!-- deltas: N -->` header and its generated `retros/INDEX.md` row, printing
  one OK/MISMATCH line per retro and a TOTAL line to quote; exit 1 on any mismatch, 3 when no `INDEX.md` is beside a
  retro (generate it with `toolbelt/kit-state.sh --write`).
