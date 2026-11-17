### Fixed — kit-cite-resolve no longer reads the text between two code spans as a repo path

- In live prose, a backtick that closes one code span no longer opens a `repo-path` pointer: in
  `` `--src`go/x.sh`--y` `` the `go/x.sh` between the spans is prose, not a cited path.
