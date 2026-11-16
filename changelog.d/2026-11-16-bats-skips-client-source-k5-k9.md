### Changed — check-bats-skips and check-client-source read what they used to refuse or misread

- `scripts/check-bats-skips.sh`: `^*` is refused only where it opens a branch (the line's start, after `(` or
  `|`, after anchors only), where GNU `grep -E` warns "* at start of expression"; `a^*`, `(a)^*` and `b|a^*`,
  which Go and GNU grep read alike without a warning, are allowed again. `^*a` stays refused.
- `scripts/check-client-source.sh --build-state`: a BUILD-STATE saved as UTF-16 with its byte-order mark (either
  byte order) is decoded and declares what it declares; UTF-16 that cannot be decoded (no mark, so NUL bytes, or
  an odd byte count) is exit 3 with a message saying to save it as UTF-8. A UTF-8 byte-order mark that opens a
  line other than the first is exit 3 naming the line. Each of these read "undeclared", exit 0.
