<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 5 -->
# 2026-10-21 · kit · modular-line-efficient-quality

**Session**: niagara-research quality round 8 (`odd/tasks/quality-round8.md`, task T7), branch
`docs/retro-modular-line-efficient` from origin/main c4b0ead
**Delta count**: 5

## What happened
The user set a standing authoring rule on 2026-10-21: always write code that is modular and line-efficient,
with fewer lines but never at the cost of quality, and modular so every part stays maintainable. Quality
round 8 applied it to the niagara-go tool suite. Its read-only inventory found no file over 488 lines and no
dead code, so splitting large files would not have reduced the line count. The reduction came from duplicated
helpers and from site data hard-coded in logic. The kit has an advisory size smell (`lint-size.sh`) but no
authoring rule for duplication, site data or how lines may be cut. This retro proposes those rules for N4 module
code (rt/ux/wb Java and ux JS). It changes no kit core file.

## Evidence
- Inventory 2026-10-21, quality round 8: no file over 488 lines, no dead code outside another session's worktree;
  "the gain is in duplicates" `[ev: odd/tasks/quality-round8.md § Why]`
- Duplicated helpers found: zip / CP437 member readers in 3-4 copies, a Python-style directory walk in 4 copies,
  base64 decoding in 2 copies, `Readdirnames` list wrappers in 9 copies; each becomes one shared, unit-tested
  helper (tasks T2-T6) `[ev: odd/tasks/quality-round8.md § Tasks]`
- Hard-coded site data: obix-nav carries the PANCCADIA room tables, defrost types and defaults inline in logic,
  plus a stale scratchpad default path for the password file; T1 moves them into one `site` table
  `[ev: odd/tasks/quality-round8.md § Tasks T1]`
- Each task keeps output byte-identical (all goldens via `make check`) and writes the shared helper's unit test
  RED before callers move to it `[ev: odd/tasks/quality-round8.md § Constraints]`
- The kit already treats size as an advisory review question only (`METHODOLOGY.md` § Conformance rules, size smell)
  `[ev: retro decision-logic-decomposition Δ2]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | **One shared helper per concern before a second copy lands.** When a change needs logic that already exists in the module (or a sibling module of the same GROUP), it calls or extracts the existing helper; it never pastes a second copy. Why: in round 8 every line saved came from removing copies, not from splitting files. Reviewer check: search the diff for a second definition of the same shape (same name, or same parsing/decoding body); a second copy is a review finding. | `METHODOLOGY.md` § rt (components); `types/logic.md` § Composition & organization | `[ev: odd/tasks/quality-round8.md § Tasks T2-T6]` |
| 2 | **Site and station data live in one data table, never inline in logic.** Room counts, ORDs, IPs, point names, defrost types and per-site defaults sit in one named table (a constants class, a `BFacets`/config slot set, or a JSON/CSV resource); control and render code reads that table. Why: inline site data is copied into every branch that needs it and makes the module unusable for a second site. Reviewer check: a literal room count, ORD, IP or site name outside the table is a finding. | `METHODOLOGY.md` § Domain correctness; `types/dashboard.md` § Facade contract | `[ev: odd/tasks/quality-round8.md § Tasks T1]` |
| 3 | **Never cut lines by deleting tests, comments or blank lines, or by minifying.** Line-efficient means less duplicated logic, not denser text. Why: those cuts lower the count and the quality together, and the rule is "fewer lines, always with quality". Reviewer check: a diff that removes tests, assertions, comments or formatting without removing logic is rejected. | `METHODOLOGY.md` § Conformance rules (next to the size smell) | `[ev: odd/tasks/quality-round8.md § Constraints]` |
| 4 | **A refactor keeps observable output, and a new helper is unit-tested first.** Before callers move to a shared helper, its unit test is written and seen RED, then GREEN; the refactor is proven by an unchanged golden or structural test of the callers' output (servlet JSON, rendered HTML, slot values, lint rows). Why: a dedupe that changes behavior is a regression, not a cleanup. Reviewer check: the PR shows the helper's RED→GREEN test and an output check that did not change. | `BUILD-LOOP.md` §5 (verify); `types/moduleTest.md` | `[ev: odd/tasks/quality-round8.md § Constraints]` |
| 5 | **No machine- or session-specific default paths in source.** A default that points at a developer home, a scratchpad, a temporary directory or one station's install path is removed; the value comes from a slot, a station-relative ORD, an environment variable, or the site table of Δ2. Why: obix-nav kept a stale scratchpad path as a default after the session that created it had ended. Reviewer check: search the diff for `/home/`, `/tmp/`, `C:\Users\`, scratchpad and station-name literals in default values. | `METHODOLOGY.md` § rt (components); `types/issues-and-gotchas.md` | `[ev: odd/tasks/quality-round8.md § Tasks T1]` |

None of the five is applied; a fold review decides each one and flips this retro. Δ1, Δ2 and Δ5 are candidates for
an advisory lint (duplicate method body, inline site literal, home/tmp default path); Δ3 and Δ4 stay human-review
items.

## Lessons
- Measure before restructuring: when no file is large, the line savings are in duplicated helpers, and splitting
  files only adds lines.
- Site data inline in logic is both a duplication source and a portability defect; one table fixes both.
- "Fewer lines" is only a quality rule when it is paired with unchanged output and tested helpers; without those
  checks it rewards deleting tests and comments.

---
**Status**: PENDING — INDEX row appended: `| 2026-10-21-modular-line-efficient-quality.md | kit | 2026-10-21 | pending | 5 |`
