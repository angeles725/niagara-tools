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
- [ ] **T3 — write tools with safety layers.**
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
- [ ] **T4 — destructive tools + rollback + save.**
  - Content:
    - `remove_component`, `rollback(batch_id)` from the journal;
    - `save_station` (destructive class; optional persistence check by `config.bog` mtime/sha when a station home is configured).
  - Route: delegated writer.
  - Checks: unittest.
- [ ] **T5 — skill + installer + METHODOLOGY.**
  - Content:
    - `mcp-n4-kit/METHODOLOGY.md`: safety layers L1-L8, live rules from B1199, route ladder;
    - `mcp-n4-kit/skill/SKILL.md` thin launcher (`$MCP_N4_KIT` → default path → fd) with the mandatory checklist;
    - `scripts/install-skill.sh --skill mcp-n4` support (bats-tested, no change to default behavior);
    - MCP client registration docs.
  - Route: delegated writer.
  - Checks: bats + shellcheck.
- [ ] **T6 — live smoke runner + release.**
  - Content:
    - `mcp-n4-kit/tools/live-smoke.py`: scratch folder → kitControl thermostat → logic read-back → rollback, off unless `--apply` + `--expected-station`;
    - VERSION/CHANGELOG release.
  - Route: inline or delegated.
  - Checks: unittest; the live run only with an operator-supplied credential (pending if absent).

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

- [ ] **T8 — hygiene sweep of accumulated advisory findings.** Content: T1c advisory (KeyboardInterrupt/SystemExit cleanup and docstring, ord-grammar helper for child ORDs, invalidate_handles prefix boundary, stale-handle double wait, test names), plus T2+ advisory that is not fixed in its own slice. Also add MCP protocol versions 2025-11-25/2026-07-28 once their semantics are implemented (T2 declares up to 2025-06-18). Route: delegated writer. Checks: unittest.

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
| T2 | delegated writer | 386d5be | 965 (size:exception — one honest slicing pass: protocol vs tools splits each stay >400 because tests follow their code) | pending | unittest 118 OK (writer + parent); sourceOrd `h:xxxx` + root `stationName` certified against B1199 live transcript |

## Next step
T2 RDD + PR, then T3 (write tools + safety layers). Previously: T1c RDD + PR, then T2 (spec drafted from scratch after a classifier cut the first T2 brief; operator said proceed on my recommendations).
