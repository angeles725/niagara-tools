### Fixed — install-skill: review findings of #694 (retro row T10 of the 2026-11-15 coordination stretch)

- A diverged prompt of a skill whose kit has no `prompt.md` names its template as "the built-in default (the kit
  has no <path>)" instead of a `prompt.md` path that does not exist.
- New tests pin the dry-run digest labels (rendered for a prompt, tracked for a launcher), the refusal of a whole
  `--harness all` run before any harness is written when one harness is diverged or holds a symbolic link, and
  that a refused `--home ""` writes nothing anywhere.
