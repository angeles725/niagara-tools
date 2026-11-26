### Fixed

- mcp-n4 `base` (`bql.BaseSlot`, shared by `n4_bql_query`, `n4_inventory`, the audits and the other tools that take a
  `base`) refuses a slot path that holds anything but letters, digits, `_`, `$` and `/` (issue #1151, follow-up of #938).
  The station matches an ORD slot path in its escaped form, so a raw space, quote, `&`, `.`, `-` or non-ASCII character
  named a component that does not exist and the read failed at the station without saying why. The refusal keeps the
  existing `base must be a plain slot path such as station:|slot:/Drivers, got '...'` text and adds the escaped-form
  hint (a space is `$20`: `Cold$20Room`, not `Cold Room`); `station:|slot:/Plant/Cold$20Room` is accepted and composed
  unchanged, as is every base that was valid. Same rule as `obix-link-audit` (#938). The frozen Python differential is
  untouched: none of its base cases changes.
