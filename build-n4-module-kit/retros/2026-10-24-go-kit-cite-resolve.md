<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-24 · kit · go-kit-cite-resolve

**Session**: niagara-research quality round 10, branch `feat/r10-kit-cite-resolve` from origin/main 2285da0
**Delta count**: 1

## What happened
The round-10 citation audit needed a pre-filter: a tool that resolves every evidence pointer of the kit
cards so a reviewer only judges whether the cited source says the claim. New Go tool `go/cmd/kit-cite-resolve`
(`go/internal/citeparse`, the pure grammar; `go/internal/citeresolve`, one resolver per pointer kind over
injected roots) behind `toolbelt/kit-cite-resolve.sh`. It reads the clauses of ev and deferred tags (retro
deltas, corpus blocks with sections, rows, claims and rules, code, devguide/guide/niagara-help lines, memories,
root-relative paths), per-card path aliases (`V=`, `G/` =, `cited CL:`), ellipsis prefixes and prose citations,
and prints one row per pointer with a status and an 80-character excerpt. The first real run over the 426 kit
cards reported 2046 non-RESOLVED pointers, of which a large share were false: nine classes were fixed, each
pinned by a case, before the first push.

## Evidence
- Parser and resolver table tests cover every case of the spec (valid retro Δ, `B1255 §1255.4`,
  a devguide line range, a V-alias line citation, a two-clause tag, a prose citation; `corpus OBS-2`
  BARE_ID, `B1023 §ND4` MISSING_SECTION, a range past EOF, a retro lacking Δ9, an unreadable research root
  exit 3, an unterminated ev tag UNPARSED) `[ev: go/internal/citeresolve/resolve_test.go TestRetroAndCorpus]`
  `[ev: go/internal/citeresolve/command_test.go TestMainUnreadableRoot]`
- Observed mutations: a number-prefix section match fails 2 cases; a Δ matched anywhere on a line fails
  "retro lacking the delta"; dropping the --strict exit fails 1 case; dropping the ENOTDIR guard fails 3
  `[ev: go/internal/citeresolve/corpus.go numbered]`
- Real run, 8001 pointers on 426 cards: first run RESOLVED 5955, MISSING_FILE 574, MISSING_SECTION 424,
  MISSING_DELTA 83, LINE_OUT_OF_RANGE 49, BARE_ID 8, UNPARSED 889, TOOL_ERROR 19 (exit 3); after the truth
  fixes RESOLVED 6515, MISSING_FILE 372, MISSING_SECTION 151, MISSING_DELTA 0, LINE_OUT_OF_RANGE 73, BARE_ID 8,
  UNPARSED 882, about 6 s `[ev: retro go-kit-cite-resolve]`
- The false classes: a plain file in `organized/` made every decompile search TOOL_ERROR (ENOTDIR); deltas
  written as numbered rows of a `| Δ |` or `| # |` delta table; § ranges (`958.3–958.4`, `867.2-3`, `1-2`);
  unquoted titles running into prose; `code <part> <ClassName>`; guide-relative, organized-relative and
  `javax/...` package paths; an absolute path joined to a root; `dir/…/suffix` `[ev: go/internal/citeresolve/paths_test.go TestFilePointers]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **A new checker runs once on the real tree before its first push, and every false-finding class it shows becomes a pinned case before release.** Why: kit-cite-resolve's fixture suite was green while the first real run reported 2046 non-RESOLVED pointers, 9 classes of them false (RESOLVED 5955 → 6515 of 8001 after the fixes). | `CONTRIBUTING.md` § 2.2 Go module | `[ev: retro go-kit-cite-resolve]` |

## Lessons
- A citation grammar written from a spec meets many more shapes in real cards (wrapped tags, delta tables, §
  ranges, titles that run into prose); the real run is the acceptance test, the fixtures pin what it found.
- A plain file in a directory of directories is a normal state, not a tool failure: treat ENOTDIR as absent.
- RESOLVED only says the cited place exists; the excerpt is what the human judges.

---
**Status**: FOLDED 2026-10-24 (branch `docs/r10-fold-go-infra`) — Δ1 `BUILD-LOOP.md` § 5 K22 real-tree line (any new checker).
