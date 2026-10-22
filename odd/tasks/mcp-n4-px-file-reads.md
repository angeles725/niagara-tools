# mcp-n4-px-file-reads — read Px views and station files through mcp-n4-kit

## Objective
Let an agent list a station's Px views and read or summarize their `.px` files (and other
station files) through read-only mcp-n4 tools, instead of by hand.

## Problem / Why
- The kit's tools read only components (BOX, BQL). Px screens are files (`file:^px/X.px`), so an
  operator who asked to see a live station's PxEditor graphics could not be served by the kit.
- The route was proven by hand on a live station: BQL over `baja:PxView` through
  `GET /ord/<ORD>` listed the views (`slotPath` came back `NULL`), and `GET /file/px/General.px`
  returned the raw Px XML (an absent `/file/nav/NavFile.nav` answered 404).
- Source facts (N4 web-rt): `FileServlet` is mapped to `/file/*`; `file:^a/b` is `/file/a/b`;
  `FileServlet.canRead` needs read permission and refuses directories and `.bog`/`.bog.gz`;
  `BPxView` is a `BStruct` (property value), `pxFile` a `BOrd` property.

## Scope
- In scope: `mcp-n4-kit/mcp_n4/**`, `mcp-n4-kit/tests/**`, `mcp-n4-kit/README.md`,
  `mcp-n4-kit/METHODOLOGY.md`, `mcp-n4-kit/skill/**`, this document.
- Out of scope: any write path; contacting a live station; root `CHANGELOG.md` / `VERSION`;
  `mcp-n4-kit/tools/live_smoke.py` read-tool list; the installed skill copy.
- No client names, hosts or credentials in artifacts.

## Constraints
- Python 3.10+, stdlib only, `unittest` (CI: `python3 -m unittest discover -s mcp-n4-kit/tests -v`).
- Test-first: RED observed before implementation.
- 401 keeps the no-retry auth latch; credentials only from the environment.

## Tasks
- [x] **T1 — `n4_read_file`** (route: delegated). `GET /file/<path>`, per-segment URL encoding,
  client-side refusal of `..`, `|`, absolute/home/drive paths, backslashes, empty segments,
  directories, `.bog`/`.bog.gz`; `max_bytes` cap (default 2 MB, max 16 MB); text vs binary
  (size + sha256 + content type); optional private `output_file` under `<state-dir>/files/`;
  403/404 typed `status`, 401 through the latch. New `BoxClient.get_file`, `box.HttpStatusError`.
- [x] **T2 — `n4_list_px_views`** (route: delegated). BQL
  `select slotPath, displayName, pxFile, media from baja:PxView` via the `bql.py` machinery;
  `owner_path` derived only when `slotPath` is not `NULL`, else null with `owner_note`.
- [x] **T3 — `n4_px_summary`** (route: delegated). Pure `files.parse_px` (stdlib XML, refuses
  DOCTYPE/ENTITY): media, imports, canvases/viewSize, layers, widget and binding counts, bound
  ORDs grouped by component, images, included Px, charts, other ORDs, labels; bounded lists.
- [x] **T4 — docs** (route: delegated). README tool list/usage, METHODOLOGY section 9, skill
  checklist line.
- [x] **T5 — tests** (route: delegated). `tests/test_files.py` (29 tests), fake station serves
  `/file/` and `/ord/`, synthetic `tests/fixtures/synthetic_view.px` written for the tests.
- [x] **T6 — this document, version bump** (route: delegated). Kit `__version__` 0.7.3 -> 0.8.0.
  No retro: the repository's retro gate (`.githooks/pre-push`) covers `build-n4-module-kit/`
  and `scripts/` only; mcp-n4-kit retros are session retros.

Route evidence: one bounded writer delegated by the parent session for 2+ non-trivial files.

## Acceptance
- All mcp-n4-kit unit tests green; new tests RED before the implementation.
- Refused paths send no request; a 401 sends exactly one request and pauses later calls.

## Progress / evidence
- Baseline (origin/main 55f7e1c): `python3 -m unittest discover -s mcp-n4-kit/tests` -> 538 tests OK.
- RED: `tests/test_files.py` failed to import `mcp_n4.files` (module absent) before T1-T3.
- GREEN: 567 tests OK (538 + 29; `test_server` tool count 10 -> 13).
- Mutation checks (`tools/mutation_check.py`): disabling the `.bog` refusal -> RED
  (`test_bog_files_are_refused_like_the_file_servlet` and one more); making a `/file/` 401 a plain
  `BoxError` -> RED (`test_a_401_on_the_file_path_latches_and_is_not_retried`); restored -> GREEN.
- Commits: `4763356` feat(mcp-n4): read station files and summarize Px views;
  `228969e` docs(mcp-n4): document Px view and station file reads.

## Open / next step
- `parent.slotPath`-style owner projection for `baja:PxView` is not live-verified; until then
  `owner_path` stays null for struct views (documented, not guessed).
- 403 on `/file/` is treated as a permission denial (no latch); confirm on a live station what
  `FileServlet` answers for a denied read (403 vs 404).
- After merge: root `CHANGELOG.md`/`VERSION` per repository policy; reinstall the skill copy
  (`scripts/install-skill.sh --skill mcp-n4 --force`); consider adding the three tools to
  `tools/live_smoke.py` `READ_TOOLS`.
- Push / PR: human decision.
