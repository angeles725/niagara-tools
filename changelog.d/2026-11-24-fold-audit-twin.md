### Fixed — the sweep-fold-audit CI step and its local twin give one verdict (follow-up to #995)

- `sweep-fold-audit.sh --no-research-probe` skips only the `DEGRADED external-roots-unconfigured` probe of
  `niagara-research/retros`; ci.yml's step passes it, so `scripts/ci-local.sh --changed` and the pre-push hook no
  longer fail on a machine that has that directory while CI (which never has it) passes the same diff.
- A manual run without the option keeps the operator-facing DEGRADED line and its `--strict` exit 1.
