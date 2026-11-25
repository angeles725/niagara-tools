### Added — lint-bql-unbounded

- New `n4lint lint-bql-unbounded [--strict] <src-root>...` rule and `toolbelt/lint-bql-unbounded.sh` wrapper: WARN
  (FAIL under `--strict`) a `bql:` query literal in module Java or JavaScript whose select reads every row: no
  `top <n>` (`no-top`), a `history:<path>` query with no top and no timestamp window (`history-no-window`: no
  `?period=`/`start=`/`end=` and no `timestamp` predicate), an `alarm:` query with no top and no index path
  (`alarm-unindexed`: not `from openAlarms`/`ackPendingAlarms`, `uuid =`, `source =` or a timestamp range), and an
  `order by` with no top (`order-by-no-top`). A string built with `+`, `+=` or a template substitution is counted in
  the one `SUMMARY` row (`non-literal-skipped`) and never judged. Waive one query with
  `// lint-bql-unbounded: <reason>` on its line or the comment-only line above; a bare marker is a FAIL
  (`waiver-no-reason`). Evidence per check in the wrapper header (`SelectQuery.java`, `BTopTable.java`,
  `BHistoryDbTable.java:232-262`, `AlarmDbConnection.java:52-110`; `types/queries.md` §3, §6). The rule is a Go
  package, `go/internal/lint/rules/bqlunbounded`, with golden cases; the parent registers it in `rules.go`.
  [ev: corpus API survey 2026-11-24 media-cache-db §5.3-5.4, finding U8]
