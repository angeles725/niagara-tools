### Fixed — ng-deploy: station-first expected signer, NIAGARA_HOME must be a directory

- With no `niagaraSigningAlias` configured, the deploy gate's expected signer is now taken from the module's jars
  the station holds before the copy, and from the jars that were in `<NIAGARA_HOME>/modules` only when none of the
  station's is signed; the two sources are never mixed.
- A `NIAGARA_HOME` that names no directory, or names a file, is refused with exit 10 before the backup and before
  gradle runs, with a line that names the value.
