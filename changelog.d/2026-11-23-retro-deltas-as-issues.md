### Added — retro deltas are GitHub issues

- `retro-issues.sh` (`go/cmd/retro-issues`): files the mcp-n4 / build-n4-module delta rows of retros as GitHub issues,
  dry run by default, `--apply` to file them, `--write-back` to append `#N` to a kit retro's row, `--check` to list the
  retro-delta issues still open. Titles are `<area>: <summary>` (at most 100 characters), bodies carry the delta, target,
  evidence, `retro <stem> Δ<id>` and a hidden `<!-- retro-delta: ... -->` marker; labels `retro-delta` and
  `area:mcp-n4` / `area:build-n4-module`.
- Search before filing: an exact marker (in a body or a comment) is `exists #N`; a match by meaning (key terms scored
  against open and closed issues) is a probable duplicate and becomes a comment on that issue, a closed match is
  reported as already fixed and never reopened, and deltas of several retros that describe one change share one issue;
  `--new <id>` overrides the meaning match.
- Every title, body and comment is checked against the private client denylist before anything is searched or filed;
  a hit refuses that issue naming the field and the rule number only. A retro outside the kit needs
  `--alias <stem>=<ALIAS>` so its name never reaches GitHub.
- `sweep-fold-audit.sh`: a delta row ending in `#N` counts as cited when issue #N is closed (gh unreachable prints
  `UNVERIFIED` and the delta stays uncited); `--external-root <dir>` and `fold-roots.conf` also audit retros of other
  projects that name mcp-n4 / build-n4-module, reported by alias (`--alias <stem>=<ALIAS>`).
- `new-retro` stub: a comment under the delta heading states the rule. CONTRIBUTING.md § 6.3 documents it.
