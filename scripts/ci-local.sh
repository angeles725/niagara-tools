#!/usr/bin/env bash
# ci-local.sh — replay .github/workflows/ci.yml locally with the CI tools and semantics.
#
# Usage: scripts/ci-local.sh [--list] [--only <substring>] [-j N] [--ci-file <path>] [--cache-dir <path>]
#        scripts/ci-local.sh --changed [--range <from>..<to>] [--ci-file <path>] [--cache-dir <path>]
#
# The steps come from ci.yml itself (go/cmd/ci-steps, built on demand by go/go-tool.sh), so the replay
# cannot drift from CI:
#   - every `run:` step runs as GitHub runs it: `bash -e -c` (errexit, NO pipefail) from the repo root,
#     with CI=true and RUNNER_TEMP set; a failure is reported and the next step still runs;
#   - steps with a local equivalent are skipped by the SKIP_RULES below (shellcheck install -> the pinned
#     v0.10.0 binary from a sha256-checked cache, prepended to PATH; bats install -> bats on PATH;
#     actions/setup-go -> the go on PATH);
#   - a step it can neither run nor skip on purpose (sudo/apt-get, an unknown `uses:` action, a step-level
#     env:/shell:/if:) is UNCOVERED: --list and the run both stop with exit 2 instead of skipping it silently.
# --list prints RUN/SKIP per step; --only runs the steps whose name contains <substring>; -j N (default 3)
# turns `bats tests/*.bats` into `bats -j N tests/*.bats`, and CI's `bats -j 4 --no-parallelize-within-files `
# (four files at a time) into `bats -j N --no-parallelize-within-files `, when GNU parallel is on PATH; without
# parallel it runs as plain `bats ` (bats refuses --no-parallelize-within-files without -j).
# A step that runs bats runs the bats-core version ci.yml installs (its "Install bats*" step: BATS_VER=,
# BATS_SHA256=), fetched when the bats on PATH is another version (lib/ci-local-tools.sh). (retro 2026-11-15 coordination-stretch C8)
# A Go step run without the npm step (--only "Go") on a host without node or acorn runs with KIT_REQUIRE_JS_TOOLS
# unset and a NOTE, so its vendorfloor tests skip instead of failing (lib/ci-local-js.sh). (retro 2026-11-15 coordination-stretch T12)
# --changed runs no step: it runs the pinned shellcheck, the kit's bats lints and gofmt over the files changed
# against origin/main, or over the commit range --range names (.githooks/pre-push calls it with the pushed
# range); lib/ci-local-changed.sh. (retro 2026-11-15 coordination-stretch C3) A kit *.md, ledger.d or changelog.d
# change also runs tests/kit-links.bats on a snapshot of it; lib/ci-local-kitlinks.sh (T25). Its bats not provisioned
# at the ci.yml pin (the download error, or one line when the provisioned bats is not the pinned version), or a git
# error while detecting the change (one line), exits 3 (T32).
# The tarballs are cached under ${XDG_CACHE_HOME:-$HOME/.cache}/niagara-tools/shellcheck-v0.10.0/ (--cache-dir)
# and their sha256 is checked on every use; never installed system-wide, never sudo.
#
# Exit: 0 every run step (or --changed check) passed; 1 a step or check failed; 2 usage error or an UNCOVERED /
#       unparsable step;
#       3 the pinned shellcheck could not be provisioned (sha256 mismatch, download failed, no sha256 tool),
#         the ci-steps parser or host-budget could not be built (no `go` on PATH, build failure), or host-budget
#         could not read the cgroup root;
#       4 host budget exceeded (go/cmd/host-budget FAIL row): a cgroup under ${CI_LOCAL_CGROUP_ROOT:-/sys/fs/cgroup}
#         uses more than half of its pids.max (a full init.scope rejected every fork and aborted all sessions on
#         2026-10-22), the system thread count is above half of threads-max, or / or /mnt/c is above 95% used (DISK_PCT);
#         wait (or free disk) and retry. --list never runs the check.
# [ev: retro 2026-10-22-ci-local-replay] [ev: retro go-ci-steps] [ev: retro r8-t47-host-budget]
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via ROOT
. "$ROOT/scripts/lib/sha256.sh" || exit 3
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via ROOT
. "$ROOT/go/go-tool.sh" || exit 3
# shellcheck disable=SC1091  # sibling libs, resolved at runtime via ROOT
. "$ROOT/scripts/lib/ci-local-tools.sh" || exit 3
# shellcheck disable=SC1091
. "$ROOT/scripts/lib/ci-local-changed.sh" || exit 3
# shellcheck disable=SC1091
. "$ROOT/scripts/lib/ci-local-js.sh" || exit 3
# shellcheck disable=SC1091
. "$ROOT/scripts/lib/ci-local-host.sh" || exit 3
# shellcheck disable=SC1091
. "$ROOT/scripts/lib/ci-local-kitlinks.sh" || exit 3

# "<step-name glob>|<reason>": CI steps replaced by a local equivalent. `uses:` refs match the same way.
SKIP_RULES=(
  "actions/checkout@*|the working tree is already checked out"
  "Cache the two pinned test jars*|a CI cache of downloads; the local run uses what is already on disk"
  "Set up Java 8*|uses the JDK on PATH (CI pins Temurin 8)"
  "Pre-fetch JUnit*|network download into ~/.gradle; run-pure-test.bats needs junit-4.13.2 there (one gradle build fetches it)"
  "Install bats*|uses bats (and GNU parallel, when on PATH, for -j) on PATH"
  "Install shellcheck*|replaced by the cached, sha256-checked shellcheck ${SC_VER} prepended to PATH"
  "Set up Go*|uses the go on PATH (CI installs the go/go.mod version; local go must exist)"
  "Set up CPython 3.15*|uses python3 on PATH (CI installs 3.15, the reference host's)"
  "Set up Node.js*|uses node and npm on PATH (CI installs the pinned Node.js; the npm tools step then runs as in CI)"
)

usage() {
  printf 'usage: %s [--list] [--only <substring>] [-j N] [--ci-file <path>] [--cache-dir <path>]\n' "$0" >&2
  printf '       %s --changed [--range <from>..<to>] [--ci-file <path>] [--cache-dir <path>]\n' "$0" >&2
  exit 2
}

CHANGED=0 RANGE="" LIST=0 ONLY="" JOBS=3 CI_FILE="$ROOT/.github/workflows/ci.yml"
CACHE="${XDG_CACHE_HOME:-$HOME/.cache}/niagara-tools/shellcheck-${SC_VER}"
while [ $# -gt 0 ]; do
  case "$1" in
    --list) LIST=1 ;;
    --changed) CHANGED=1 ;;
    --range) [ $# -ge 2 ] || usage; RANGE="$2"; shift ;;
    --only) [ $# -ge 2 ] || usage; ONLY="$2"; shift ;;
    -j) [[ "${2:-}" =~ ^[1-9][0-9]*$ ]] || usage; JOBS="$2"; shift ;;
    --ci-file) [ $# -ge 2 ] || usage; CI_FILE="$2"; shift ;;
    --cache-dir) [ $# -ge 2 ] || usage; CACHE="$2"; shift ;;
    *) usage ;;
  esac
  shift
done
[ -z "$RANGE" ] || [ "$CHANGED" -eq 1 ] || usage
[ "$CHANGED" -eq 0 ] || { [ "$LIST" -eq 0 ] && [ -z "$ONLY" ]; } || usage

# classify <kind> <name> <body> — print "RUN", "SKIP|<reason>" or "UNCOVERED|<reason>".
classify() {
  local rule
  for rule in "${SKIP_RULES[@]}"; do
    # shellcheck disable=SC2053  # the rule is a glob on purpose
    if [[ "$2" == ${rule%%|*} ]]; then printf 'SKIP|%s\n' "${rule#*|}"; return; fi
  done
  case "$1" in
    bad) printf 'UNCOVERED|%s\n' "$3" ;;
    uses) printf 'UNCOVERED|action %s has no local equivalent; add a SKIP_RULES entry\n' "$3" ;;
    *) if grep -qE '(^|[^[:alnum:]_])(sudo|apt-get)([^[:alnum:]_]|$)' <<< "$3"; then
         printf 'UNCOVERED|needs a system install; add a SKIP_RULES entry with its local equivalent\n'
       else printf 'RUN\n'; fi ;;
  esac
}

# Parse once into parallel arrays; a parse error is a drift failure (exit 2).
KINDS=() NAMES=() BODIES=() VERDICTS=()
parsed="$(mktemp)"; WORK="$(mktemp -d)"
trap 'rm -rf "$parsed" "$WORK"' EXIT
CI_STEPS="$(go_tool_bin "$ROOT/go" ci-steps)" || exit 3
"$CI_STEPS" "$CI_FILE" > "$parsed" || exit 2
while IFS= read -r -d '' kind && IFS= read -r -d '' name && IFS= read -r -d '' body; do
  KINDS+=("$kind") NAMES+=("$name") BODIES+=("$body") VERDICTS+=("$(classify "$kind" "$name" "$body")")
done < "$parsed"

uncovered=0
for i in "${!NAMES[@]}"; do
  v="${VERDICTS[$i]}"
  case "$v" in
    RUN) [ "$LIST" -eq 1 ] && printf 'RUN %s\n' "${NAMES[$i]}" ;;
    SKIP*) [ "$LIST" -eq 1 ] && printf 'SKIP %s — %s\n' "${NAMES[$i]}" "${v#*|}" ;;
    *) printf 'UNCOVERED %s — %s\n' "${NAMES[$i]}" "${v#*|}"; uncovered=1 ;;
  esac
done
if [ "$LIST" -eq 1 ]; then
  for rule in "${SKIP_RULES[@]}"; do
    hit=0
    for n in "${NAMES[@]}"; do
      # shellcheck disable=SC2053  # the rule is a glob on purpose
      [[ "$n" == ${rule%%|*} ]] && hit=1
    done
    [ "$hit" -eq 1 ] || printf 'STALE skip rule matches no step: %s\n' "${rule%%|*}" >&2
  done
fi
[ "$uncovered" -eq 0 ] || exit 2
[ "$LIST" -eq 0 ] || exit 0

check_host_budget

# --changed runs no workflow step: the lints of the change, after the same host-budget gate, then kit-links.
if [ "$CHANGED" -eq 1 ]; then
  rc=0 kl=0
  run_changed || rc=$?
  run_kit_links || kl=$?  # Mutation: CL30 -- dropping this call lets a dangling kit reference through to CI.
  exit $((kl > rc ? kl : rc))
fi

selected=()
for i in "${!NAMES[@]}"; do
  [ "${VERDICTS[$i]}" = RUN ] && [[ "${NAMES[$i]}" == *"$ONLY"* ]] && selected+=("$i")
done
npm_selected=0
for i in "${selected[@]}"; do [[ "${BODIES[$i]}" != *"npm ci "* ]] || npm_selected=1; done
for i in "${selected[@]}"; do
  if grep -q shellcheck <<< "${BODIES[$i]}"; then provision_shellcheck; break; fi
done
for i in "${selected[@]}"; do
  if grep -qE '(^|[;&|[:space:]])bats[[:space:]]' <<< "${BODIES[$i]}"; then provision_bats; break; fi
done

BATS_ALL='bats tests/*.bats' BATS_CI_J='bats -j 4 --no-parallelize-within-files '
BATS_J="$BATS_ALL" BATS_LOCAL_J='bats '
if command -v parallel >/dev/null 2>&1; then BATS_J="bats -j $JOBS tests/*.bats" BATS_LOCAL_J="bats -j $JOBS --no-parallelize-within-files "; fi
mkdir -p "$WORK/runner-temp"
passed=0 failed=0
cd "$ROOT"
for i in "${selected[@]}"; do
  printf '== %s\n' "${NAMES[$i]}"
  body="$(js_tools_guard "${NAMES[$i]}" "${BODIES[$i]}" "$npm_selected")"
  body="${body//"$BATS_ALL"/"$BATS_J"}"
  body="${body//"$BATS_CI_J"/"$BATS_LOCAL_J"}"
  rc=0
  CI=true RUNNER_TEMP="$WORK/runner-temp" bash -e -c "$body" || rc=$?
  if [ "$rc" -eq 0 ]; then printf 'PASS %s\n' "${NAMES[$i]}"; passed=$((passed + 1))
  else printf 'FAIL %s (exit %d)\n' "${NAMES[$i]}" "$rc"; failed=$((failed + 1)); fi
done
printf 'ci-local: %d passed, %d failed, %d skipped\n' "$passed" "$failed" "$(( ${#NAMES[@]} - ${#selected[@]} ))"
[ "$failed" -eq 0 ]
