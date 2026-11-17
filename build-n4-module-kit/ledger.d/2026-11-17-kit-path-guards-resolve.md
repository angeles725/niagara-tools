<!-- ledger: kit -->
<!-- date: 2026-11-17 -->
<!-- last-change-note: path guards follow symlinks after ".." and create nothing before every check -->
log: 2026-11-17 · `orient-guard.sh` and `mirror-niagara-home.sh` resume following symlinks after a `..` re-enters existing directories (realpath -m); the mirror tool guards, wipes and rebuilds the judged path, creates nothing before every check passes, and refuses an option-like path; orient-guard keeps a final newline in a name. Correction of the independent verifier of #785. [ev: retro go-toolbelt-small-port Δ1, Δ2]
