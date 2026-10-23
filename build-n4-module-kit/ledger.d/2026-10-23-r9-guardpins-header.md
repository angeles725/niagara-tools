<!-- ledger: kit -->
<!-- date: 2026-10-23 -->
<!-- last-change-note: quality round 9: lint-guard-pins reads the whole leading comment block -->
log: 2026-10-23 · quality round 9 → `go/internal/guardpins` `Declarations` reads a lint's `# Mutation:` lines from its leading comment block instead of a fixed lines 1-60 window (`HeaderLines` removed), so the 8 declarations on `lint-config-sanity.sh:61-68` are now resolved (8 new MATCH rows, all fixtures found; no other output change); header and SKILL wording updated; closes the go-lint-ord-ports open issue. Retro retros/2026-10-23-r9-guardpins-header.md (1Δ, pending). [ev: retro r9-guardpins-header Δ1]
