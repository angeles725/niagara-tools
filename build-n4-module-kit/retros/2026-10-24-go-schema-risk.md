<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-10-24 · kit · go-schema-risk

**Session**: niagara-research quality round 10, branch `r10/schema-risk-go` from origin/main c894b37
**Delta count**: 3

## What happened
`toolbelt/schema-risk.sh` (bash + three gawk programs + a CSV heredoc) is now the thin `go/go-tool.sh` wrapper
over `go/cmd/schema-risk` (`go/internal/schemarisk`). It is a standalone command, not an n4lint rule: it diffs two
snapshot directories and exits on the verdict scale 0 SAFE / 1 LOSSY / 2 OUTAGE / 3 usage / 4 env, which no lint
engine rule kind (Rule, TreeRule, RootRule; rows `SEV check path:line msg`, exit 0/1/3) can express. The B795
§795.4 table is embedded (`b795-795.4.csv`) and SR-CSV compares it with the fixture oracle. The walk reuses
`lint.Walk` and annotation depth `javasrc.ParenDepth`.

One declared difference, its own commit with RED: several rows of one category (removed, added, kind swap,
retype) print in declaration order; bash printed gawk `for (nm in …)` hash order, which nobody can predict.

## Evidence
- RED (package missing) → GREEN; 8 fixture goldens recorded from bash byte-identical `[ev: go/internal/schemarisk/command_test.go TestGolden]`
- 330 real snapshot pairs (14 Cliente modules crossed, 67 consecutive git-revision pairs of 10 tracked modules, both
  ways), stdout + stderr + exit: port commit 142 byte-identical + 188 order-only, 0 other; after the order fix 140 +
  190, 0 other (every order-only pair permutes rows inside one category only)
- Order fix RED: name order put `Multi.alpha` before `Multi.zeta` `[ev: go/internal/schemarisk/testdata/multi_row.golden]`
- SR-multi fails on the bash body before the swap `[ev: tests/schema-risk.bats SR-multi]`
- Truth fixes T1-T3 (Δ1-Δ3 below), each one commit with its RED, a Go test and a bats pin `[ev: tests/schema-risk.bats SR-empty-before]` `[ev: tests/schema-risk.bats SR-unreadable-java]` `[ev: tests/schema-risk.bats SR-unbalanced]` `[ev: tests/schema-risk.bats SR-type-name]`
- 330 real pairs after T1-T3, against the origin/main bash: 140 byte-identical, 145 order-only, 45 changed beyond order, all 45 T1: the five slotless -ux/-wb modules (UmbrellaDashboard-ux, Apillm-wb, two client dashboard -ux modules, the client-E dashboard -ux) as before against the nine slotted ones, LOSSY remove rows in bash, the same slots as add_slot SAFE now. T2 and T3 change no real row: no client Java file is unreadable or holds an unparsed annotation, and every client module-include.xml parses and names the same type

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | DONE (T1, this branch): an empty before snapshot (no slot harvested) read the after slots as **removed** (awk `NR == FNR` is true for the second file when the first is empty); every after slot is now add_slot, SAFE. Closed (by design): a pair with no slot, or no `*.java`, on either side stays SAFE with no row, since schema-risk classifies changes and a missing snapshot directory is already env (4) | `go/internal/schemarisk/schemarisk.go` Diff | `[ev: go/internal/schemarisk/command_test.go TestMainEdges]` |
| Δ2 | DONE (T2, this branch): an unreadable `*.java` stopped the harvest (awk fatal on stderr) and the verdict came from the slots read before it (fail-open); it now exits 4 with `error: cannot read <file>: <reason>` and no row | `go/internal/schemarisk/command.go` javaLines | `[ev: go/internal/schemarisk/command_test.go TestUnreadableJava]` |
| Δ3 | DONE (T3, this branch, two commits): an annotation that never balances in its file or has no name is now an UNKNOWN row (OUTAGE) as L3 said, and the type name comes from the first `<type>` element (xmlsrc; malformed XML is env 4), never a `<dependency>` | `build-n4-module-kit/toolbelt/schema-risk.sh` header | `[ev: go/internal/schemarisk/schemarisk.go Parse]` |

## Lessons
- A port of an awk program that iterates `for (k in arr)` cannot be byte-identical once an array holds two keys; measure old vs new as "byte-identical or order-only inside one category" and fix the order in its own commit.
- A tool whose exit scale is a verdict belongs beside the lint engine as a `cli.Main` command, not inside it.
- Δ1-Δ3 are applied in this unit as code fixes; the retro stays pending until a fold review decides whether any of them becomes doctrine.

---
**Status**: PENDING
