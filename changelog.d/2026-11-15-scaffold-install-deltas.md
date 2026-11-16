### Changed — scaffold-module writes option values as typed

- `--target-version` and `--plugin-version` are literal text: `&`, `|` and `\` are written as typed. Before, a
  value was read as sed replacement text (`&` was the replaced token, `\&` a literal `&`, `\n` a newline) and a
  value with an unescaped `|`, a trailing `\` or `\1`-`\9` was refused. A value that holds a newline is exit 2 with
  `scaffold-module: <option> value must be a single line (it holds a newline): <value>` and nothing written; the
  message `<option> value cannot be substituted (...)` is gone. `--vendor` keeps its own rule (one Java package
  segment once lowercased, exit 2). Retro `2026-11-14-go-scaffold-module-port` Δ1.
- Inserted text is never rewritten by another substitution: `--plugin-version 4.14 --target-version 4.15` keeps
  plugin `4.14` (it became `4.15`), and a ModuleName that holds the fixture vendor (`AngelesBox --vendor Acme`)
  keeps its name (it became `AcmeBox`). Ordinary values emit the same tree byte for byte.
