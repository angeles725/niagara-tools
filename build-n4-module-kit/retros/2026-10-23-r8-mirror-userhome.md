<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-23 · kit · r8-mirror-userhome

**Session**: niagara-research quality round 8, mirror-build isolation guard, branch `fix/r8-mirror-userhome` from origin/main 1e39076
**Delta count**: 1

## What happened
An offline build against a mirror Niagara home (`toolbelt/mirror-niagara-home.sh`) set
`org.gradle.jvmargs=-Duser.home=<scratch>` in a scratch `gradle.properties` to keep the build away from the
real user home. The Gradle 7.6 daemon's command line carried no `user.home`, so the niagara-signing plugin's
default-profile lookup (`user.home/.tridium/security/niagara.signing.xml`, SigningProfileFactory in
niagara-plugins 7.6.17) read the real `~/.tridium/security/niagara.signing.{xml,jceks}`. Read only; nothing
was written. `JAVA_TOOL_OPTIONS=-Duser.home=<scratch>/mirror-user-home` with `GRADLE_USER_HOME` pinned to the
real `~/.gradle` isolated it and kept offline dependency resolution working.

## Evidence
- Local mirror build 2026-10-22: daemon command line without `user.home`, real keystore read; isolated run with `JAVA_TOOL_OPTIONS` `[ev: CERT-live local build 2026-10-22]`
- RED 2/3 then GREEN 3/3: `tests/build-sh.bats` BS-mirror-userhome, BS-mirror-userhome-guh (BS-mirror-userhome-control stays green); `tests/mirror-niagara-home.bats` M6 `[ev: tests/build-sh.bats]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | `org.gradle.jvmargs -Duser.home` (and `systemProp.user.home`) in `gradle.properties` does not reach an already-running Gradle daemon; isolate user.home via `JAVA_TOOL_OPTIONS` + `--no-daemon` for mirror builds | `toolbelt/build.sh` (mirror mode) + `build-verify.md` § Mirror | `[ev: CERT-live local build 2026-10-22]` |

## Lessons
- A JVM argument in `gradle.properties` (`org.gradle.jvmargs=-Duser.home=…` or `systemProp.user.home=…`) only reaches a daemon that starts after it; a reused daemon keeps its own command line.
- Review correction: the mirror path cannot contain whitespace (exit 33), a dev-key signature in the mirror user home is a WARN, and `maven.repo.local` is pinned to the real `~/.m2`.
- Moving `user.home` also moves the default Gradle user home, so pin `GRADLE_USER_HOME` before isolating it.

---
**Status**: FOLDED (same unit) — Δ1 `toolbelt/build.sh` mirror-mode isolation + `build-verify.md` § Mirror.
