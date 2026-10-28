### Fixed — course-coverage run A review fix-forward

- `lint-structure` L17 FAILs `<symbol>:B<Type>` only when the symbol maps to this module; another module's symbol
  (or one the palette never maps) is not judged.
- `lint-charset-default` CD1 judges `getBytes()` only on a provable String receiver (a custom `getBytes()` is
  clean), and the byte/char ambiguity guard reads C-style `x[]` declarators on both sides.
