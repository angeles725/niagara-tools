# mcp-n4-bulk-read-deltas — fold the 8 deltas of the first remote inventory retro into mcp-n4-kit

## Objective
Fold the eight proposed deltas (D1-D8) of `mcp-n4-kit/retros/2026-10-02-customer-remote-01-session.md`
into the kit. The headline gap: the kit's read path is unfit for inventory work over a WAN. A
recursive navigate/read crawl ran more than 16 minutes unfinished, while projected BQL over HTTP
returned the same inventory in about 3 s.

## Problem / Why
- No bulk read tool: every component costs one BOX round trip (D1).
- `load_tree` polls on a fixed 0.5 s delay, and its latency is invisible (D2).
- No guidance that says "inventory = BQL first" (D3, D7, D8).
- Complex address slots read as `None` (D4).
- A bare `HTTP 401` gives no actionable hint (D5).
- The skill launcher fails silently when the kit checkout is missing (D6).

## Scope (authorized by the parent session, 2026-10-02)
- In scope: `mcp-n4-kit/**`, `CHANGELOG.md`, `VERSION`, and this document.
- Out of scope: other kits, the installed skill copy under `~/.claude/skills`, and any write to a
  station. Live validation is read-only only.
- The repository is public: no customer, station, host, IP or user names in any file.

## Constraints
- Python 3.10+, stdlib only, `unittest`. No import side effects.
- `n4_bql_query` stays read-only:
  - `select` only;
  - no `|` in caller-supplied text;
  - path-form `GET /ord/<url-encoded ORD>` (the `/ord?` query form answers 400);
  - no redirects carrying credentials;
  - an auth failure is never retried.
- Test-first: RED, then GREEN, wherever a deterministic test is possible.

## Tasks
- [x] **D1 — `n4_bql_query`.** Projected BQL against a subtree.
  - Acceptance:
    - rows come back as a list of dicts, plus the columns;
    - anything but `select`, and any `|`, is refused before a request is sent;
    - the row cap (default 5000) sets `truncated`;
    - a timeout is passed to the transport;
    - control characters and the BOM are stripped;
    - `$xx` decoding lives in a helper;
    - the read is recorded (session observation; plus an `audit.jsonl` line with outcome `read`
      when the write machinery is active);
    - tests use a fake HTTP opener and an anonymized recorded CSV fixture.
- [x] **D2 — adaptive load polling.**
  - Acceptance:
    - the first poll delay is 0.1 s, then backoff 0.2, 0.4, 0.8, capped at 0.8 s;
    - total sleep is bounded (default 3 s);
    - `n4_navigate` and `n4_read_slots` return `elapsed_ms`;
    - tests inject the sleep and the clock.
- [x] **D3 — METHODOLOGY: inventory = BQL first.** Navigate only for targeted components.
  - Acceptance: a METHODOLOGY section that cites a test.
- [x] **D4 — display fallback in `n4_read_slots`.**
  - Acceptance:
    - a complex slot with no decodable value (FlexAddress, BacnetAddress) returns its display
      string in `value`;
    - every slot that carries one also returns `value_display`;
    - covered by tests.
- [x] **D5 — actionable 401.**
  - Acceptance:
    - the 401 `AuthError` names `HTTPBasicScheme`, the password, and "do not retry (lockout)";
    - covered by tests on both the BOX path and the BQL path.
- [x] **D6 — launcher and README.**
  - Acceptance:
    - when the kit is missing, the launcher says so and offers `git worktree add <path> origin/main`
      or an installed copy;
    - the README documents a per-project `.mcp.json` registration.
- [x] **D7 — progress for long reads.**
  - Acceptance:
    - a METHODOLOGY rule;
    - `n4_bql_query` and `n4_inventory` report counts;
    - `n4_inventory` appends progress lines to the operator's `--progress-file` when it is set.
- [x] **D8 — local devices.**
  - Acceptance:
    - `n4_inventory` flags the `localDevice` slot as `local: true`;
    - local devices are counted separately from field devices;
    - a pure helper, covered by tests;
    - METHODOLOGY inventory guidance.
- [x] **Close.**
  - Retro INDEX row and review-status marker set to `folded 2026-10-02 · kit b13747e` (the
    parent fills the SHA).
  - `VERSION` 0.26.0, a CHANGELOG entry, and kit `__version__` 0.2.0.

## Checks
- `cd mcp-n4-kit && python3 -m unittest discover -s tests -q`
- `bats tests/install-skill.bats` (the launcher is installed byte-identical, SK9)
- Live, read-only, optional: `n4_bql_query` for devices under `/Drivers`, and `n4_read_slots` on a
  Modbus proxy extension (`dataAddress` display, `elapsed_ms`).

## Route
- Delegated writer: 2+ non-trivial files.
- The parent commits the work units, opens the PRs and merges.

## Progress
| Task | Route (trigger) | Commit | Authored lines | RDD tier / outcome | Checks |
|---|---|---|---|---|---|
| D2+D4+D5 | delegated writer | b13747e (with D1/D7/D8; hunks shared box.py/tools_read.py) | 781 (shared with D1/D7/D8) | medium, granted, approved + acknowledged (review-18213260b4c048dd) | unittest 433 OK; live read-only: n4_inventory 507 points about 1.2 s, elapsed_ms, Decimal display |
| D1+D8+D7 | delegated writer | b13747e | (in b13747e) | medium, granted, approved + acknowledged (same lineage) | unittest |
| D3+D6+D7 docs, close | delegated writer | docs commit on this branch | 221 | covered by the same range review | unittest (citation gate) + bats install-skill |

## Next step
The parent commits the suggested work units, runs the native review per unit, and fills the
`b13747e` placeholders.
