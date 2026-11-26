### Fixed

- build-n4-module: `build.sh` fast-forwards a clean kit-main worktree strictly behind origin/main, as `kit-resolve.sh` does, instead of refusing it as stale (exit 10). It names its kit through `BUILD_N4_KIT`, which the resolver treated as another clone; it now recognizes the kit-main kit by path. A dirty, ahead, diverged or other clone is still refused (#1122).
