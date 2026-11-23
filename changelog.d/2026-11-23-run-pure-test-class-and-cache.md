### Fixed — run-pure-test: default-package classes, a symlinked Gradle cache, the jar copy it picks

- `run-pure-test.sh <module-rt-dir> <Class>` runs a test in the default package
  (`srcTest[/test]/<Class>.java`); it looked for `<Class>/<Class>.java` and always ended "test source not found".
  The not-found message names the path it looked for.
- A `~/.gradle` that is a symlink (a shared or relocated cache) is followed, as `find -H` does; before, JUnit was
  reported missing (exit 3) from a cache that held it.
- With several copies of `junit-4.13.2.jar` or `hamcrest-core-1.3.jar` under `~/.gradle`, a NOTICE line on stderr
  names the jar, how many copies there are and the one used (still the first in name order). A cache with one copy
  prints nothing new. Retro `2026-11-14-go-toolbelt-small-port` Δ3.
