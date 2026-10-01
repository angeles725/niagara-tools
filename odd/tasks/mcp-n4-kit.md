# mcp-n4-kit — an MCP server + skill that lets an AI agent read AND write any Niagara N4 station safely

## Objective
Ship `mcp-n4-kit/`, a sibling of `build-n4-module-kit/`. It provides:
- a stdlib-Python MCP stdio server over the BOX JSON protocol (route R-A);
- a thin `mcp-n4` Claude Code skill launcher.

With them, an agent can navigate, read, create components, place them on the wire sheet, link, write, invoke actions, save and roll back, with the safety layers designed in niagara-research B1197 and the lessons certified live in B1199.

## Problem / Why
The read+write path has been proven live (B1199: create, link, kitControl logic, save, rollback on station `LLM`), but only as throwaway scripts. Agents need a tested, safe, reusable tool, plus a skill that encodes the mandatory checklist, like `/build-n4-module`.

## Scope (authorized 2026-10-01 by the operator)
Commit, push, PR, PR view, issues and merge are authorized. Work runs as ODD+RDD, with RDD candidate consent granted for every candidate. It runs in an automatic chain, and the improvements already seen are applied.

In scope:
- Kit docs: README, METHODOLOGY (safety layers + live rules) and the skill launcher.
- Installer support for the second skill.
- Python BOX client library with unit tests against a fake station.
- MCP server with read tools, then write tools behind dry-run, confirmation token, journal/audit and ORD allowlist.
- A dangling-output check (B1199-G2).
- A live smoke runner (off by default).
- A release.

Out of scope:
- Java Fox sidecar (P5);
- in-station module (P7);
- security/platform/spy families (P8);
- BQL/history/alarm/schedule families (later, B1199-G3).

## Constraints
- Python 3.10+, stdlib only. The MCP SDK and pytest are not installed, so tests use `unittest`.
- No side effects on import: every CLI has a `__main__` guard, and every mutating step needs an explicit tool call plus a confirmation token (B1199 retro #1).
- Identity check before the first write (retro #2): `expected_station` must match `stationName`, or writes are refused.
- Status slots are written whole (B1199 §1199.4). Default-omitted values are resolved from the type default (§1199.5).
- One op per `syncTo`. Read-back after every write.
- No login retry loop (B1179 §1179.2).
- Credentials come from env vars or a 0600 file, are never logged or echoed, and are HTTPS only.
- Repo rules: Conventional Commits, NO Co-Authored-By / AI attribution (CONTRIBUTING §6), branch → PR → merge.

## TDD
- Mode: ON.
- Source: user global CLAUDE.md "Strict TDD Mode: enabled".
- Runner: `python3 -m unittest discover -s mcp-n4-kit/tests -v`.
- Plus the repo gate: `bats tests/*.bats`, and `shellcheck` on any changed `.sh`.

## Delivery
- Strategy: `auto-chain`, chain strategy `stacked-to-main`.
- Each task is one work-unit PR of about 400 authored lines or fewer, merged before the next starts.
- Running authored-line count: see the Progress table.

## Tasks
- [x] **T1 — kit scaffold + BOX client library.**
  - Content:
    - `mcp-n4-kit/README.md` (METHODOLOGY.md moved to T5);
    - `mcp-n4-kit/mcp_n4/box.py`: frame, session make/makessc/callssc, syncTo, checkLinks, invokeAction, pollchgs, tree walk, status helpers;
    - `mcp-n4-kit/tests/` with a fake BOX station (`http.server` on localhost, self-signed not required: plain HTTP allowed only for tests);
    - a CI step that runs unittest.
  - Route: delegated writer (2+ non-trivial files).
  - Checks: the unittest runner, plus `bats tests/*.bats` still green.
- [x] **T1b — harden the BOX client (RDD T1 follow-ups, lineage review-e2b9c59fdafb98d5).** Content: no redirect following with credentials (R1-001); malformed replies and transport errors → BoxError, reply seq check (R3/R4); open() cleans up its session on partial failure (R3/R4); load_tree filters load ops by requested handle/ord and does not drop unrelated events (R3/R4); named constants for `cs1` and the root handle, documented return shapes (R2-002..004); tests for 403/500/timeout/bad JSON/retry path and server-assigned names (R2-005, R3). Route: delegated writer. Checks: unittest.
- [x] **T1c — BOX client follow-ups (RDD T1b advisory, lineage review-236d2aad767b3300).** Content: no cleanup `del` after AuthError (lock-out), bounded/drainable pending events, handle-cache invalidation, docstring/label fixes. Route: delegated writer. Checks: unittest 62 OK.
- [x] **T2 — MCP stdio server core + read tools.**
  - Content:
    - JSON-RPC 2.0 over stdio: `initialize`, `tools/list`, `tools/call`;
    - read-only default;
    - tools `connect`/`describe_session` (detects stationName, version tier), `navigate`, `read_slots`, `list_links`, `find_dangling_outputs`.
  - Route: delegated writer.
  - Checks: unittest with the fake station; an end-to-end stdio test.
- [x] **T3 — write tools with safety layers** (delivered as T3a primitives #163 + T3 tools #164 + T3b station policy/lifecycle #165).
  - Content:
    - `dry_run` default true;
    - HMAC single-use confirmation token bound to tool + canonical args + plan hash + expiry;
    - ORD allowlist;
    - max ops per batch;
    - journal (inverse ops) + JSON-lines audit (redacted);
    - tools `create_component` (type + optional wsAnnotation), `set_slot` (whole-Status enforcement), `invoke_action`, `create_link` (`checkLinks c:true`);
    - identity check;
    - read-back verdict `{requested, accepted, observed, verdict}`.
  - Route: delegated writer.
  - Checks: red-then-green per safety layer.
- [x] **T4 — destructive tools + rollback + save** (delivered as T4a write-ahead journal hardening #166 + T4 tools).
  - Content:
    - `remove_component`, `rollback(batch_id)` from the journal;
    - `save_station` (destructive class; optional persistence check by `config.bog` mtime/sha when a station home is configured).
  - Route: delegated writer.
  - Checks: unittest.
- [x] **T5 — METHODOLOGY + skill launcher + installer** (PR #168).
  - `mcp-n4-kit/METHODOLOGY.md`, concise. Each rule names the code or test that enforces it, or is marked "manual". Sections:
    1. **Route ladder:**
       - R-A BOX JSON (default, live-certified in niagara-research B1199);
       - R-B Java Fox sidecar;
       - R-F offline BOG-XML (niagara-research `tools/bog-nav.py`/`bog-write.py`);
       - R-E oBIX for values only;
       - R-D in-station module, last resort;
       - always give a ladder, never a bare "cannot".
    2. **Safety layers L1-L8** (B1197 §1197.4), mapped to their code: dedicated station user, server-side RBAC, dry-run, confirmation token, write scope and limits, write-ahead journal and rollback, audit, operator-controlled station policy (`--station`/`--credential-env`/`--insecure-tls`, HTTPS only, no login retry).
    3. **Live rules (B1199):**
       - one op per syncTo;
       - read back every write;
       - write Status slots whole;
       - an absent value means the type default;
       - check dangling outputs;
       - lock-out is 5 failures in 30 s;
       - no import side effects;
       - confirm a save on disk;
       - identity before writes;
       - declare probe-only elements.
    4. **Mandatory session checklist:** connect to a configured station, navigate/read, dry-run, show the plan to the human, execute with the token, read back, check dangling outputs, save with persistence evidence, keep the batch_id, write the session retro (T7).
    5. **Version tiers** (B1197): A 4.13/4.14 full; B 4.15/4.3 start read-only; C others need a scratch write first.
    6. **Evidence index:** B1177, B1179, B1192, B1197, B1199.
  - `mcp-n4-kit/skill/SKILL.md`, a thin launcher modeled on `build-n4-module-kit/skill/SKILL.md`:
    - Frontmatter: `name: mcp-n4`, `description: "Trigger: …"`, `license: Apache-2.0`, metadata `version "0.1"`.
    - Kit resolution, in order:
      1. `$MCP_N4_KIT` containing `METHODOLOGY.md`;
      2. the default `/home/cristian/modulos_niagara_n4/niagara-tools/mcp-n4-kit`;
      3. `fd`, requiring `mcp_n4/server.py`; never `$HOME` or `/`; ask when ambiguous.
    - Body: read METHODOLOGY first, register the server (README), the checklist, and the hard rules:
      - never write without showing the dry-run plan;
      - no credentials in tool args or chat;
      - never retry a failed login;
      - report `mismatch`/`failed`/`unverified` honestly;
      - close with the session retro.
  - `scripts/install-skill.sh --skill <build-n4-module|mcp-n4>`:
    - default `build-n4-module`, with behavior and exit codes unchanged;
    - an unknown name exits 2;
    - TDD in `tests/install-skill.bats`, RED first: install, current, diverged without `--force` (exit 1), `--force`, `--dry-run`, unknown;
    - shellcheck stays clean.
  - README: links to METHODOLOGY and the skill, plus the install command.
  - Route: delegated writer.
  - Checks:
    - `bats tests/install-skill.bats`
    - `bats tests/*.bats`
    - `shellcheck scripts/*.sh build-n4-module-kit/toolbelt/*.sh tests/*.bats tests/helpers/*.bash`
    - the unittest runner
- [ ] **T6 — live smoke runner + release.**
  - `mcp-n4-kit/tools/live_smoke.py` drives the REAL MCP server over stdio (subprocess `python3 -m mcp_n4.server ...`), not the box library directly, so the whole stack is exercised.
    - Mode:
      - default: plan only, printing the scenario;
      - `--apply`: executes, and requires `--station NAME=URL` and `--station-home NAME=PATH`;
      - `--write-scope` is set to the scratch folder ORD plus the root for the folder create.
    - Credentials: only through the env vars the server reads (`--credential-env`); the runner never prints them.
    - Scenario: the kitControl thermostat from niagara-research B1199. Every step uses dry-run, then the token, then execute, and checks the verdict:
      1. connect, with `expected_station` = NAME;
      2. create the folder `McpSmoke`;
      3. create `Temp` and `Setpoint` (NumericWritable), `Compare` (`kitControl:GreaterThan`) and `Cooling` (BooleanWritable), each with a wire-sheet position;
      4. create 3 links: Temp.out→Compare.inA, Setpoint.out→Compare.inB, Compare.out→Cooling.in10;
      5. `find_dangling_outputs` returns `[Cooling]` only, the expected terminal output;
      6. set Temp=30 and Setpoint=25 through the `set` action, then read Compare.out=true and Cooling.out=true;
      7. Temp=20, then read false;
      8. save; `persisted` must be true;
      9. remove the folder;
      10. roll back the remove, which re-creates the nested subtree. This is the T4 claim that has not been proven live; record the observed verdict honestly;
      11. remove it again;
      12. save; `persisted` must be true;
      13. check that the folder is absent.
    - Output: a JSON report with each step, its verdict and its batch_id. Exit 0 only when every required step is `verified`.
    - Probe-only elements are declared in the report (B1199 retro #3).
  - Unit tests run the runner against the fake station. No live contact in tests.
  - Release: `VERSION` and `CHANGELOG.md`, new section `[v0.25.0]` with `### Added` mcp-n4-kit (T1-T6 PRs) and a References subsection. Follow the CONTRIBUTING §5 order: tag `v0.25.0` after the merge, then push the tags.
  - Live run (operator-authorized destination: the localhost station `LLM`, as in AM20): the parent runs it after the merge, with the credential in a 0600 file read into env for that command only and deleted afterward. The result is recorded in this document and in niagara-research.
  - Route: delegated writer for the runner and tests; the parent does the live run and the release.

- [ ] **T7 — usage retros and kit deltas (operator request 2026-10-01).** Every session that uses `mcp_n4` on a station ends with a retro that proposes kit deltas. This mirrors `/research-sdd` §18 and the build-n4-module `retros/` flow: propose, never apply.
  - Content:
    - `mcp-n4-kit/retros/` with `INDEX.md` (`| file | Station | Date | pending|folded | deltas |`);
    - `mcp-n4-kit/templates/retro.template.md` with a `## Proposed kit deltas` table: change · target file · evidence (audit `batch_id` / tool call) · type · priority;
    - a server tool `session_retro_draft`, which reads the session's audit/journal JSON-lines and drafts a retro pre-filled with evidence-backed candidate deltas:
      - refused writes (no token, identity mismatch, allowlist);
      - read-back verdict mismatches;
      - BOX errors by op;
      - partial-status rejections;
      - dangling outputs;
      - unsupported types/ops;
    - a CLI `mcp-n4-kit/tools/new-retro.py` (writes the file, adds the INDEX row, is idempotent);
    - a skill step: "at session close, write the retro (or the honesty line `no new deltas`)";
    - staging deltas as GitHub issues reuses the research-sdd `stage-retro-issues.sh` pattern (dry-run by default).
  - Route: delegated writer.
  - Checks: unittest, covering both a draft from a synthetic audit log and the honesty line when there is no friction.

- [ ] **T8 — hygiene sweep of accumulated advisory findings.** Content: T1c advisory (KeyboardInterrupt/SystemExit cleanup and docstring, ord-grammar helper for child ORDs, invalidate_handles prefix boundary, stale-handle double wait, test names), plus T2+ advisory that is not fixed in its own slice. Also: a unittest that every `test_*` name cited in METHODOLOGY.md exists (T5 R3-003); explain or parameterize the default kit path in skill/SKILL.md (T5 R1-001/R2-002); one source of truth for skill names in install-skill.sh (R2-004); assert SK7 first-run status (R2-005/R3-001); T4a inverse None normalization + tighten existing journal/audit file modes. Also add MCP protocol versions 2025-11-25/2026-07-28 once their semantics are implemented (T2 declares up to 2025-06-18). Route: delegated writer. Checks: unittest.

## Acceptance criteria
- All unittest + bats + shellcheck are green locally and in CI.
- The server lists tools, refuses writes without a token, refuses on a station-name mismatch, and writes + verifies against the fake station.
- The skill installs with `scripts/install-skill.sh --skill mcp-n4`.

## Progress
| Task | Route (trigger) | Commit | Authored lines | RDD tier / outcome | Checks |
|---|---|---|---|---|---|
| T1 | delegated writer (2+ non-trivial files) | b4bf886 | 704 (size:exception — library + its fake station + tests are one cohesive unit; METHODOLOGY moved to T5) | high → granted → approved + acknowledged (lineage review-e2b9c59fdafb98d5; 7 WARNING + 5 SUGGESTION advisory → T1b) | unittest 24 OK (writer + parent re-run); bats 689 OK (writer) |
| T1b | delegated writer | ab15787 (PR #160, merge 3aaa717) | 402 | high → granted → approved + acknowledged (lineage review-236d2aad767b3300; advisory → T1c) | unittest 47 OK; CI pass |
| T1c | delegated writer | cdb647f (PR #161, merge 7ab0526) | 298 | high → granted → approved + acknowledged (lineage review-78277d6addbdda3f; advisory → T8 hygiene) | unittest 62 OK; CI pass |
| T2 | delegated writer | 386d5be (PR #162, merge 570a81f) | 965 (size:exception — one honest slicing pass: protocol vs tools splits each stay >400 because tests follow their code) | pending | unittest 118 OK (writer + parent); sourceOrd `h:xxxx` + root `stationName` certified against B1199 live transcript; RDD high → approved + acknowledged (review-7438fc563c04844a; R1-001 credential-exfil risk → T3b) |
| T3a | delegated writer | eee825b (PR #163, merge d76e07d) | 345 | medium, under_budget → reviewed together with T3 | unittest 145 OK; CI pass |
| T3 | delegated writer | 0acc462 (PR #164, merge 811d2ed) | 1106 (size:exception — shared pipeline) | medium slice_budget_reached (base 570a81f, covers T3a) → approved + acknowledged (review-9e93dcb1fdb7152a; send-before-journal → T4a) | unittest 191 OK; CI pass |
| T3b | delegated writer | 72d2c70 (PR #165, merge fc64eda) | 515 | high → approved + acknowledged (review-a23ad5de00ce84eb; ambiguous link reply → T4a) | unittest 218 OK; CI pass; found real defect: refused link was journaled |
| T4a | delegated writer | be62b8a (PR #166, merge 12ba86d) | 555 | medium → approved + acknowledged (review-eb1f15fef8474243) | unittest OK; CI pass |
| T4 | delegated writer | bac77b5 (PR #167, merge f8be126) | 729 | medium → approved + acknowledged (review-ecd62053ad516359; relink/retry/outgoing → T4b) | unittest 275 OK; CI pass |
| T5 | delegated writer | a11881c+dba5a64 (PR #168, merge 2c1291d) | 293 | high → approved + acknowledged (review-242f411d683addc9; suggestions → T8) | bats 14/14 + 698; unittest 275 |

## Next step
T6c (this commit): live smoke runner hardening from the T6 native review (review-7cbd7c1ccb5695dd); then T6 RDD + PR, then T7/T8. Earlier steps (T1-T5) are merged; see the ledger above.
