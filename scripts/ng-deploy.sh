#!/usr/bin/env bash
# scripts/ng-deploy.sh — Niagara N4 module deploy wrapper
# Usage: ng-deploy.sh [--mode A|B|C] [--env-file PATH] [--no-backup --i-know-what-im-doing]
#        ng-deploy.sh [--no-deploy] [--help]
# See docs/GOTCHAS.md and CLAUDE.md for decisions and invariants.

set -euo pipefail
IFS=$'\n\t'

# ---------------------------------------------------------------------------
# Globals (resolved by parse_args + load_env_file + validate_required)
# ---------------------------------------------------------------------------
MODE="A"
NO_BACKUP=0
NO_DEPLOY=0
IKWID=0      # i-know-what-im-doing companion flag
ENV_FILE=".env.local"

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
  --help                   Print this help and exit 0

Required env vars (from .env.local or --env-file):
  MODULE_NAME, GRADLEW_PATH, NIAGARA_HOME, NIAGARA_USER_HOME,
  JAVA_HOME, STATION_MODULES_DIR
  EXPECTED_RT_TYPES  (required for mode A or C)
  EXPECTED_UX_TYPES  (required for mode A or B)

Optional:
  BUILD_ID  — when set, verifies index.html in ux jar contains ?v=$BUILD_ID

Exit codes:
  0   success (or --help)
  10  missing required env var or invalid path
  20  backup failed, or --no-backup without --i-know-what-im-doing
  30  build (gradlew) returned non-zero
  40  copy of jar to STATION_MODULES_DIR failed
  50  verify failed: type count mismatch or BUILD_ID not found in index.html
EOF
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

    # Step 6: Post-deploy hint
    print_restart_reminder "$MODE"
    exit 0
}

# Guard: when BATS_TEST_MODE=1, functions are sourced individually for testing
# and main() is NOT auto-invoked.
if [[ "${BATS_TEST_MODE:-0}" -eq 0 ]]; then
    main "$@"
fi
