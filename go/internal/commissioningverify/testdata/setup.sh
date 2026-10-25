# shellcheck shell=bash disable=SC2034  # RUN and SKIP are read by record.sh
# setup.sh — record.sh hooks of the commissioning-verify cases: the script runs as commissioning-verify.sh of
# <case>/toolbelt, a copy of this checkout's toolbelt (every member a symlink, the case's toolbelt/<member>
# stubs kept), as the Go test's toolbeltCopy builds it. The lints it runs are this checkout's Go wrappers.
TB="$REPO/build-n4-module-kit/toolbelt"
setup_once() { # the wrappers then find current binaries under the case's minimal PATH
  # shellcheck disable=SC1091  # resolved at runtime
  . "$REPO/go/go-tool.sh"; go_tool_bin "$REPO/go" n4lint >/dev/null; go_tool_bin "$REPO/go" bog-audit >/dev/null
}
setup_case() {
  local m; SKIP=(toolbelt/commissioning-verify.sh); [ -d "$D/toolbelt" ] || SKIP+=(toolbelt)
  mkdir -p "$D/toolbelt"
  for m in "$TB"/*; do
    m="$(basename "$m")"; [ -e "$D/toolbelt/$m" ] || [ -L "$D/toolbelt/$m" ] || { ln -s "$TB/$m" "$D/toolbelt/$m"; SKIP+=("toolbelt/$m"); }
  done
  rm -f "$D/toolbelt/commissioning-verify.sh"; cp "$WORK/bin/commissioning-verify.sh" "$D/toolbelt/"; RUN="$D/toolbelt/commissioning-verify.sh"
}
