### Fixed — path guards of orient-guard and mirror-niagara-home

- `orient-guard.sh` resolves the longest existing ancestor of the module root and judges the rest by its text,
  so a new directory several levels deep inside the prefix passes, spelled absolute or relative. The FAIL row
  names the prefix that was compared (the seam one when set), every final slash of `HOME` is dropped, and an
  argument starting with `-` is a usage error (exit 3; spell such a path `./-name`).
- `mirror-niagara-home.sh` runs guard A on a mirror dir that is not there yet before it creates anything, so a
  refused mirror dir leaves no directory behind; one that cannot be resolved is refused (exit 20). A mirror dir
  that passes is made where guard A judged it, so `<src>/new/../../out` no longer creates `<src>/new`, and
  guard B judges that same path, so `missing/../dir` no longer writes into a non-empty `dir` without the marker. A symlinked
  mirror dir is refused (exit 20) whatever its spelling: `link`, `link/` and `link/.` no longer replace the link
  or empty its target.
