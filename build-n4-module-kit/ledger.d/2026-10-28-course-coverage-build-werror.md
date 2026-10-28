<!-- ledger: kit -->
<!-- date: 2026-10-28 -->
<!-- last-change-note: kit-backlog B11.1: build.sh --werror compiles with -Xlint:all -Werror -->
log: 2026-10-28 · kit-backlog B11.1 (course coverage run B) → `toolbelt/build.sh --werror` passes `--init-script toolbelt/lib/werror.init.gradle`, which adds `-Xlint:all -Werror` to every `JavaCompile` (Groovy `tasks.withType`, Gradle 4.0.2 and 7.x); off by default; a failing `--werror` build prints the remedy (fix the warning or a scoped `@SuppressWarnings`) before exit 30. `METHODOLOGY.md` "Compiles warning-clean", `build-verify.md` § Raw gradle; `tests/build-sh.bats` BS-werror, BS-werror-off, BS-werror-hint. Not yet run against a real module: the `-Xlint:all` set on a real Slot-o-Matic module is unmeasured. [ev: retro course-coverage Δ22]
