### Added — build.sh --werror

- New `toolbelt/build.sh --werror`: an opt-in build that compiles every part with `-Xlint:all -Werror` through the
  gradle init script `toolbelt/lib/werror.init.gradle`; a javac warning fails the build (exit 30) with the remedy
  printed (fix it, or a scoped `@SuppressWarnings`). Off by default.
