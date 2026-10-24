<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-23 · kit · r9-followups

**Session**: niagara-research quality round 9 (Ayudante3), branch `r9/followups` from origin/main 94a6a8d
**Delta count**: 1

## What happened
The advisory follow-ups of the round-9 reviews (#341, #344, #355) were closed mostly as test pins of behaviour
that was already right. One kit behaviour changed: lint-guard-pins printed two rows for a lint whose only
`# Mutation:` lines sit below the header. The first row named each stray line. The second row said "no # Mutation:
line ... add one", which hid that the lines exist. The second row now says `no # Mutation: line in the header
(K24(7): move the declarations below it above the first code line)`. A lint with no declaration anywhere keeps the
old row. No lint in the real tree is in that state, so `lint-guard-pins.sh .` prints the same 218 rows before and
after.

gen.File's newline-trim asymmetry was kept. Under TrimNewlines, same() trims ALL trailing newlines from the committed
file and ONE from the content. Both index generators render content that ends in exactly one newline, so the
asymmetry is unreachable today. A test now documents it: content ending in "\n\n" is never fresh, not even right
after a write.

## Evidence
- RED: TestMainOnlyStrayDeclarations got the generic "add '# Mutation: ...'" row; GREEN after the switch in
  command.go `[ev: go/internal/guardpins/command_test.go TestMainOnlyStrayDeclarations]`
- Real tree: `bash build-n4-module-kit/toolbelt/lint-guard-pins.sh .` byte-identical before/after (218 rows, exit 0)
- Teeth of the pure pins, each by a hand mutation that was then reverted: dropping either `|| refuse` or the
  origin/HEAD fallback in .githooks/pre-push fails H22/H23/H24. Pointing lint-no-system-out at `*.javax` fails the
  exact AL23 and passes the loosened one. Dropping HAD_ENV=1 from the member exit-3 relay fails RM23b. TrimRight on
  both sides of same() fails TestFileSameTrimAsymmetry. Requiring '#' in blankOrBareHash fails both new TestExtract
  cases `[ev: tests/build-retro-sync.bats H22]`
- buildstate surprise: an extra trailing newline on BUILD-STATE.md is FRESH, not stale. The generator rewrites only
  the kit block and carries the rest of the committed bytes through. TestMainCommittedBytes pins that
  `[ev: go/internal/buildstate/command_test.go TestMainCommittedBytes]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | List the new `no # Mutation: line in the header` row variant in the lint-guard-pins.sh header's row list | `toolbelt/lint-guard-pins.sh` header (row shapes) | `[ev: go/internal/guardpins/command_test.go TestMainOnlyStrayDeclarations]` |

## Fold
- Δ1 → already in code (`toolbelt/lint-guard-pins.sh` header row list); doctrine line `METHODOLOGY.md` K24 (7).

## Lessons
- A pin needs a mutation that the old, looser assertion lets pass. Otherwise it only proves that the code runs.
- Check what a generator copies through before you pin its comparison rule. BUILD-STATE.md is byte-exact, but
  its extra trailing newline still compares fresh.

---
**Status**: FOLDED 2026-10-24 (branch `docs/r10-fold-meta`) — INDEX row: `| 2026-10-23-r9-followups.md | kit | 2026-10-23 | folded | 1 |`
