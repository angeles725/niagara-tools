### Fixed — `build.sh --werror` survives a build that replaces the compiler args

- `toolbelt/lib/werror.init.gradle` adds `-Xlint:all -Werror` at `taskGraph.whenReady`, only when absent, so a build
  script or plugin that sets `options.compilerArgs = [...]` no longer turns `--werror` into a silent no-op.
