### Changed — lint-ntpl-manifest-deps is bounded for the whole run

- `lint-ntpl-manifest-deps.sh`: every template of a template module jar and every manifest, template.bog and Px
  entry of every archive now inflate from one budget of 128 MiB for the whole run, instead of 128 MiB per
  archive. The entry the budget leaves no room for is one FAIL row (`the templates and their entries inflate
  past 134217728 bytes in all, the bound of one run -- <entry> and everything after it not read`) and nothing
  more is read: a small jar of many amplifying templates ends in seconds instead of minutes.
- Behaviour change: a module or a template module jar with more than 1000 templates now FAILs (exit 1). The
  first 1000 are checked, in path order (entry-name order in a jar), and one FAIL ntpl-entries row names the
  first template not read.
- Behaviour change: an unreadable jar over 32 MiB is now the size-bound FAIL row (exit 1); it was exit 3. A jar
  over 32 MiB is refused from its metadata before it is opened, as a .ntpl already was.
