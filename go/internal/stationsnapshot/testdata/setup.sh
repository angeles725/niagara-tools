# shellcheck shell=bash disable=SC2034  # RUN and SKIP are read by record.sh
# setup.sh — record.sh hooks of the station-snapshot cases: the script runs as <case>/toolbelt/station-snapshot.sh
# (the prefix of the shell's own error messages, the STATION_SNAPSHOT_SCRIPT the Go test passes); <case>/.tools
# links the system tools TOOLS and holds a fake `date` printing 20260102T030405Z, the UTC stamp the Go test
# injects; <case>/.tools-nosha is the same without sha256sum. A case's @env PATH picks one of them.
TOOLS=(bash sh dirname basename mkdir mktemp rm cp chmod sha256sum cut wc tr find sort awk)
setup_case() {
  local t dir; mkdir -p "$D/.tools" "$D/.tools-nosha" "$D/toolbelt"
  for t in "${TOOLS[@]}"; do
    for dir in .tools .tools-nosha; do
      [ "$dir/$t" != .tools-nosha/sha256sum ] || continue
      if [ -e "/usr/bin/$t" ]; then ln -s "/usr/bin/$t" "$D/$dir/$t"; elif [ -e "/bin/$t" ]; then ln -s "/bin/$t" "$D/$dir/$t"; fi
    done
  done
  for dir in .tools .tools-nosha; do printf '#!/bin/sh\nprintf "20260102T030405Z\\n"\n' > "$D/$dir/date"; chmod +x "$D/$dir/date"; done
  cp "$WORK/bin/station-snapshot.sh" "$D/toolbelt/station-snapshot.sh"
  RUN="$D/toolbelt/station-snapshot.sh"; SKIP=(.tools/ .tools-nosha/ toolbelt/)
}
