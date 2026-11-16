### Added — verify-module --expected-signer, passed by build.sh

- `verify-module.sh --expected-signer ALIAS` FAILs the `signed` row unless the jar's complete signatures are exactly
  that one signer, compared as jarsigner names its files (upper-cased, 8 characters, other characters as `_`:
  `acmesigner` is `META-INF/ACMESIGN.SF`). Before, a client module signed with the default development key passed
  the gate and a station's Software Manager rejected it as self-signed. Without the option nothing changes.
- `build.sh` passes it with the configured `niagaraSigningAlias` (`--gradle-prop`, `ORG_GRADLE_PROJECT_`, or a
  `gradle.properties`), else with the signer of the deployed jars its drift gate snapshots; with neither it prints
  one `WARN — no niagaraSigningAlias is configured` line and keeps its exit code. External session retro, delta B5.
- The deployed-jar signer is never the signing plugin's default development signer (`NIAGARA4`): such a deployed jar
  gives no expected signer and the WARN names it. A verify gate FAIL now restores the pre-build `modules/` jar the
  drift gate backed up (exit 50 as before), so a bare re-run fails for the same reason; `--no-drift-check` has no
  backup and no deployed-jar signer.
