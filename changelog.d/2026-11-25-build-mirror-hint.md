### Changed

- build-n4-module: when `build.sh` stops because `niagara_home` is missing or names a real installation, it lists the mirrors it finds under `~/niagara-mirror-*` (those made from the given installation first) and a `try:` command line naming the first; with none and no `niagara_home` it says how to make one. The build still stops: it never picks a mirror itself.
