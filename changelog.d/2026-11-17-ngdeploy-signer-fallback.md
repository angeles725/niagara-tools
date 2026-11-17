### Fixed — ng-deploy: the home's signer is kept when the station gives none

- With no `niagaraSigningAlias`, the deploy gate again expects the signer of the jars that were in
  `<NIAGARA_HOME>/modules` whenever the station's jars give none (unsigned, two signers, or only the development
  signer); before, a station jar signed with the development key or two station signers left the gate with none.
- A station jar whose signer cannot be told (unreadable, not a zip, a `.SF` without its block file) gives no
  expected signer and a WARN naming it, and the home's signer is not taken in its place.
- `NIAGARA_HOME` reaches gradle as an absolute path, and one under a directory the deploy may not search is
  refused as a permission error.
