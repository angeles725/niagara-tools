<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-11-16 · kit · ntpl-budget-t26

**Session**: kit row T26 of the 2026-11-15 coordination-stretch retro: the bounds of lint-ntpl-manifest-deps,
and the native-review advisories of #710.
**Delta count**: 3

## What happened
The #710 verifier (A4) built a jar of about 114 KB holding 200 templates, each with a template.bog that
inflates to the full 32 MiB per-entry bound. The jar's own bounds (32 MiB per template, 128 MiB for all
templates) held, but each template kept its own 128 MiB budget for its bog and Px entries: the run asked for
200 x 32 MiB of reads and took 132.6 s (RSS 183 MB). A module directory of many small .ntpl files had the same
cost class with no count bound. The native review of #710 also noted that a jar over 32 MiB was opened before
its size bound (the .ntpl path checks the size first), that jar entry names might be matched case-sensitively,
and that the WARN rows of a jar had no test.

## Evidence
- RED before GREEN: the amplifying jar (200 templates, a 32 MiB bog each, built in t.TempDir) exits 0 with no
  row on the base after minutes, and ends in about 4.5 s with one FAIL ntpl-uses row on the candidate; a module
  and a jar of 1001 small templates exit 0 on the base and give one FAIL ntpl-entries row; an unreadable jar
  over 32 MiB is exit 3 on the base and the size-bound row (exit 1) on the candidate.
  `[ev: go/internal/lint/rules/ntplmanifest/budget_test.go]`
- The case and WARN advisories were already met on the base: `T.NTPL` and `U.Ntpl` entries are read (the
  `templateNames` patterns are letter-case classes), and a jar without an installation gives the noHome,
  ntpl-deps and ntpl-uses WARN rows with exit 0. Both now have a test that pins them.
- Profile of the amplifying jar: 91% of the time is the XML parse of the bog and Px entries (about 30 MiB/s),
  so the run budget is 128 MiB, the bound one archive had before, not 512 MiB (16 s of parse).

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-16. One inflate budget of 128 MiB for the whole run: every template of a jar and every manifest, bog and Px entry draw on it; the entry it leaves no room for is one FAIL row and nothing more is read. At most 1000 templates are read per run, past them one FAIL ntpl-entries row | Go `internal/lint/rules/ntplmanifest` `budget.go`, `archive.go`, `usesread.go`, `jar.go`, `ntplmanifest.go`; `toolbelt/lint-ntpl-manifest-deps.sh` header | `[ev: go/internal/lint/rules/ntplmanifest/budget.go]` |
| Δ2 | APPLIED 2026-11-16. A jar's size bound is read from its metadata before it is opened (`openBounded`, shared with the .ntpl path) | Go `internal/lint/rules/ntplmanifest` `archive.go`, `jar.go` | `[ev: go/internal/lint/rules/ntplmanifest/jar.go]` |
| Δ3 | APPLIED 2026-11-16. Tests pin the any-case jar entry names and the WARN rows of a jar run without an installation (no code change: both already held) | `budget_test.go` | `[ev: go/internal/lint/rules/ntplmanifest/budget_test.go]` |
