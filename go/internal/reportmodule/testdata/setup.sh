# shellcheck shell=bash disable=SC2034  # RUN and SKIP are read by record.sh
# setup.sh — record.sh hooks of the report-module cases: the script runs as report-module.sh of <case>/toolbelt,
# a copy of this checkout's toolbelt as the Go test's toolbeltCopy builds it: every entry a symlink, the case's
# toolbelt/<entry> sections kept. A case holding toolbelt/.stub-rest gets, for every other *.sh member, a stub
# that prints nothing and exits 0 (an orchestration case that should not run the real members).
TB="$REPO/build-n4-module-kit/toolbelt"
setup_once() { # the member wrappers then find current binaries under the case's minimal PATH
  local t
  # shellcheck disable=SC1091  # resolved at runtime
  . "$REPO/go/go-tool.sh"
  # every Go binary a wrapper execs, read from the wrappers (the Go test's memberTools reads the same)
  for t in $(sed -n 's/.*go_tool_bin "\$GO_DIR" \([A-Za-z0-9-]*\)).*/\1/p' "$TB"/*.sh | sort -u); do
    go_tool_bin "$REPO/go" "$t" >/dev/null
  done
}
setup_case() {
  local m stub=0; SKIP=(toolbelt/report-module.sh); [ -d "$D/toolbelt" ] || SKIP+=(toolbelt)
  [ ! -f "$D/toolbelt/.stub-rest" ] || stub=1
  mkdir -p "$D/toolbelt"
  for m in "$TB"/*; do
    m="$(basename "$m")"
    if [ -e "$D/toolbelt/$m" ] || [ -L "$D/toolbelt/$m" ]; then continue; fi
    if [ "$stub" = 1 ] && [[ "$m" == *.sh ]]; then
      printf '#!/usr/bin/env bash\nexit 0\n' > "$D/toolbelt/$m"; chmod 755 "$D/toolbelt/$m"
    else
      ln -s "$TB/$m" "$D/toolbelt/$m"
    fi
    SKIP+=("toolbelt/$m")
  done
  rm -f "$D/toolbelt/report-module.sh"; cp "$WORK/bin/report-module.sh" "$D/toolbelt/"; RUN="$D/toolbelt/report-module.sh"
}
