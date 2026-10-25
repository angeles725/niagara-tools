# shellcheck shell=bash
# go-tool.sh — build-on-demand for the thin shell wrappers over go/cmd/<tool> (CONTRIBUTING.md § 2.2).
# Sourced, never run: toolbelt/gen-retro-index.sh, toolbelt/gen-build-state.sh and scripts/ci-local.sh
# (ci-steps) share it.
# It lives in go/ beside the module it builds, so a fixture that symlinks the repository go/ next to a
# copied wrapper (tests/build-retro-sync.bats _fixture_top) still finds it. [ev: retro go-ci-steps]
#
# go_tool_bin <go-dir> <tool> — make <go-dir>/bin/<tool> current and print its path on stdout.
#   The binary is rebuilt when it is missing or older than any source under <go-dir>: every file except
#   bin/, testdata/ directories and *_test.go, which `go build` never compiles. Non-.go files count, so an
#   edited //go:embed asset (registry.json, usage.txt, a template) rebuilds too; tests/go-tool.bats GT14
#   fails if a non-test file ever embeds from testdata/. The first call after a source change pays one
#   `go build` (seconds, cached after); that is the price of never running stale logic. Skipping the test
#   files and fixtures halves the walk (~22 -> ~10 ms here). Build output goes to stderr.
#   [ev: retro r10-go-tool-walk-scope]
#   Returns 3 with a `<tool>: ...` message when <go-dir> has no go.mod, `go` is not on PATH, the
#   staleness check cannot read <go-dir> or the build fails; the old binary is then left in place and
#   no temporary file remains.
#   With GO_TOOL_ANNOUNCE_FD=<n> set, a rebuild first prints "<tool>: rebuilding <bin>…" to fd <n>: a
#   caller that captures stderr (the .githooks) still shows that a build is happening.
go_tool_bin() {
  local go_dir="$1" tool="$2" bin="$1/bin/$2" tmp newer=""
  if [ -x "$bin" ]; then
    # Fail closed: a find error (an unreadable source dir) must never pass as "nothing newer".
    # Mutation: GT9 -- piping this find into `head` (or `2>/dev/null || true`) hides its exit status and reuses a stale binary.
    newer="$(find -H "$go_dir" \( -path "$go_dir/bin" -o -name testdata \) -prune -o ! -type d ! -name '*_test.go' -newer "$bin" -print -quit)" \
      || { printf '%s: cannot check whether %s is current (find failed under %s)\n' "$tool" "$bin" "$go_dir" >&2; return 3; }
  fi
  if [ ! -x "$bin" ] || [ -n "$newer" ]; then
    [ -f "$go_dir/go.mod" ] || { printf '%s: no Go module at %s\n' "$tool" "$go_dir" >&2; return 3; }
    command -v go >/dev/null 2>&1 || { printf '%s: go is not on PATH; it is needed to build %s\n' "$tool" "$bin" >&2; return 3; }
    [ -z "${GO_TOOL_ANNOUNCE_FD:-}" ] || printf '%s: rebuilding %s…\n' "$tool" "$bin" >&"$GO_TOOL_ANNOUNCE_FD"
    # Build to a private name, then rename: concurrent callers never exec a half-written binary.
    # -buildvcs=false: no VCS stamping, so a test's fake `git` on PATH cannot fail the rebuild.
    tmp="$bin.tmp.$$"
    if ! (mkdir -p "$go_dir/bin" && cd "$go_dir" && go build -buildvcs=false -o "$tmp" "./cmd/$tool") >&2; then
      rm -f "$tmp"; printf '%s: go build failed in %s\n' "$tool" "$go_dir" >&2; return 3
    fi
    mv -f "$tmp" "$bin"
  fi
  printf '%s\n' "$bin"
}
