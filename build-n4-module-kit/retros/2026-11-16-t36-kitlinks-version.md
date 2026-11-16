<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-11-16 · kit · t36-kitlinks-version

**Session**: row T36 of the 2026-11-15 coordination-stretch retro: the native-review advisories of niagara-tools #719
(WARNING `scripts/lib/ci-local-kitlinks.sh:82-83`, SUGGESTION `:83`, SUGGESTION `scripts/lib/ci-local-tools.sh:25`).
**Delta count**: 3

## What happened
The review record of #719 was not retrievable (no PR comments; the burned review artifacts are gone), so the three
findings were re-derived from the code. The pinned-version check asked `bats --version` in the caller's directory,
then kit-links ran a bare `bats` inside the snapshot: a relative PATH entry could resolve to another binary there,
and kit-links passed on a bats nobody checked. Now the check resolves the binary first on PATH to an absolute path,
compares that binary's version, and kit-links runs that same path. A mismatch line now names the pin, the version
found and its path. `curl --max-time 120` is per try, so `--retry 3` allowed four full tries (~8 minutes); the
download now also passes `--retry-max-time 60`, about 3 minutes in all.

## Evidence
- RED before GREEN: CL38 (a wrong-version stub reached through `PATH=tests/stubbin:…` from another directory) passed
  kit-links on origin/main and the stub ran; CL39 (mismatch line names pin, version and path) and CL24
  (`--retry-max-time 60`) failed on origin/main. `[ev: tests/ci-local-kitlinks.bats]` `[ev: tests/ci-local-pins.bats]`
- Mutations, not committed: running a bare `bats` again turns CL38 red; dropping the version comparison turns CL35
  and CL39 red; dropping `--retry-max-time` turns CL24 red. `[ev: tests/ci-local-kitlinks.bats]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-16. kit-links runs the bats binary whose version was checked, by its absolute path | `scripts/lib/ci-local-kitlinks.sh` | `[ev: tests/ci-local-kitlinks.bats]` |
| Δ2 | APPLIED 2026-11-16. A bats version mismatch names the pin, the version found and its path in its one line | `scripts/lib/ci-local-kitlinks.sh` | `[ev: tests/ci-local-kitlinks.bats]` |
| Δ3 | APPLIED 2026-11-16. A pinned-tarball download starts no retry after 60 s (`--retry-max-time 60`) | `scripts/lib/ci-local-tools.sh` | `[ev: tests/ci-local-pins.bats]` |
