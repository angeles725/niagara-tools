<!-- ledger: kit -->
<!-- date: 2026-10-22 -->
<!-- last-change-note: quality round 8 T8k-1: Go javasrc scanner -->
log: 2026-10-22 · quality round 8 T8k-1 → new Go package `go/internal/javasrc` (`Strip`, `Methods`, `Lines`) with the semantics of `toolbelt/lib/method-boundary.sh` `mb_strip` + `mb_parse`; `TestDifferentialAwk` proves 0 diffs vs awk, mawk and gawk --posix over the repo Java corpus (and 302,813 client lines when `JAVASRC_DIFF_ROOTS` is set); two lib quirks pinned, not fixed; no lint uses it yet; retro go-javasrc-scanner
