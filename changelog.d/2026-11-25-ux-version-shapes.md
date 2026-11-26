### Changed

- `toolbelt/lint-ux-version.sh` (`go/internal/lint/rules/uxversion`): three shapes of a dashboard that embeds a
  separately built app no longer give a false positive, and one forgotten bump is now seen. Issue #1117.
  - A content-hashed name (`-[0-9a-f]{8,}.(js|css|mjs)`, e.g. `app3d/dist/main-18a30d6b12.js`) gets no row: the
    hash is a stronger cache key than `?v=`.
  - A sub-tree with its own version is declared in a scanned file with `<!-- ux-version: app3d/ = 1.1 -->`. A
    reference whose path starts with `app3d/` (or holds `/app3d/`) is judged against 1.1, not
    `defaultModuleVersion`; one that differs is a new FAIL `subtree-version`, so the iframe and its preloads must
    agree. Two declarations of one directory with different versions are a FAIL too. An undeclared directory with
    another version is still `stale-version`.
  - A `?v=` built in JavaScript (`"view3d.html?v=" + NAME`, also `+ encodeURIComponent(NAME)`, in a `.js` file or an
    inline `<script>`) is read: `NAME` resolved from a `var|let|const NAME = "<ver>"` in the same file is judged like
    a literal `?v=`; otherwise a new WARN `dynamic-version: ... dynamic version, check by hand` (FAIL under
    `--strict`; waivable with `lint-ux-version: allow dynamic-version <reason>`).
  - Changed behaviour: `<iframe src>` is now a read reference (before, an iframe was not read), so a local iframe
    with no `?v=` is a `no-version` WARN. A `"x.js?v=" + NAME` that was silent before is a `dynamic-version` WARN
    unless the file declares `NAME`.
  - Goldens `hashed-names`, `subtree-*`, `js-const-*`, `js-dynamic`, `iframe-ref` pin them; `types/dashboard.md` and
    `types/frontend-standard.md` document the declaration.
