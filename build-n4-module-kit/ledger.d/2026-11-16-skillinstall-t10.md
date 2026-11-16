<!-- ledger: kit -->
<!-- date: 2026-11-16 -->
<!-- last-change-note: install-skill names the built-in default template of a diverged prompt, and its review findings of #694 each have a test -->
log: 2026-11-16 · install-skill T10 → a diverged prompt of a skill whose kit has no `prompt.md` names "the built-in default (the kit has no <path>)", not the missing file; new tests pin the dry-run rendered and tracked digest labels, the refusal of a whole `--harness all` run before any harness is written, and that a refused `--home ""` writes nothing in `$HOME`, the fixture or the working directory, each proven by a mutant in a scratch copy. Retro skillinstall-t10 (Δ1-Δ2 applied, pending review). [ev: retro skillinstall-t10 Δ1] [ev: retro skillinstall-t10 Δ2]
