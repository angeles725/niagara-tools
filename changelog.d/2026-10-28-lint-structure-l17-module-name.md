### Fixed — `lint-structure` L17 knows its own module by its declared name

- The B-prefix FAIL resolves this module from the gradle `moduleName` (part directory, else module root, else the
  directory name) and compares ignoring case, so a part directory named differently from its module is still judged.
