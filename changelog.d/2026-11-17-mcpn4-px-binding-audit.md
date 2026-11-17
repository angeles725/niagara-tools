### Added — mcp-n4: `n4_px_binding_audit`, the Px bindings whose ORD does not resolve (station-audits S15)

- `n4_px_binding_audit(root, file_ord, base, max_views=50, max_bindings=500)` audits, read-only, every Px view
  below `root` (relative ORDs read against the component the view is on) or one `file_ord` with an optional
  `base`. Each distinct target is loaded once: `resolves` when it loads, `missing` (listed under `broken`) only
  when the deepest ancestor that loads holds no slot by that name, loaded or declared; everything else
  (another scheme, a chained ORD, a relative ORD with no base, a handle, a failed load whose parent still holds
  the name) is `unknown` with a `why` and never counted as broken. `n4_px_summary` is unchanged.
