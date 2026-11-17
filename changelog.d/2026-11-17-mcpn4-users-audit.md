### Added — mcp-n4: `n4_users_audit`, the station's user accounts without a credential (station-audits S14)

- `n4_users_audit(unused_days=90, max_users=500)` lists, read-only, every `baja:User` under
  `Services/UserService` with its name, enabled, lock-out and expiration state, roles (resolved against
  `Services/RoleService`, `admin_role` when a role is `baja:AdminRole` or grants `super`), auth scheme and
  `last_login` (the newest `Login` record of `history:/<station>/SecurityHistory`), and flags `disabled`,
  `expired`, `locked`, `no_recorded_login`, `unused`, `admin`, `unknown_role`, plus `unused`, `counts`,
  `masked_slots` and `notes`. Fail closed: only allowlisted slots are returned, `email` and `cellPhoneNumber`
  show only `<set>` or `""`, and the authenticator, password hash, permissions and any other slot are only
  named in `masked_slots`. A verdict the evidence cannot support is null, never guessed.
- An allowlisted user slot typed `baja:Password` or holding a non-scalar value is masked instead of echoed.
- `no_recorded_login` (was `never_logged_in`) and `unused` no longer trust a rolled SecurityHistory: they claim only what the records read cover, and `logins_since` names that window.
