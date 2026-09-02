#!/usr/bin/env bash
# scripts/ng-deploy.sh — Niagara N4 module deploy wrapper
# Usage: ng-deploy.sh [--mode A|B|C] [--env-file PATH] [--no-backup --i-know-what-im-doing]
#        ng-deploy.sh [--no-deploy] [--with-slotomatic] [--strict-slotomatic] [--no-gate]
#        ng-deploy.sh [--help] [--version]
# See docs/GOTCHAS.md and CLAUDE.md for decisions and invariants.

set -euo pipefail
IFS=$'\n\t'

# ---------------------------------------------------------------------------
# Version (resolved CWD-agnostically from VERSION file next to the script)
# ---------------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
readonly VERSION_FILE="${SCRIPT_DIR}/../VERSION"
# verify-module gate binary — env-overridable (tests point it at a stub)
VERIFY_MODULE_BIN="${VERIFY_MODULE_BIN:-${SCRIPT_DIR}/../build-n4-module-kit/toolbelt/verify-module.sh}"
SCRIPT_VERSION="$(cat "${VERSION_FILE}" 2>/dev/null || echo "unknown")"
readonly SCRIPT_VERSION

# ---------------------------------------------------------------------------
# Globals (resolved by parse_args + load_env_file + validate_required)
# ---------------------------------------------------------------------------
MODE="A"
NO_BACKUP=0
NO_DEPLOY=0
IKWID=0      # i-know-what-im-doing companion flag
ENV_FILE=".env.local"
WITH_SLOTOMATIC=0
STRICT_SLOTOMATIC=0
NO_GATE=0

# ---------------------------------------------------------------------------
# print_usage — print help text to stdout (does NOT exit)
# ---------------------------------------------------------------------------
print_usage() {
    cat << 'EOF'
Usage: ng-deploy.sh [OPTIONS]

Deploy a Niagara N4 module jar to a station modules directory.

Options:
  --mode <A|B|C>           Build mode (default: A)
                             A = rt + ux jars (both)
                             B = ux jar only  (JS/CSS/Java servlet changes)
                             C = rt jar only  (Java BComponent changes)
  --env-file <PATH>        Source env from PATH instead of .env.local
  --no-deploy              Build only; do not copy or verify (jars stay in build/libs/)
  --no-backup              WARNING: skip backup step (dangerous — live station has no rollback)
  --i-know-what-im-doing   Required companion for --no-backup
  --with-slotomatic        Run :MODULE-rt:slotomatic (and :MODULE-ux:slotomatic when -ux is annotated) BEFORE build_jars (opt-in; ignored on mode B)
  --strict-slotomatic      Abort with exit 15 if annotation changes detected without --with-slotomatic
  --no-gate                Skip the verify-module.sh gate (default: on for mode A/C)
  --help                   Print this help and exit 0
  --version                Print SCRIPT_VERSION and exit 0

Required env vars (from .env.local or --env-file):
  MODULE_NAME, GRADLEW_PATH, NIAGARA_HOME, NIAGARA_USER_HOME,
  JAVA_HOME, STATION_MODULES_DIR
  EXPECTED_RT_TYPES  (required for mode A or C)
  EXPECTED_UX_TYPES  (required for mode A or B)

Optional:
  BUILD_ID             — when set, verifies index.html in ux jar contains ?v=$BUILD_ID
  SLOTOMATIC_DETECTION — warn|strict|off (default: warn); controls annotation-change heuristic

Exit codes:
  0   success (or --help)
  10  missing required env var or invalid path
  15  slotomatic failed, or annotation changes detected in strict mode
  20  backup failed, or --no-backup without --i-know-what-im-doing
  30  build (gradlew) returned non-zero
  40  copy of jar to STATION_MODULES_DIR failed
  50  verify failed: type count mismatch, BUILD_ID not found, or verify-module gate failed
EOF
}

# ---------------------------------------------------------------------------
# print_version — print SCRIPT_VERSION to stdout (does NOT exit)
# ---------------------------------------------------------------------------
print_version() {
    printf '%s\n' "${SCRIPT_VERSION}"
}

# ---------------------------------------------------------------------------
# die <code> <message> — print error to stderr and exit with code
# ---------------------------------------------------------------------------
die() {
    local code="$1"
    local msg="$2"
    printf '[ng-deploy] ERROR %s\n' "$msg" >&2
    exit "$code"
}

# ---------------------------------------------------------------------------
# load_env_file <path> — source env file if it exists; tolerate missing
# ---------------------------------------------------------------------------
load_env_file() {
    local path="$1"
    [[ -f "$path" ]] || return 0
    set -a
    # why: path is dynamic (user-supplied); not following is expected
    # shellcheck disable=SC1090
    source "$path"
    set +a
}

# ---------------------------------------------------------------------------
# parse_args — while-case flag parser; sets globals
# ---------------------------------------------------------------------------
parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --help)
                print_usage
                exit 0
                ;;
            --version)
                print_version
                exit 0
                ;;
            --mode)
                [[ $# -ge 2 ]] || die 10 "--mode requires an argument"
                MODE="$2"
                shift 2
                ;;
            --env-file)
                [[ $# -ge 2 ]] || die 10 "--env-file requires an argument"
                ENV_FILE="$2"
                shift 2
                ;;
            --no-backup)
                NO_BACKUP=1
                shift ;;
            --i-know-what-im-doing)
                IKWID=1
                shift ;;
            --no-deploy)
                NO_DEPLOY=1
                shift ;;
            --with-slotomatic)
                WITH_SLOTOMATIC=1
                shift ;;
            --strict-slotomatic)
                STRICT_SLOTOMATIC=1
                shift ;;
            --no-gate)
                NO_GATE=1
                shift ;;
            *)
                printf '[ng-deploy] ERROR unknown flag: %s\n' "$1" >&2
                print_usage
                exit 10
                ;;
        esac
    done

    # Validate mode value
    case "$MODE" in
        A|B|C) ;;
        *) die 10 "invalid --mode '$MODE'; valid values are A, B, C" ;;
    esac
}

# ---------------------------------------------------------------------------
# validate_required — check all required env vars; exit 10 on first missing
# ---------------------------------------------------------------------------
validate_required() {
    local required_all=("MODULE_NAME" "GRADLEW_PATH" "NIAGARA_HOME" "NIAGARA_USER_HOME" "JAVA_HOME" "STATION_MODULES_DIR")

    for var in "${required_all[@]}"; do
        if [[ -z "${!var:-}" ]]; then
            die 10 "$var is required but not set"
        fi
    done

    # Mode-conditional type count vars
    if [[ "$MODE" == "A" || "$MODE" == "C" ]]; then
        if [[ -z "${EXPECTED_RT_TYPES:-}" ]]; then
            die 10 "EXPECTED_RT_TYPES is required for mode $MODE"
        fi
    fi
    if [[ "$MODE" == "A" || "$MODE" == "B" ]]; then
        if [[ -z "${EXPECTED_UX_TYPES:-}" ]]; then
            die 10 "EXPECTED_UX_TYPES is required for mode $MODE"
        fi
    fi

    # Path checks for STATION_MODULES_DIR
    if [[ "${STATION_MODULES_DIR}" == *\\* ]]; then
        die 10 "STATION_MODULES_DIR must use WSL /mnt/c/... paths, not Windows-style backslashes"
    fi
    if [[ ! -d "${STATION_MODULES_DIR}" ]]; then
        die 10 "STATION_MODULES_DIR path not found: ${STATION_MODULES_DIR}"
    fi
}

# ---------------------------------------------------------------------------
# guard_no_backup — refuse --no-backup without companion
# ---------------------------------------------------------------------------
guard_no_backup() {
    if [[ "$NO_BACKUP" -eq 1 && "$IKWID" -eq 0 ]]; then
        die 20 "--no-backup requires --i-know-what-im-doing; refusing to skip backup without it"
    fi
}

# ---------------------------------------------------------------------------
# backup — tar deployed jars from STATION_MODULES_DIR into _backups/
# ---------------------------------------------------------------------------
backup() {
    local ts
    ts="$(date +%Y%m%d-%H%M%S)"
    local bk="_backups/${MODULE_NAME}-pre-${ts}.tar.gz"
    mkdir -p _backups
    printf '[ng-deploy] backup: %s\n' "$bk"
    # why: glob intentional — matches module jars by pattern; brace would be needed for portability
    # shellcheck disable=SC2086
    tar -czf "$bk" -C "$(dirname "$STATION_MODULES_DIR")" \
        "$(basename "$STATION_MODULES_DIR")" 2>/dev/null \
        || die 20 "backup failed: $bk"
    printf '[ng-deploy] backup ok: %s\n' "$bk"
}

# ---------------------------------------------------------------------------
# build_jars <mode> — invoke gradlew with tasks for the selected mode
# ---------------------------------------------------------------------------
build_jars() {
    local mode="$1"
    local tasks=()
    case "$mode" in
        A)
            # why: word-splitting intentional for gradle task list
            # shellcheck disable=SC2206
            tasks=(:${MODULE_NAME}-rt:clean :${MODULE_NAME}-ux:clean :${MODULE_NAME}-rt:jar :${MODULE_NAME}-ux:jar)
            ;;
        B)
            # shellcheck disable=SC2206
            tasks=(:${MODULE_NAME}-ux:clean :${MODULE_NAME}-ux:jar)
            ;;
        C)
            # shellcheck disable=SC2206
            tasks=(:${MODULE_NAME}-rt:clean :${MODULE_NAME}-rt:jar)
            ;;
    esac

    printf '[ng-deploy] build: mode %s — %s\n' "$mode" "${tasks[*]}"
    "${GRADLEW_PATH}" \
        -Pniagara_home="${NIAGARA_HOME}" \
        "-Pniagara_user_home=${NIAGARA_USER_HOME}" \
        -Porg.gradle.java.installations.paths="${JAVA_HOME}" \
        "${tasks[@]}" \
        || die 30 "build failed (gradlew exited non-zero)"
    printf '[ng-deploy] build ok\n'
}

# ---------------------------------------------------------------------------
# copy_jars <mode> — cp built jar(s) to STATION_MODULES_DIR
# ---------------------------------------------------------------------------
copy_jars() {
    local mode="$1"
    local src_dir
    src_dir="$(dirname "$GRADLEW_PATH")"

    case "$mode" in
        A)
            cp "${src_dir}/${MODULE_NAME}/${MODULE_NAME}-rt/build/libs/${MODULE_NAME}-rt.jar" \
               "${STATION_MODULES_DIR}/" \
               || die 40 "copy failed for ${MODULE_NAME}-rt.jar"
            cp "${src_dir}/${MODULE_NAME}/${MODULE_NAME}-ux/build/libs/${MODULE_NAME}-ux.jar" \
               "${STATION_MODULES_DIR}/" \
               || die 40 "copy failed for ${MODULE_NAME}-ux.jar"
            ;;
        B)
            cp "${src_dir}/${MODULE_NAME}/${MODULE_NAME}-ux/build/libs/${MODULE_NAME}-ux.jar" \
               "${STATION_MODULES_DIR}/" \
               || die 40 "copy failed for ${MODULE_NAME}-ux.jar"
            ;;
        C)
            cp "${src_dir}/${MODULE_NAME}/${MODULE_NAME}-rt/build/libs/${MODULE_NAME}-rt.jar" \
               "${STATION_MODULES_DIR}/" \
               || die 40 "copy failed for ${MODULE_NAME}-rt.jar"
            ;;
    esac
    printf '[ng-deploy] copy ok\n'
}

# ---------------------------------------------------------------------------
# verify_jar <jar_path> <expected_count> — check <type count in module.xml
# ---------------------------------------------------------------------------
verify_jar() {
    local jar="$1"
    local expected="$2"
    local actual
    actual="$(unzip -p "$jar" META-INF/module.xml | grep -c "<type")"
    if [[ "$actual" -ne "$expected" ]]; then
        die 50 "verify failed: expected $expected <type entries in $jar, found $actual"
    fi
    printf '[ng-deploy] verify ok (%s: %s/%s)\n' "$(basename "$jar")" "$actual" "$expected"
}

# ---------------------------------------------------------------------------
# verify_cachebuster <jar_path> — check ?v=$BUILD_ID in index.html (optional)
# ---------------------------------------------------------------------------
verify_cachebuster() {
    local jar="$1"
    # Only runs when BUILD_ID is set in the environment
    [[ -n "${BUILD_ID:-}" ]] || return 0
    local content
    content="$(unzip -p "$jar" rc/index.html 2>/dev/null || true)"
    if [[ "$content" != *"?v=${BUILD_ID}"* ]]; then
        die 50 "verify failed: index.html in $jar does not contain ?v=${BUILD_ID} (BUILD_ID check)"
    fi
    printf '[ng-deploy] verify ok (cache-buster ?v=%s found)\n' "${BUILD_ID}"
}

# ---------------------------------------------------------------------------
# print_restart_reminder <mode> — post-deploy action hint
# ---------------------------------------------------------------------------
print_restart_reminder() {
    local mode="$1"
    case "$mode" in
        A|C)
            printf '[ng-deploy] station restart required (Java classes deployed)\n'
            ;;
        B)
            printf '[ng-deploy] browser hard-reload only (no station restart needed)\n'
            ;;
    esac
}

# ---------------------------------------------------------------------------
# read_baseline_sha — reads .last-deploy-sha, validates with cat-file, fallback HEAD~1
# Always returns 0. Sets BASELINE_SHA global.
# ---------------------------------------------------------------------------
read_baseline_sha() {
    local sha_file
    sha_file="$(pwd)/.last-deploy-sha"
    BASELINE_SHA=""
    if [[ -f "$sha_file" ]]; then
        local candidate
        candidate="$(cat "$sha_file" 2>/dev/null || true)"
        candidate="${candidate%$'\n'}"
        if [[ -n "$candidate" ]] && git cat-file -e "${candidate}^{commit}" 2>/dev/null; then
            BASELINE_SHA="$candidate"
            return 0
        fi
    fi
    # fallback: HEAD~1
    BASELINE_SHA="HEAD~1"
    return 0
}

# ---------------------------------------------------------------------------
# warn_slotomatic_recommended — multi-line WARN to stderr
# ---------------------------------------------------------------------------
warn_slotomatic_recommended() {
    cat >&2 << 'WARN'
[ng-deploy] WARN: annotation changes detected in Java sources.
[ng-deploy] WARN: Run with --with-slotomatic to regenerate slot code, or
[ng-deploy] WARN: set SLOTOMATIC_DETECTION=off to suppress this warning.
[ng-deploy] WARN: Deploying without slotomatic may cause BComponent type errors.
WARN
}

# ---------------------------------------------------------------------------
# detect_annotation_changes — heuristic: git diff baseline..HEAD on *.java
# Returns 0 if @Niagara* annotation changes found, 1 otherwise.
# Respects SLOTOMATIC_DETECTION=off.  Never calls die directly.
# ---------------------------------------------------------------------------
detect_annotation_changes() {
    # off-guard: env var
    if [[ "${SLOTOMATIC_DETECTION:-warn}" == "off" ]]; then
        return 1
    fi

    # git availability guard
    if ! command -v git > /dev/null 2>&1; then
        printf '[ng-deploy] NOTICE: git not found; skipping annotation change detection\n' >&2
        return 1
    fi

    # repo guard: must be inside a git working tree
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        printf '[ng-deploy] NOTICE: not a git repository; skipping annotation change detection\n' >&2
        return 1
    fi

    read_baseline_sha

    local diff_output
    # why: glob in git diff path spec is intentional; no brace expansion needed
    # shellcheck disable=SC2086
    diff_output="$(git diff "${BASELINE_SHA}..HEAD" -- '*/src/com/**/*.java' 2>/dev/null || true)"
    if printf '%s\n' "$diff_output" | grep -qE '^[+-][[:space:]]*@Niagara(Type|Property|Action|Topic|Singleton)'; then
        return 0   # annotation changes found
    fi
    return 1       # no annotation changes
}

# ---------------------------------------------------------------------------
# ux_has_annotations — true (0) if the -ux profile source tree holds a
# @Niagara(Type|Property|Action|Topic|Singleton) annotation. Presence-based,
# not git-diff: a fresh checkout has no diff but its -ux slots still need regen.
# ---------------------------------------------------------------------------
ux_has_annotations() {
    local ux_src
    ux_src="$(dirname "$GRADLEW_PATH")/${MODULE_NAME}/${MODULE_NAME}-ux/src"
    [[ -d "$ux_src" ]] || return 1
    grep -rqE '@Niagara(Type|Property|Action|Topic|Singleton)' "$ux_src" --include='*.java' 2>/dev/null
}

# ---------------------------------------------------------------------------
# run_gate — run the verify-module.sh gate on the built jars (die 50 on fail).
# Skipped by --no-gate and for mode B; a missing gate binary warns, not blocks.
# ---------------------------------------------------------------------------
run_gate() {
    local jars=( "$@" )
    if [[ ! -x "$VERIFY_MODULE_BIN" ]]; then
        printf '[ng-deploy] WARN gate skipped: verify-module not executable at %s\n' "$VERIFY_MODULE_BIN" >&2
        return 0
    fi
    local module_dir
    module_dir="$(dirname "$GRADLEW_PATH")/${MODULE_NAME}"
    printf '[ng-deploy] gate: verify-module on %s\n' "${jars[*]##*/}"
    "$VERIFY_MODULE_BIN" --src "$module_dir" "${jars[@]}" \
        || die 50 "verify-module gate failed; jars not fit to deploy (use --no-gate to override)"
    printf '[ng-deploy] gate ok\n'
}

# run_slotomatic — invoke gradlew :MODULE_NAME-rt:slotomatic with 3 -P overrides
# die 15 on non-zero gradlew exit.
# ---------------------------------------------------------------------------
run_slotomatic() {
    local tasks=( ":${MODULE_NAME}-rt:slotomatic" )
    # P2: a -ux profile carrying @Niagara annotations needs its slots regenerated too
    # (only meaningful in mode A — mode C is rt-only, mode B never reaches here).
    if [[ "$MODE" == "A" ]] && ux_has_annotations; then
        tasks+=( ":${MODULE_NAME}-ux:slotomatic" )
        printf '[ng-deploy] slotomatic: running :%s-rt:slotomatic :%s-ux:slotomatic (-ux has @Niagara annotations)\n' "${MODULE_NAME}" "${MODULE_NAME}"
    else
        printf '[ng-deploy] slotomatic: running :%s-rt:slotomatic\n' "${MODULE_NAME}"
    fi
    "${GRADLEW_PATH}" \
        -Pniagara_home="${NIAGARA_HOME}" \
        "-Pniagara_user_home=${NIAGARA_USER_HOME}" \
        -Porg.gradle.java.installations.paths="${JAVA_HOME}" \
        "${tasks[@]}" \
        || die 15 "slotomatic failed (gradlew exited non-zero); deploy aborted"
    printf '[ng-deploy] slotomatic ok\n'
}

# ---------------------------------------------------------------------------
# write_last_deploy_sha — write git rev-parse HEAD to .last-deploy-sha
# Silent on any git error (non-critical).
# ---------------------------------------------------------------------------
write_last_deploy_sha() {
    local sha
    sha="$(git rev-parse HEAD 2>/dev/null || true)"
    sha="${sha%$'\n'}"
    if [[ -n "$sha" ]]; then
        printf '%s\n' "$sha" > "$(pwd)/.last-deploy-sha" 2>/dev/null || true
    fi
}

# ---------------------------------------------------------------------------
# main — orchestrator
# ---------------------------------------------------------------------------
main() {
    parse_args "$@"
    load_env_file "$ENV_FILE"
    validate_required
    guard_no_backup

    # Step 2: Backup (skippable with IKWID-guarded --no-backup)
    if [[ "$NO_BACKUP" -eq 0 ]]; then
        backup
    fi

    # Step 2.4: Annotation change detection (mode A/C only; skipped in mode B — ux-only)
    if [[ "$MODE" != "B" ]]; then
        if detect_annotation_changes; then
            if [[ "$STRICT_SLOTOMATIC" -eq 1 || "${SLOTOMATIC_DETECTION:-warn}" == "strict" ]]; then
                die 15 "annotation changes detected; --with-slotomatic required (strict mode)"
            elif [[ "$WITH_SLOTOMATIC" -eq 0 ]]; then
                warn_slotomatic_recommended
            fi
        fi
        # Step 2.5: Run slotomatic (opt-in via --with-slotomatic)
        if [[ "$WITH_SLOTOMATIC" -eq 1 ]]; then
            run_slotomatic
        fi
    elif [[ "$WITH_SLOTOMATIC" -eq 1 ]]; then
        printf '[ng-deploy] WARN --with-slotomatic ignored for mode B (ux-only): ng-deploy runs slotomatic for -rt only.\n' >&2
        printf '[ng-deploy] WARN to regenerate -ux slots, deploy with --mode A --with-slotomatic, or build via toolbelt/build.sh.\n' >&2
    fi

    # Step 3: Build (always; --no-deploy stops AFTER build)
    build_jars "$MODE"

    if [[ "$NO_DEPLOY" -eq 1 ]]; then
        printf '[ng-deploy] no-deploy: jars in build/libs/\n'
        exit 0
    fi

    # Step 4: Copy
    copy_jars "$MODE"

    # Step 5: Verify
    local src_dir
    src_dir="$(dirname "$GRADLEW_PATH")"
    local rt_jar="${src_dir}/${MODULE_NAME}/${MODULE_NAME}-rt/build/libs/${MODULE_NAME}-rt.jar"
    local ux_jar="${src_dir}/${MODULE_NAME}/${MODULE_NAME}-ux/build/libs/${MODULE_NAME}-ux.jar"

    case "$MODE" in
        A)
            verify_jar "$rt_jar" "$EXPECTED_RT_TYPES"
            verify_jar "$ux_jar" "$EXPECTED_UX_TYPES"
            [[ -n "${BUILD_ID:-}" ]] && verify_cachebuster "$ux_jar"
            ;;
        B)
            verify_jar "$ux_jar" "$EXPECTED_UX_TYPES"
            [[ -n "${BUILD_ID:-}" ]] && verify_cachebuster "$ux_jar"
            ;;
        C)
            verify_jar "$rt_jar" "$EXPECTED_RT_TYPES"
            ;;
    esac

    # Step 5.5: verify-module gate (default on for mode A/C; --no-gate to skip; mode B has none)
    if [[ "$NO_GATE" -eq 0 ]]; then
        case "$MODE" in
            A) run_gate "$rt_jar" "$ux_jar" ;;
            C) run_gate "$rt_jar" ;;
        esac
    fi

    # Step 6: Write deploy SHA (after successful verify + gate; silent if git absent)
    write_last_deploy_sha

    # Step 7: Post-deploy hint
    print_restart_reminder "$MODE"
    exit 0
}

# Guard: when BATS_TEST_MODE=1, functions are sourced individually for testing
# and main() is NOT auto-invoked.
if [[ "${BATS_TEST_MODE:-0}" -eq 0 ]]; then
    main "$@"
fi
