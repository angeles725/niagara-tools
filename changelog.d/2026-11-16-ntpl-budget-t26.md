### Changed — lint-ntpl-manifest-deps is bounded for the whole run

- `lint-ntpl-manifest-deps.sh`: every template of a template module jar and every manifest, template.bog and Px
  entry of every archive now inflate from one budget of 128 MiB for the whole run, instead of 128 MiB per
  archive. The entry the budget leaves no room for is one FAIL row (`the templates and their entries inflate
  past 134217728 bytes in all, the bound of one run -- <entry> and everything after it not read`) and nothing
  more is read: a small jar of many amplifying templates ends in seconds instead of minutes.
- At most 1000 templates are read per run, in a module or a jar; past them one FAIL ntpl-entries row names the
  first template not read.
- A jar over 32 MiB is refused from its metadata before it is opened, as a .ntpl already was: an unreadable one
  is the size row (exit 1), no longer exit 3.
