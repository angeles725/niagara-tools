### Fixed

- `templates/dashboard-login` (#1119): the "never throws at station start" promise now holds for an `Error`. The
  activator's first use of the template type runs its class initializer, and a type that fails to load (`INSTANCE`
  after `TYPE`, a missing `web-rt` class) throws `ExceptionInInitializerError` or `NoClassDefFoundError`, which
  `catch (Exception)` let escape `stationStarted()`. The new pure `LoginActivation.guarded` catches `Throwable` in
  `LoginActivator.apply` and in the service's save scheduling, logs it as a WARNING and leaves the login as it was;
  only a `VirtualMachineError` is rethrown. `LoginActivationTest` pins it with an injected `Error`, a type whose
  initializer fails (first and second use), an `Exception`, a normal answer and the rethrow.
- The known-non-digest-user oracle (an LDAP user gets Niagara's page, an unknown user the custom one) is documented
  as an accepted limitation in `templates/dashboard-login/README.md` and `types/dashboard.md`: the custom page runs
  SCRAM and never posts the password, so showing it to a non-digest user would break that user's sign-in.
