### Fixed — review advisories of the 2026-11-23 fold campaign

- `OfPath` (the mount table behind the 9p/drvfs gates) takes the LAST entry of a mountpoint listed twice, the topmost
  mount, as `df` answers; it took the first.
- `ModuleVersionFromZip` bounds the bytes it reads over all duplicate `META-INF/module.xml` entries together
  (one shared limit of 8 MiB), not per entry.
- `mcp-n4` read retry: a `loadSlots` asked again after a timeout counts as one more load outstanding (a timeout does
  not prove the station dropped the first), so a stray second load event is never trusted as the one asked for.
- `lint-gradle-userhome` continues a `gradle.properties` line only on an ODD number of trailing backslashes, as
  `java.util.Properties` does (`\\\` continues, `\\` and `\\\\` do not); it treated three and five as a pair.
- `modulejars.Dir` and `Suffix` are used by the mirror, the jar-lock check and `lint-ntpl-manifest-deps` instead of
  their own `"modules"` / `".jar"` copies.
- Tests only, no behaviour change: one-argument `set*` calls on a servlet response, a request or a StringBuilder are no
  write for `lint-servlet-write-guards`; a zip with zero entries never reads as equal in the drift gate; mode D of
  `ng-deploy` gates and backs up the `-wb` jar, refuses a left-over `-wb` `.ng-deploy-old` and a wrong
  `EXPECTED_WB_TYPES`; a station that is `<NIAGARA_HOME>/modules` skips the target-version check; `UndescribedEnd` flags
  an error entry; an interrupted client does not retry a read; an unlistable `modules/` is the list error.
