<!-- ledger: kit -->
<!-- date: 2026-10-25 -->
<!-- last-change-note: quality round 10: LTF1 reads a class javadoc that names a READONLY link target past @NiagaraType -->
log: 2026-10-25 · quality round 10 → `go/internal/lint/rules/linktargetflags` LTF1: every comment block, when it ends (at `@NiagaraType` too), marks the slots it names in code formatting (`` `status` `` or `{@code status}`) next to a link keyword, one name per keyword; a READONLY marked slot FAILs as `(class comment)`. A keyword with no named slot marks none. Goldens classdoc-*, bats LTF-classdoc / LTF-classdoc-named. Expected new hit: `fixtures/MinimalDash` BMinimalDash `status` (fixed on another branch). Retro retros/2026-10-25-r10-ltf1-marker-reset.md (pending).
