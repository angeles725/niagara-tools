<!-- review-status: pending -->
<!-- Marker lifecycle: the maintainer flips 'pending' above to 'folded <date> · kit <sha>' once the proposed deltas are reviewed and applied (or dismissed) in the kit. This retro only PROPOSES; kit changes are human-reviewed and human-committed. -->
# Retro — kit-dev · 2026-10-02 · mcp-n4 development retrospective (tests and tier gate)

> Written by hand from the three merged pull requests, following `mcp_n4/templates/retro.template.md`. This was
> kit development, not a station session: no station was contacted, so `audit.jsonl` / `journal.jsonl` hold no
> evidence and `n4_session_retro_draft` does not apply. The reviewer keeps, edits or drops each delta. Stage
> GitHub issues only by hand; nothing here is applied automatically.

## Session summary

Three mcp-n4 work units landed on 2026-10-02:

1. **MWU1, PR #201 (`ec4bd50`), closes #190.** Test-only change in `tests/test_tools_destructive.py`.
   R3-001: a frozen-check read error that carries the session secret is scrubbed to `***` in the rollback
   output, and the secret appears nowhere in the result. R3-002: an unread frozen child beside a real config
   gap yields `unverified`; a skipped relink beside a read error yields `mismatch`. Both are checked in the
   result and in the journal. A mutation check confirmed the tests bite: dropping `_scrub_gap_errors`, or
   swapping either precedence pair, turns at least one test red. The live-station items of #190
   (B1200-G1 restore half, B1200-G3 `reg.loadContract`) moved to #200 because no offline test can cover them.
   No version bump. 459 tests OK. RDD: reliability lens approved and acknowledged, two non-blocking advisories.
2. **MWU2, PR #203 (`201c1f1`).** Fixes both MWU1 advisories forward. A `DestructiveCase.patch(obj, attr, new)`
   helper wraps `mock.patch.object` and registers `addCleanup(stop)`. `fail_frozen_reads` became a
   `contextlib.contextmanager` that patches `client.load_tree` and `FROZEN_CHECK_DEPTH` together and restores
   both; before this, `load_tree` stayed patched after the block. The same leak pattern was fixed at about
   16 other sites in the file (`fake._sync`, `fake._check_link`, `fake.hook`, `box.load_tree`, a `tamper`
   try/finally). The verdict-precedence test now asserts exactly 2 gaps and the exact key set of each.
   Verification: 459 OK, and 459 OK under shuffled order with seeds 1, 7 and 42.
3. **MWU3, PR #205 (`ac41750`), mcp-n4-kit v0.4.1 → v0.5.0.** The version-tier gate in METHODOLOGY section 5
   is now enforced instead of manual. `n4_connect` reads oBIX `productVersion` from `/obix/about/` (one GET,
   never retried) and reports `version`, `version_source`, `tier`, `tier_writes` and, on failure,
   `version_error`; `n4_describe_session` repeats them. Classification is pure (`mcp_n4/tiers.py`):
   A = 4.13/4.14, B = 4.15/4.3, C = anything else or undetected. The gate sits in `tools_write._process` after
   the identity check and before the budget, scope and token checks. Tier B executions need
   `--allow-tier-b NAME`, tier C needs `--allow-tier-c NAME`; opt-in names must be configured stations or the
   server refuses to start. Dry runs stay allowed and carry a `tier_gate` annotation outside the plan hash;
   reads are never gated. Refusals start with `safety.REASON_TIER` and the session retro classifies them.
   TDD: RED was an `ImportError` on `tiers` plus a connect-contract failure, then 3 `KeyError: 'version_error'`;
   GREEN was 484 tests OK (baseline 459). RDD on the feature commit approved and acknowledged; its two
   advisories were fixed forward in a second commit: a 401 on `/obix/about/` was silently swallowed although
   it may count toward the account lock-out (now reported as `version_error`, and METHODOLOGY tells the
   operator to fix the oBIX permission before reconnecting), and the failure paths were untested.

Lessons:

- A review advisory is cheapest to close as the next small PR (MWU2 after MWU1, the second MWU3 commit).
  Neither needed a new review cycle.
- `mock.patch` without cleanup leaks across tests and stays hidden while the suite runs in file order.
  A shuffled-order run is the cheap check that exposes it.
- Detection side effects matter on a live station even when the detection is "only a read": a 401 on the
  version probe is a failed login to the station's lock-out counter. Fail-closed classification (undetected
  means tier C) keeps the server safe, but the operator must see the error to avoid a second attempt.
- PR #205 carried its CHANGELOG entry in the PR body "to fold into root CHANGELOG.md by the release owner"
  because another writer owned the concurrent kit release. It was folded, but nothing in `mcp-n4-kit/`
  says where mcp-n4 changes are recorded.

## Proposed kit deltas

| # | Delta | Evidence | Proposed change | Cost |
|---|---|---|---|---|
| D1 | Where mcp-n4 changes are recorded is undocumented | There is no `mcp-n4-kit/CHANGELOG.md`; mcp-n4 entries live in the root `CHANGELOG.md` under `### <kind> — \`mcp-n4-kit\` vX.Y.Z` headings, and the version lives in `mcp_n4/__init__.py`. Neither `mcp-n4-kit/README.md` nor `CONTRIBUTING.md` says so. PR #205 shipped its entry in the PR body for the release owner to fold | Add a short "Versioning" note to `mcp-n4-kit/README.md`: version in `mcp_n4/__init__.py`; entries go in the root `CHANGELOG.md` with that heading form; when another release is in flight, put the entry in the PR body for the release owner to fold. Do not add a kit-local CHANGELOG (it would split history) | S |
| D2 | No upgrade note for deployments that wrote to tier B or C stations before v0.5.0 | Before v0.5.0 the tier gate was manual, so a server pointed at a 4.3 (EC-Net) or 4.15 station could execute writes. From v0.5.0 the same command line refuses every execution on that station until the operator adds `--allow-tier-b NAME` (or `--allow-tier-c NAME`). README documents the flags but not the migration | Add an "Upgrading to v0.5.0" note to the README: list the tier-B and tier-C stations that already write; add the opt-in only for a build where a PoC already matched (for tier C, where the `reg.loadContract` + `loadRoot` + scratch-write probe already ran); otherwise run the PoC or probe first. Never add the opt-in just to clear a refusal | S |
| D3 | Test-isolation conventions live only in one test file | MWU2 fixed about 16 leaking patch sites in `test_tools_destructive.py` by hand; the `patch` helper and the shuffled-order check exist only there and in the PR body. `test_tools_write.py` keeps fixture overrides that are safe only because `setUp` rebuilds them | Under "Run the tests" in the README: patch module or object attributes only through `mock.patch` with `addCleanup` (or a context manager), never by bare assignment; before merging a test-heavy change, run the suite in a shuffled order. Optionally move the `patch` helper to a shared test base class | S |

## Already covered (dedupe)

- Test-runner command: `python3 -m unittest discover -s mcp-n4-kit/tests -v` is in `mcp-n4-kit/README.md`
  ("Run the tests") and in CI (`.github/workflows/ci.yml`, step `mcp-n4-kit unittest`).
- A 401/403 on `/obix/about/` may count toward the lock-out: METHODOLOGY section 5 says so and tells the
  operator to fix the user's oBIX permission before reconnecting; the reply carries `version_error`.
- The opt-in is not a way past a refusal: the skill's Hard Rules say never to suggest a tier opt-in without
  the PoC or probe that tier requires.
- Live-station verification that offline tests cannot provide is tracked in #200, not dropped.

## Honest verdict

A clean day. Every unit ran RED before GREEN, review advisories were fixed forward in small PRs instead of
reopening review, and the gate fails closed: an undetected version is tier C. The gaps are process documentation
and an upgrade note, not code: D2 matters most, because an operator who already writes to a 4.3 station sees
refusals after the upgrade with no note explaining them. Recommended fold order: D2, then D1, then D3.
