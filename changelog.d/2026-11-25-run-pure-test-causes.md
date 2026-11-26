### Changed

- toolbelt: `run-pure-test.sh` names the cause of a class JUnit cannot load: a class name built from the path (`test.com.x.FooTest`) instead of the file's `package` line stops with exit 3 and `class <name> not found; the package line of <file> says <pkg>`, and a relative part dir is made absolute (#1115).
- toolbelt: `run-pure-test.sh --all <part dir>` runs every `*Test.java` under `srcTest/` in one javac and one JVM and prints `run-pure-test: N classes, M tests, F failures`; the exit code is JUnit's (#1115).
