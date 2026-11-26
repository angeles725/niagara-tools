### Added

- build-n4-module: `toolbelt/preview-shot.sh` (`go/internal/previewshot`) takes a headless screenshot of a dashboard
  preview. `preview-shot.sh <url> <file> [--size WxH] [--budget ms]` shoots a URL already served;
  `preview-shot.sh --serve <rc-dir> <file> [--path /hmi] [--prefix /mount] [--mock f.json]` starts
  `dashboard-preview` on a port the OS hands out, records its PID in a state file, waits until it answers, shoots, and
  stops exactly that PID, also when the shot fails. The browser gets `--use-gl=angle --use-angle=swiftshader`
  (software WebGL; on Chrome 151-159 `--use-gl=swiftshader` alone gives `getContext('webgl') === null`, so
  `PREVIEW_SHOT_GL` can swap the flags) and `--virtual-time-budget=25000` (the shot is taken before the polls fill
  the cards without it) at 1280×800.
  The browser is `PREVIEW_SHELL`, else the newest `~/.cache/ms-playwright/chromium_headless_shell-*`, else
  `chrome-headless-shell` on PATH; none is exit 3 naming `npx playwright install chromium-headless-shell`. A missing
  or empty PNG is exit 1. Documented in `BUILD-LOOP.md` §3 next to the preview rule (issue #1126).
