### Added — rc-scan `status-default-ok` (advisory)

- `rc-scan` WARNs a status read that falls back to `"ok"` when the status is missing (`p.st || "ok"`,
  `?? 'ok'`, `||= "ok"`, `??= "ok"` after a status name: `st`, `stat`, `sts`, `status` or a `*Status` name).
  An unlinked or uncommissioned point then renders as healthy. Advisory: WARN under every mode, `--strict`
  included; `// rc-scan: allow status-default-ok` on the line silences it.
