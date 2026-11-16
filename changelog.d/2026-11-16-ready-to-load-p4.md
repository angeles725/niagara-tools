### Added — `report-module.sh --ready`, the ready-to-load report of a built module

- `report-module.sh <module-root> --ready [--niagara-home <mirror>] [--expect-niagara <ver>] [--expect-signer <alias>] [--json]`
  prints one row per check for each built jar: module, niagara-target, niagara-home, signer, verify-module,
  schema-risk, sha256 and mirror. It ends with `READY TO LOAD: <MOD> <version> for Niagara <x> (sha256 <12>)`
  and the read-back to run on the station after loading, or `NOT READY` with the failing rows. Exit 0 ready,
  1 not ready, 3 env. It never contacts a station. See `build-verify.md` § Ready to load.
- `--ready` refuses every lint-run option (`--lint-only`, `--write-baseline`, `--no-baseline`, `--console-dir`,
  `--profile`, `--legacy`, `--wiring-map`) with exit 2 and a first line naming it, instead of ignoring it.
