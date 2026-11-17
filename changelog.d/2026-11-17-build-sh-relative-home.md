### Fixed — build.sh passes gradle an absolute niagara_home

- `build.sh … ../mirror-415` (a relative `niagara_home`) failed in gradle, which runs in the gradle root where the
  relative path names another directory. build.sh now hands gradle the absolute path; its own messages still name
  `niagara_home` as given.
