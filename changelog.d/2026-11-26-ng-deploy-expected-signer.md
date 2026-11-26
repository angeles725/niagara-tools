### Added — ng-deploy reads the recorded expected signer

- `ng-deploy.sh` reads the module's optional `expected_signer` from its `build-state.v1` envelope, as `build.sh` does
  (issue #1149, follow-up of #1131): new `--build-state <file>`, else the kit's `BUILD-STATE.src.md` (command line
  only: `--gradle-prop` and `--plugin-version` have no env-file key either). Same precedence: a configured
  `niagaraSigningAlias` must equal the record; with none configured the record is the expected signer (one
  `expected signer: …` line) and gradle builds with it as `-PniagaraSigningAlias`, so the build and the gate agree;
  only with no record does the signer of the jars already deployed decide, as before. A configured alias that
  differs from the record, a malformed record (the four kinds `build.sh` refuses) and a `--build-state` file that
  cannot be read are exit 10 before the build and the copy, naming both aliases or the file; an empty
  `--build-state` is exit 10. A missing default file, `unknown` or no field behave as before. Documented in
  `usage.txt` and the `ng-deploy.sh` header, and in `BUILD-LOOP.md` §4.
