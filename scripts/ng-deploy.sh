#!/usr/bin/env bash
# scripts/ng-deploy.sh — Niagara N4 module deploy wrapper
# Usage: ng-deploy.sh [--mode A|B|C] [--env-file PATH] [--no-backup] [--full-backup] [--keep N]
#        ng-deploy.sh [--no-deploy] [--with-slotomatic] [--strict-slotomatic] [--no-gate]
#        ng-deploy.sh [--help] [--version]
# See docs/GOTCHAS.md and CLAUDE.md for decisions and invariants.
set -u

# Implementation: the Go module at <repo>/go (go/cmd/ng-deploy, go/internal/ngdeploy). This script is only the
# launcher: go/go-tool.sh `go_tool_bin` builds go/bin/ng-deploy when the binary is missing or older than a Go
# source, then this script execs it. Needs `go` on PATH to build; no Go toolchain or a failed build is exit 3.
# --help prints go/internal/ngdeploy/usage.txt (options, env variables, exit codes).
# The env file is read as plain KEY=value assignments (go/internal/envfile), never sourced: $NAME, ${NAME} and a
# leading ~ expand as under bash, and a line that would run a command is refused with its file and line number.
# A flag wins over the same option variable in the env file, and NO_GATE=1 or NO_BACKUP=1 there is refused
# (exit 10) unless --no-gate or --no-backup is given.
# readlink -f: a linked copy of this file still finds ../go, while VERSION and the default verify-module and
# schema-risk tools are taken from the directory of the INVOKED path.
NG_DEPLOY_GO="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")/.." && pwd)/go"
[ -f "$NG_DEPLOY_GO/go-tool.sh" ] || { printf 'ng-deploy.sh: no Go module at %s (go-tool.sh missing)\n' "$NG_DEPLOY_GO" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via NG_DEPLOY_GO
. "$NG_DEPLOY_GO/go-tool.sh"
NG_DEPLOY_BIN="$(go_tool_bin "$NG_DEPLOY_GO" ng-deploy)" || exit 3
NG_DEPLOY_SCRIPT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/$(basename "${BASH_SOURCE[0]}")"
export NG_DEPLOY_SCRIPT
exec "$NG_DEPLOY_BIN" "$@"
