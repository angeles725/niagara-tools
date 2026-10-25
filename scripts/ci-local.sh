#!/usr/bin/env bash
# ci-local.sh — replay .github/workflows/ci.yml locally with the CI tools and semantics.
#
# Usage: scripts/ci-local.sh [--list] [--only <substring>] [-j N] [--ci-file <path>] [--cache-dir <path>]
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
# The shellcheck tarball is cached under ${XDG_CACHE_HOME:-$HOME/.cache}/niagara-tools/shellcheck-v0.10.0/
# and its sha256 is checked on every use; never installed system-wide, never sudo.
#
# Exit: 0 every run step passed; 1 a step failed; 2 usage error or an UNCOVERED / unparsable step;
#       3 the pinned shellcheck could not be provisioned (sha256 mismatch, download failed, no sha256 tool),
#         the ci-steps parser or host-budget could not be built (no `go` on PATH, build failure), or host-budget
#         could not read the cgroup root;
#       4 host budget exceeded (go/cmd/host-budget FAIL row): a cgroup under ${CI_LOCAL_CGROUP_ROOT:-/sys/fs/cgroup}
#         uses more than half of its pids.max (a full init.scope rejected every fork and aborted all sessions on
#         2026-10-22), the system thread count is above half of threads-max, or / or /mnt/c is above 90% used;
#         wait (or free disk) and retry. --list never runs the check.
# [ev: retro 2026-10-22-ci-local-replay] [ev: retro go-ci-steps] [ev: retro r8-t47-host-budget]
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via ROOT
. "$ROOT/scripts/lib/sha256.sh" || exit 3
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via ROOT
. "$ROOT/go/go-tool.sh" || exit 3

SC_VER=v0.10.0
SC_TAR="shellcheck-${SC_VER}.linux.x86_64.tar.xz"
SC_URL="https://github.com/koalaman/shellcheck/releases/download/${SC_VER}/${SC_TAR}"
# Computed 2026-10-22 from a download of SC_URL (2,404,716 bytes).
SC_SHA256=6c881ab0698e4e6ea235245f22832860544f17ba386442fe7e9d629f8cbedf87

# "<step-name glob>|<reason>": CI steps replaced by a local equivalent. `uses:` refs match the same way.
SKIP_RULES=(
  "actions/checkout@*|the working tree is already checked out"
  "Set up Java 8*|uses the JDK on PATH (CI pins Temurin 8)"
  "Pre-fetch JUnit*|network download into ~/.gradle; run-pure-test.bats needs junit-4.13.2 there (one gradle build fetches it)"
  "Install bats*|uses bats (and GNU parallel, when on PATH, for -j) on PATH"
  "Install shellcheck*|replaced by the cached, sha256-checked shellcheck ${SC_VER} prepended to PATH"
  "Set up Go*|uses the go on PATH (CI installs the go/go.mod version; local go must exist)"
  "Set up CPython 3.15*|uses python3 on PATH (CI installs 3.15, the reference host's)"
)

usage() { printf 'usage: %s [--list] [--only <substring>] [-j N] [--ci-file <path>] [--cache-dir <path>]\n' "$0" >&2; exit 2; }

LIST=0 ONLY="" JOBS=3 CI_FILE="$ROOT/.github/workflows/ci.yml"
CACHE="${XDG_CACHE_HOME:-$HOME/.cache}/niagara-tools/shellcheck-${SC_VER}"
while [ $# -gt 0 ]; do
  case "$1" in
    --list) LIST=1 ;;
    --only) [ $# -ge 2 ] || usage; ONLY="$2"; shift ;;
    -j) [[ "${2:-}" =~ ^[1-9][0-9]*$ ]] || usage; JOBS="$2"; shift ;;
    --ci-file) [ $# -ge 2 ] || usage; CI_FILE="$2"; shift ;;
    --cache-dir) [ $# -ge 2 ] || usage; CACHE="$2"; shift ;;
    *) usage ;;
  esac
  shift
done

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

# check_host_budget — refuse to start the steps while go/cmd/host-budget reports a FAIL row: a cgroup above
# 50% of a numeric pids.max (or the system thread count above 50% of threads-max), or / or /mnt/c above 90%
# used. WARN rows (pids above 40%, disk above 80%, GOCACHE above 20 GB) are printed and do not block.
# A cgroup root that does not exist at all (macOS, cgroup-less containers) is a stderr note and the run
# proceeds, as the pre-T47 inline loop did; a root that exists but cannot be read stays exit 3.
# CI_LOCAL_HOST_BUDGET_ARGS holds extra host-budget arguments (word-split; no spaces inside one argument),
# so the tests can point --proc-root, --gocache and --disk-pct at fixtures instead of this machine.
# Mutation: CL10 -- dropping this call lets the steps run on a host near its fork limit.
check_host_budget() {
  local bin rc=0 root="${CI_LOCAL_CGROUP_ROOT:-/sys/fs/cgroup}" extra=()
  if [ ! -e "$root" ]; then
    printf 'ci-local: no cgroup root at %s; host budget not checked\n' "$root" >&2
    return 0
  fi
  read -r -a extra <<< "${CI_LOCAL_HOST_BUDGET_ARGS:-}"
  bin="$(go_tool_bin "$ROOT/go" host-budget)" || exit 3
  "$bin" --cgroup-root "$root" ${extra[@]+"${extra[@]}"} >&2 || rc=$?
  case "$rc" in
    0) ;;
    1) printf 'ci-local: host budget exceeded (FAIL row above); wait for other sessions and retry\n' >&2; exit 4 ;;
    *) printf 'ci-local: host-budget failed (exit %d: 3 = cannot read host / usage, other = crash/signal)\n' "$rc" >&2; exit 3 ;;
  esac
}
check_host_budget

# provision_shellcheck — verify (and on first use download) the pinned tarball, unpack it into $WORK, PATH it.
provision_shellcheck() {
  local tar="$CACHE/$SC_TAR" got
  sha256_require ci-local || exit 3
  [ "$(uname -s)/$(uname -m)" = Linux/x86_64 ] || { printf 'ci-local: the pinned shellcheck is linux x86_64 only\n' >&2; exit 3; }
  local src="$tar"
  if [ ! -f "$tar" ]; then
    src="$tar.partial"
    mkdir -p "$CACHE"
    curl -fsSL -o "$src" "$SC_URL" || { rm -f "$src"; printf 'ci-local: download failed: %s\n' "$SC_URL" >&2; exit 3; }
  fi
  got="$(sha256_of "$src")"
  if [ "$got" != "$SC_SHA256" ]; then
    [ "$src" = "$tar" ] || rm -f "$src"
    printf 'ci-local: sha256 mismatch for %s (got %s, pinned %s); remove it to re-download\n' "$src" "$got" "$SC_SHA256" >&2
    exit 3
  fi
  [ "$src" = "$tar" ] || mv "$src" "$tar"
  tar -xJf "$tar" -C "$WORK"
  PATH="$WORK/shellcheck-${SC_VER}:$PATH"
  printf 'ci-local: %s\n' "$(shellcheck --version | grep '^version:')"
}

selected=()
for i in "${!NAMES[@]}"; do
  [ "${VERDICTS[$i]}" = RUN ] && [[ "${NAMES[$i]}" == *"$ONLY"* ]] && selected+=("$i")
done
for i in "${selected[@]}"; do
  if grep -q shellcheck <<< "${BODIES[$i]}"; then provision_shellcheck; break; fi
done

BATS_ALL='bats tests/*.bats' BATS_CI_J='bats -j 4 --no-parallelize-within-files '
BATS_J="$BATS_ALL" BATS_LOCAL_J='bats '
if command -v parallel >/dev/null 2>&1; then BATS_J="bats -j $JOBS tests/*.bats" BATS_LOCAL_J="bats -j $JOBS --no-parallelize-within-files "; fi
mkdir -p "$WORK/runner-temp"
passed=0 failed=0
cd "$ROOT"
for i in "${selected[@]}"; do
  printf '== %s\n' "${NAMES[$i]}"
  body="${BODIES[$i]//"$BATS_ALL"/"$BATS_J"}"
  body="${body//"$BATS_CI_J"/"$BATS_LOCAL_J"}"
  rc=0
  CI=true RUNNER_TEMP="$WORK/runner-temp" bash -e -c "$body" || rc=$?
  if [ "$rc" -eq 0 ]; then printf 'PASS %s\n' "${NAMES[$i]}"; passed=$((passed + 1))
  else printf 'FAIL %s (exit %d)\n' "${NAMES[$i]}" "$rc"; failed=$((failed + 1)); fi
done
printf 'ci-local: %d passed, %d failed, %d skipped\n' "$passed" "$failed" "$(( ${#NAMES[@]} - ${#selected[@]} ))"
[ "$failed" -eq 0 ]
