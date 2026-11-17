### Added — preflight remote-backup row

- `preflight.sh` WARNs (`remote-backup`) when the gradle root's git checkout has no remote, or its branch has no
  upstream, so its commits exist on one disk only; pushing or backing up stays the operator's decision. Silent
  outside git, on a detached HEAD and with an upstream.
