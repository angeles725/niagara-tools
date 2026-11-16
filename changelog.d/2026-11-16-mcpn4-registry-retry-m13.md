### Fixed — mcp-n4: a failed type-registry call is retried once (retro row M13)

- Right after a station starts or a module first loads, one type-registry request (`reg` / `loadTypes`) could get
  no answer, and the first `n4_set_slot` of a slot still at its declared default failed with it. `box.Client.LoadTypes`
  now asks again once, after 0.5 s, when a request gets no answer (an HTTP error status, a transport failure or a
  malformed reply), so every caller inherits it: `n4_set_slot` and the other declared-default writes, the create,
  link, alarm, history and flags prechecks, and the `n4_read_slots` / `n4_navigate` listings.
- An answer is never retried: the station's own error reply, an `isErr` entry for a type the station does not know
  (still refused as "not known to the station" with the stale-registry route) and an AuthError.
- Two requests without an answer refuse with "the station's type registry did not answer (it may have just
  started): retry", naming the last failure.
