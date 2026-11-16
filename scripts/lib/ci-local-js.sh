#!/usr/bin/env bash
# shellcheck shell=bash
# lib/ci-local-js.sh — the npm prerequisite of ci.yml's Go step for scripts/ci-local.sh. Sourced only, with ROOT
# (the repo root) set by the caller.
# The Go step exports KIT_REQUIRE_JS_TOOLS=1, which fails the vendorfloor tests when node or acorn is missing; CI
# installs acorn in the npm step just before it. A run that selects the Go step without that npm step (--only "Go")
# on a host without acorn would fail on the missing tool, not on the code, so the export is replaced by an unset
# and a NOTE says so. A host that resolves acorn, or a run that includes the npm step, keeps CI's strict setting.
# (retro 2026-11-15 coordination-stretch T12)

JS_REQUIRE_EXPORT='export KIT_REQUIRE_JS_TOOLS=1'

# acorn_available — the lookup order of go/internal/vendorfloor's tests: KIT_ACORN, the toolbelt's npm install,
# then a node that resolves acorn on its own.
acorn_available() {
  command -v "${KIT_NODE:-node}" >/dev/null 2>&1 || return 1
  [ -z "${KIT_ACORN:-}" ] || return 0
  [ ! -d "$ROOT/build-n4-module-kit/toolbelt/eslint/node_modules/acorn" ] || return 0
  "${KIT_NODE:-node}" -e 'require.resolve("acorn")' >/dev/null 2>&1
}

# js_tools_guard <name> <body> <npm step selected: 0|1> — print the body to run: unchanged, or with the export
# replaced by an unset (and a NOTE on stderr) when the npm step is not part of the run and acorn is missing.
js_tools_guard() {
  local name="$1" body="$2"
  if [ "$3" -eq 1 ] || [[ "$body" != *"$JS_REQUIRE_EXPORT"* ]] || acorn_available; then
    printf '%s' "$body"
    return 0
  fi
  printf "ci-local: NOTE: node or acorn is not installed and the npm step is not part of this run: KIT_REQUIRE_JS_TOOLS unset for '%s' (its vendorfloor tests skip; run without --only, or npm ci --prefix build-n4-module-kit/toolbelt/eslint)\n" "$name" >&2
  printf '%s' "${body//"$JS_REQUIRE_EXPORT"/unset KIT_REQUIRE_JS_TOOLS}"
}
