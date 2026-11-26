#!/usr/bin/env bash
# preview-shot.sh — headless screenshot of a dashboard preview, WebGL and polled cards included (issue #1126).
#
# Usage: preview-shot.sh <url> <file> [--size WxH] [--budget ms]
#        preview-shot.sh --serve <rc-dir> <file> [--path /hmi] [--prefix /mount] [--mock f.json]
#                        [--size WxH] [--budget ms]
#   First form: shoot a URL that is already served. Second form: start dashboard-preview (--rc <rc-dir>) on a port the
#   OS hands out, record its PID in <state>/server-<port>.pid, wait until the page answers, shoot <path> (default
#   /hmi), then stop exactly that process (by the PID it started, never by a command-line pattern) and remove the
#   file, also when the shot fails or the run is interrupted. A SIGKILL of preview-shot leaves the server running:
#   its PID is in the state file; check it with `ps -p <pid>` before `kill <pid>`.
#   The browser is run as: --use-gl=angle --use-angle=swiftshader --enable-unsafe-swiftshader (software WebGL: the
#   canvas is black without it, and on Chrome 151-159 `--use-gl=swiftshader` alone gives no WebGL at all)
#   --virtual-time-budget=<ms> (without it the shot is taken before the page's polls fill it; default 25000)
#   --window-size=<W,H> (default 1280x800) --screenshot=<file> <url>. A file left by an earlier run is deleted
#   first, so only a fresh PNG passes.
# Env:   PREVIEW_SHELL (browser; else the newest ~/.cache/ms-playwright/chromium_headless_shell-*, under
#        PLAYWRIGHT_BROWSERS_PATH when set; else chrome-headless-shell on PATH) · PREVIEW_SERVER (dashboard-preview;
#        else PATH, else ~/.local/lib/niagara-go/bin) · PREVIEW_SHOT_GL (other GL flags, e.g. --use-gl=swiftshader
#        for an old build) · PREVIEW_SHOT_STATE (PID file directory; default
#        ~/.local/state/niagara-tools/preview-shot)
#        · PREVIEW_SHOT_WAIT_MS (wait for the server; default 30000)
# exit:  0 shot written · 1 the shot or the server failed (no file, empty file, server died or never answered)
#        · 3 usage, or no browser (install one: npx playwright install chromium-headless-shell) or no
#          dashboard-preview
#
# Implementation: the Go module at <repo>/go (go/internal/previewshot); go/go-tool.sh builds go/bin/preview-shot.
# VCS-free by design: version control is never invoked (kit-links L2).
set -u

here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'preview-shot: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" preview-shot)" || exit 3
exec "$BIN" "$@"
