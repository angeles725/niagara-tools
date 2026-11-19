<!-- ledger: kit -->
<!-- date: 2026-11-19 -->
<!-- last-change-note: mcp-n4 second session loss named; BQL warns on a NULL column that is no slot -->
log: 2026-11-19 · mcp-n4: a second "Could not find Server Session" in a row after the one reconnect is a named error, never the raw BOX text or a loop; a write execution that loses its session says "session lost ... redo the dry run". `n4_bql_query` warns on a column NULL on every row that is no slot of the queried type (facet?). Retro mcpn4-session-bql (Δ1, Δ2 applied; Δ3 open, pending review). [ev: retro mcpn4-session-bql Δ1]
