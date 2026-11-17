### Changed — scaffold-module: `--vendor` is required and version options are shape-checked (retro build-kit-bench-session Δ4, Δ8, Δ11, Δ17, Δ18)

- **Breaking:** `scaffold-module.sh` no longer guesses the vendor. A run without `--vendor` exits 2 with
  `scaffold-module: --vendor is required: …` and the usage, before anything is written. `--target-version` must be
  `<major>.<minor>` and `--plugin-version` `<major>.<minor>.<patch>` (digits only); another shape is exit 2 naming
  the option. Every kit caller, bats case and the CI scaffold-diff steps pass `--vendor`.
- Every scaffold type passes `report-module.sh --lint-only` with zero FAIL and zero WARN, and verify-module's
  `facets-req` and `palette-root` checks with no WARN: MIN facet and floor guard on the logic `interval`, unit
  facets on the `setpoint` slots, `b:UnrestrictedFolder` palette roots, `<Type>.displayName` lexicon keys.
- Every fixture, so every scaffold, carries a `.gitignore` (`.gradle/`, `build/`, `_backups/`, `.last-deploy-sha`).
- `skill/SKILL.md` points only at files that exist, holds no per-user path, maps each decision-table row to a
  scaffold `--type`, and describes the scaffold once; `BUILD-LOOP.md` § 1.a matches what the scaffold emits.
