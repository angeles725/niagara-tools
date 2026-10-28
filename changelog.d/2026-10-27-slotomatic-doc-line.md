### Added — lint-slotomatic-markers `slotomatic-doc-line` (WARN)

- `lint-slotomatic-markers` WARNs a `//` comment block of two or more lines directly above a `@NiagaraProperty`
  whose last line continues the line before it: Slot-o-Matic keeps only that last line as the slot's generated
  Javadoc. The row is a WARN (exit 0; exit 1 under `--strict`, as every WARN of this lint), relayed by
  `report-module`.
