### Changed

- `build.sh` (retro Δ28): every mirror build now opens, before gradle, with a warning that the jar is signed with
  the throwaway key of the mirror user home (not your signing profile) and cannot be installed on a station. It names
  the build that works: the same gradle tasks with `-Pniagara_home=<mirror>` under the real `HOME`. Until now the
  warning came only after the build, and only when the plugin had created a profile there.
- `run-pure-test.sh` (retro Δ30): with `NIAGARA_HOME` set it also compiles every source under the part's `src/`
  (`javac -proc:none`, jars of `<NIAGARA_HOME>/modules` on the class path), so a helper deleted from a Baja adapter
  fails the pure run instead of only gradle. A compile error is exit 3 naming the file, before the tests run; a
  `NIAGARA_HOME` with no jar in `modules/` is exit 3. Without `NIAGARA_HOME` one NOTICE on stderr says the adapters
  were not compiled; the exit code is unchanged.
- `build.sh` (retro Δ29): the plugin version auto-detected from the module's pin (`gradle.properties`, else the
  `settings.gradle.kts` default) no longer fails the build when the niagara_home ships another one. When the pin is
  not under `etc/m2/repository` and the home ships exactly one plugin version, that one is used (a NOTICE names both)
  and preflight gets it too; with several or none the build ends with exit 10 naming the pin and the versions found.
  An explicit `--plugin-version` is never replaced.
