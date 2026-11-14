### Changed — pre-push comments and test titles describe the Go gate

- `scripts/changelog-gate.sh`'s header now says that ci.yml runs it and that the Go pre-push gate runs the same
  check in process. The H29 comment no longer describes the hook as bash, and the LC8 title names the early-pass
  exit instead of the bash function `pass_early`. Retro `2026-11-14-go-pre-push-port` Δ4.
