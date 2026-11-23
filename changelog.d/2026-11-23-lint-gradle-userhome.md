### Added — lint-gradle-userhome

- `build-n4-module-kit/toolbelt/lint-gradle-userhome.sh [--strict] <root>` warns when `gradle.properties` sets `user.home` (`org.gradle.jvmargs` with `-Duser.home=`, or `systemProp.user.home`); advisory, not in `report-module`.
