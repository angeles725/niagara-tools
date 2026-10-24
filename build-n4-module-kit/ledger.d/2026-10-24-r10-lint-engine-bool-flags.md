<!-- ledger: kit -->
<!-- date: 2026-10-24 -->
<!-- last-change-note: quality round 10: lint engine boolean flags; G65 wrapper drops its flag translation -->
log: 2026-10-24 · quality round 10 → the Go lint engine's Mode Valued takes boolean flags (`Spec.Flags`, read as `Args.Flags` in `Configure`; anywhere in argv, `<flag>=<v>` a usage error), so `lint-private-api-import.sh` (G65) declares `--no-default-allow` natively and is the plain `exec n4lint` its siblings are; the engine option `--default-allow on|off` is removed. [ev: retro r10-lint-engine-bool-flags]
