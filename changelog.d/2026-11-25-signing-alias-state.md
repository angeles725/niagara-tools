### Added — expected signer in the build-state envelope

- A module's `build-state.v1` envelope may carry an optional `expected_signer: <alias>` field (issue #1131): the
  signing alias its releases use. `build.sh` reads the envelope of `<MOD>` (new `--build-state <file>`, else the
  kit's `BUILD-STATE.src.md`) before gradle. With no alias configured it hands the recorded one to gradle as
  `-PniagaraSigningAlias`, so the signing block check, the key listing and the verify gate's `signed` row use it
  (one `==> expected signer: …` line); an alias configured by `--gradle-prop`,
  `ORG_GRADLE_PROJECT_niagaraSigningAlias` or `gradle.properties` that differs from the record is exit 10 before
  gradle, naming both. Precedence: configured alias (must equal the record) > record > the deployed jars' signer.
  `unknown` or no field behaves as before. `sweep-build-state.sh` fails (exit 1) a value that is not one alias
  token (letters, digits, `.`, `_`, `-`), and `build.sh` refuses it with exit 10. Documented in `build.sh`'s
  header, `BUILD-LOOP.md` §4 and `skill/SKILL.md`.
