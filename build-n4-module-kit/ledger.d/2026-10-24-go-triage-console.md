<!-- ledger: kit -->
<!-- date: 2026-10-24 -->
<!-- last-change-note: quality round 10: triage-console runs the Go triage -->
log: 2026-10-24 · quality round 10 → `toolbelt/triage-console.sh` is the thin wrapper over `go/cmd/triage-console` (`go/internal/triageconsole`: the awk channels C1-C3 and `--site` classes, own rows in gawk for-in order via `rules/awkorder`, awk `-v` escapes); 70 goldens recorded from the bash byte-identical through `golden.Runner`, the default package/tag/module-prefix lists stay in the wrapper and reach Go as `TRIAGE_CONSOLE_DEFAULT_*`; an unreadable console or an option without its value is exit 3, a row with a non-UTF-8 byte prints under a UTF-8 locale, an assignment-shaped console name is read (truth goldens). [ev: retro go-triage-console]
