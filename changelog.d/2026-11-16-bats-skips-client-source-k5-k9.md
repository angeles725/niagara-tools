### Changed — check-bats-skips and check-client-source read what they used to refuse or misread

- `scripts/check-bats-skips.sh`: `^*` is refused only where it opens a branch (the line's start, after `(` or
  `|`, after anchors only), where GNU `grep -E` warns "* at start of expression"; `a^*`, `(a)^*` and `b|a^*`,
  which Go and GNU grep read alike without a warning, are allowed again. `^*a` stays refused.
