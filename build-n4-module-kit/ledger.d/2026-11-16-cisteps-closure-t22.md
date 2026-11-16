<!-- ledger: kit -->
<!-- date: 2026-11-16 -->
<!-- last-change-note: the pre-push shellcheck closure reads compound source lines and falls back on any unresolved one in the set -->
log: 2026-11-16 · pre-push shellcheck closure hardening → `ci-steps changed` reads a `.`/`source` command after a separator or keyword (`[ -f x ] && . x`, `then . x`), checks CI's whole shellcheck set when any source line of the set names no file of it, keeps a `# shellcheck source=` directive over comment lines and leaves out a changed path the set does not hold; CL23 clears the host's acorn variables and HB9 pins the `--disk-pct` default. Retro cisteps-closure-t22 (Δ1-Δ3 applied, Δ4 declined, pending review). [ev: retro cisteps-closure-t22 Δ1] [ev: retro cisteps-closure-t22 Δ3]
