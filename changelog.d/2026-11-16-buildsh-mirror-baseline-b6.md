### Fixed — build.sh: a mirror keeps the jar a build replaced; a missing `.deploy-baseline` names its command (retro row B6)

- `build.sh` against a mirror: gradle's `:jar` step replaces `<mirror>/modules/<MOD>-<p>.jar`, so the mirror (also the
  drift baseline of what the station runs) stopped matching the deployed jar. A run that ends with a `modules/` jar
  different from the drift gate's pre-build snapshot now keeps the snapshot as
  `<mirror>/.mirror-replaced/<MOD>-<p>.jar.<sha12>` and ends stdout with one line: `==> mirror: <MOD>-<p>.jar in
  <mirror>/modules replaced (vendorVersion <old> sha256:<old> -> <new> sha256:<new>); the replaced jar is kept — put it
  back with: <toolbelt>/build.sh --restore-mirror <kept>`. Exit codes are unchanged; nothing is kept for a real
  `niagara_home`, under `--no-drift-check`, on a first deploy, or when the run already restored `modules/`.
- New `build.sh --restore-mirror <kept>`: copies a kept jar back into its mirror's `modules/` and builds nothing; it
  refuses a path not under `.mirror-replaced/` (exit 2), a missing kept jar or a home without `.niagara-mirror`
  (exit 10); a failed copy is exit 1.
- `report-module.sh`: the `schema-risk` SKIP row of an artifact without `.deploy-baseline` names the exact command that
  creates it from the deployed commit (`mkdir -p <artifact>/.deploy-baseline && git -C <artifact> archive
  <deployed-ref> . | tar -x -C <artifact>/.deploy-baseline`); the verdict stays INCOMPLETE (exit 4, `build.sh` 53).
