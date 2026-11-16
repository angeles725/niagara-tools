<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 4 -->
# 2026-11-16 · kit · bats-skips-client-source-k5-k9

**Session**: kit rows K5 and K9 of the niagara-research retro "coordination-stretch-deltas" (2026-11-15): where `check-bats-skips` refuses `^*`, and how `check-client-source` reads a BUILD-STATE an editor saved as UTF-16 or joined from two files.
**Delta count**: 4

## What happened
K5 asked to refuse `^*a`; the base already did (gate-followups-and-template-lints Δ16), and it stays refused.
K9 found that the refusal reached too far (`a^*`, which both matchers read alike) and that `check-client-source`
read a UTF-16 BUILD-STATE, and a byte-order mark in front of a key in the middle of the file, as "undeclared",
exit 0. Both halves are fixed, each RED first.

## Evidence
- Differential, base `dc5f4304` binaries against the branch, over ten allowlist patterns, the committed
  allowlist and seven BUILD-STATE encodings: every difference is one the rows name; UTF-8, UTF-8 with a mark
  and CRLF files read as before. `[ev: go/internal/checkbatsskips/dialect.go]`
- GNU grep 3.11 (`/usr/bin/grep`) on `^*a`, `a^*`, `(^*a)`, `x|^*a`, `(a)^*`, `b|a^*`, `^^*a`: the warning
  "* at start of expression" appears exactly for a `^*` that opens a branch; the lines matched equal those of
  Go's `regexp.CompilePOSIX` in every case tried. `[ev: go/internal/checkbatsskips/dialect_test.go]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-16. K5, K9: `check-bats-skips` refuses `^*` only where it opens a branch (the line's start, after `(` or `|`, after anchors only); `a^*`, `(a)^*`, `b|a^*` are allowed; `^*a` stays refused | `tests/bats-skip-allowlist.txt` header · Go `internal/checkbatsskips` `dialect.go` | `[ev: go/internal/checkbatsskips/dialect.go]` |
| Δ2 | APPLIED 2026-11-16. K9: `check-client-source` decodes a UTF-16 BUILD-STATE with its byte-order mark (either byte order) and refuses, exit 3, UTF-16 it cannot decode (no mark: NUL bytes; an odd byte count) and a UTF-8 mark that opens a line other than the first (naming the line) | Go `internal/checkclientsource` `encoding.go`, `command.go` | `[ev: go/internal/checkclientsource/encoding.go]` |
| Δ3 | OPEN. Gate-followups-and-template-lints Δ16 says GNU grep "reads a literal star" for `^*`; on 3.11 it warns but matches the same lines as Go in every case tried. The refusal rests on the warning (POSIX leaves the star undefined there), not on a different reading; correct that wording where it is repeated | `build-n4-module-kit/retros/2026-11-15-gate-followups-and-template-lints.md` Δ16 (as a later note) | `[ev: go/internal/checkbatsskips/dialect.go]` |
| Δ4 | OPEN. On the dev host `grep` on PATH is ugrep, which rejects `^*` outright: a comparison with GNU grep must call `/usr/bin/grep` (or check `grep --version`) | `CONTRIBUTING.md` § 2 (test runner setup) | `[ev: go/internal/checkbatsskips/dialect_test.go]` |
