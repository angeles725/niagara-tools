# shellcheck shell=bash disable=SC2034  # RUN and SKIP are read by record.sh
# setup.sh — record.sh hooks of the kit-ticket cases: the script runs as <case>/toolbelt/kit-ticket.sh (its
# default kit root is then the case dir, the KIT_TICKET_DEFAULT_ROOT the Go test passes); <case>/.tools links
# the system tools TOOLS and holds a fake `date` printing 2026-01-02, the date the Go test injects. No case
# PATH holds the system gh: a gh-present case puts its own fake gh in the tree (ghbin/gh) and on its PATH.
TOOLS=(bash sh awk sed grep head cat mkdir mv rm printf dirname)
setup_case() {
  local t; mkdir -p "$D/.tools" "$D/toolbelt"
  for t in "${TOOLS[@]}"; do if [ -e "/usr/bin/$t" ]; then ln -s "/usr/bin/$t" "$D/.tools/$t"; elif [ -e "/bin/$t" ]; then ln -s "/bin/$t" "$D/.tools/$t"; fi; done
  printf '#!/bin/sh\nprintf "2026-01-02\\n"\n' > "$D/.tools/date"; chmod +x "$D/.tools/date"
  cp "$WORK/bin/kit-ticket.sh" "$D/toolbelt/kit-ticket.sh"
  RUN="$D/toolbelt/kit-ticket.sh"; SKIP=(.tools/ toolbelt/)
}
