### Fixed — build.sh: a mirror keeps the jar a build replaced; a missing `.deploy-baseline` names its command (retro row B6)

- `build.sh` against a mirror: gradle's `:jar` step replaces `<mirror>/modules/<MOD>-<p>.jar`, so the mirror (also the
  drift baseline of what the station runs) stopped matching the deployed jar. A run that ends with a `modules/` jar
  different from the drift gate's pre-build snapshot now keeps the snapshot as
  `<mirror>/.mirror-replaced/<MOD>-<p>.jar.<sha12>` (never overwritten or deleted) and ends stdout with one line:
  `==> mirror: <MOD>-<p>.jar in <mirror>/modules replaced (vendorVersion <old> sha256:<old> -> <new> sha256:<new>);
  first replaced this cycle: vendorVersion <v> sha256:<s>; restore with: <toolbelt>/build.sh --restore-mirror <kept>`.
  After two builds the restore command still names the jar from before the first one; a cycle ends only at
  `--restore-mirror`, so after a deploy delete `<mirror>/.mirror-replaced/<MOD>-<p>.jar.first`. A jar or a cycle
  pointer that cannot be written is one WARN (and then no jar is named as first). Printed paths are shell-quoted. Exit codes are unchanged; nothing is kept for a real `niagara_home`, under `--no-drift-check`, on a first
  deploy, or when the run already restored `modules/`.
- New `build.sh --restore-mirror <kept>`: copies a kept jar back into its mirror's `modules/` and builds nothing; it
  refuses a path not under `.mirror-replaced/` (exit 2), a missing kept jar, one whose sha256 prefix is not its
  `<sha12>`, or a home without `.niagara-mirror` (exit 10, nothing written); a failed copy is exit 1. It ends the cycle.
- `report-module.sh`: the `schema-risk` SKIP row of an artifact without `.deploy-baseline` names the exact command that
  creates it from the deployed commit (archive to a temporary tar, then `mkdir <artifact>/.deploy-baseline` and
  extract, so a wrong ref or a pasted `<deployed-ref>` leaves no empty baseline); the verdict stays INCOMPLETE (exit 4, `build.sh` 53), and
  `build.sh`'s INCOMPLETE line points at that row only when an artifact lacks `.deploy-baseline`.
