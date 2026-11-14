### Changed — scaffold-module refuses a --vendor that is not one Java package segment

- `--vendor` is checked before anything is written: lowercased, it names the vendor's Java package directory,
  so it must be an ASCII letter or `_` first, then ASCII letters, digits or `_`. Anything else (`/`, `.`, `..`,
  whitespace, a newline, a leading digit, a non-ASCII byte, `&`, `|`, `\`, empty) exits 2 naming the value. Before,
  such a value emitted a broken or escaping package path. Retro `2026-11-14-go-scaffold-module-port` Δ3.
