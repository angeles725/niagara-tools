# shellcheck shell=bash
# go-tool.sh — build-on-demand for the thin shell wrappers over go/cmd/<tool> (CONTRIBUTING.md § 2.2).
# Sourced, never run: toolbelt/gen-retro-index.sh and scripts/ci-local.sh (ci-steps) share it.
# It lives in go/ beside the module it builds, so a fixture that symlinks the repository go/ next to a
# copied wrapper (tests/build-retro-sync.bats _fixture_top) still finds it. [ev: retro go-ci-steps]
#
# go_tool_bin <go-dir> <tool> — make <go-dir>/bin/<tool> current and print its path on stdout.
#   The binary is rebuilt when it is missing or older than any *.go or go.mod under <go-dir>
#   (bin/ excluded). The first call after a source change pays one `go build` (seconds, cached after);
#   that is the price of never running stale logic. Build output goes to stderr.
#   Returns 3 with a `<tool>: ...` message when <go-dir> has no go.mod, `go` is not on PATH or the
#   build fails; the old binary is then left in place and no temporary file remains.
go_tool_bin() {
  local go_dir="$1" tool="$2" bin="$1/bin/$2" tmp
  if [ ! -x "$bin" ] || [ -n "$(find -H "$go_dir" -path "$go_dir/bin" -prune -o \( -name '*.go' -o -name go.mod \) -newer "$bin" -print 2>/dev/null | head -1)" ]; then
    [ -f "$go_dir/go.mod" ] || { printf '%s: no Go module at %s\n' "$tool" "$go_dir" >&2; return 3; }
    command -v go >/dev/null 2>&1 || { printf '%s: go is not on PATH; it is needed to build %s\n' "$tool" "$bin" >&2; return 3; }
    # Build to a private name, then rename: concurrent callers never exec a half-written binary.
    tmp="$bin.tmp.$$"
    if ! (mkdir -p "$go_dir/bin" && cd "$go_dir" && go build -o "$tmp" "./cmd/$tool") >&2; then
      rm -f "$tmp"; printf '%s: go build failed in %s\n' "$tool" "$go_dir" >&2; return 3
    fi
    mv -f "$tmp" "$bin"
  fi
  printf '%s\n' "$bin"
}
