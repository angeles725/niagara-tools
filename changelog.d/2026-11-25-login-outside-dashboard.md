### Added

- `toolbelt/lint-structure.sh` (`go/internal/lint/rules/structure`): new code L18 FAILs `login outside the dashboard
  module`. Issue #1120.
  - A class that extends `BLoginTemplate` must stand in a dashboard `-rt`: the `-rt` part that holds a `BWebServlet`
    itself, or whose `<module>-ux` sibling does. A separate `<X>Login` module (in its own root or beside the
    dashboard), or a login in a `-ux`/`-wb` part, is the retired design; the row names the class line and points to
    `templates/dashboard-login` and `types/dashboard.md` § The module ships its own login.
  - Not flagged: a dashboard on Niagara's stock login (a servlet and no template), which `types/dashboard.md` supports
    (`applyLogin` false, an empty `loginTemplate` slot); a servlet and a login in one `-rt`; a template named only in
    a comment.
  - `BUILD-LOOP.md` gains a "porting from a sibling module" checklist: run the lints
    on the source at the revision being copied, check that the sibling has not retired the design, prefer the kit
    template over a copy.
  - `types/structure.md` has the L18 row; `types/dashboard.md` names the rule.
