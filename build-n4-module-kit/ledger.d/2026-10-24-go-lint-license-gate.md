<!-- ledger: kit -->
<!-- date: 2026-10-24 -->
<!-- last-change-note: quality round 10: lint-license-isoperational-gate ported to Go -->
log: 2026-10-24 · quality round 10 port → `toolbelt/lint-license-isoperational-gate.sh` execs `n4lint lint-license-isoperational-gate` (rule package go/internal/lint/rules/licensegate): byte-identical to the awk on 39 recorded goldens, Cliente and sources/decompiled (±--strict); Cliente 7.8 s → 0.11 s, decompiled 30.2 s → 3.4 s. Truth fix: the rule reads code with literal contents removed (a quoted gate, a quoted license declaration and a `;` in a string no longer count); real trees unchanged (Cliente 0, decompiled 31 rows). Retro retros/2026-10-24-go-lint-license-gate.md (2Δ, pending). [ev: retro go-lint-license-gate Δ1]
