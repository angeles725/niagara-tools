### Changed — merge-when-green exit 10 for a merge call lost on the network (retro row T33b of the 2026-11-15 coordination stretch)

- `scripts/merge-when-green.sh` exits 10, not 8, when every send of `gh pr merge` failed on a network error and the
  PR is still OPEN: GitHub refused nothing and nothing merged, so run it again. Exit 8 stays GitHub's refusal; every
  other exit is unchanged.
