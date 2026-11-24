#!/usr/bin/env bash
# ci-local-golinks.bash — shared bats helper of the ci-local fixtures that run `ci-local.sh --changed` in a throwaway
# repository. Loaded with `load helpers/ci-local-golinks`.

# ci_local_link_go <repo> <fixture> — give <fixture>/go the entries of <repo>/go as links, its README.md copied: a
# changed kit .md runs the cardwidth gate, which counts the fixture's live cards and holds go/README.md among them.
ci_local_link_go() {
  mkdir -p "$2/go"
  ln -s "$1"/go/* "$2/go/"
  rm "$2/go/README.md"
  cp "$1/go/README.md" "$2/go/"
}
