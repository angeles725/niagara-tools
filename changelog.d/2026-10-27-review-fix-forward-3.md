### Fixed — sweep ledger tags, pre-push ref names, bats skip allowlist

- `sweep-build-state` names the real opener of an open ledger tag (`unbalanced [deferred: tag` for a
  `[deferred:` tag, open or swallowing the next one) and no longer FAILs a closed tag whose text quotes `` `[ev:` ``
  as code. It reads citeparse's new typed `Unterminated` flag instead of the free-text note.
- `.githooks/pre-push` prints `pre-push: refused <local ref> -> <remote ref> (exit N)` when a ref fails, so a
  multi-ref push says which ref was refused.
- `scripts/check-bats-skips.sh` reads a skip whose reason quotes ` # skipped` (it was dropped, so the run passed
  unchecked) and passes each allowlist line to grep as its own pattern.
