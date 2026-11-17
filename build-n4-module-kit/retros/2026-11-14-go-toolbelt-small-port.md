<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 5 -->
# 2026-11-14 · kit · go-toolbelt-small-port

**Session**: "all tool logic in Go" pass, wave 3: `toolbelt/orient-guard.sh`, `mirror-niagara-home.sh` and `run-pure-test.sh` become launchers over Go commands; `lib/method-boundary.sh` and `lib/signing-block.sh` are deleted.
**Delta count**: 5

## What happened
The last three toolbelt scripts with logic of their own were ported with a differential of 133 cases (stdout,
stderr, exit code, resulting tree): 116 identical, 17 declared. Eight of the declared cases are false successes
turned into refusals: the location guard passed any path it could not resolve on its raw text, so
`<prefix>/New/../../elsewhere` was a bypass, and the mirror tool printed "mirror ready" over links that all
dangled. Nine defects of the bash tools were kept byte-identical and are grouped below. Two bash libs lost their
last caller and went, with the awk oracle test that ran one of them.

## Evidence
- `orient-guard`: a relative path with two missing levels is compared as raw text and FAILs inside the prefix; the
  FAIL row names the default prefix even when the seam prefix was compared; a home directory ending in `//` gives a
  prefix that never matches; an option-like argument is compared raw; a trailing newline in the path is dropped.
  Kept in `go/internal/orientguard/canon.go` and `guard.go`. `[ev: 90bc6d1b]`
- `mirror-niagara-home`: the parent of the mirror dir is created before guard A, so a refused mirror path inside
  the source install leaves an empty directory there; a symlinked mirror dir spelled with a final slash empties
  the link's target. Kept in `go/internal/mirrorhome/paths.go` and `build.go`. `[ev: f81997e5]`
- `run-pure-test`: a class without a package is looked up under a directory of its own name; a symlinked Gradle
  cache is not followed, so JUnit is reported missing; with several copies of a jar the first in name order is
  taken. Kept in `go/internal/puretest/locate.go`. `[ev: bca3dcc9]`
- A launcher run under a changed or unset home directory rebuilds its binary with a fresh build cache; the three
  bats suites gained a `setup_file` prebuild to stay fast and hermetic. `[ev: 90bc6d1b]`
- `BUILD-LOOP.md` and `skill/SKILL.md` carry single lines of several thousand characters: a one-phrase edit shows
  the whole line as added, and a search truncated to a terminal width misses mentions inside them. `[ev: 3eabc38b]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-17. The location guard resolves the longest existing ancestor and judges the remainder lexically, so a new directory several levels deep inside the prefix passes; the FAIL row names the prefix that was compared; a trailing slash run in the home directory is normalised; an option-like argument is a usage error. Choices: an argument starting with `-` (but `-` alone) is exit 3 with a stderr line that says to spell it `./-name`, before any row; a path that cannot be judged (link loop, unsearchable directory, empty) FAILs naming it as given; every final slash of HOME is dropped | `go/internal/orientguard/canon.go` · `guard.go` · `tests/orient-guard.bats` | `[ev: 90bc6d1b]` `[ev: 5767f044]` |
| Δ2 | APPLIED 2026-11-17. The mirror tool runs guard A on the lexical path before it creates anything, and treats a symlinked mirror dir as "exists and is not a directory made by this tool" whatever its spelling. Choices: guard A judges a mirror dir that is not there yet by `orientguard.Reachable` (longest existing ancestor resolved, rest by text) before its parent is made, and refuses one that cannot be resolved (exit 20); a symlink at the mirror dir (`link`, `link/`, `link/.`, dangling) is a guard B refusal, exit 20, after guard A, so a link to the source keeps guard A's message and a looping link the unresolved one; a mirror dir that is not there yet is made where guard A judged it, never along its spelling, so `<src>/new/../../out` creates nothing inside the source, and guard B judges that path too (`missing/../dir` cannot write into a non-mirror `dir`); a dangling link is refused before anything is made | `go/internal/mirrorhome/paths.go` · `guard.go` | `[ev: f81997e5]` `[ev: 5767f044]` |
| Δ3 | `run-pure-test` accepts a class in the default package, follows a symlinked Gradle cache, and names the jar it picked when more than one copy exists | `go/internal/puretest/locate.go` · `messages.go` | `[ev: bca3dcc9]` |
| Δ4 | Every bats suite of a Go launcher prebuilds its binary in `setup_file`; a lint refuses a suite that runs a launcher under a changed home directory without it | `tests/shell-hygiene.bats` · `tests/test_helper` | `[ev: 90bc6d1b]` |
| Δ5 | The oversized lines of `BUILD-LOOP.md` and `skill/SKILL.md` are wrapped to the repository width, and a lint bounds the line length of live doctrine cards | `build-n4-module-kit/BUILD-LOOP.md` · `skill/SKILL.md` · kit lints | `[ev: 3eabc38b]` |

## Lessons
- A guard that falls back to the raw text when it cannot resolve a path is open exactly where it matters; resolve
  what exists and refuse what cannot be judged.
- A tool that reports "ready" must have checked the thing it built, not only that its commands ran.
- A test that executes a retired implementation as an oracle keeps that implementation alive; find such callers
  before calling code dead.
- Search for a retired name with untruncated output.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-11-14-go-toolbelt-small-port.md | kit | 2026-11-14 | pending | 5 |`
