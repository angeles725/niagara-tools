#!/usr/bin/env bash
# stub-toolbelt.bash — shared bats helper: a toolbelt copy with one member replaced by a stub.
# Loaded with `load helpers/stub-toolbelt` from tests/report-module.bats and tests/commissioning-verify.bats.

# stub_toolbelt <dir> <member> <stub-body> — symlink every build-n4-module-kit/toolbelt entry into
# <dir>, then replace <member> (e.g. lint-link-target-flags.sh) with a bash script whose body is
# <stub-body>. An orchestrator invoked as <dir>/<script> resolves TOOLBELT to <dir> (the directory of
# the invoked path), so it runs the stub with no env override in the production script.
stub_toolbelt() {
  local dir="$1" member="$2" body="$3" f
  mkdir -p "$dir"
  for f in "$KIT/toolbelt"/*; do ln -s "$f" "$dir/"; done
  rm "$dir/$member"
  printf '#!/usr/bin/env bash\n%s\n' "$body" > "$dir/$member"
  chmod +x "$dir/$member"
}
