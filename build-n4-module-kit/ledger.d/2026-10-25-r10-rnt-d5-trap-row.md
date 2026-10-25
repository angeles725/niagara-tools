<!-- ledger: kit -->
<!-- date: 2026-10-25 -->
<!-- last-change-note: quality round 10: run-niagara-test names the plugin module-include trap in the gradle-root FAIL row -->
log: 2026-10-25 · quality round 10 → `run-niagara-test run` (`go/internal/niagaratest/gradleroot.go`): when the build changed a `*/module-include.xml` in the gradle root, the `gradle root changed by the build` FAIL row names the plugin-version trap (7.3.40 and 7.6.22, observed), points at `--restore-gradle-root` and `types/moduleTest.md`, and says whether the restore checked it out. Other files keep the generic row; the gate and exit codes are unchanged. Retro retros/2026-10-25-r10-rnt-d5-trap-row.md (1Δ, folded). [ev: retro r10-rnt-d5-trap-row Δ1]
