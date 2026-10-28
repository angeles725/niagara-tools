### Added — ledger-check and a scaffolded client version ledger

- New `toolbelt/ledger-check.sh <VERSION-LEDGER.md>` (Go `go/cmd/ledger-check`) checks a client's version ledger
  against `templates/VERSION-LEDGER.md`, reading the contract from the template: title, sections, the Versions
  header, the Status vocabulary, real dates, classes declared in the ledger's own Root-cause classes table, and a
  defect with its root cause and class (FAIL); a template placeholder left in a row is a WARN.
- `scaffold-module.sh` emits `<out-dir>/<ModuleName>/odd/VERSION-LEDGER.md` for every type (the template with its
  sample row dropped); it passes `ledger-check.sh` clean.
