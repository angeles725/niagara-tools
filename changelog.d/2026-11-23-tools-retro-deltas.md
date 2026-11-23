### Added — build.sh checks its kit, warns on a wrong schema baseline; the fold audit reports a missing external root

- `build.sh` runs `scripts/kit-resolve.sh` on its own kit first and prints `==> kit: KIT=<dir> SHA=<sha> (origin/main,
  clean)` as its first stdout line. A kit behind origin/main (or diverged from it) is `build.sh: FAIL — the kit is
  behind origin/main ...` and exit 10 before gradle; a dirty or ahead kit, or a missing resolver, is `==> kit: <dir>
  freshness unchecked — <why>` and the run goes on. `BUILD_N4_KIT_CHECK=0` skips the check; `-h` and `--restore-mirror`
  never run it.
- `build.sh --schema-baseline-ref <ref>` compares the module version the ref declares (`defaultModuleVersion`) with the
  vendorVersion of each deployed jar and WARNs when they differ (a baseline of 2.3.0 with 2.3.1 deployed). The exit code
  does not change. The replaced mirror jar was already kept with a final `==> mirror: ...` line naming both versions
  and the restore command.
- `sweep-fold-audit.sh` prints `fold-audit: DEGRADED external-roots-unconfigured: niagara-research/retros has <R>
  retro(s) with <D> pending kit delta(s); not audited — ...` when no external root is configured but such a directory
  holds retros with pending mcp-n4 / build-n4-module deltas. It fails `--strict`. `--external-root` and
  `fold-roots.conf` were already shipped.
