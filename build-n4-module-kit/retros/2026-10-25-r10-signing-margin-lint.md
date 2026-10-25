<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-25 · kit · r10-signing-margin-lint

**Session**: niagara-tools quality round 10, lint-structure L16 FAILs a build whose declared LocalSigningProfile has
validity - refresh below 32 days, one predicate with the run-niagara-test gate, branch `r10/signing-margin-lint`
from 04f031a1 (origin/main)
**Delta count**: 1

## What happened
`run-niagara-test run` already refuses a LocalSigningProfile with validity - refresh < 32 (retro
r10-scaffold-station-test Δ5), but only when a station test runs: a plain `build.sh` of the same tree still signed
jars that the NRE then drops. The rule now lives once, in `go/internal/signingprofile` (`Parse`, `Profile.Safe`,
`Profile.Refusal`, `GradleProperty`, `MinMargin = 32`), and two callers read it:
1. `run-niagara-test` (`go/internal/niagaratest/signingprofile.go`), behaviour and messages unchanged, except that a
   validity or refresh outside the plugin's 32-bit Int is now "not a whole number of days" (a 64-bit pair could
   overflow validity - refresh into a pass).
2. `lint-structure.sh` L16 (`go/internal/lint/rules/structure/module.go` `signingMargin`). It reads the
   `niagaraSigningProfile` that the gradle root's `gradle.properties` names (the directory of the settings.gradle.kts
   that L14 and L15 already find), resolves a relative path against that root, and FAILs a refused profile on the
   `gradle.properties` row, naming both values and the reason. No property: no row. A named profile that does not
   exist: WARN, margin unchecked. A profile that is not properties XML, or a non-integer day count: exit 3 with
   `lint-structure: cannot scan <profile>: …`, as for any file the lint cannot read.

Why lint-structure and not a verify-module row: verify-module judges a built jar and reads `--src` per profile; it
reads no gradle.properties and no gradle root. lint-structure already reads every gradle.properties up to the `.git`
sentinel (L10, L12, L13) and the gradle root's signing block (L15), so L16 needs no new discovery. The user-level
`GRADLE_USER_HOME/gradle.properties` stays out of a source lint; the runner keeps reading it first.

## Evidence
- RED on origin/main 04f031a1 with the new tests and no L16: `TestSigningMargin` 7 of 12 cases FAIL (30/15, 214/183,
  no profileType entry, a malformed value, a value beyond Int, a non-XML file, a missing profile); bats `LS16-margin`
  and `LS16-malformed` FAIL (`[ "$status" -eq 1 ]`, `[ "$status" -eq 3 ]`). GREEN after the change; the runner's
  `TestSigningProfileRefusal`, `TestSigningProfileRefusalNonLocalType` and `TestRunRefusals` (the 30/15 case) stay green unchanged.
  `[ev: code go/internal/signingprofile/signingprofile.go]`
- Mutation `MinMargin` 32 → 31: `TestRefusal`, `TestSigningProfileRefusal/31 days apart is refused`,
  `TestSigningMargin/214/183 (margin 31) FAILs` and bats `LS16-margin` turn red.
- Generic fixtures only (synthetic trees under a temp dir). A read-only sweep found no kit fixture, template or
  client tree that declares a LocalSigningProfile, so this is prevention.

## Vendor defect
The niagara signing plugin computes a LocalSigningProfile key's expiry wrongly. Facts, from `niagara-plugins` 7.6.22
`LocalSigningProfile.kt` read with `javap -c -l` (line numbers are the class file's LineNumberTable):
- `:373-374` `generateCert` refuses only `validity <= refresh` ("The validity period must be greater than the
  refresh period").
- `:386-388` the defect: notAfter is `Calendar.getInstance()` then `cal.set(Calendar.DAY_OF_MONTH, validity)`. That
  SETS the day of the current month to `validity`, which the lenient Calendar rolls forward; it does not add
  `validity` days. A key minted on day d of a month has `validity - d` days left.
- `:216-225` `getCertInfo` deletes the chain and mints a new key when the whole days left to notAfter are fewer
  than `refresh`.

Predicate: a fresh key is due for refresh at once, so every signing re-mints it, whenever `validity - d < refresh`.
d reaches 31 and the day count is floored, so a profile is safe on every date only when
**validity - refresh >= 32** (`signingprofile.MinMargin`). Such jars carry a signature block no JVM verifies
(`cannot verify signature block file`), the NRE drops them and test.exe prints only `invalid test target(s)`.

Corroboration (keytool -list -v on the minted key, 25 October): a 30/15 profile gave notAfter 30 October, 5 days
left, so a new key at every signing; 365/183 gave notAfter 30 September of the next year, 340 days left, stable.

**Re-check on plugin upgrade:** when the kit moves to a `niagara-plugins` newer than 7.6.22, run `javap -c -l` on its
`LocalSigningProfile` class and read `generateCert`'s notAfter. If it adds days (`cal.add(Calendar.DAY_OF_YEAR,
validity)` or an equivalent) the safe margin is `validity > refresh` and `MinMargin` can drop for that plugin
version; if it still calls `cal.set(Calendar.DAY_OF_MONTH, …)`, keep 32. Re-run the keytool check on a 30/15
throwaway profile either way and record the result here.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | `lint-structure.sh` L16 FAILs a gradle root whose `gradle.properties` names (`niagaraSigningProfile`) a LocalSigningProfile with validity - refresh < 32, naming both values; a missing profile WARNs, a malformed one is exit 3. The predicate is `go/internal/signingprofile`, shared with `run-niagara-test run` | `build-verify.md` § Signing per deploy target; `toolbelt/lint-structure.sh` header | `[ev: code go/internal/signingprofile/signingprofile.go]` |

## Lessons
- A gate that runs only in one tool leaves every other path open: the runner refused the profile, but a plain build
  of the same tree still signed jars the station would drop. Put the predicate in a package and call it from each
  place that reads the input.
- In bats, `[ a ] && [[ b ]]` in the middle of a test never fails it (`set -e` ignores a failing command that is not
  last in an `&&` list): the first mutation run of `LS16-margin` stayed green until the line was split.

## Fold
- The rule is code in the branch that wrote this retro (`go/internal/signingprofile`, lint-structure L16).
- Δ1 is cited in `build-verify.md` § Signing per deploy target (the LocalSigningProfile margin bullet). Nothing owed.

---
**Status**: FOLDED 2026-10-25 (branch `r10/signing-margin-lint`; log in `ledger.d/2026-10-25-r10-signing-margin-lint.md`) — Δ1 in `build-verify.md` § Signing per deploy target.
