### Added — lint-module-map MM7: one row per concern folder

- `lint-module-map` MM7: a concern folder (the directory right under a role folder: rt `model/` or `adapter/`;
  ux `route/`, `service/` or `model/`; wb and se `model/`) whose files two rows list, or a non-plumbing row
  listing files in concern folders of two names, is a WARN in a `modularity: legacy` module and a FAIL in a
  `modularity: strict` one. Concern folder names are free per module (rt `model/battery/` and
  `adapter/battery/` are one concern). A module with no concern sub-folder gets no MM7 row.
