### Added — lint-lexicon-apostrophe

- New `toolbelt/lint-lexicon-apostrophe.sh [--strict] <module-root>` (WARN), the `lint-lexicon-ascii` companion: a
  lexicon value with a MessageFormat argument (`{0}`) and a single `'`, which MessageFormat drops; write `''`.
  Not yet run by `report-module`.
