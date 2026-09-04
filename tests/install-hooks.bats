#!/usr/bin/env bats
# RED-first tests for scripts/install-hooks.sh — the opt-in installer that ACTIVATES the
# retro-enforcement pre-push gate by pointing git at the kit's hooks: `git config core.hooksPath
# .githooks` (Campaign 5, AG-PR1). The gate ships inert; this is what makes it bite in a live repo.
#
# The script does not exist yet: each test SKIPS until it lands (this suite's red-first convention),
# then must pass. git is REAL here (a throwaway repo), which is allowed — install-hooks.sh lives
# under scripts/, not toolbelt/, so the kit-links L2 rule ("no toolbelt script invokes git") does
# not apply to it.
#
# Contract these tests define:
#   * run from a repo root  -> sets --local core.hooksPath = .githooks, exit 0
#   * hooksPath unset OR already .githooks -> set/keep .githooks, exit 0 (idempotent)
#   * hooksPath set to some OTHER (custom) value -> REFUSE: leave it untouched, exit non-zero
#     (fail loud so a user's custom hooks are never silently clobbered; --force is the override)
#   * --uninstall -> unset the core.hooksPath we set (restore the default), exit 0

setup() {
  REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  INSTALLER="$REPO/scripts/install-hooks.sh"
  # a throwaway git repo with a .githooks/ dir for the installer to point at
  GR="$BATS_TEST_TMPDIR/repo"
  mkdir -p "$GR/.githooks"
  printf '#!/usr/bin/env bash\nexit 0\n' > "$GR/.githooks/pre-push"
  chmod +x "$GR/.githooks/pre-push"
  git -C "$GR" init -q
  git -C "$GR" config user.email t@e.st
  git -C "$GR" config user.name  test
}

hookspath() { git -C "$GR" config --local --get core.hooksPath 2>/dev/null || true; }

@test "I1: install sets core.hooksPath to .githooks (activates the gate)" {
  [ -f "$INSTALLER" ] || skip "scripts/install-hooks.sh not implemented yet (red-first)"
  ( cd "$GR" && bash "$INSTALLER" )
  [ "$(hookspath)" = ".githooks" ]
}

@test "I2: re-running install is idempotent — no error, still .githooks" {
  [ -f "$INSTALLER" ] || skip "scripts/install-hooks.sh not implemented yet (red-first)"
  ( cd "$GR" && bash "$INSTALLER" )
  run bash -c "cd '$GR' && bash '$INSTALLER'"
  [ "$status" -eq 0 ]
  [ "$(hookspath)" = ".githooks" ]
}

@test "I3: --uninstall restores the default (unsets the core.hooksPath we set)" {
  [ -f "$INSTALLER" ] || skip "scripts/install-hooks.sh not implemented yet (red-first)"
  ( cd "$GR" && bash "$INSTALLER" )
  [ "$(hookspath)" = ".githooks" ]
  ( cd "$GR" && bash "$INSTALLER" --uninstall )
  [ -z "$(hookspath)" ]
}

@test "I4: install REFUSES to clobber a pre-existing CUSTOM core.hooksPath (guards it, exits non-zero)" {
  [ -f "$INSTALLER" ] || skip "scripts/install-hooks.sh not implemented yet (red-first)"
  git -C "$GR" config --local core.hooksPath .customhooks
  run bash -c "cd '$GR' && bash '$INSTALLER'"
  [ "$status" -ne 0 ]                     # must NOT silently overwrite a user's hooks
  [ "$(hookspath)" = ".customhooks" ]     # the custom path survives untouched
}
