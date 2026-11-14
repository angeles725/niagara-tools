package ngdeploy

import "strings"

// stub is every fake command, with LOGDIR replaced by the fixture directory: it logs its name and arguments
// to calls.log and the environment the deploy controls to env.log, then runs the bash text of
// "LOGDIR/do/<name>" when a test wrote one.
const stub = `#!/bin/bash
n=${0##*/}
{ printf '%s' "$n"; printf ' <%s>' "$@"; echo; } >> "LOGDIR/calls.log"
echo "$n MODE=[${MODE-unset}] FD=[${FIRST_DEPLOY-unset}] FILEKEY=[${FILEKEY-unset}] MN=[${MODULE_NAME-unset}] SELF=[${NG_DEPLOY_SCRIPT-unset}]" >> "LOGDIR/env.log"
[ ! -f "LOGDIR/do/$n" ] || . "LOGDIR/do/$n"
`

// The default behaviour of the stubs, driven by FAKE_* variables a test sets.
var stubDo = map[string]string{
	"unzip": `case "$*" in
  */modules/*) [ -z "${FAKE_OLD_VERSION:-}" ] || printf '<module name="demo-rt" vendorVersion="%s">\n' "$FAKE_OLD_VERSION" ;;
  *) [ -z "${FAKE_NEW_VERSION:-}" ] || printf '<module name="demo-rt" vendorVersion="%s">\n' "$FAKE_NEW_VERSION" ;;
esac
echo '<types>'
for ((i = 0; i < ${FAKE_UNZIP_TYPES:-9}; i++)); do printf '<type name="T%s"/>\n' "$i"; done
[ -z "${FAKE_INDEX:-}" ] || printf '%s\n' "$FAKE_INDEX"
[ -z "${FAKE_UNZIP_ERR:-}" ] || echo "unzip: $FAKE_UNZIP_ERR" >&2
exit "${FAKE_UNZIP_EXIT:-0}"`,
	"tar": `[ "${FAKE_TAR_EXIT:-0}" -eq 0 ] || { echo "tar: boom" >&2; exit "$FAKE_TAR_EXIT"; }
: > "$2"`,
	"git": `case "$1" in
  diff) [ "${FAKE_GIT_DIFF_EXIT:-0}" -eq 0 ] || { echo "fatal: bad revision" >&2; exit "$FAKE_GIT_DIFF_EXIT"; }
        printf '%s\n' "${FAKE_GIT_DIFF:-}" ;;
  rev-parse) if [ "$2" = --git-dir ]; then [ -z "${FAKE_GIT_NOREPO:-}" ] || exit 128; echo .git
             else [ -z "${FAKE_GIT_HEAD_EXIT:-}" ] || exit "$FAKE_GIT_HEAD_EXIT"; printf '%s\n' "${FAKE_GIT_HEAD-deadbeef00}"; fi ;;
  cat-file) exit "${FAKE_GIT_CAT_FILE_EXIT:-0}" ;;
esac`,
	"gradlew": `case "$*" in *slotomatic*)
  [ -z "${FAKE_SLOT_OUT:-}" ] || { printf '%s\n' "$FAKE_SLOT_OUT"; echo "gradle stderr line" >&2; }
  exit "${FAKE_SLOT_EXIT:-0}" ;;
esac
exit "${FAKE_GRADLEW_EXIT:-0}"`,
	"vm.sh": `exit "${FAKE_VM_EXIT:-0}"`,
	"sr.sh": `echo "schema-risk says"; exit "${FAKE_SR_EXIT:-0}"`,
}

// fakeVariables are the variables the stubs and the deploy read; each test starts with all of them unset.
var fakeVariables = strings.Fields(`FAKE_OLD_VERSION FAKE_NEW_VERSION FAKE_UNZIP_TYPES FAKE_INDEX FAKE_UNZIP_ERR
	FAKE_UNZIP_EXIT FAKE_TAR_EXIT FAKE_GIT_DIFF_EXIT FAKE_GIT_DIFF FAKE_GIT_NOREPO FAKE_GIT_HEAD_EXIT FAKE_GIT_HEAD
	FAKE_GIT_CAT_FILE_EXIT FAKE_SLOT_OUT FAKE_SLOT_EXIT FAKE_GRADLEW_EXIT FAKE_VM_EXIT FAKE_SR_EXIT
	MODULE_NAME GRADLEW_PATH NIAGARA_HOME NIAGARA_USER_HOME JAVA_HOME STATION_MODULES_DIR EXPECTED_RT_TYPES
	EXPECTED_UX_TYPES FIRST_DEPLOY SCHEMA_BASELINE_DIR BUILD_ID SLOTOMATIC_DETECTION FULL_BACKUP KEEP_N FILEKEY
	MODE NO_BACKUP NO_DEPLOY NO_GATE WITH_SLOTOMATIC STRICT_SLOTOMATIC ACCEPT_LOSSY ENV_FILE ` + StepEnv)
