### Changed

- `types/dashboard.md`: a lock indicator on a control row never disables its HOA buttons when HOA is a release path, with a structural test (issue #870); `ORCHESTRATION.md` § 3.b item 6 points to it.
- `types/dashboard.md`: an HMI change to an operator flow lists every surface that renders it and opens the parity item before the build (issue #871).
- `types/dashboard.md`: the reader emits an unlinked numeric slot as `null`, never `0.0` with status ok, so averages skip it (issue #880).
- `types/dashboard.md`: a device placeholder for "no reading" is shown as "Sin sensor", never as a measurement (issue #897).
- `types/dashboard.md` § Write path: a multi-part write rehearses on `newCopy()` before touching live components (issue #901).
- `types/dashboard.md` § Write path: a timed override never becomes permanent by accident when its schedule is empty (issue #910).
- `types/dashboard.md`: a label derives from the station (`displayName`, then the owning device's name), the static catalog is the default only (issue #922).
- `types/dashboard.md`: after a three.js major bump, compare a before/after render and recalibrate `envScale` (issue #945).
