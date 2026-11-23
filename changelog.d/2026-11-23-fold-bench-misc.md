### Added — a throwaway signing profile for a first build

- `build-n4-module-kit/templates/signing-profile.xml`: a commented profile with the fully qualified `profileType`, the store keys and the `keytool` command with the code-signing key usage; the scaffold's signing line and `BUILD-LOOP.md` § 1.a point at it.

### Changed — smaller kit messages and checks

- `toolbelt/verify-module.sh` usage opens with its own title, without the shellcheck directive and its rationale.
- `toolbelt/count-retro-deltas.sh` says on stderr to regenerate the index when it has no row for a retro; `tests/count-retro-deltas.bats` CRD4 runs it over every retro dated after 2026-10-24 (two named mismatches excepted).
- `tests/bats-skip-allowlist.txt` header: `^*` is refused because POSIX leaves it undefined, not because GNU grep reads it differently; compare with `/usr/bin/grep`, since `grep` may be ugrep.
