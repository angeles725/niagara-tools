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
#   - bats is the version ci.yml installs (provision_bats), as the full replay runs it.
# changelog.d fragments reach no file L1 scans (CHANGELOG.md takes them only at release); a change to one still
# runs kit-links, so a fragment cannot be the one change a push leaves unchecked.
# Returns 0 pass or nothing to check, 1 kit-state or kit-links failed, 3 the snapshot could not be made.

# kit_docs_changed — 0 when the change touches (deletions too) a kit *.md, a ledger.d or a changelog.d file; a git
# error also reads as changed, so kit-links runs rather than being skipped.
kit_docs_changed() {
  local s=(':(glob)build-n4-module-kit/**/*.md' changelog.d) b
  if [ -n "$RANGE" ]; then ! git -C "$ROOT" diff --quiet --no-renames "${RANGE%%..*}" "${RANGE#*..}" -- "${s[@]}"; return; fi
  b="$(git -C "$ROOT" merge-base origin/main HEAD)" && git -C "$ROOT" diff --quiet "$b" -- "${s[@]}" || return 0
  [ -n "$(git -C "$ROOT" ls-files --others --exclude-standard -- "${s[@]}")" ]
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

# kit_state_write <kit-root> — materialize the generated kit state there; the paths it prints are noise here.
kit_state_write() { bash "$ROOT/build-n4-module-kit/toolbelt/kit-state.sh" --write "$1" >/dev/null; }

run_kit_links() {
  local snap="$WORK/kit-links"
  kit_docs_changed || return 0
  kit_links_snapshot "$snap" || { printf 'ci-local: --changed: cannot snapshot the change for kit-links\n' >&2; return 3; }
  provision_bats "$(bats_install_body)"
  kit_links_check "kit-state --write (the generated BUILD-STATE.md kit-links scans)" "$snap" \
    kit_state_write "$snap/build-n4-module-kit" || return 1
  kit_links_check "kit-links (tests/kit-links.bats over the changed kit docs)" "$snap" bats tests/kit-links.bats || return 1
}
