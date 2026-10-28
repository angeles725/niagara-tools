### Fixed — run-3 review fix-forward: permissions-xml, adapter-cfg-default, logger-name

- `lint-permissions-xml`: a `module-permissions.xml` with no root element (empty, declaration-only or comment-only)
  FAILs instead of passing clean; ISO-8859-1 and US-ASCII files are decoded (a non-ASCII byte used to fail the parse
  as invalid UTF-8), any other declared encoding is exit 3; the absolute-path row names a FILE_READ or FILE_WRITE
  grant.
- `lint-adapter-cfg-default`: Cfg fields and slots are matched by (class, name) through the receiver's declared type
  and the class holding the statement, so two Cfg classes or two components sharing a name no longer hide or
  fabricate a row; a link it cannot resolve to one field and one slot gets no row.
- `lint-logger-name`, `lint-permissions-xml`, `lint-adapter-cfg-default`: only a part's own `test/` and `build/`
  directories are pruned, so a Java package named `test` or `build` under `src/` is read.
