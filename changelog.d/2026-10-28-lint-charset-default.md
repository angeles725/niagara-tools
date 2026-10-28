### Added — lint-charset-default

- New `toolbelt/lint-charset-default.sh [--strict] <src-root>`: WARNs text bytes encoded or decoded with the platform
  default charset — a no-argument `getBytes()`, a `new String` over a byte array without a charset, a one-argument
  `InputStreamReader` / `OutputStreamWriter`, and any `FileReader` / `FileWriter` (no charset overload on Java 8).
  Not yet run by `report-module`.
