### Changed — run-niagara-test names the plugin module-include trap in the gradle-root FAIL row

- `run-niagara-test run`: when the build changed a `*/module-include.xml` in the gradle root, the
  `gradle root changed by the build` FAIL row now names the known cause (niagara gradle plugin 7.3.40 and 7.6.22,
  observed, append the test types to the production `module-include.xml` in the moduleTest compile when the test
  sources changed), points at `--restore-gradle-root` and `types/moduleTest.md` (plugin-version trap), and says
  whether the restore checked it out. Other files keep the generic row; the run stays FAIL with the same exit code.
