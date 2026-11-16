<!-- ledger: kit -->
<!-- date: 2026-11-14 -->
<!-- last-change-note: the five install and bootstrap scripts are launchers over Go commands -->
log: 2026-11-14 · all-Go pass, wave 3 → `scripts/install-skill.sh`, `check-skill-drift.sh`, `kit-resolve.sh`, `install-hooks.sh` and `install-client-hooks.sh` are launchers over `go/cmd/<name>` (`go/internal/skillinstall`, `kitresolve`, `hookinstall`, shared `installtools`). Differential against the bash originals over 143 rows: 125 identical, 18 declared; nine false successes are refusals now (an unreadable kit, source, target or hooks directory, an empty `$HOME`, a corrupt index under `--offline-ok`). `scripts/lib/sha256.sh` stays for `ci-local.sh` and the two skill launchers.
open_issue: none (retro go-install-tools-port rows 1-4 applied)
