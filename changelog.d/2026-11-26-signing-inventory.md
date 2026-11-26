### Added — signing-inventory: which signing certificates exist, where, until when, for which module

- `toolbelt/signing-inventory.sh` (Go, `go/internal/signinginventory`, issue #1160) is a read-only inventory of
  the signing material a developer with several Niagara installs and profiles loses track of. Per signing profile
  (`--profile`, else the user's `~/.tridium/security` and `~/.config/n4-signing` `*.signing.xml`, plus the `etc/`
  and `security/` of each `--home`, whose Niagara version it prints) it lists the keystore and each alias with the
  certificate subject and issuer, the days left (WARN under 60, FAIL expired), whether it is a code-signing
  certificate and not a CA (the `signing-doctor.sh` rules, now `signingdoctor.CheckX509`) and the file it lives
  in. The store is listed with the keytool of `$JAVA_HOME`, else the one on PATH (`go/internal/keytool`), opened
  with the profile's own store password through the environment; a store it cannot open is one `locked` or
  `unreadable` WARN row with its path only. No password, key or keytool message is ever printed. `--module <root>
  [--module-name <MOD>] [--build-state <file>]` adds the recorded `expected_signer` and `niagara_target` (#1158),
  the signer and compiled-for Niagara version of each `build/libs/*.jar`, and FAILs when the recorded alias is in
  no listed store (or only expired), a built jar carries another signer, or it was compiled for another
  major.minor than the recorded target. A profile that is not a regular file of at most 1 MiB is a WARN row (never
  read). Exit 0 / 1 a FAIL row / 2 usage / 3 no Go toolchain. Named in `skill/SKILL.md` and `build-verify.md`.
