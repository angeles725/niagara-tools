#!/usr/bin/env bash
# shellcheck shell=bash
# lib/ci-local-kitlinks.sh — the kit-links step of `scripts/ci-local.sh --changed [--range <from>..<to>]`. Sourced
# only, by ci-local.sh, after lib/ci-local-tools.sh, with ROOT, WORK, RANGE, NAMES and BODIES set. It runs only when
# kit_docs_changed finds a changed (or deleted) kit *.md, ledger.d or changelog.d file; any other change is unchanged.
#
# #706 failed CI on tests/kit-links.bats L1 ("dangling reference: prompt.md"): a ledger.d fragment named a bare
# prompt.md, the fragment is copied into the generated BUILD-STATE.md, and L1 scans the kit's *.md. The pre-push
# replay ran no kit-links, so the writer saw it first in CI. (retro 2026-11-15 coordination-stretch T25)
# run_kit_links runs tests/kit-links.bats on a snapshot of the change, never on the checkout itself:
#   - the snapshot holds every non-ignored file but go/ (a link to $ROOT/go): with --range the bytes of the range's
#     end commit (what a push sends), without it the work tree, untracked files included (as ci-steps changed);
#   - toolbelt/kit-state.sh --write materializes BUILD-STATE.md, retros/INDEX.md and toolbelt/INDEX.md in the
#     snapshot's kit, from its own ledger.d/ and BUILD-STATE.src.md (a fragment kit-state refuses fails here, as
#     CI's gen-build-state --check fails it);
#   - the snapshot's own toolbelt/kit-state.sh does it, the launcher the push sends and kit-links.bats itself runs
#     (its go/ is the checkout's either way);
#   - bats is the version ci.yml installs (provision_bats), as the full replay runs it; a provisioning that fails or
#     leaves another version first on PATH is one stderr line and exit 3, never a run on the wrong bats.
# changelog.d fragments reach no file L1 scans (CHANGELOG.md takes them only at release); a change to one still
# runs kit-links, so a fragment cannot be the one change a push leaves unchecked.
# Returns 0 pass or nothing to check, 1 kit-state or kit-links failed, 3 the snapshot could not be made, bats could
# not be provisioned, or git failed while detecting the change. (retro 2026-11-15 coordination-stretch T32)

# kit_docs_changed — 0 when the change touches (deletions too) a kit *.md, a ledger.d or a changelog.d file, 1 when
# it does not, 2 with one stderr line when git fails (a failed detection is never read as "no kit change").
kit_docs_changed() {
  local s=(':(glob)build-n4-module-kit/**/*.md' changelog.d) b u rc=0
  if [ -n "$RANGE" ]; then
    git -C "$ROOT" diff --quiet --no-renames "${RANGE%%..*}" "${RANGE#*..}" -- "${s[@]}" || rc=$?
  else
    b="$(git -C "$ROOT" merge-base origin/main HEAD)" || rc=128
    [ "$rc" -ne 0 ] || git -C "$ROOT" diff --quiet "$b" -- "${s[@]}" || rc=$?
    if [ "$rc" -eq 0 ]; then
      u="$(git -C "$ROOT" ls-files --others --exclude-standard -- "${s[@]}")" || rc=128
      [ -z "$u" ] || rc=1
    fi
  fi
  case "$rc" in 0) return 1 ;; 1) return 0 ;; esac
  printf 'ci-local: --changed: kit-links: cannot tell whether the change touches a kit doc (git exit %d)\n' "$rc" >&2
  return 2
}

# kit_links_snapshot <dir> — write the change's non-ignored files (go/ left out) under <dir>.
kit_links_snapshot() {
  local dir="$1" ex=':(exclude)go'
  mkdir -p "$dir" || return 3
  if [ -n "$RANGE" ]; then
    git -C "$ROOT" archive "${RANGE#*..}" -- . "$ex" | tar -x -C "$dir" || return 3
  else
    # --cached lists a file deleted in the work tree too; tar reads only the ones that still exist.
    git -C "$ROOT" ls-files -z --cached --others --exclude-standard -- . "$ex" \
      | (cd "$ROOT" && while IFS= read -r -d '' f; do [ ! -e "$f" ] || printf '%s\0' "$f"; done) \
      | tar -C "$ROOT" --null -T - -cf - | tar -x -C "$dir" || return 3
  fi
  ln -s "$ROOT/go" "$dir/go" || return 3
  [ -f "$dir/tests/kit-links.bats" ] || { printf 'ci-local: --changed: no tests/kit-links.bats in the change\n' >&2; return 3; }
}

# kit_links_check <name> <dir> <cmd...> — run one check in <dir> and print PASS/FAIL as changed_check does.
kit_links_check() {
  local name="$1" dir="$2" rc=0
  shift 2
  printf '== %s\n' "$name"
  (cd "$dir" && "$@") || rc=$?
  if [ "$rc" -eq 0 ]; then printf 'PASS %s\n' "$name"; return 0; fi
  printf 'FAIL %s (exit %d)\n' "$name" "$rc"
  return 1
}

# kit_state_write <kit-root> — materialize the generated kit state there with that kit's own launcher; the paths it
# prints are noise here.
kit_state_write() { bash "$1/toolbelt/kit-state.sh" --write "$1" >/dev/null; }

# kit_links_bats — provision the bats ci.yml pins and confirm it is the one first on PATH (provision_bats ends on a
# printf, so a failed tar inside it would otherwise leave the bats already on PATH to run).
kit_links_bats() {
  local body ver
  body="$(bats_install_body)"
  provision_bats "$body" || return 1
  ver="$(sed -nE 's/^[[:space:]]*BATS_VER=([0-9.]+)[[:space:]]*$/\1/p' <<< "$body" | head -n 1)"
  [ -z "$ver" ] || [ "$(bats --version 2>/dev/null)" = "Bats $ver" ]
}

run_kit_links() {
  local snap="$WORK/kit-links" rc=0
  kit_docs_changed || rc=$?
  case "$rc" in 0) ;; 1) return 0 ;; *) return 3 ;; esac
  kit_links_snapshot "$snap" || { printf 'ci-local: --changed: cannot snapshot the change for kit-links\n' >&2; return 3; }
  kit_links_bats || { printf 'ci-local: --changed: kit-links: bats could not be provisioned at the version ci.yml pins\n' >&2; return 3; }
  kit_links_check "kit-state --write (the generated BUILD-STATE.md kit-links scans)" "$snap" \
    kit_state_write "$snap/build-n4-module-kit" || return 1
  kit_links_check "kit-links (tests/kit-links.bats over the changed kit docs)" "$snap" bats tests/kit-links.bats || return 1
}
