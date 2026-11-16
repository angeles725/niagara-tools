<!-- ledger: kit -->
<!-- date: 2026-11-16 -->
<!-- last-change-note: merge-when-green exits 10 when every merge send was lost on the network and the PR stays OPEN -->
log: 2026-11-16 · merge-when-green T33b → when every send of `gh pr merge` fails on a network error and the PR stays OPEN, `scripts/merge-when-green.sh` exits 10 with "no send of gh pr merge … got an answer from GitHub … GitHub refused nothing and nothing merged", not 8 (a refusal GitHub never made); help and CONTRIBUTING.md § 8 name exit 10; every other exit unchanged (stub-gh differential against origin/main). Retro t33b-mergewait-network (Δ1 applied, pending review). [ev: retro t33b-mergewait-network Δ1]
