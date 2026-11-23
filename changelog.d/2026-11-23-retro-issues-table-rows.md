### Fixed — retro-issues sees each row of a table-style retro issue (issue #946)

- `retro-issues` scored an existing issue on its title and only the first 800 characters of its body. A delta that
  matched a row further down a table-style issue (one row per delta, no markers) got a new issue of its own. Each table
  row is now also scored together with the issue title, so such a delta is reported as a probable duplicate of the
  table issue. Unrelated rows do not raise the score.
