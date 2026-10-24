# shellcheck shell=bash disable=SC2034  # SKIP and RUN are read by record.sh
# setup.sh — record.sh hooks of the stored-repack cases (golden_test.go mirrors them in buildJars and listOut).
# A jar cannot be spelled as a record.sh tree section, so a case spells each input jar as <path>.spec (the
# jarlist spec format) and setup_case builds <path> from it. A repacked jar's bytes are no contract (zip(1) and
# archive/zip lay headers out differently), so the run goes through a wrapper that writes the jarlist listing of
# the output jar (the second argument) to <out>.listing; the jars themselves are skipped from the tree.
setup_once() {
  (cd "$REPO/go" && go build -buildvcs=false -o "$WORK/jartool" ./internal/storedrepack/testdata/jartool) || die "cannot build jartool"
  cat > "$WORK/rec-run.sh" <<EOF
/bin/bash "$WORK/bin/$(basename "$SCRIPT")" "\$@"; rc=\$?
if [ \$# -ge 2 ] && [ -f "\$2" ]; then "$WORK/jartool" list "\$2" > "\$2.listing" 2>/dev/null || rm -f "\$2.listing"; fi
exit \$rc
EOF
}
setup_case() {
  local spec
  SKIP=(out.jar sub/out.jar)
  while IFS= read -r spec; do
    "$WORK/jartool" build "$D/$spec" "$D/${spec%.spec}" || die "cannot build ${spec%.spec}"
    SKIP+=("${spec%.spec}")
  done < <(cd "$D" && find . -name '*.spec' -type f | sed 's#^\./##' | LC_ALL=C sort)
  RUN="$WORK/rec-run.sh"
}
