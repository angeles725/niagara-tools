#!/usr/bin/env bash
# station-pull.sh — copy a platform-daemon station to a local directory, verify it and zip it (issue #1169).
#
# Usage: station-pull.sh [--env FILE] [--home DIR] [--dry-run] [--exclude-files 'a;b'] [--exclude-dirs 'x;y']
#                        --station NAME --dest DIR
#   1. Refuses a non-empty or non-directory --dest (exit 3); NAME must match ^[A-Za-z0-9_.-]+$.
#   2. plat fget -recurse -overwrite:none ... ~stations/NAME <dest>/station (credential from the mode-0600 env
#      file of plat-read.sh; --home or NIAGARA_HOME locates bin/plat.exe or bin/plat; with plat.exe under WSL
#      --dest must be under /mnt/<drive>).
#   3. plat flist -recurse ~stations/NAME, and every CRC it lists is checked against the copied file. plat
#      exits 0 and prints a `niagarad version ?` exception after a good copy, so the files decide, not plat.
#   4. Zips <dest>/station to <dest>/<NAME>-<UTC yyyymmddTHHMMSSZ>.zip without the BackupService defaults
#      (files *.hdb;*.adb;*.lock;*backup*;console.*;config.bog.b*;config_backup*, directories
#      history;alarm;webFileCache;dataRecovery; --exclude-files / --exclude-dirs replace them) and prints one
#      summary line: archive, copied, verified, zipped, bytes, excluded. The password is masked everywhere.
#   --dry-run prints the two plat command lines (password masked) and runs nothing.
# exit:  0 ok · 1 plat copied nothing, a file is missing or a CRC differs (the names are printed) · 3 usage/env
#
# Implementation: the Go module at <repo>/go (go/internal/stationpull); go/go-tool.sh builds go/bin/station-pull.
# This script is VCS-free by design — version control is never invoked (kit-links L2).
set -u

here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'station-pull: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" station-pull)" || exit 3
exec "$BIN" "$@"
