### Added — preflight --toolchains

- New opt-in `toolbelt/preflight.sh --toolchains` row for the raw or Windows `gradlew` route: PASS when
  `gradle.properties` pins `org.gradle.java.installations.paths` to a full JDK 8, else runs
  `gradlew -q javaToolchains -Porg.gradle.java.installations.auto-detect=true` and FAILs when no full JDK 8
  (Language Version 8, Is JDK true) is listed; SKIP when `gradlew` is not executable or lists nothing.
