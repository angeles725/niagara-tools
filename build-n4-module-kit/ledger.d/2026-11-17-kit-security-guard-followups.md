<!-- ledger: kit -->
<!-- date: 2026-11-17 -->
<!-- last-change-note: security guard follow-ups: commit-msg reads comment lines and fake cut lines; servlet write lint receiver match -->
log: 2026-11-17 · `.githooks/pre-commit` message mode reads comment lines and drops below a `commit -v` cut only the lines of the diff git appends; `lint-servlet-write-guards` catches self-permission through aliases and `super`, matches a check to the written receiver, needs its result used, treats `invoke` as a write, verifies a declared `requireTargetWrite`, and needs a `verifyCsrfToken(` call. Retro kit-p1-security-guards (Δ8-Δ10 applied, Δ11 open). [ev: retro kit-p1-security-guards Δ8]
