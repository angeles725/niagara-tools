<!-- ledger: kit -->
<!-- date: 2026-11-17 -->
<!-- last-change-note: build/deploy safety: interrupt restore, --no-deploy same-dir refusal, env-file gate keys -->
log: 2026-11-17 · build.sh restores the pre-build modules/ jars on an interrupt before the gate passed; ng-deploy refuses `--no-deploy` into `<NIAGARA_HOME>/modules` and `MODE=B` / `VERIFY_MODULE_BIN` in the env file without the flag. Retro p1-build-deploy-safety (Δ1-Δ3 applied, Δ4-Δ7 open, pending review). [ev: retro p1-build-deploy-safety Δ1 Δ2 Δ3]
