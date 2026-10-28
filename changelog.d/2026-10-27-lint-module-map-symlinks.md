### Fixed — lint-module-map symlinked directories and its exit-3 contract

- `lint-module-map` no longer descends into a symlinked directory under a part: a link back up the tree no longer
  loops to an ELOOP exit 3 (or grows the walk exponentially), and a symlinked sibling directory no longer lists its
  target's files a second time as unlisted or duplicate MM4 rows. A symlinked file is still listed.
- The exit-3 contract now reads the same in the wrapper header, the rule's doc comment and `types/structure.md`:
  usage, a root with no part, an unreadable directory under a part, or an unreadable map file; the MM3 row also
  names a directory, absolute or root-escaping entry.
