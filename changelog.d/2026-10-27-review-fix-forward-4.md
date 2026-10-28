### Fixed — run-2 review fix-forward: ledger-check, rc-scan provenance, slotomatic-doc-line

- `ledger-check` LC9 no longer skips every row holding `<...>` text: only the template's exact sample row is a
  placeholder row (WARN, not checked further); a row with one placeholder cell left WARNs and is still checked by
  LC4-LC8. Rows after a blank, comment or prose line inside `## Versions` are checked up to the next heading.
- `rc-scan` innerhtml-server provenance: an object literal with a token that starts no expression no longer loops
  forever; a page function declaration no longer stands in for a local binding that shadows its name; a `.map`
  verdict taken while writes were still being marked is re-checked once every write is marked; the index parameter
  of an iterator callback counts as a number only on a literal array receiver (a `Map.forEach` key does not).
- `lint-slotomatic-markers` `slotomatic-doc-line` is an ADVISORY row (no exit effect, `--strict` included, so
  `report-module` no longer relays it), and a blank `//` separator or an indented list item before the slot is not
  read as a continued sentence.
