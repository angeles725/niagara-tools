# Dashboard modules ship their own login (issue #847)

Implements GitHub issue #847 (retro-delta) in the build-n4-module kit. The live reference is a private client module,
proven on a 4.14 lab station on 2026-11-23. No client name may appear in this repository.

## Specs

- S1 (doctrine) — verbatim from #847: "**Doctrine** in `types/dashboard.md` (and a pointer from SKILL.md / BUILD-LOOP.md): a dashboard module carries a `BLoginTemplate` subclass in its `-rt` part (not a separate module), selected through `Services/WebService.loginTemplate`."
- S2 (template behaviour) — verbatim: "every rendering points the session's origin URI at the dashboard, so a successful sign-in lands on it; assets (CSS, JS, images, fonts) are served from the module jar through `resourceToOrd` (no remote URLs); screens it cannot draw itself (two-factor, a non-digest scheme, a network user's expiry notice) fall back to Niagara's own template; the SCRAM/digest sign-in flow and the forced password change / expiry screens keep working."
- S3 (activation rule) — verbatim: "**Activation rule** at station start (service `stationStarted`): set `loginTemplate` only when it is empty, the stock one, or a retired type of the same module; never overwrite another custom template; a boolean slot (default true) turns this off; the station is saved after the change. Document the rollback (flag off + clear `loginTemplate`) and that removing the module with the slot still set gives HTTP 500 on `/login`."
- S4 (scaffold) — verbatim: "**Scaffold**: the dashboard scaffold generates the template skeleton, the asset prefix, the activation rule with its pure tests, and a source-order test."
- S5 (gate) — verbatim: "**Gate**: a lint for `@NiagaraType` classes that declare a static `INSTANCE` after `TYPE` (the type then fails to load on a station: \"INSTANCE field is null; insure getType() comes last\"; pure tests, build, verify and report all pass while the station refuses the type)."
- S6 (acceptance) — verbatim: "`types/dashboard.md` documents the rule; the scaffold emits a working login for a new dashboard module; the INSTANCE-before-TYPE lint has a fixture and a golden." The live lab check ("the template is selected automatically on a station with no custom login, kept on a station that has one, and sign-in lands on the dashboard") was done on the reference module on 2026-11-23 (user sign-in confirmed); the scaffold output is checked by build/pure tests only.

## Tasks

| ID | Specs | Route | Commit |
|---|---|---|---|
| T1 | S5, S6 (lint part) | worker, Go lint rule + fixture + golden | feat(lint) lint-instance-type-order |
| T2 | S1-S4, S6 (doctrine + scaffold part) | worker, kit docs + scaffold-module.sh templates | docs(kit) + feat(scaffold) dashboard login |

## Log

- L1 (2026-11-23, operator, verbatim): "Una cosa, para que se quede como issue, los modulos deben de tener su propio login dirigidos al dashboard, en caso de que sea un modulo para un dashboard"
- L2 (2026-11-23, operator, verbatim): "Trabaja con las issues, por favor"
- L3 (2026-11-23): live root cause behind S5 — the reference module declared `INSTANCE` after `TYPE` in its `BLoginTemplate` subclass; the station logged `TypeIntrospectionException: INSTANCE field is null; insure getType() comes last`; moving `INSTANCE` above `TYPE` fixed it (reference module 2.5.1).
- L4 (2026-11-23): T1 and T2 done; the scaffold commit is ~1500 lines, mostly generated template files (one reviewable unit: the overlay and its tests). The scaffold output was not compiled against a Niagara install; the source-order test and the lint catch the INSTANCE/TYPE trap.
