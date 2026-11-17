<!-- ledger: kit -->
<!-- date: 2026-11-17 -->
<!-- last-change-note: n4_rollback outside the write budget; in-doubt replication rollback removes found accepted copies -->
log: 2026-11-17 · mcp-n4 n4_rollback is outside --max-writes like the save (recovery always reachable); it offers an in-doubt n4_replicate_points batch, removing only accepted copies the read-back finds and no later batch wrote under, with per-copy verdicts; the replication dry run notes registers held by points already on a target. [ev: retro mcpn4-replicate-points Δ5]
