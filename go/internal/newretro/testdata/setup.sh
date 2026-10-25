# shellcheck shell=bash disable=SC2034  # RUN, SKIP and SETUP_ENV are read by record.sh
# setup.sh — record.sh hooks of the new-retro cases: the script runs as <case>/toolbelt/new-retro.sh (its
# default kit root is then the case dir, the NEW_RETRO_DEFAULT_ROOT the Go test passes) beside this checkout's
# gen-retro-index.sh, and <case>/fakebin/date prints 2026-01-02, the date the Go test injects.
GEN="$REPO/build-n4-module-kit/toolbelt/gen-retro-index.sh"
setup_once() { bash "$GEN" --definitely-not-an-option >/dev/null 2>&1 || true; } # builds go/bin/gen-retro-index
setup_case() {
  mkdir -p "$D/toolbelt" "$D/fakebin"
  cp "$WORK/bin/new-retro.sh" "$D/toolbelt/new-retro.sh"; ln -s "$GEN" "$D/toolbelt/gen-retro-index.sh"
  printf '#!/bin/sh\nprintf "2026-01-02\\n"\n' > "$D/fakebin/date"; chmod +x "$D/fakebin/date"
  RUN="$D/toolbelt/new-retro.sh"; SKIP=(toolbelt/ fakebin/); SETUP_ENV=(PATH="$D/fakebin:/usr/bin:/bin")
}
