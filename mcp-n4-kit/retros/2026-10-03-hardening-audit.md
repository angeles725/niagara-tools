<!-- review-status: pending -->
<!-- Marker lifecycle: the maintainer flips 'pending' above to 'folded <date> · kit <sha>' once the proposed deltas are reviewed and applied (or dismissed) in the kit. This retro only PROPOSES; kit changes are human-reviewed and human-committed. -->
# Retro — kit-dev · 2026-10-03 · mcp-n4 development retrospective (hardening audit F1-F17)

> Written by hand from the seven merged pull requests, following `mcp_n4/templates/retro.template.md`. This was
> kit development, not a station session: no station was contacted, so `audit.jsonl` / `journal.jsonl` hold no
> evidence and `n4_session_retro_draft` does not apply. The reviewer keeps, edits or drops each delta. Stage
> GitHub issues only by hand; nothing here is applied automatically.

## Session summary

The 2026-10-03 audit of `mcp-n4-kit` listed 17 findings (F1-F17). They landed as seven work units
(`odd/tasks/mcp-n4-hardening-2026-10-03.md`), one PR each, each from the latest `origin/main`:

1. **H1, PR #229, v0.5.1.** The write plan, and so the confirmation token, names
   `station: {name, base_url, session_id}`; the write scope accepts only plain `station:|slot:/...` ORDs and
   checks `--write-scope` prefixes at startup; `--token-ttl` and `--max-writes` must be >= 1. RED showed a
   token from station A executing on station B.
2. **H2, PR #231, v0.5.2.** Any `AuthError` (login, oBIX about, `/ord` GET, BOX) latches every station call
   for `--auth-cooldown` (>= 30 s); only other credentials may reconnect meanwhile.
3. **H3, PR #232, v0.5.3.** Nested `n4_set_slot` loads one level deeper (the previous value was the type
   default); rollback compares before restoring and refuses to overwrite a later change; clearer "slot not
   found"; H2a closed a fail-open the H2 review found (the fallback latch skipped a second credential).
4. **H4, PR #234, v0.5.4.** Characterization tests for seven untested write branches. One of them found a
   real bug: a component-intent journal failure was reported only as `ToolError`.
5. **H5, PR #236, v0.6.0.** Repeated CSV headers are kept (`Type#2`); columns are resolved by select-list
   position, so localized headers work; `n4_bql_query` `output_file`.
6. **H6, PR #238, v0.7.0.** `n4_list_batches`; `snapshot_truncated`; `--load-wait` / `--http-timeout`; H5a
   fixed the two regressions the H5 review found.
7. **H7, this PR, v0.7.1.** The write budget is per server process; docs drift; FlexAddress/BacnetAddress
   as values; H6a corrected an inexact truncation count ("at least").

Every unit ran RED before GREEN where a behavior changed, pinned each new guard with an observed mutation,
and passed a native RDD review (medium: one lens; high: four lenses). Six review advisories became sub-tasks
under the anti-cascade policy (one fail-open, two regressions, one inexact claim); the rest are parked.

Lessons:

- A mutation restored with a plain copy within the same second, with the same file size, left a stale
  `__pycache__` `.pyc`: the "restored" code still ran the mutant, and the next shuffled run failed for no
  visible reason.
- Characterization tests are not only debt payment: the H4 test for an "obvious" branch exposed a swallowed
  error reason.
- Under one-PR-per-unit delivery, `review assess` reported `under_budget` (not due) for medium slices that
  were then merged, so the pending slice never reached the budget. The reviews ran anyway (consent granted);
  without that, medium safety changes would have shipped unreviewed.

## Proposed kit deltas

| # | Delta | Evidence | Proposed change | Cost |
|---|---|---|---|---|
| D1 | Mutation checks can run stale bytecode | H5: after a mode mutation (0o600 -> 0o644, same size) was restored by `shutil.copy` within the same second, the next run still wrote 0644 files; clearing `__pycache__` fixed it | README "Test isolation rules": run mutation checks with `PYTHONDONTWRITEBYTECODE=1` or clear `mcp-n4-kit/**/__pycache__` after each restore; restore with `cp -p` only if the mtime differs | S |
| D2 | Untested branches are found by hand | F13 listed seven write branches with no test; the H4 test for one of them found a swallowed error reason | Add a stdlib branch audit to the README (`python3 -m trace --count --missing` over the suite, or a periodic review of `tools_write.py` raise sites without a test), run before each minor release | M |
| D3 | Per-unit PRs never reach the `under_budget` review threshold | H1, H5, H6: `gentle-ai review assess --committed-only` returned `review_due: false, under_budget` for medium safety changes that were merged as their own PR | In the kit's delivery notes (README "Versioning" or CONTRIBUTING): when each work unit merges as its own PR, review a medium candidate at its PR instead of waiting for the slice budget | S |

## Already covered (dedupe)

- Live-station certification (F2 chained ORD on a real station, F3 nested Status depth, F10 type-default
  omission, F12 localized headers, F9 override inverse) cannot be done on the fake station: it is tracked in
  #200, as the kit already does for B1200-G1/G3.
- Test isolation through `mock.patch` + `addCleanup` and shuffled-order runs: README "Test isolation rules";
  H4 moved `test_auth_latch` to that style.
- CHANGELOG hand-off in the PR body while another writer owns the root release: README "Versioning".

## Honest verdict

A productive, clean run: seven PRs, each small, RED first, mutation-pinned, reviewed and green in CI. The two
high-severity safety gaps (token station binding, chained-ORD scope bypass) and the lock-out risk are closed in
code. The remaining risk is live-only behavior that no offline test can prove; it is listed on #200. Recommended
fold order: D1 (cheap, prevents false mutation results), D3, then D2.
