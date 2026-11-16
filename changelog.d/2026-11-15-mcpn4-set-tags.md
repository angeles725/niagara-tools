### Added — n4_set_tags writes direct tags checked against the installed dictionaries

- `n4_set_tags` adds, changes and removes direct tags (metadata slots named `ns$3aname`) on 1 to 50
  components in one batch of at most 200 ops, with the usual dry run, confirmation token, write scope,
  read-back and `n4_rollback`.
- A defined tag takes its definition's value type and default; an undefined one needs `allow_ad_hoc` and a
  namespace no dictionary installs (an installed namespace is refused, as the Tagging guide advises); a
  type change is refused. The read-back lists values the station normalized under `coerced`.
