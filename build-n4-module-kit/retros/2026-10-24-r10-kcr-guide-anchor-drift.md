<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-24 · kit · r10-kcr-guide-anchor-drift

**Session**: niagara-research quality round 10, kit issue #511 (guides-clean `:line` drift), branch
`feat/kcr-anchor-drift` stacked on `fix/kcr-false-positives`, origin/main ceec6877 merged in
**Delta count**: 1

## What happened
The niagara-help `guides-clean` copies are rendered text, and a new renderer moves their lines. A `path:line`
pointer whose line moved still resolved: the file exists and the line is in range, so `kit-cite-resolve` said
RESOLVED while the line held other text. Issue #511 asked for a content anchor and one sweep.

1. Design. A quoted phrase right after a pointer's lines is its anchor: `guide <page>.txt:12 "phrase"` in a tag, or
   `` `G/<page>.txt:12` "phrase" `` in prose. The kit had no anchor convention before, and a quoted phrase after a
   backticked pointer was already the commonest way prose quoted a guide. The phrase must sit on a cited line,
   give or take 2 lines (whitespace and zero-width spaces ignored). Otherwise the row is DRIFT, naming the lines
   that hold the phrase now, or saying no line does. DRIFT is a strict finding on a live card and HISTORICAL_DRIFT on
   a history card.
2. The first GREEN run checked anchors in every file and turned 6 kit rows to DRIFT. All 6 were paraphrases of a
   claim after a kit or task-card pointer, not quotes of the cited line. The check now runs only in a guides-clean
   page; any other anchor is shown in the locator and noted as unchecked.
3. Sweep. The old 4.15.6.20 rendering is not in git (`versions/*/` is ignored). It was rebuilt in a scratch tree by
   running the renderer as of niagara-help a8135f106 (the last commit before the 2026-10-16 readable-text change)
   over the same `versions/4.15.6.20/guides/` sources. That rebuild put the Absolute Logoff paragraph back at the
   cited :30-41, for example. Each citation was then placed in a render era by blame date and by how well its
   claim matched the old and the current text. The 2026-10-15 retros, the older register lines and one
   2026-10-23 retro had read the old rendering; the type cards and recent retros had read the current one. Each
   old-era line was found again in the current file, and every citation got an anchor taken from the text it
   cited.

## Evidence
- RED then GREEN `[ev: code go/internal/citeresolve/anchor_test.go]` `[ev: code go/internal/citeparse/source_test.go]`
  (fixture `Platform/Rerendered.txt`): anchor moved, anchor gone, anchor on several lines, inline backticked anchor,
  and history scope all failed before the change. Anchor on the line, within the window, inside a range, and no
  anchor passed before and after.
- Sweep: 161 guides-clean line citations (159 rows the tool saw, plus a pointer with a space after `niagara-help`
  and a bare `:84-85` in the same retro). 45 were repointed: 44 old-era and 1 off-by-one live pointer. 158 got an
  anchor. 2 were left as they were, because each sits on a line that names a client. The mapping helper was a
  throwaway script and is not committed.
- Kit-wide run over 585 tracked kit cards, before → after: LIVE strict 1 → 1, HISTORY strict 167 → 167,
  HISTORICAL_DRIFT 248 → 226 (22 rows now RESOLVED), no DRIFT row, and no row went from passing to failing.
  Mutation: moving one swept pointer by 10 lines turns it into DRIFT, naming the line that holds the anchor.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | A guides-clean line pointer carries a quoted anchor; kit-cite-resolve reports DRIFT when the anchor is not on or within 2 lines of a cited line (HISTORICAL_DRIFT on a history card), and shows but does not check anchors outside guides-clean | `BUILD-LOOP.md` § Citation pre-filter; `go/internal/citeresolve/anchor.go` | `[ev: code go/internal/citeresolve/anchor.go]` |

## Lessons
- An ignored, regenerated data tree has no history, so a re-render deletes the evidence of what an old pointer
  read. Rebuilding the old rendering with the old renderer from git recovered it here. A pinned copy of the
  rendered text, or an anchor written at citation time, avoids needing that.
- A quoted phrase after a pointer is not always a quote of the pointed line: kit and task cards use it for
  paraphrase. A check that reads a new meaning into existing syntax has to be limited to the tree where that
  meaning holds, and a kit-wide before/after row diff is what showed the 6 false DRIFT rows.

## Owed
- `toolbelt/kit-cite-resolve.sh` header lists the statuses and does not name DRIFT yet. It was outside this
  change's edit surface.
- The research corpus has the same drift: about 360 old-era guides-clean citations no longer hold their line. The
  list, with old → new lines, was handed over and not applied. Its quoted phrases often elide with `…` and
  escape `|`, so an anchor check over the corpus would need `…` as a gap before it can be adopted there.
- Two kit citations were not swept because their lines name a client: a register line citing
  `Obix/LicenseRequirement-OBIXDriver-9F25675C.txt:19` (now line 9) and a `types/dashboard.md` line citing the
  dashboard pane guide page at `:7,22` (still correct, no anchor yet).
