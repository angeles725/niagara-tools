# templates/dashboard-login

The overlay `scaffold-module.sh --type dashboard` adds to the `-rt` part: a `BLoginTemplate` subclass, the pure
classes behind it (`login/model/`), the service that applies it at station start and the page assets.
`files/` is copied into the module; `fragments/` is merged into its plumbing files. This README is not emitted.
The doctrine is in `types/dashboard.md` § The module ships its own login.

## Start never fails

`BModuleAService.stationStarted()` calls `LoginActivator.apply` and the station-save scheduling through
`LoginActivation.guarded`. Any `Throwable` (an `Error` from a template type that fails to load, such as
`ExceptionInInitializerError` or `NoClassDefFoundError`) is logged as a WARNING and the station keeps the login it
had. Only a `VirtualMachineError` is rethrown. `LoginActivationTest` pins it with an injected `Error`.
Request handling (`write`, `LoginRequest`) is not part of the start path and is unchanged.

## Accepted limitation: a known non-digest user is visible

An unknown user name gets the custom password page, so a guess does not show which names exist. A known user on a
scheme that is not digest (LDAP and similar) gets Niagara's own page (`LoginView.Screen.STOCK`), so an anonymous
client can tell that such an account exists.

It is not fixed on purpose. The custom page runs SCRAM (`login.js`, `wireLogin`) and then goes to
`/j_security_check/` without posting the password, which only a digest scheme answers; shown to an LDAP user it
would make their sign-in fail. The server's scheme has to stay the one that authenticates. If the exposure matters
on a station, set `applyLogin` false and clear `Services/WebService.loginTemplate`.
